// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, reactive, ref, nextTick } from 'vue'
import ReplyBox from './ReplyBox.vue'

const mocks = vi.hoisted(() => ({ api: {}, store: null, draft: null, files: null }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 42, can: () => true }) }))
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
  useIsComposerCramped: () => ref(false)
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
