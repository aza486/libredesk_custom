// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive, ref } from 'vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import ConversationListItem from './ConversationListItem.vue'
import { createI18n } from 'vue-i18n'
import de from '../../../../../../../i18n/de-DE.json'

const mocks = vi.hoisted(() => ({ store: null, route: null, settings: null }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/stores/appSettings', () => ({ useAppSettingsStore: () => mocks.settings }))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route }))
vi.mock('@/composables/useBulkActionPermissions', () => ({
  useBulkActionPermissions: () => ({ canBulkAct: ref(true) })
}))

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
      last_message_at: '2026-10-23T10:00:00Z',
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
  app.use(createI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } }))
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
  expect(card.textContent).toContain('12')
  expect(host.querySelector('[data-conversation-preview] p').textContent).toContain(
    props.conversation.last_message
  )
  expect(host.querySelector('[data-conversation-preview] > svg.lucide-reply')).not.toBeNull()
  const status = host.querySelector('[data-conversation-status]')
  expect(status.textContent.trim()).toBe('Offen')
  expect(status.classList.contains('whitespace-nowrap')).toBe(true)
  expect(status.closest('[data-conversation-tags]')).toBeNull()
  props.currentConversation = { uuid: 'ticket' }
  await nextTick()
  expect(card.classList.contains('bg-primary/15')).toBe(true)
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
    'Schreibt …'
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
    const badge = host.querySelector('[data-conversation-status]')
    expect(badge.classList.contains(color)).toBe(true)
    expect(badge.textContent.trim()).toBe(({ Open: 'Offen', Closed: 'Geschlossen', Snoozed: 'Schlummernd', Resolved: de['globals.terms.resolved'] })[status] || status)
    expect(props.conversation.status).toBe(status)
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
    expect(card.classList.contains('bg-primary/15')).toBe(true)
    props.currentConversation = null
    mocks.store.selected = true
    await nextTick()
    expect(card.classList.contains('bg-primary/5')).toBe(true)
    expect(card.classList.contains('hover:bg-primary/10')).toBe(true)
  }
})

test('status is the first footer item with its own dot and no tag placeholder when tags are absent', async () => {
  const status = host.querySelector('[data-conversation-status]')
  const footer = status.parentElement
  expect(footer.firstElementChild).toBe(status)
  expect(status.querySelector('[aria-hidden=true]').classList.contains('bg-current')).toBe(true)
  expect(status.classList.contains('shrink-0')).toBe(true)
  expect(status.classList.contains('text-[11px]')).toBe(true)
  props.conversation.tags = []
  await nextTick()
  expect(footer.children).toHaveLength(1)
  expect(footer.firstElementChild).toBe(status)
  expect(host.querySelector('[data-conversation-tags]')).toBeNull()
  expect(host.querySelector('[data-tag-separator]')).toBeNull()
})

test('keeps title, contact, preview and footer in one content column and unread with metadata', () => {
  const content = host.querySelector('.card-content')
  expect(content.querySelector('[data-conversation-title]')).not.toBeNull()
  expect(content.querySelector('.card-contact')).not.toBeNull()
  expect(content.querySelector('[data-conversation-preview]').parentElement).toBe(content)
  expect(content.querySelector('.card-footer').parentElement).toBe(content)
  const metadata = content.querySelector('.card-metadata')
  expect(metadata.querySelector('[data-conversation-unread]').textContent.trim()).toBe('12')
  expect(metadata.querySelector('[data-conversation-time]')).toBeNull()
  const time = content.querySelector('[data-conversation-preview] [data-conversation-time]')
  expect(time).not.toBeNull()
  expect(time.parentElement.classList.contains('truncate')).toBe(true)
  expect(content.querySelector('[data-conversation-preview] [data-conversation-unread]')).toBeNull()
  expect(content.querySelector('.card-avatar')).toBeNull()
})

test('uses last_message_at before the snippet and updates it with the latest list data', async () => {
  props.conversation.created_at = '2020-01-01T00:00:00Z'
  // Local dates keep expectations independent of the machine's timezone.
  props.conversation.last_message_at = new Date(2026, 10, 9, 19, 11).toISOString()
  await nextTick()
  const time = host.querySelector('[data-conversation-time]')
  expect(time.textContent).toBe('09.11.2026, 19:11 · ')
  expect(time.nextElementSibling.textContent).toBe(props.conversation.last_message)
  expect(time.classList.contains('text-muted-foreground/80')).toBe(true)
  props.conversation = {
    ...props.conversation,
    last_message_at: new Date(2027, 0, 1, 0, 9).toISOString(),
    last_message: 'Newest message'
  }
  await nextTick()
  expect(time.parentElement.textContent).toBe('01.01.2027, 00:09 · Newest message')
})

test.each([null, undefined, '', 'invalid'])(
  'shows only the snippet when last_message_at is %s',
  async (value) => {
    props.conversation.last_message_at = value
    props.conversation.created_at = '2020-01-01T00:00:00Z'
    await nextTick()
    expect(host.querySelector('[data-conversation-time]')).toBeNull()
    expect(host.querySelector('[data-conversation-preview] p').textContent).toBe(
      props.conversation.last_message
    )
  }
)

test.each([9, 10, 27, 125, 1024])(
  'shows the full unread count %s without a fixed width',
  async (count) => {
    props.conversation.unread_message_count = count
    await nextTick()
    const badge = host.querySelector('[data-conversation-unread]')
    expect(badge.textContent.trim()).toBe(String(count))
    expect(badge.classList.contains('w-5')).toBe(false)
    expect(badge.classList.contains('shrink-0')).toBe(true)
    expect(badge.classList.contains('whitespace-nowrap')).toBe(true)
  }
)

test('keeps active styling across refreshes and folder navigation, and separates bulk selection and focus', async () => {
  props.currentConversation = { uuid: 'ticket' }
  await nextTick()
  const card = host.querySelector('[data-conversation-card]')
  const expectActive = () => {
    expect(card.classList.contains('bg-primary/15')).toBe(true)
    expect(card.classList.contains('dark:bg-primary/20')).toBe(true)
    expect(card.classList.contains('before:bg-primary')).toBe(true)
    expect(card.classList.contains('hover:bg-accent/40')).toBe(false)
    expect(card.classList.contains('bg-primary/5')).toBe(false)
    expect(card.classList.contains('focus-visible:ring-2')).toBe(true)
  }
  expectActive()
  props.conversation = { ...props.conversation, last_message: 'Refreshed' }
  props.currentConversation = { uuid: 'ticket' }
  mocks.route.query = { folder: '11', folderName: 'Custom' }
  mocks.store.selected = true
  await nextTick()
  expectActive()
  expect(JSON.parse(card.dataset.route).query).toEqual({
    folder: '11',
    folderName: 'Custom',
    scrollTo: 'mention'
  })
  mocks.route.params = { type: 'assigned' }
  await nextTick()
  expectActive()
  expect(JSON.parse(card.dataset.route).name).toBe('inbox-conversation')
  props.currentConversation = { uuid: 'other' }
  await nextTick()
  expect(card.classList.contains('bg-primary/15')).toBe(false)
  expect(card.classList.contains('bg-primary/5')).toBe(true)
})
