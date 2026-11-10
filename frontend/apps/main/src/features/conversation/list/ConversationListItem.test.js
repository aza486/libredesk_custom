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

test('keeps routes and unread indication while replacing previews with independent times', async () => {
  const card = host.querySelector('[data-conversation-card]')
  expect(JSON.parse(card.dataset.route)).toEqual({
    name: 'personal-inbox-conversation',
    params: { uuid: 'ticket', inboxID: '7' },
    query: { status: 'Open', scrollTo: 'mention' }
  })
  expect(card.textContent).toContain('12')
  expect(host.querySelector('[data-conversation-preview]')).toBeNull()
  expect(card.textContent).not.toContain(props.conversation.last_message)
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

test('preserves compact typing and draft indicators, title fallback and status colours', async () => {
  mocks.store.draft = { content: '<p>Draft preview</p>' }
  await nextTick()
  expect(host.querySelector('[data-conversation-editing]').textContent).toContain(de['globals.terms.draft'])
  expect(host.textContent).not.toContain('Draft preview')
  mocks.store.typingByUUID.ticket = true
  await nextTick()
  expect(host.querySelector('[data-conversation-editing]').textContent).toContain(
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

test('places customer reply left and activity right below badges, with responsive equal columns', () => {
  const content = host.querySelector('.card-content')
  const times = content.querySelector('[data-conversation-times]')
  expect(content.querySelector('[data-conversation-title]')).not.toBeNull()
  expect(content.querySelector('.card-contact')).not.toBeNull()
  expect(times.previousElementSibling.classList.contains('card-footer')).toBe(true)
  expect(times.firstElementChild.matches('[data-customer-reply]')).toBe(true)
  expect(times.lastElementChild.matches('[data-last-activity]')).toBe(true)
  expect(times.firstElementChild.textContent).toContain('Letzte Kundenantwort')
  expect(times.lastElementChild.textContent).toContain('Letzte Aktivität')
  expect(times.classList.contains('grid-cols-2')).toBe(true)
  expect(times.classList.contains('min-w-0')).toBe(true)
  expect(times.classList.contains('border-t')).toBe(true)
  expect(times.lastElementChild.classList.contains('border-l')).toBe(true)
  expect(content.querySelector('.card-metadata [data-conversation-unread]')).not.toBeNull()
})

test('formats local German dates and updates each timestamp independently', async () => {
  props.conversation.created_at = new Date(2026, 10, 7, 9, 15).toISOString()
  props.conversation.last_message_at = new Date(2026, 10, 9, 19, 11).toISOString()
  await nextTick()
  const customer = host.querySelector('[data-customer-reply-time]')
  const activity = host.querySelector('[data-activity-time]')
  expect(customer.textContent).toBe('07.11.2026, 09:15')
  expect(activity.textContent).toBe('09.11.2026, 19:11')
  props.conversation.last_message_at = new Date(2026, 10, 9, 20, 12).toISOString()
  await nextTick()
  expect(customer.textContent).toBe('07.11.2026, 09:15')
  expect(activity.textContent).toBe('09.11.2026, 20:12')
  props.conversation.last_customer_reply_at = props.conversation.last_message_at
  await nextTick()
  expect(customer.textContent).toBe(activity.textContent)
  expect(customer.dateTime).toBe(props.conversation.last_customer_reply_at)
})

test.each([null, undefined, ''])(
  'falls back to creation for missing customer or activity timestamp %s', async (value) => {
    props.conversation.created_at = new Date(2026, 10, 7, 9, 15).toISOString()
    props.conversation.last_customer_reply_at = value
    props.conversation.last_message_at = value
    await nextTick()
    expect(host.querySelector('[data-customer-reply-time]').textContent).toBe('07.11.2026, 09:15')
    expect(host.querySelector('[data-activity-time]').textContent).toBe('07.11.2026, 09:15')
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
    expect(card.classList.contains('before:border-primary')).toBe(true)
    expect(card.classList.contains('before:border-l-2')).toBe(true)
    expect(card.classList.contains('before:inset-0')).toBe(true)
    expect(card.classList.contains('before:rounded-[inherit]')).toBe(true)
    expect(card.classList.contains('overflow-hidden')).toBe(false)
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
  expect(card.classList.contains('before:border-l-2')).toBe(false)
  expect(card.classList.contains('bg-primary/5')).toBe(true)
})
