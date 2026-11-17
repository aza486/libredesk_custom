import { onBeforeUnmount, ref } from 'vue'
import api from '@main/api'

const MAX_BYTES = 8 * 1024 * 1024
const MAX_MS = 120000
const MIME_TYPES = ['audio/webm;codecs=opus', 'audio/ogg;codecs=opus', 'audio/mp4']
let activeCancel

// One microphone owner, including pending permissions. Each operation owns its
// callbacks; cancellation invalidates late permission, stop and HTTP responses.
export function useVoiceInput(onTranscript) {
  const state = ref('ready')
  const error = ref('')
  let revision = 0,
    recorder,
    stream,
    controller,
    timer
  function release() {
    clearTimeout(timer)
    stream?.getTracks().forEach((track) => track.stop())
    stream = undefined
    if (recorder) {
      recorder.ondataavailable = recorder.onstop = recorder.onerror = null
      if (recorder.state !== 'inactive') recorder.stop()
      recorder = undefined
    }
  }
  function cancel() {
    revision++
    controller?.abort()
    release()
    state.value = 'ready'
    error.value = ''
    if (activeCancel === cancel) activeCancel = undefined
  }
  function fail(code) {
    cancel()
    error.value = code
    state.value = 'error'
  }
  async function start() {
    if (['requesting', 'recording', 'processing'].includes(state.value)) return
    activeCancel?.()
    activeCancel = cancel
    const run = ++revision
    error.value = ''
    state.value = 'requesting'
    controller = new AbortController()
    timer = setTimeout(() => fail('timeout'), 15000)
    try {
      if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') {
        fail('unsupported')
        return
      }
      const mimeType = MIME_TYPES.find((type) => MediaRecorder.isTypeSupported(type))
      if (!mimeType) {
        fail('unsupported')
        return
      }
      const config = await api.voiceInputConfig(controller.signal)
      if (run !== revision) return
      if (!config.data.data?.enabled) {
        fail('unavailable')
        return
      }
      const acquired = await navigator.mediaDevices.getUserMedia({ audio: true })
      if (run !== revision) {
        acquired.getTracks().forEach((track) => track.stop())
        return
      }
      clearTimeout(timer)
      stream = acquired
      recorder = new MediaRecorder(stream, { mimeType, audioBitsPerSecond: 64000 })
      const chunks = []
      let bytes = 0,
        completed = false
      recorder.ondataavailable = ({ data }) => {
        if (run !== revision || !data?.size) return
        bytes += data.size
        if (bytes > MAX_BYTES) {
          fail('tooLarge')
          return
        }
        chunks.push(data)
      }
      recorder.onerror = () => {
        if (run === revision) fail('recordingFailed')
      }
      recorder.onstop = async () => {
        if (run !== revision || completed) return
        completed = true
        clearTimeout(timer)
        stream?.getTracks().forEach((track) => track.stop())
        stream = undefined
        state.value = 'processing'
        timer = setTimeout(() => {
          if (run === revision) fail('timeout')
        }, 50000)
        try {
          const blob = new Blob(chunks, { type: mimeType })
          chunks.length = 0
          if (!blob.size) {
            fail('recordingFailed')
            return
          }
          const response = await api.transcribeAudio(blob, controller.signal)
          if (run !== revision) return
          const text = response?.data?.data?.text
          if (typeof text !== 'string' || !text.trim()) {
            fail('transcriptionFailed')
            return
          }
          onTranscript(text)
          release()
          state.value = 'success'
          if (activeCancel === cancel) activeCancel = undefined
        } catch (e) {
          if (run === revision)
            fail(
              e?.response?.status === 504 || e?.code === 'ECONNABORTED'
                ? 'timeout'
                : e?.response?.status === 503
                  ? 'unavailable'
                  : 'transcriptionFailed'
            )
        }
      }
      recorder.start(1000)
      state.value = 'recording'
      timer = setTimeout(() => {
        if (run === revision) stop()
      }, MAX_MS)
    } catch (e) {
      if (run !== revision) return
      fail(
        e?.code === 'ECONNABORTED'
          ? 'timeout'
          : e?.name === 'NotAllowedError'
            ? 'denied'
            : e?.name === 'NotFoundError'
              ? 'noMicrophone'
              : 'recordingFailed'
      )
    }
  }
  function stop() {
    if (state.value !== 'recording' || recorder?.state === 'inactive') return
    clearTimeout(timer)
    state.value = 'processing'
    recorder.stop()
    stream?.getTracks().forEach((track) => track.stop())
  }
  onBeforeUnmount(cancel)
  return { state, error, start, stop, cancel }
}
