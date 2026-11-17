// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import { useVoiceInput } from './useVoiceInput'
const mocks = vi.hoisted(() => ({ config: vi.fn(), transcribe: vi.fn() }))
vi.mock('@main/api', () => ({
  default: { voiceInputConfig: mocks.config, transcribeAudio: mocks.transcribe }
}))
let apps = [],
  instances,
  tracks,
  gum
const flush = async () => {
  for (let i = 0; i < 8; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
class Recorder {
  static isTypeSupported = (type) => type.startsWith('audio/ogg')
  state = 'inactive'
  constructor(stream, options) {
    this.options = options
    instances.push(this)
  }
  start() {
    this.state = 'recording'
  }
  stop() {
    this.state = 'inactive'
    this.ondataavailable?.({ data: new Blob(['OggSsynthetic audio']) })
    this.onstop?.()
  }
}
function mount() {
  let voice
  const result = vi.fn()
  const el = document.createElement('div')
  const app = createApp({
    setup() {
      voice = useVoiceInput(result)
      return () => h('div')
    }
  })
  app.mount(el)
  apps.push(app)
  return { voice, result, app }
}
beforeEach(() => {
  vi.useFakeTimers()
  instances = []
  tracks = { stop: vi.fn() }
  gum = vi.fn().mockResolvedValue({ getTracks: () => [tracks] })
  vi.stubGlobal('MediaRecorder', Recorder)
  vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: gum } })
  mocks.config.mockResolvedValue({ data: { data: { enabled: true } } })
  mocks.transcribe.mockResolvedValue({ data: { data: { text: 'Dictated text' } } })
})
afterEach(() => {
  apps.forEach((a) => a.unmount())
  apps = []
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})
test('explicit start, stop and transcript; no duplicate stop results', async () => {
  const { voice, result } = mount()
  await voice.start()
  expect(gum).toHaveBeenCalledWith({ audio: true })
  expect(instances[0].options.mimeType).toBe('audio/ogg;codecs=opus')
  expect(voice.state.value).toBe('recording')
  expect(mocks.transcribe).not.toHaveBeenCalled()
  voice.stop()
  voice.stop()
  await flush()
  expect(result).toHaveBeenCalledExactlyOnceWith('Dictated text')
  expect(tracks.stop).toHaveBeenCalled()
  expect(voice.state.value).toBe('success')
})
test('cancel recording releases tracks without upload or callback', async () => {
  const { voice, result } = mount()
  await voice.start()
  voice.cancel()
  await flush()
  expect(tracks.stop).toHaveBeenCalled()
  expect(result).not.toHaveBeenCalled()
  expect(mocks.transcribe).not.toHaveBeenCalled()
})
test('unmount cancels in-flight transcription and ignores its late result', async () => {
  let done
  mocks.transcribe.mockReturnValue(new Promise((resolve) => (done = resolve)))
  const { voice, result, app } = mount()
  await voice.start()
  voice.stop()
  await flush()
  app.unmount()
  expect(mocks.transcribe.mock.calls[0][1].aborted).toBe(true)
  done({ data: { data: { text: 'late' } } })
  await flush()
  expect(result).not.toHaveBeenCalled()
})
test('late permission after cancellation is released', async () => {
  let done
  gum.mockReturnValue(new Promise((resolve) => (done = resolve)))
  const { voice } = mount()
  const pending = voice.start()
  await flush()
  voice.cancel()
  done({ getTracks: () => [tracks] })
  await pending
  expect(tracks.stop).toHaveBeenCalled()
  expect(instances).toHaveLength(0)
})
test('denied permission and missing microphone remain manual-only', async () => {
  for (const [name, code] of [
    ['NotAllowedError', 'denied'],
    ['NotFoundError', 'noMicrophone']
  ]) {
    gum.mockRejectedValueOnce({ name })
    const { voice, result } = mount()
    await voice.start()
    expect(voice.error.value).toBe(code)
    expect(result).not.toHaveBeenCalled()
  }
})
test('unconfigured service never requests microphone', async () => {
  mocks.config.mockResolvedValue({ data: { data: { enabled: false } } })
  const { voice } = mount()
  await voice.start()
  expect(voice.error.value).toBe('unavailable')
  expect(gum).not.toHaveBeenCalled()
})
test('transcription failure and retry preserve lifecycle', async () => {
  mocks.transcribe.mockRejectedValueOnce(new Error('network'))
  const { voice, result } = mount()
  await voice.start()
  voice.stop()
  await flush()
  expect(voice.error.value).toBe('transcriptionFailed')
  expect(result).not.toHaveBeenCalled()
  await voice.start()
  voice.stop()
  await flush()
  expect(result).toHaveBeenCalledTimes(1)
})
test('only one microphone owner and no double start', async () => {
  const one = mount(),
    two = mount()
  await one.voice.start()
  await one.voice.start()
  await two.voice.start()
  expect(instances).toHaveLength(2)
  expect(one.voice.state.value).toBe('ready')
  expect(two.voice.state.value).toBe('recording')
})
test('two minute limit stops and transcribes exactly once', async () => {
  const { voice, result } = mount()
  await voice.start()
  await vi.advanceTimersByTimeAsync(120000)
  await flush()
  expect(voice.state.value).toBe('success')
  expect(result).toHaveBeenCalledTimes(1)
})
test('unsupported browser and overlarge chunks fail safely', async () => {
  const { voice } = mount()
  vi.stubGlobal('MediaRecorder', undefined)
  await voice.start()
  expect(voice.error.value).toBe('unsupported')
  vi.stubGlobal('MediaRecorder', Recorder)
  await voice.start()
  instances[0].ondataavailable({ data: { size: 8 * 1024 * 1024 + 1 } })
  expect(voice.error.value).toBe('tooLarge')
  expect(mocks.transcribe).not.toHaveBeenCalled()
})

test('provider timeout reports timeout and releases the microphone', async () => {
  mocks.transcribe.mockRejectedValueOnce({ response: { status: 504 } })
  const { voice, result } = mount()
  await voice.start()
  voice.stop()
  await flush()
  expect(voice.error.value).toBe('timeout')
  expect(tracks.stop).toHaveBeenCalled()
  expect(result).not.toHaveBeenCalled()
})
