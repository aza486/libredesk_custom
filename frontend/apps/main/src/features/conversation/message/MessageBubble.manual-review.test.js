// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import MessageBubble from './MessageBubble.vue'
const state = vi.hoisted(() => ({ store: null, can: true }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => state.store }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 42, can: () => state.store.allowed }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: key => key }) }))
vi.mock('vue-letter', () => ({ Letter: defineComponent({ setup: () => () => h('div') }) }))
vi.mock('./attachment/BubbleAttachmentPreview.vue', () => ({ default: { render: () => null } }))
vi.mock('./CSATResponseDisplay.vue', () => ({ default: { render: () => null } }))
vi.mock('@shared-ui/components/ui/tooltip', () => {
 const wrapper = defineComponent({ setup: (_, { slots }) => () => h('span', slots.default?.()) })
 return { Tooltip: wrapper, TooltipTrigger: wrapper, TooltipContent: wrapper }
})
const uuid = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'
let app, root
async function mount(kind, changes = {}) {
 state.can = true
 state.store = reactive({ allowed: true, current: { uuid, inbox_channel: 'email' }, messages: { version: 0, data: { getAllPagesMessages: () => [] } } })
 const message = { uuid, type: 'outgoing', sender_type: 'agent', private: true, sender_id: 41,
   content: 'Note', created_at: '2026-10-26T09:00:00Z', status: 'sent', attachments: [],
   meta: { ai_analysis: { kind, version: 1, original_message_uuid: uuid, ...(kind === 'n8n_ai_analysis' ? { suggested_message: 'Proposal' } : {}) } }, ...changes }
 root = document.createElement('div'); document.body.append(root)
 const emit = vi.fn()
 app = createApp(MessageBubble, { message, direction: 'outgoing', onComposeReply: emit })
 app.component('router-link', defineComponent({ setup: (_, { slots }) => () => h('span', slots.default?.()) }))
 app.mount(root); await nextTick(); return emit
}
afterEach(() => { app?.unmount(); root?.remove() })
test('manual typed note exposes only annotation action and uses existing compose event', async () => {
 const emit = await mount('n8n_manual_review')
 const button = [...root.querySelectorAll('button')].find(b => b.textContent === 'Feedback erstellen')
 expect(button).toBeTruthy(); button.click()
 expect(emit).toHaveBeenCalledWith({ manualReviewNote: { noteUUID: uuid, originalUUID: uuid }, mode: 'manual' })
 expect(root.textContent).not.toContain('conversation.ai.useReply')
 expect(root.textContent).not.toContain('conversation.ai.writeOwnReply')
 expect(root.textContent).not.toContain('Stil anpassen')
 state.store.allowed = false; await nextTick(); expect(root.textContent).not.toContain('Feedback erstellen')
})
test('regular analysis retains all original actions', async () => {
 await mount('n8n_ai_analysis')
 expect(root.textContent).toContain('conversation.ai.useReply')
 expect(root.textContent).toContain('conversation.ai.writeOwnReply')
 expect(root.textContent).toContain('Stil anpassen')
 expect(root.textContent).not.toContain('Feedback erstellen')
})
test('ordinary HTML, public and deleted notes expose no manual action', async () => {
 await mount('n8n_manual_review', { meta: {}, content: 'n8n_manual_review Human needed' })
 expect(root.textContent).not.toContain('Feedback erstellen')
})

test('public manual-looking note has no button', async () => {
  await mount('n8n_manual_review', { private: false })
  expect(root.textContent).not.toContain('Feedback erstellen')
})
test('deleted manual note has no button', async () => {
  await mount('n8n_manual_review', { meta: { deleted_at: 'now', ai_analysis: { kind: 'n8n_manual_review', version: 1, original_message_uuid: uuid } } })
  expect(root.textContent).not.toContain('Feedback erstellen')
})
test('widgets cannot expose annotation action', async () => {
  await mount('n8n_manual_review')
  state.store.current.inbox_channel = 'livechat'; await nextTick()
  expect(root.textContent).not.toContain('Feedback erstellen')
})
