// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, reactive, ref, nextTick } from 'vue'
import ReplyBox from './ReplyBox.vue'

const mocks = vi.hoisted(() => ({ api: {}, store: null, draft: null, files: null, cramped: null, user: null }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('@main/stores/inbox', () => ({ useInboxStore: () => ({ inboxes: [] }) }))
vi.mock('@main/stores/notification', () => ({
  useNotificationStore: () => ({ markAssignmentAsReadForConversation: () => {} })
}))
vi.mock('@main/stores/aiPrompt', () => ({
  useAiPromptStore: () => ({ prompts: [], fetchPrompts: () => {} })
}))
vi.mock('@main/composables/useEmitter', () => ({
  useEmitter: () => ({ on: () => {}, off: () => {}, emit: () => {} })
}))
vi.mock('@main/composables/useIsComposerCramped', () => ({
  useIsComposerCramped: () => mocks.cramped
}))
vi.mock('@main/composables/useVisualViewportHeight', () => ({ useVisualViewportHeight: () => {} }))
vi.mock('@main/composables/useDraftManager', () => ({ useDraftManager: () => mocks.draft }))
vi.mock('@main/composables/useFileUpload', () => ({ useFileUpload: () => mocks.files }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
// Keep the real ReplyBox orchestration, with lightweight editor/design stubs.
vi.mock('./ReplyBoxContent.vue', () => ({
  default: defineComponent({
    inheritAttrs: false,
    props: ['htmlContent', 'isSending'],
    emits: ['send'],
    setup(props, { emit, expose }) {
      expose({ focus: () => {} })
      return () =>
        h('section', [
          h('div', { 'data-editor': '' }, props.htmlContent),
          h(
            'button',
            { 'data-send': '', disabled: props.isSending, onClick: () => emit('send') },
            'Send'
          )
        ])
    }
  })
}))
vi.mock('@shared-ui/components/ui/dialog', () => {
  const wrapper = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
  return {
    Dialog: defineComponent({
      props: ['open'],
      setup(props, { slots }) {
        return () => (props.open ? h('div', { 'data-dialog': '' }, slots.default?.()) : null)
      }
    }),
    DialogContent: wrapper,
    DialogHeader: wrapper,
    DialogTitle: wrapper,
    DialogDescription: wrapper,
    DialogFooter: wrapper
  }
})
vi.mock('@shared-ui/components/ui/alert-dialog', () => {
  const wrapper = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
  return Object.fromEntries(
    [
      'AlertDialog',
      'AlertDialogAction',
      'AlertDialogCancel',
      'AlertDialogContent',
      'AlertDialogDescription',
      'AlertDialogFooter',
      'AlertDialogHeader',
      'AlertDialogTitle'
    ].map((name) => [
      name,
      name === 'AlertDialog'
        ? defineComponent({
            props: ['open'],
            setup(props, { slots }) {
              return () => (props.open ? h('div', slots.default?.()) : null)
            }
          })
        : wrapper
    ])
  )
})

let app, root, exposed
const originalUUID = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'
const outgoingUUID = 'bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb'
const note = {
  noteUUID: 'cccccccc-cccc-4ccc-cccc-cccccccccccc',
  originalUUID,
  suggestion: 'Hallo\nKI-Vorschlag'
}
const flush = async () => {
  for (let i = 0; i < 15; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
function click(text) {
  const button = [...root.querySelectorAll('button')].find((b) => b.textContent === text)
  expect(button).toBeTruthy()
  button.click()
}

beforeEach(async () => {
  mocks.user = reactive({ userID: 42, can: () => true })
  mocks.cramped = ref(false)
  const map = new Map()
  mocks.store = reactive({
    current: {
      uuid: 'conversation',
      inbox_channel: 'email',
      contact: { email: 'contact@example.com' }
    },
    messages: { version: 0 },
    currentTo: ['contact@example.com'],
    draftsReady: Promise.resolve(),
    resolveDraftType: () => 'reply',
    setSelectedDraftType: () => {},
    getMacro: () => ({}),
    resetMacro: () => {},
    getDraft: (uuid, type) => map.get(`${uuid}:${type}`),
    setDraft: (uuid, type, draft) => map.set(`${uuid}:${type}`, draft),
    removeDraft: (uuid, type) => map.delete(`${uuid}:${type}`),
    addPendingMessage: vi.fn(() => 'echo-1'),
    removePendingMessage: vi.fn(),
    refreshCurrentConversation: vi.fn().mockResolvedValue({})
  })
  mocks.draft = {
    htmlContent: ref('Existing employee answer'),
    textContent: ref('Existing employee answer'),
    isLoading: ref(false),
    aiResponse: ref(null),
    clearDraft: vi.fn(),
    loadedAttachments: ref([]),
    loadedMacroActions: ref([]),
    loadedMacroID: ref(0)
  }
  mocks.files = {
    uploadingFiles: ref([]),
    mediaFiles: ref([]),
    clearMediaFiles: vi.fn(),
    setMediaFiles: vi.fn()
  }
  mocks.api.getOutgoingTemplates = vi.fn().mockResolvedValue({ data: { data: [] } })
  mocks.api.saveDraft = vi.fn().mockResolvedValue({})
  mocks.api.deleteDraft = vi.fn().mockResolvedValue({})
  mocks.api.sendMessage = vi
    .fn()
    .mockResolvedValue({
      data: {
        data: { uuid: outgoingUUID, meta: { ai_feedback_original_message_uuid: originalUUID } }
      }
    })
  mocks.api.saveAIFeedback = vi
    .fn()
    .mockResolvedValue({ data: { data: { id: outgoingUUID, committed: true } } })
  mocks.api.getConversationMessage = vi.fn(async (uuid, id) => ({
    data: {
      data:
        id === originalUUID
          ? {
              uuid: originalUUID,
              conversation_uuid: uuid,
              type: 'incoming',
              sender_type: 'contact',
              private: false
            }
          : {
              uuid: outgoingUUID,
              conversation_uuid: uuid,
              type: 'outgoing',
              sender_type: 'agent',
              sender_id: 42,
              private: false,
              status: 'pending',
              meta: { ai_feedback_original_message_uuid: originalUUID }
            }
    }
  }))
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(ReplyBox)
  app.config.globalProperties.$t = (key) => key
  exposed = app.mount(root)
  await flush()
})
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

test('opening and cancelling own-answer modal preserves editor and never sends', async () => {
  await exposed.openReply({ aiNote: note, mode: 'own' })
  await flush()
  expect(root.textContent).toContain('Eigene Antwort verfassen')
  expect(root.textContent).toContain(note.suggestion)
  click('Abbrechen')
  await flush()
  expect(mocks.draft.htmlContent.value).toBe('Existing employee answer')
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
})
test('adoption is explicit, editable and causes neither send nor feedback', async () => {
  await exposed.openReply({ aiNote: note, mode: 'adopt' })
  await flush()
  expect(mocks.draft.htmlContent.value).toBe('Existing employee answer')
  click('KI-Antwort übernehmen')
  await flush()
  expect(mocks.draft.htmlContent.value).toBe('Hallo<br>KI-Vorschlag')
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
})
test('own answer keeps editor, passes original UUID, and double-click queues once', async () => {
  await exposed.openReply({ aiNote: note, mode: 'own' })
  await flush()
  const textarea = root.querySelector('textarea')
  textarea.value = 'Different answer because…'
  textarea.dispatchEvent(new Event('input'))
  await flush()
  click('Weiter zur Antwort')
  await flush()
  expect(mocks.draft.htmlContent.value).toBe('Existing employee answer')
  const send = root.querySelector('[data-send]')
  expect(send).toBeTruthy()
  send.click()
  send.click()
  await flush()
  expect(mocks.api.sendMessage).toHaveBeenCalledTimes(1)
  expect(mocks.api.sendMessage.mock.calls[0][1]).toMatchObject({
    message: 'Existing employee answer',
    ai_feedback_original_message_uuid: originalUUID,
    ai_response: { response_source: 'human', source_message_uuid: originalUUID, note_uuid: note.noteUUID },
    private: false
  })
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
  expect(mocks.store.getDraft('conversation', 'reply').meta.ai_feedback.reason).toBe(
    'Different answer because…'
  )
})
test('missing original reference cannot open the action modal', async () => {
  mocks.api.getConversationMessage.mockResolvedValue({
    data: {
      data: {
        uuid: originalUUID,
        conversation_uuid: 'other',
        type: 'incoming',
        sender_type: 'contact'
      }
    }
  })
  await exposed.openReply({ aiNote: note, mode: 'own' })
  await flush()
  expect(root.querySelector('[data-dialog]')).toBeNull()
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
})
test('changing conversation closes modal and prevents applying an old proposal', async () => {
  await exposed.openReply({ aiNote: note, mode: 'own' })
  await flush()
  mocks.store.current.uuid = 'other'
  await flush()
  expect(root.querySelector('[data-dialog]')).toBeNull()
  expect(mocks.draft.htmlContent.value).toBe('Existing employee answer')
})

test('sent status event saves feedback and retry never posts another message', async () => {
  await exposed.openReply({ aiNote: note, mode: 'own' })
  await flush()
  click('Weiter zur Antwort')
  await flush()
  root.querySelector('[data-send]').click()
  await flush()
  mocks.api.saveAIFeedback.mockRejectedValueOnce(new Error('502'))
  mocks.api.getConversationMessage.mockImplementation(async (uuid, id) => ({
    data: {
      data:
        id === originalUUID
          ? {
              uuid: originalUUID,
              conversation_uuid: uuid,
              type: 'incoming',
              sender_type: 'contact',
              private: false
            }
          : {
              uuid: outgoingUUID,
              conversation_uuid: uuid,
              type: 'outgoing',
              sender_type: 'agent',
              sender_id: 42,
              private: false,
              status: 'sent',
              meta: { ai_feedback_original_message_uuid: originalUUID }
            }
    }
  }))
  mocks.store.messages.version++
  await flush()
  expect(root.textContent).toContain('Feedback konnte nicht gespeichert werden')
  const getCalls = mocks.api.getConversationMessage.mock.calls.length
  click('Feedback erneut speichern')
  await flush()
  expect(mocks.api.sendMessage).toHaveBeenCalledTimes(1)
  expect(mocks.api.saveAIFeedback).toHaveBeenCalledTimes(2)
  expect(mocks.api.getConversationMessage).toHaveBeenCalledTimes(getCalls)
  expect(root.textContent).toContain('Feedback gespeichert')
})

test.each([['ai', undefined], ['ai_adapted', 'Angepasster Text']])('adoption confirms composer replacement and sends %s provenance without feedback', async (source, adaptedText) => {
  await exposed.openReply({ aiNote: note, mode: 'adopt', adaptedText })
  await flush()
  expect(mocks.draft.htmlContent.value).toBe('Existing employee answer')
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
  click('KI-Antwort übernehmen'); await flush()
  expect(mocks.draft.htmlContent.value).toBe(adaptedText || 'Hallo<br>KI-Vorschlag')
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
  root.querySelector('[data-send]').click(); await flush()
  expect(mocks.api.sendMessage).toHaveBeenCalledTimes(1)
  expect(mocks.api.sendMessage.mock.calls[0][1]).toMatchObject({ ai_response: { response_source: source, source_message_uuid: originalUUID, note_uuid: note.noteUUID } })
  expect(mocks.api.sendMessage.mock.calls[0][1].ai_feedback_original_message_uuid).toBeUndefined()
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
})


test('normal employee reply sends without a client-selected source or classification', async () => {
  root.querySelector('[data-send]').click(); await flush()
  expect(mocks.api.sendMessage).toHaveBeenCalledTimes(1)
  const body = mocks.api.sendMessage.mock.calls[0][1]
  expect(body.reply_to_message_uuid).toBeUndefined()
  expect(body.ai_response).toBeUndefined()
  expect(body.ai_feedback_original_message_uuid).toBeUndefined()
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
})


test('cancelled human feedback resumes normal backend assignment without reclassification', async () => {
  await exposed.openReply({ aiNote: note, mode: 'own' }); await flush()
  click('Weiter zur Antwort'); await flush()
  click('Feedback abbrechen (Antwort behalten)'); await flush()
  root.querySelector('[data-send]').click(); await flush()
  const body = mocks.api.sendMessage.mock.calls[0][1]
  expect(body.reply_to_message_uuid).toBeUndefined()
  expect(body.ai_response).toBeUndefined()
  expect(body.ai_feedback_original_message_uuid).toBeUndefined()
})


test('desktop and mobile composers have no source selector or mandatory selection', async () => {
  expect(root.querySelector('[data-editor]')).toBeTruthy()
  expect(root.textContent).not.toContain('Antwort auf Kundenmail')
  expect(root.querySelector('[aria-label="Ursprüngliche Kundenmail auswählen"]')).toBeNull()
  mocks.cramped.value = true; await flush()
  await exposed.openReply({ content: 'Mobile reply without source selection' }); await flush()
  expect(root.querySelector('[data-dialog]')).toBeTruthy()
  expect(root.querySelector('[data-editor]').textContent).toBe('Mobile reply without source selection')
  expect(root.textContent).not.toContain('Antwort auf Kundenmail')
  expect(root.querySelector('[aria-label="Ursprüngliche Kundenmail auswählen"]')).toBeNull()
})

const manualNote = { noteUUID: note.noteUUID, originalUUID }
const manualResponse = (reason = null, editable = true) => ({ data: { data: {
  note_uuid: note.noteUUID, original_message_uuid: originalUUID, conversation_uuid: 'conversation',
  agent_id: 42, feedback_reason: reason, editable
} } })
function mockManual(reason = null, editable = true) {
  mocks.api.getManualReviewReason = vi.fn().mockResolvedValue(manualResponse(reason, editable))
  mocks.api.saveManualReviewReason = vi.fn().mockResolvedValue(manualResponse('saved'))
  mocks.api.deleteManualReviewReason = vi.fn().mockResolvedValue(manualResponse())
}
function reasonInput(value) {
  const input = root.querySelector('textarea')
  input.value = value
  input.dispatchEvent(new Event('input'))
}
test('manual variant reuses feedback dialog without proposal or composer changes', async () => {
  mockManual('Previously saved')
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  expect(root.textContent).toContain('Feedback erstellen')
  expect(root.textContent).toContain('Was soll die KI beim nächsten Mal anders machen?')
  expect(root.textContent).toContain('tatsächlich gesendete Mitarbeiterantwort')
  expect(root.textContent).not.toContain(note.suggestion)
  expect(root.querySelector('[aria-label="Ursprünglicher KI-Antwortvorschlag"]')).toBeNull()
  expect(root.querySelector('textarea').value).toBe('Previously saved')
  expect(document.activeElement).toBe(root.querySelector('textarea'))
  click('Abbrechen'); await flush()
  expect(mocks.api.saveManualReviewReason).not.toHaveBeenCalled()
  expect(mocks.draft.htmlContent.value).toBe('Existing employee answer')
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
})
test('manual save persists annotation only; normal reply remains unclassified', async () => {
  mockManual()
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  reasonInput('Short reply and apology'); await flush()
  click('Feedbackgrund speichern'); await flush()
  expect(mocks.api.saveManualReviewReason).toHaveBeenCalledWith('conversation', note.noteUUID, { feedback_reason: 'Short reply and apology' })
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
  expect(root.querySelector('[data-dialog]')).toBeNull()
  root.querySelector('[data-send]').click(); await flush()
  const body = mocks.api.sendMessage.mock.calls[0][1]
  expect(body.message).toBe('Existing employee answer')
  expect(body.ai_response).toBeUndefined()
  expect(body.ai_feedback_original_message_uuid).toBeUndefined()
})
test('manual empty reason deletes saved annotation and enforces Unicode length', async () => {
  mockManual('old')
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  reasonInput('🙂'.repeat(2501)); await flush()
  expect([...root.querySelectorAll('button')].find(b => b.textContent === 'Feedbackgrund speichern').disabled).toBe(true)
  expect(root.textContent).toContain('2501 / 2500 Zeichen')
  reasonInput('   '); await flush(); click('Feedbackgrund speichern'); await flush()
  expect(mocks.api.deleteManualReviewReason).toHaveBeenCalledTimes(1)
  expect(mocks.api.saveManualReviewReason).not.toHaveBeenCalled()
})
test('manual saving waits for confirmation, blocks double clicks, retains failures', async () => {
  mockManual()
  let reject
  mocks.api.saveManualReviewReason.mockReturnValue(new Promise((_, fail) => { reject = fail }))
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  reasonInput('retain'); await flush(); click('Feedbackgrund speichern'); await flush()
  expect(root.querySelector('[data-dialog]')).toBeTruthy()
  expect(root.textContent).not.toContain('Feedbackgrund gespeichert')
  expect(mocks.api.saveManualReviewReason).toHaveBeenCalledTimes(1)
  reject(new Error('network')); await flush()
  expect(root.querySelector('textarea').value).toBe('retain')
  expect(root.textContent).toContain('konnte nicht gespeichert')
  mocks.api.saveManualReviewReason.mockRejectedValue({ response: { status: 409 } })
  click('Feedbackgrund speichern'); await flush()
  expect(root.textContent).toContain('nicht mehr geändert')
  expect(root.querySelector('textarea').disabled).toBe(true)
})
test('manual dialog cannot edit after SENT or a live status update', async () => {
  mockManual('frozen', false)
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  expect(root.querySelector('textarea').disabled).toBe(true)
  click('Abbrechen'); await flush()
  mockManual('draft')
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  reasonInput('unsaved'); await flush()
  mocks.api.getManualReviewReason.mockResolvedValue(manualResponse('frozen', false))
  mocks.store.messages.version++; await flush()
  expect(root.querySelector('textarea').value).toBe('unsaved')
  expect(root.querySelector('textarea').disabled).toBe(true)
})
test('manual load failures and ticket switches never show false success', async () => {
  mockManual()
  mocks.api.getManualReviewReason.mockRejectedValue(new Error('network'))
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  expect(root.textContent).toContain('konnte nicht geladen')
  expect(root.querySelector('textarea').disabled).toBe(true)
  click('Abbrechen'); await flush()
  let resolve
  mocks.api.getManualReviewReason.mockReturnValue(new Promise(ok => { resolve = ok }))
  const opening = exposed.openReply({ manualReviewNote: manualNote }); await flush()
  mocks.store.current.uuid = 'other'; await flush()
  resolve(manualResponse('must not leak')); await opening; await flush()
  expect(root.querySelector('[data-dialog]')).toBeNull()
  expect(root.textContent).not.toContain('must not leak')
})

const manualTitle = 'Feedback für manuelle Antwort gespeichert'
function loadManualNote() {
  mocks.store.conversationMessages = [{
    uuid: note.noteUUID, private: true, type: 'outgoing', sender_type: 'agent',
    meta: { ai_analysis: { kind: 'n8n_manual_review', version: 1, original_message_uuid: originalUUID } }
  }]
  mocks.store.messages.version++
}
test('manual banner uses the existing status area only after server confirmation and disappears on delete', async () => {
  mockManual()
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  expect(root.textContent).not.toContain(manualTitle)
  let resolve
  mocks.api.saveManualReviewReason.mockReturnValue(new Promise(ok => { resolve = ok }))
  reasonInput('saved reason'); await flush(); click('Feedbackgrund speichern'); await flush()
  expect(root.textContent).not.toContain(manualTitle)
  resolve(manualResponse('saved reason')); await flush()
  const banner = root.querySelector('[role="status"]')
  expect(banner.textContent).toContain(manualTitle)
  expect(banner.className).toBe('mx-2 mt-2 rounded-md border p-3 text-sm')
  expect(banner.textContent).toContain('Private Notizen erzeugen kein Feedback.')
  mocks.api.getManualReviewReason.mockResolvedValue(manualResponse('saved reason'))
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  reasonInput(''); await flush(); click('Feedbackgrund speichern'); await flush()
  expect(root.textContent).not.toContain(manualTitle)
  expect(mocks.api.sendMessage).not.toHaveBeenCalled()
  expect(mocks.api.saveAIFeedback).not.toHaveBeenCalled()
})
test('manual banner restores from GET on reopening and isolates employees and tickets', async () => {
  mockManual('persisted')
  loadManualNote(); await flush()
  expect(root.textContent).toContain(manualTitle)
  mocks.store.current.uuid = 'other'; await flush()
  expect(root.textContent).not.toContain(manualTitle)
  mocks.store.current.uuid = 'conversation'; await flush()
  expect(root.textContent).toContain(manualTitle)
  mocks.user.userID = 43; await flush()
  expect(root.textContent).not.toContain(manualTitle)
  mocks.user.userID = 42; await flush()
  expect(root.textContent).toContain(manualTitle)
  mocks.api.getManualReviewReason.mockRejectedValue(new Error('network'))
  mocks.store.messages.version++; await flush()
  expect(root.textContent).not.toContain(manualTitle)
})
test.each(['SENT', 'new incoming', 'incompatible source'])('manual banner hides when server eligibility changes: %s', async () => {
  mockManual('persisted')
  loadManualNote(); await flush()
  expect(root.textContent).toContain(manualTitle)
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  mocks.api.getManualReviewReason.mockResolvedValue(manualResponse('persisted', false))
  mocks.store.messages.version++; await flush()
  expect(root.textContent).not.toContain(manualTitle)
  expect(root.querySelector('textarea').disabled).toBe(true)
})
test('stale GET cannot resurrect a deleted annotation or cross ticket and employee changes', async () => {
  mockManual('persisted')
  await exposed.openReply({ manualReviewNote: manualNote }); await flush()
  let resolve
  mocks.api.getManualReviewReason.mockReturnValue(new Promise(ok => { resolve = ok }))
  mocks.store.messages.version++; await flush()
  reasonInput(''); await flush(); click('Feedbackgrund speichern'); await flush()
  expect(root.textContent).not.toContain(manualTitle)
  resolve(manualResponse('stale')); await flush()
  expect(root.textContent).not.toContain(manualTitle)
  loadManualNote(); await flush()
  mocks.user.userID = 43; await flush()
  mocks.store.current.uuid = 'other'; await flush()
  resolve(manualResponse('stale')); await flush()
  expect(root.textContent).not.toContain(manualTitle)
})
test('regular own-answer banner retains priority and its existing cancellation action', async () => {
  mockManual('persisted')
  loadManualNote(); await flush()
  await exposed.openReply({ aiNote: note, mode: 'own' }); await flush()
  click('Weiter zur Antwort'); await flush()
  expect(root.querySelectorAll('[role="status"]')).toHaveLength(1)
  expect(root.textContent).toContain('Eigene Antwort: optionales Feedback vorgemerkt')
  expect(root.textContent).not.toContain(manualTitle)
  expect([...root.querySelectorAll('button')].some(b => b.textContent === 'Feedback abbrechen (Antwort behalten)')).toBe(true)
})

test('a different composer source suppresses the manual confirmation', async () => {
  mockManual('persisted')
  loadManualNote(); await flush()
  expect(root.textContent).toContain(manualTitle)
  mocks.draft.aiResponse.value = { source_message_uuid: outgoingUUID, response_source: 'ai' }
  await flush()
  expect(root.textContent).not.toContain(manualTitle)
})
test('dialog shares an in-flight hydration GET and duplicate notes do not duplicate requests', async () => {
  mockManual()
  let resolve
  mocks.api.getManualReviewReason.mockReturnValue(new Promise(ok => { resolve = ok }))
  loadManualNote()
  mocks.store.conversationMessages.push({ ...mocks.store.conversationMessages[0], uuid: outgoingUUID })
  await flush()
  const opening = exposed.openReply({ manualReviewNote: { ...manualNote, noteUUID: outgoingUUID } })
  await flush()
  expect(mocks.api.getManualReviewReason).toHaveBeenCalledTimes(1)
  resolve({ data: { data: { ...manualResponse('persisted').data.data, note_uuid: outgoingUUID } } })
  await opening; await flush()
  expect(root.textContent).toContain(manualTitle)
  expect(root.querySelector('textarea').value).toBe('persisted')
})
