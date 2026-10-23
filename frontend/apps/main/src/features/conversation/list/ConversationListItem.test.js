// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive, ref } from 'vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import ConversationListItem from './ConversationListItem.vue'

const mocks = vi.hoisted(() => ({ store: null, route: null, settings: null }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/stores/appSettings', () => ({ useAppSettingsStore: () => mocks.settings }))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route }))
vi.mock('@/composables/useBulkActionPermissions', () => ({
  useBulkActionPermissions: () => ({ canBulkAct: ref(true) })
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))

let app, host, props
beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      disconnect() {}
    }
  )
  vi.stubGlobal('requestAnimationFrame', () => 1)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  mocks.store = reactive({
    typingByUUID: {},
    selectedCount: 0,
    selected: false,
    draft: null,
    markAsUnread: vi.fn(),
    toggleSelect: vi.fn(),
    isSelected: () => mocks.store.selected,
    conversationHasDraft: () => Boolean(mocks.store.draft),
    conversationDraftPreview: () => mocks.store.draft
  })
  mocks.route = reactive({ params: { inboxID: '7' }, query: { status: 'Open' } })
  mocks.settings = reactive({ settings: {} })
  props = reactive({
    conversation: {
      uuid: 'ticket',
      contact: { first_name: 'Customer', avatar_url: '' },
      subject: '[INTERN] Aufgaben – #252',
      status: 'Open',
      priority: 'High',
      last_message: 'Full message '.repeat(30),
      last_message_sender: 'agent',
      unread_message_count: 12,
      inbox_channel: 'email',
      inbox_name: 'Support',
      tags: ['Intern', 'Support'],
      mentioned_message_uuid: 'mention'
    },
    contactFullName: 'Very long contact name',
    currentConversation: null
  })
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({
    render: () => h(TooltipProvider, null, { default: () => h(ConversationListItem, props) })
  })
  app.config.globalProperties.$t = (key) => key
  app.component(
    'router-link',
    defineComponent({
      props: ['to'],
      setup:
        (p, { slots }) =>
        () =>
          h('a', { href: '#ticket', 'data-route': JSON.stringify(p.to) }, slots.default?.())
    })
  )
  app.mount(host)
})
afterEach(() => {
  app.unmount()
  host.remove()
  vi.unstubAllGlobals()
})

test('keeps routes, unread indication, full preview text and reply marker while separating status from tags', async () => {
  const card = host.querySelector('[data-conversation-card]')
  expect(JSON.parse(card.dataset.route)).toEqual({
    name: 'personal-inbox-conversation',
    params: { uuid: 'ticket', inboxID: '7' },
    query: { status: 'Open', scrollTo: 'mention' }
  })
  expect(card.textContent).toContain('9+')
  expect(host.querySelector('[data-conversation-preview] p').textContent).toBe(
    props.conversation.last_message
  )
  expect(host.querySelector('[data-conversation-preview] > svg.lucide-reply')).not.toBeNull()
  const status = host.querySelector('[data-conversation-status]')
  expect(status.textContent.trim()).toBe('Open')
  expect(status.classList.contains('whitespace-nowrap')).toBe(true)
  expect(status.closest('[data-conversation-tags]')).toBeNull()
  props.currentConversation = { uuid: 'ticket' }
  await nextTick()
  expect(card.classList.contains('bg-accent')).toBe(true)
  mocks.store.selected = true
  mocks.store.selectedCount = 1
  await nextTick()
  host
    .querySelector('button[role=checkbox]')
    .parentElement.dispatchEvent(new MouseEvent('click', { bubbles: true, shiftKey: true }))
  expect(mocks.store.toggleSelect).toHaveBeenCalledWith('ticket', true)
})

test('preserves typing and draft precedence, title fallback and status colours', async () => {
  mocks.store.draft = { content: '<p>Draft preview</p>' }
  await nextTick()
  expect(host.querySelector('[data-conversation-preview]').textContent).toContain('Draft preview')
  expect(host.querySelector('[data-conversation-preview] > svg.lucide-reply')).toBeNull()
  mocks.store.typingByUUID.ticket = true
  await nextTick()
  expect(host.querySelector('[data-conversation-preview]').textContent).toContain(
    'globals.terms.typing'
  )
  mocks.settings.settings['app.show_conversation_subject'] = false
  await nextTick()
  expect(host.querySelector('[data-conversation-title]').textContent.trim()).toBe(
    props.contactFullName
  )
  for (const [status, color] of [
    ['Open', 'text-red-700'],
    ['Resolved', 'text-green-700'],
    ['Snoozed', 'text-blue-700'],
    ['Closed', 'text-violet-700'],
    ['Custom', 'text-muted-foreground']
  ]) {
    props.conversation.status = status
    await nextTick()
    expect(host.querySelector('[data-conversation-status]').classList.contains(color)).toBe(true)
  }
})

test('warning tags never recolour cards or override hover, active, selected and unread states', async () => {
  const card = host.querySelector('[data-conversation-card]')
  for (const tags of [
    [],
    ['😎 Mensch erforderlich'],
    ['🛑 Verarbeitungsfehler'],
    ['😎 Mensch erforderlich', '🛑 Verarbeitungsfehler']
  ]) {
    props.conversation.tags = tags
    props.currentConversation = null
    mocks.store.selected = false
    await nextTick()
    expect(card.className).not.toMatch(/ticket-highlight|bg-\[#/)
    expect(card.classList.contains('hover:bg-accent/40')).toBe(true)
    expect(host.querySelector('[data-conversation-title]').classList.contains('font-bold')).toBe(
      true
    )
    props.currentConversation = { uuid: 'ticket' }
    await nextTick()
    expect(card.classList.contains('bg-accent')).toBe(true)
    props.currentConversation = null
    mocks.store.selected = true
    await nextTick()
    expect(card.classList.contains('bg-primary/5')).toBe(true)
    expect(card.classList.contains('hover:bg-primary/10')).toBe(true)
  }
})
