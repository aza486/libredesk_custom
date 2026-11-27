// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, ref, computed, nextTick } from 'vue'
import { useDraftManager } from './useDraftManager'
import { useFormRecipient } from './useFormRecipient'
const mocks = vi.hoisted(() => ({ store: null, api: { saveDraft: vi.fn().mockResolvedValue({}), deleteDraft: vi.fn().mockResolvedValue({}) } }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/api', () => ({ default: mocks.api }))
let app, host
const flush = async () => { for (let i = 0; i < 10; i++) { await Promise.resolve(); await nextTick() } }
afterEach(() => { app?.unmount(); host?.remove(); vi.useRealTimers(); vi.clearAllMocks() })
function mount(drafts = [], ready = Promise.resolve()) {
  vi.useFakeTimers()
  const map = new Map(drafts)
  mocks.store = { macros: {}, draftsReady: ready, getMacro: () => ({}), getDraft: (u, t) => map.get(`${u}:${t}`),
    setDraft: (u, t, d) => map.set(`${u}:${t}`, d), removeDraft: (u, t) => map.delete(`${u}:${t}`) }
  const conversation = ref({ uuid: 'a', reply_recipient: { managed: true, email: 'form@example.com' } })
  const mode = ref('reply'), fields = [ref(''), ref(''), ref('')]
  let manager, form
  app = createApp(defineComponent({ setup() {
    manager = useDraftManager(computed(() => conversation.value.uuid), mode)
    form = useFormRecipient(conversation, fields, vi.fn(), { draft: manager.replyRecipients })
    return () => h('div')
  } }))
  host = document.createElement('div'); document.body.append(host); app.mount(host)
  return { map, conversation, mode, fields, manager, form }
}
test('recipient-only drafts persist deliberate clearing, survive mode/conversation changes and reload', async () => {
  const f = mount(); await flush()
  f.fields[0].value = ''
  f.fields[1].value = 'cc@example.com'
  await flush(); await vi.advanceTimersByTimeAsync(600); await flush()
  expect(f.map.get('a:reply').meta.reply_recipients).toMatchObject({ to: '', cc: 'cc@example.com', edited: { to: true, cc: true } })
  expect(mocks.api.saveDraft).toHaveBeenCalled()
  f.mode.value = 'private_note'; await flush()
  expect(f.manager.replyRecipients.value).toBeNull()
  f.mode.value = 'reply'; await flush()
  expect(f.fields.map(r => r.value)).toEqual(['', 'cc@example.com', ''])
  f.conversation.value = { uuid: 'b', reply_recipient: { managed: true, email: 'b@example.com' } }; await flush()
  expect(f.fields[0].value).toBe('b@example.com')
  f.conversation.value = { uuid: 'a', reply_recipient: { managed: true, email: 'new@example.com' } }; await flush()
  expect(f.fields.map(r => r.value)).toEqual(['', 'cc@example.com', ''])
  const saved = f.map.get('a:reply')
  app.unmount(); host.remove()
  const next = mount([['a:reply', saved]]); await flush()
  expect(next.fields.map(r => r.value)).toEqual(['', 'cc@example.com', ''])
})
test('legacy draft recipient arrays including an empty TO override the form suggestion', async () => {
  const f = mount([['a:reply', { content: 'Draft', meta: { to: [], cc: ['saved@example.com'], bcc: ['hidden@example.com'] } }]])
  await flush()
  expect(f.fields.map(r => r.value)).toEqual(['', 'saved@example.com', 'hidden@example.com'])
})
test('a manual decision made before draft prefetch finishes wins over saved recipients', async () => {
  let resolve
  const ready = new Promise(r => { resolve = r })
  const f = mount([['a:reply', { content: 'Draft', meta: { reply_recipients: { to: 'old@example.com', cc: '', bcc: '', edited: { to: true } } } }]], ready)
  f.fields[0].value = 'manual@example.com'
  resolve(); await flush()
  expect(f.fields[0].value).toBe('manual@example.com')
  expect(f.manager.replyRecipients.value.to).toBe('manual@example.com')
})
