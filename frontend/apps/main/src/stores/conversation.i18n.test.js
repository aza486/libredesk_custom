// @vitest-environment jsdom
import de from '../../../../../i18n/de-DE.json'
import { beforeEach, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { initI18n } from '../i18n'
import { useConversationStore } from './conversation'

const mocks = vi.hoisted(() => ({ api: {}, emit: vi.fn() }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: mocks.emit }) }))
vi.mock('@main/stores/user', () => ({
  useUserStore: () => ({ userID: 1, roles: ['Admin'], user: { id: 1 } })
}))
vi.mock('@main/stores/notification', () => ({ useNotificationStore: () => ({ markAssignmentAsReadForConversation: vi.fn() }) }))
vi.mock('@main/websocket', () => ({
  subscribeToConversation: vi.fn(), sendTypingIndicator: vi.fn(), subscribeListReplace: vi.fn()
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ currentRoute: { value: { params: {}, query: {} } } })
}))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

test('German labels leave status names, IDs and assigned-inbox filter payloads intact', async () => {
  const instance = initI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } })
  const statuses = [{ id: 1, name: 'Open' }, { id: 2, name: 'Closed' },
    { id: 3, name: 'Snoozed' }, { id: 4, name: 'Resolved' }, { id: 8, name: 'Open Projects' }]
  mocks.api.getStatuses = vi.fn().mockResolvedValue({ data: { data: statuses } })
  mocks.api.getAssignedConversations = vi.fn().mockResolvedValue({
    data: { data: { results: [{ uuid: 'original', status: 'Open', subject: 'Open Projects', assigned_user_id: 1 }],
      page: 1, total: 1, total_pages: 1 } }
  })
  const store = useConversationStore()
  await store.fetchStatuses()
  expect(store.statusOptions).toEqual(statuses.map((status) => ({
    label: status.name, value: String(status.id),
    displayLabel: ({ Open: 'Offen', Closed: 'Geschlossen', Snoozed: 'Schlummernd', Resolved: de['globals.terms.resolved'] })[status.name] || status.name
  })))
  expect(store.statusOptionsNoSnooze.map((option) => option.label)).toEqual(['Open', 'Closed', 'Resolved', 'Open Projects'])
  expect(instance.global.t('globals.terms.closed')).toBe('Geschlossen')
  store.conversations.status = 'Open'
  await store.fetchConversationsList(false, 'assigned')
  expect(mocks.api.getAssignedConversations).toHaveBeenCalledOnce()
  const request = mocks.api.getAssignedConversations.mock.calls[0][0]
  expect(JSON.parse(request.filters)).toContainEqual({
    model: 'conversation_statuses', field: 'name', operator: 'equals', value: 'Open'
  })
  expect(store.conversationsList[0]).toMatchObject({ status: 'Open', subject: 'Open Projects' })
  expect(store.getListSortField).not.toContain('globals.')
  // Using the untranslated option label must preserve each status-update payload.
  store.conversation.data = { uuid: 'original', status: 'Open' }
  mocks.api.updateConversationStatus = vi.fn().mockResolvedValue({})
  for (const name of ['Closed', 'Resolved', 'Open Projects']) {
    const option = store.statusOptions.find((status) => status.label === name)
    await store.updateStatus(option.label)
    expect(mocks.api.updateConversationStatus).toHaveBeenLastCalledWith('original', { status: name })
    expect(store.current.status).toBe(name)
  }
  expect(statuses.map((status) => status.name)).toEqual(['Open', 'Closed', 'Snoozed', 'Resolved', 'Open Projects'])
  instance.global.locale.value = 'en-US'
  instance.global.setLocaleMessage('en-US', { 'globals.terms.closed': 'Closed' })
  expect(store.statusOptions.find((status) => status.label === 'Closed').displayLabel).toBe('Closed')
})

test('German priority options preserve saved names, IDs, sidebar high queries and folder filters', async () => {
  initI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } })
  const rows = [{ id: 1, name: 'Low' }, { id: 2, name: 'Medium' }, { id: 3, name: 'High' },
    { id: 4, name: 'high' }, { id: 9, name: 'Customer Critical' }]
  const snapshot = structuredClone(rows)
  mocks.api.getPriorities = vi.fn().mockResolvedValue({ data: { data: rows } })
  mocks.api.updateConversationPriority = vi.fn().mockResolvedValue({})
  const list = { data: { data: { results: [], page: 1, total: 0, total_pages: 0 } } }
  mocks.api.getAssignedConversations = vi.fn().mockResolvedValue(list)
  mocks.api.getSidebarFolderConversations = vi.fn().mockResolvedValue(list)
  const store = useConversationStore()
  await store.fetchPriorities()
  expect(store.priorityOptions.map((option) => option.displayLabel)).toEqual(['Niedrig', 'Mittel', 'Hoch', 'Hoch', 'Customer Critical'])
  store.conversation.data = { uuid: 'priority-ticket', priority: 'Low', priority_id: '1' }
  for (const option of store.priorityOptions) {
    // Same raw fields used by the sidebar's selectPriority handler.
    store.current.priority = option.label
    store.current.priority_id = option.value
    await store.updatePriority(option.label)
    expect(mocks.api.updateConversationPriority).toHaveBeenLastCalledWith('priority-ticket', { priority: option.label })
    expect(store.current.priority).toBe(rows.find((row) => String(row.id) === option.value).name)
  }
  const filter = [{ model: 'conversations', field: 'priority_id', operator: 'equals', value: '2' }]
  await store.fetchConversationsList(false, 'assigned-high', 0, filter)
  expect(mocks.api.getAssignedConversations).toHaveBeenLastCalledWith(expect.objectContaining({ priority: 'high' }))
  expect(JSON.parse(mocks.api.getAssignedConversations.mock.calls[0][0].filters)).toContainEqual(filter[0])
  await store.fetchConversationsList(false, 'folder:customer:0:11', 0, filter)
  const request = mocks.api.getSidebarFolderConversations.mock.calls[0]
  expect(request.slice(0, 3)).toEqual(['customer', '0', '11'])
  // Exclusive folder definitions are evaluated by the server, without inline filters.
  expect(request[3]).not.toHaveProperty('filters')
  expect(store.conversations.listFilters).toEqual(filter)
  expect(rows).toEqual(snapshot)
  expect(store.priorityOptions.map(({ label, value }) => ({ label, value }))).toEqual(rows.map((row) => ({ label: row.name, value: String(row.id) })))
})

test('customer timestamps survive employee and status updates without per-ticket API calls', () => {
  const store = useConversationStore()
  store.conversations.data = [{ uuid: 'times', status: 'Open', created_at: '2026-11-07T09:15:00Z',
    last_customer_reply_at: '2026-11-07T09:15:00Z', last_message_at: '2026-11-07T09:15:00Z' }]
  store.conversations.status = ''
  const row = store.conversations.data[0]
  const apiCalls = () => Object.values(mocks.api).filter(vi.isMockFunction).reduce((sum, fn) => sum + fn.mock.calls.length, 0)
  const before = apiCalls()
  store.updateConversationLastMessage('times', { type: 'outgoing', sender_type: 'agent', private: false, created_at: '2026-11-08T10:00:00Z', content: 'Employee' })
  expect(row.last_customer_reply_at).toBe('2026-11-07T09:15:00Z')
  expect(row.last_message_at).toBe('2026-11-08T10:00:00Z')
  store.handleConvPush({ uuid: 'times', status: 'Closed', last_message_at: '2026-11-08T11:00:00Z', last_customer_reply_at: row.last_customer_reply_at })
  expect(row.last_customer_reply_at).toBe('2026-11-07T09:15:00Z')
  expect(row.last_message_at).toBe('2026-11-08T11:00:00Z')
  store.updateConversationLastMessage('times', { type: 'incoming', sender_type: 'contact', private: false, created_at: '2026-11-09T12:00:00Z', content: 'Customer' })
  expect(row.last_customer_reply_at).toBe('2026-11-09T12:00:00Z')
  expect(row.last_message_at).toBe(row.last_customer_reply_at)
  store.updateConversationLastMessage('times', { type: 'incoming', sender_type: 'contact', private: false, created_at: '2026-11-01T12:00:00Z', content: 'Delayed' })
  expect(row.last_customer_reply_at).toBe('2026-11-09T12:00:00Z')
  expect(row.last_message_at).toBe(row.last_customer_reply_at)
  expect(apiCalls()).toBe(before)
})


test.each([
  ['newest_customer_reply', 'desc', 'Letzte Kundenantwort', [3, 2, 1, 5, 4]],
  ['oldest_customer_reply', 'asc', 'Älteste Kundenantwort', [1, 2, 3, 4, 5]]
])('customer reply sort %s uses the shared server contract and stable realtime ordering', async (sort, order, label, expected) => {
  const instance = initI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } })
  mocks.api.getAssignedConversations = vi.fn().mockResolvedValue({ data: { data: { results: [], total: 0, page: 1, total_pages: 0 } } })
  const store = useConversationStore()
  store.conversations.sortField = sort
  expect(store.getListSortField).toBe(label)
  await store.fetchConversationsList(false, 'assigned')
  expect(mocks.api.getAssignedConversations.mock.calls[0][0]).toMatchObject({ order_by: 'customer_reply.last_customer_reply_at', order })
  store.conversations.status = ''
  store.conversations.data = [
    { id: 4, assigned_user_id: 1, uuid: 'empty', last_customer_reply_at: null, created_at: '2026-11-12T12:00:00Z' },
    { id: 2, assigned_user_id: 1, uuid: 'b', last_customer_reply_at: '2026-11-12T09:00:00Z' },
    { id: 1, assigned_user_id: 1, uuid: 'a', last_customer_reply_at: '2026-11-12T08:00:00Z', last_message_at: '2026-11-12T10:00:00Z' },
    { id: 3, assigned_user_id: 1, uuid: 'tie', last_customer_reply_at: '2026-11-12T10:00:00+01:00' },
    { id: 5, assigned_user_id: 1, uuid: 'empty2', last_customer_reply_at: null }
  ]
  expect(store.conversationsList.map(c => c.id)).toEqual(expected)
  for (const privateNote of [false, true]) {
    store.updateConversationLastMessage('a', { type: 'outgoing', sender_type: 'agent', private: privateNote, created_at: '2026-11-12T13:00:00Z' })
    expect(store.conversationsList.map(c => c.id)).toEqual(expected)
  }
  instance.global.setLocaleMessage('en-US', { 'conversation.sort.newestCustomerReply': 'Latest customer reply', 'conversation.sort.oldestCustomerReply': 'Oldest customer reply' })
  instance.global.locale.value = 'en-US'
  expect(store.getListSortField).toBe(order === 'asc' ? 'Oldest customer reply' : 'Latest customer reply')
  expect(store.conversations.sortField).toBe(sort)
})
