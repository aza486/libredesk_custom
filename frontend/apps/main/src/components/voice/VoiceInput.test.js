// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import VoiceInput from './VoiceInput.vue'
const mocks = vi.hoisted(() => ({ config: vi.fn(), transcribe: vi.fn() }))
vi.mock('@main/api', () => ({
  default: { voiceInputConfig: mocks.config, transcribeAudio: mocks.transcribe }
}))
let app, root
const flush = async () => {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})
test('closing a recording clears parent busy state; reopening remains usable and appends to live text only', async () => {
  let recorder
  const track = { stop: vi.fn() }
  vi.stubGlobal('navigator', {
    mediaDevices: { getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [track] }) }
  })
  vi.stubGlobal(
    'MediaRecorder',
    class {
      state = 'inactive'
      static isTypeSupported() {
        return true
      }
      constructor() {
        recorder = this
      }
      start() {
        this.state = 'recording'
      }
      stop() {
        this.state = 'inactive'
        this.ondataavailable?.({ data: new Blob(['synthetic audio']) })
        this.onstop?.()
      }
    }
  )
  mocks.config.mockResolvedValue({ data: { data: { enabled: true } } })
  let done
  mocks.transcribe.mockReturnValue(new Promise((resolve) => (done = resolve)))
  const props = reactive({ open: true, text: 'Original', busy: false })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      props.open
        ? h(VoiceInput, {
            modelValue: props.text,
            'onUpdate:modelValue': (v) => (props.text = v),
            onBusy: (v) => (props.busy = v)
          })
        : null
  })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await flush()
  root.querySelector('button[aria-label="voiceInput.start"]').click()
  await flush()
  expect(props.busy).toBe(true)
  props.open = false
  await flush()
  expect(track.stop).toHaveBeenCalled()
  expect(props.busy).toBe(false)
  props.open = true
  await flush()
  expect(root.querySelector('button[aria-label="voiceInput.start"]').disabled).toBe(false)
  root.querySelector('button[aria-label="voiceInput.start"]').click()
  await flush()
  root.querySelector('button[aria-label="voiceInput.stop"]').click()
  await flush()
  props.text = 'Manual edits during transcription'
  await flush()
  done({ data: { data: { text: 'Dictated' } } })
  await flush()
  expect(props.text).toBe('Manual edits during transcription Dictated')
  expect(props.busy).toBe(false)
  expect(recorder.onstop).toBeNull()
})
