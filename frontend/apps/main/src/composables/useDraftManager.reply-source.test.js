// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, ref, nextTick } from 'vue'
import { useDraftManager } from './useDraftManager'
const mocks = vi.hoisted(() => ({
  store: null,
  api: { saveDraft: vi.fn().mockResolvedValue({}), deleteDraft: vi.fn().mockResolvedValue({}) }
}))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/api', () => ({ default: mocks.api }))
let app, host
const flush = async () => {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.useRealTimers()
  vi.clearAllMocks()
})
test('server draft source survives editing and switching; old drafts remain usable', async () => {
  vi.useFakeTimers()
  const map = new Map([
    ['a:reply', { content: 'Existing draft', meta: { reply_to_message_uuid: 'customer-a' } }],
    ['b:reply', { content: 'Old draft without metadata', meta: {} }]
  ])
  mocks.store = {
    macros: {},
    draftsReady: Promise.resolve(),
    getMacro: () => ({}),
    getDraft: (u, t) => map.get(`${u}:${t}`),
    setDraft: (u, t, d) => map.set(`${u}:${t}`, d),
    removeDraft: (u, t) => map.delete(`${u}:${t}`)
  }
  const uuid = ref('a')
  let manager
  app = createApp(
    defineComponent({
      setup() {
        manager = useDraftManager(uuid, ref('reply'))
        return () => h('div')
      }
    })
  )
  host = document.createElement('div')
  document.body.append(host)
  app.mount(host)
  await flush()
  expect(manager.replyToMessageUUID.value).toBe('customer-a')
  manager.htmlContent.value = 'Edited draft'
  await flush()
  await vi.advanceTimersByTimeAsync(600)
  await flush()
  expect(mocks.api.saveDraft).toHaveBeenCalledWith('a', 'reply', {
    content: 'Edited draft',
    meta: { reply_to_message_uuid: 'customer-a' }
  })
  uuid.value = 'b'
  await flush()
  expect(manager.replyToMessageUUID.value).toBe('')
  expect(manager.htmlContent.value).toBe('Old draft without metadata')
  uuid.value = 'a'
  await flush()
  expect(manager.replyToMessageUUID.value).toBe('customer-a')
  expect(manager.aiResponse.value).toBeNull()
  manager.clearDraft()
  await flush()
  expect(manager.replyToMessageUUID.value).toBe('')
})
