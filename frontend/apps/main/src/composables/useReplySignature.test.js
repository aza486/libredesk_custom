// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, ref, computed, nextTick } from 'vue'
import { useDraftManager } from './useDraftManager'
import { useReplySignature } from './useReplySignature'

const mocks = vi.hoisted(() => ({ store: null,
  api: { saveDraft: vi.fn().mockResolvedValue({}), deleteDraft: vi.fn().mockResolvedValue({}) }
}))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/api', () => ({ default: mocks.api }))
let app, host
const flush = async () => {
  for (let i = 0; i < 10; i++) { await Promise.resolve(); await nextTick() }
}
afterEach(() => { app?.unmount(); host?.remove(); vi.useRealTimers(); vi.clearAllMocks() })

function mount(drafts = [], ready = Promise.resolve()) {
  vi.useFakeTimers()
  const map = new Map(drafts)
  mocks.store = {
    draftsReady: ready, macros: {}, getMacro: () => ({}),
    getDraft: (u, t) => map.get(`${u}:${t}`),
    setDraft: (u, t, d) => map.set(`${u}:${t}`, d),
    removeDraft: (u, t) => map.delete(`${u}:${t}`)
  }
  const conversation = ref({ uuid: 'a', inbox_id: 1, inbox_channel: 'email' })
  const inboxes = ref([{ id: 1, default_signature_id: 11 }, { id: 2, default_signature_id: 22 }])
  const templates = ref([{ id: 10, is_default: true, name: 'Global' }, { id: 11, name: 'Support' }, { id: 22, name: 'Personal' }])
  const messageType = ref('reply'), attachments = ref([])
  let manager, selected
  app = createApp(defineComponent({ setup() {
    manager = useDraftManager(computed(() => conversation.value.uuid), messageType, attachments)
    selected = useReplySignature({ conversation, inboxes, templates, messageType,
      selection: manager.outgoingSignature, isLoading: manager.isLoading,
      hasStoredDraft: manager.hasStoredDraft, htmlContent: manager.htmlContent, attachments })
    return () => h('div')
  } }))
  host = document.createElement('div'); document.body.append(host); app.mount(host)
  const go = (uuid, inboxID) => { conversation.value = { uuid, inbox_id: inboxID, inbox_channel: 'email' } }
  return { map, manager, selected, go, conversation, inboxes, templates, messageType }
}

test('public -> personal -> public selects each inbox default without copying text or creating empty drafts', async () => {
  const f = mount(); await flush()
  expect(f.selected.value).toBe(11)
  f.go('b', 2); await flush(); expect(f.selected.value).toBe(22)
  f.go('a', 1); await flush(); expect(f.selected.value).toBe(11)
  await vi.advanceTimersByTimeAsync(600)
  expect(f.map.size).toBe(0)
  expect(f.manager.htmlContent.value).toBe('')
})

test('manual selection persists even in an empty draft and wins over metadata and template refresh', async () => {
  const f = mount(); await flush(); f.selected.value = 22
  f.inboxes.value[0].default_signature_id = 10
  f.conversation.value = { ...f.conversation.value, assigned_user_id: 99, status: 'Closed' }
  f.templates.value = [...f.templates.value, { id: 33, name: 'New' }]
  await flush(); expect(f.selected.value).toBe(22)
  f.go('b', 2); await flush(); f.go('a', 1); await flush()
  expect(f.selected.value).toBe(22)
  expect(f.map.get('a:reply').meta.outgoing_signature.source).toBe('manual')
  expect(f.manager.htmlContent.value).toBe('')
})

test('saved manual draft and legacy draft without signature metadata survive defaults', async () => {
  const f = mount([
    ['a:reply', { content: '<p>Manual text</p>', meta: { outgoing_signature: { source: 'manual', template_id: 22 } } }],
    ['b:reply', { content: '<p>Legacy text without signature</p>', meta: {} }]
  ]); await flush()
  expect(f.selected.value).toBe(22)
  expect(f.manager.htmlContent.value).toBe('<p>Manual text</p>')
  f.go('b', 2); await flush()
  expect(f.selected.value).toBe(10) // prior global behavior; no retroactive inbox default
  expect(f.manager.htmlContent.value).toBe('<p>Legacy text without signature</p>')
})

test('automatic signature freezes with written content and is restored from the existing draft', async () => {
  const f = mount(); await flush(); f.manager.htmlContent.value = '<p>Written answer</p>'
  f.inboxes.value[0].default_signature_id = 22; await flush()
  expect(f.selected.value).toBe(11)
  f.go('b', 2); await flush(); f.go('a', 1); await flush()
  expect(f.selected.value).toBe(11)
  expect(f.manager.htmlContent.value).toBe('<p>Written answer</p>')
})

test('rapid switches while drafts are loading never hydrate another conversation', async () => {
  let resolve
  const f = mount([
    ['a:reply', { content: 'Draft A', meta: { outgoing_signature: { source: 'manual', template_id: 11 } } }],
    ['b:reply', { content: 'Draft B', meta: { outgoing_signature: { source: 'manual', template_id: 22 } } }]
  ], new Promise(r => { resolve = r }))
  f.go('b', 2); f.go('a', 1); f.go('b', 2)
  resolve(); await flush()
  expect(f.selected.value).toBe(22)
  expect(f.manager.htmlContent.value).toBe('Draft B')
  f.go('a', 1); await flush()
  expect(f.selected.value).toBe(11)
  expect(f.manager.htmlContent.value).toBe('Draft A')
})

test('late inbox/template data resolves only the current conversation; notes never select a signature', async () => {
  const f = mount(); f.templates.value = []; f.inboxes.value = []; await flush()
  f.go('b', 2); await flush()
  f.inboxes.value = [{ id: 1, default_signature_id: 11 }, { id: 2, default_signature_id: 22 }]
  f.templates.value = [{ id: 11 }, { id: 22 }]; await flush()
  expect(f.selected.value).toBe(22)
  f.messageType.value = 'private_note'; await flush()
  expect(f.selected.value).toBeNull()
  f.manager.htmlContent.value = '<p>Internal</p>'
  f.messageType.value = 'reply'; await flush()
  expect(f.selected.value).toBe(22)
  expect(f.map.get('b:private_note').meta.outgoing_signature).toBeUndefined()
})

test('AI adoption and manual changes preserve content; clear starts a new reply with the inbox default', async () => {
  const f = mount(); await flush()
  f.manager.htmlContent.value = '<p>AI answer</p>'
  f.manager.aiResponse.value = { response_source: 'ai', source_message_uuid: 'source' }
  f.selected.value = 22; await flush()
  expect(f.manager.htmlContent.value).toBe('<p>AI answer</p>')
  await vi.advanceTimersByTimeAsync(600)
  expect(f.map.get('a:reply').meta.ai_response.response_source).toBe('ai')
  f.manager.clearDraft(); await flush()
  expect(f.selected.value).toBe(11)
  expect(f.manager.htmlContent.value).toBe('')
})

test('missing or deleted signatures fall back cleanly; rename keeps the stable ID', async () => {
  const f = mount(); await flush(); f.selected.value = 22
  f.templates.value = f.templates.value.map(t => ({ ...t, name: 'Renamed' })); await flush()
  expect(f.selected.value).toBe(22)
  f.templates.value = f.templates.value.filter(t => t.id !== 22); await flush()
  expect(f.selected.value).toBe(10)
  f.templates.value = []; await flush(); expect(f.selected.value).toBeNull()
})

test('an automatic choice follows a deliberate sender inbox change while the reply is empty', async () => {
  const f = mount(); await flush()
  f.conversation.value = { ...f.conversation.value, inbox_id: 2 }; await flush()
  expect(f.selected.value).toBe(22)
  f.selected.value = 11
  f.conversation.value = { ...f.conversation.value, inbox_id: 1 }; await flush()
  expect(f.selected.value).toBe(11)
})

test('an inbox without a default retains the global or first-template fallback', async () => {
  const f = mount(); f.inboxes.value[0].default_signature_id = null; await flush()
  expect(f.selected.value).toBe(10)
  f.templates.value = [{ id: 22 }, { id: 11 }]; await flush()
  expect(f.selected.value).toBe(22)
})

test('initial metadata loading defers automatic selection without losing newly typed text', async () => {
  const f = mount()
  f.templates.value = []
  f.inboxes.value = []
  await flush()
  f.manager.isLoading.value = true
  f.manager.htmlContent.value = '<p>Newly typed text</p>'
  f.inboxes.value = [{ id: 1, default_signature_id: 22 }]
  f.templates.value = [{ id: 10, is_default: true }, { id: 22 }]
  await flush()
  expect(f.manager.outgoingSignature.value).toBeNull()
  f.manager.isLoading.value = false
  await flush()
  expect(f.selected.value).toBe(22)
  expect(f.manager.htmlContent.value).toBe('<p>Newly typed text</p>')
})
