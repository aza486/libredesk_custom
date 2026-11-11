// @vitest-environment jsdom
import { beforeEach, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useConversationStore } from './conversation'
const mocks = vi.hoisted(() => ({ api: {}, emit: vi.fn() }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: mocks.emit }) }))
vi.mock('@main/stores/user', () => ({
  useUserStore: () => ({ roles: ['Admin'], user: { id: 1 } })
}))
vi.mock('@main/stores/notification', () => ({ useNotificationStore: () => ({}) }))
vi.mock('@main/websocket', () => ({
  subscribeToConversation: vi.fn(),
  sendTypingIndicator: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/i18n', () => ({ getI18n: () => null }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ currentRoute: { value: { params: {}, query: {} } } })
}))
const response = (rows, page = 1, total = rows.length) => ({
  data: { data: { results: rows, page, total, total_pages: Math.ceil(total / 25) } }
})
beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  mocks.api.getSidebarFolderConversations = vi.fn()
})

test('a tag-only folder displays open and closed tickets and never appends a status filter', async () => {
  mocks.api.getSidebarFolderConversations.mockResolvedValue(
    response([
      { uuid: 'open', status: 'Open' },
      { uuid: 'closed', status: 'Closed' }
    ])
  )
  const store = useConversationStore()
  await store.fetchConversationsList(false, 'folder:customer:0:11')
  expect(store.conversations.status).toBe('')
  expect(store.conversationsList.map((c) => c.uuid)).toEqual(['open', 'closed'])
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenCalledWith(
    'customer',
    '0',
    '11',
    expect.not.objectContaining({ filters: expect.anything() })
  )
})

test('a server refresh removes moved tickets while retaining the loaded page range', async () => {
  const store = useConversationStore()
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(
    response([{ uuid: 'moved', status: 'Open' }], 1, 26)
  )
  await store.fetchConversationsList(false, 'folder:team:7:11')
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(
    response([{ uuid: 'later-page', status: 'Open' }], 2, 26)
  )
  await store.fetchConversationsList(false, 'folder:team:7:11', 0, [], 0, 2)
  expect(store.conversations.data).toHaveLength(2)
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(
    response([{ uuid: 'kept', status: 'Closed' }])
  )
  await store.fetchFirstPageConversations()
  expect(store.conversations.data.map((c) => c.uuid)).toEqual(['kept'])
  expect(store.conversations.page).toBe(2)
  expect(store.conversations.total).toBe(1)
  expect(store.conversations.hasMore).toBe(false)
})

test('responses from a previous folder cannot overwrite the newly selected scope', async () => {
  let finishOld
  mocks.api.getSidebarFolderConversations.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finishOld = resolve
      })
  )
  const store = useConversationStore()
  const pending = store.fetchConversationsList(false, 'folder:team:7:11')
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(
    response([{ uuid: 'current', status: 'Open' }])
  )
  await store.fetchConversationsList(false, 'folder:personal:8:12')
  finishOld(response([{ uuid: 'old', status: 'Open' }]))
  await pending
  expect(store.conversationsList.map((c) => c.uuid)).toEqual(['current'])
})

test('my_inbox folder requests preserve open/closed and explicitly shared personal tickets', async () => {
  mocks.api.getSidebarFolderConversations.mockResolvedValue(
    response([
      { uuid: 'closed', status: 'Closed' },
      {
        uuid: 'shared',
        status: 'Open',
        inbox_access_mode: 'personal',
        inbox_owner_user_id: 99,
        custom_attributes: { access_mode: 'personal' }
      }
    ])
  )
  const store = useConversationStore()
  await store.fetchConversationsList(false, 'folder:my_inbox:1:11')
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenCalledWith(
    'my_inbox',
    '1',
    '11',
    expect.any(Object)
  )
  expect(store.conversationsList.map((c) => c.uuid)).toEqual(['closed', 'shared'])
})

test('assigned first-page refresh removes tickets claimed by a my_inbox folder', async () => {
  const store = useConversationStore()
  mocks.api.getAssignedConversations = vi
    .fn()
    .mockResolvedValueOnce(response([{ uuid: 'claimed', status: 'Open', assigned_user_id: 1 }]))
    .mockResolvedValueOnce(response([]))
  await store.fetchConversationsList(false, 'assigned')
  expect(store.conversations.data).toHaveLength(1)
  await store.fetchFirstPageConversations()
  expect(store.conversations.data).toEqual([])
  expect(store.conversations.total).toBe(0)
})

const tickets = (start, count) => Array.from({ length: count }, (_, index) => ({
  uuid: `ticket-${start + index}`, status: 'Open', last_message_at: start + index
}))

async function loadPages(store, listType = 'folder:customer:0:11', pages = 4) {
  for (let page = 1; page <= pages; page++) {
    mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(
      response(tickets((page - 1) * 25, 25), page, Math.max(150, pages * 25 + 25))
    )
    if (page === 1) await store.fetchConversationsList(false, listType)
    else await store.fetchNextConversations()
  }
}

test('opening and switching tickets keeps all loaded pages without another list request', async () => {
  const store = useConversationStore()
  await loadPages(store)
  mocks.api.getConversation = vi.fn(async (uuid) => ({ data: { data: { uuid } } }))
  await store.fetchConversation('ticket-65')
  await store.fetchConversation('ticket-66')
  await store.fetchConversationsList(true, 'folder:customer:0:11')
  expect(store.conversations.data).toHaveLength(100)
  expect(store.conversations.page).toBe(4)
  expect(store.conversations.hasMore).toBe(true)
  expect(store.conversations.loading).toBe(false)
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenCalledTimes(4)
})

test('closing a folder ticket validates the loaded window and keeps pagination and unique rows', async () => {
  const store = useConversationStore()
  await loadPages(store)
  const kept = tickets(0, 101).filter((ticket) => ticket.uuid !== 'ticket-65')
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce({
    data: { data: { results: kept, page: 1, total: 149, total_pages: 2 } }
  })
  await store.fetchFirstPageConversations()
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenLastCalledWith(
    'customer', '0', '11', expect.objectContaining({ page: 1, page_size: 100 })
  )
  expect(store.conversations.data).toHaveLength(100)
  expect(store.conversations.data.some((ticket) => ticket.uuid === 'ticket-65')).toBe(false)
  expect(store.conversations.data.some((ticket) => ticket.uuid === 'ticket-90')).toBe(true)
  expect(store.conversations.page).toBe(4)
  expect(store.conversations.hasMore).toBe(true)
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(response(tickets(100, 25), 5, 149))
  await store.fetchNextConversations()
  expect(new Set(store.conversations.data.map((ticket) => ticket.uuid)).size)
    .toBe(store.conversations.data.length)
})

test('returning to a folder restores its loaded snapshot before revalidating it', async () => {
  const store = useConversationStore()
  await loadPages(store)
  const firstKey = store.listKey
  store.listPositions.set(firstKey, { scrollTop: 6500, candidates: [] })
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(response([{ uuid: 'other' }]))
  await store.fetchConversationsList(false, 'folder:team:7:12')
  const secondKey = store.listKey
  store.listPositions.set(secondKey, { scrollTop: 250, candidates: [] })
  let finish
  mocks.api.getSidebarFolderConversations.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
  const pending = store.fetchConversationsList(true, 'folder:customer:0:11')
  expect(store.listKey).toBe(firstKey)
  expect(store.conversations.data).toHaveLength(100)
  expect(store.conversations.page).toBe(4)
  expect(store.conversations.loading).toBe(false)
  expect(store.listPositions.get(firstKey).scrollTop).toBe(6500)
  expect(store.listPositions.get(secondKey).scrollTop).toBe(250)
  finish({ data: { data: { results: tickets(0, 100), total: 150, total_pages: 2, page: 1 } } })
  await pending
})

test('changing a search filter invalidates in-flight results even within the same list type', async () => {
  const store = useConversationStore()
  let finish
  mocks.api.getAllConversations = vi.fn()
    .mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
    .mockResolvedValueOnce(response([{ uuid: 'new-result', status: 'Open' }]))
  const filter = (value) => [{ model: 'conversations', field: 'subject', operator: 'contains', value }]
  const old = store.fetchConversationsList(false, 'all', 0, filter('old'))
  const oldKey = store.listKey
  await store.fetchConversationsList(false, 'all', 0, filter('new'))
  finish(response([{ uuid: 'old-result', status: 'Open' }]))
  await old
  expect(store.listKey).not.toBe(oldKey)
  expect(store.conversationsList.map((ticket) => ticket.uuid)).toEqual(['new-result'])
  expect(store.conversations.page).toBe(1)
})

test('an older refresh of the same list cannot overwrite a newer refresh', async () => {
  const store = useConversationStore()
  await loadPages(store, undefined, 1)
  let finish
  mocks.api.getSidebarFolderConversations
    .mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
    .mockResolvedValueOnce(response([{ uuid: 'latest' }]))
  const old = store.fetchConversationsList(false, 'folder:customer:0:11', 0, [], 0, 1, 0, true)
  await store.fetchConversationsList(false, 'folder:customer:0:11', 0, [], 0, 1, 0, true)
  finish(response([{ uuid: 'stale' }]))
  await old
  expect(store.conversations.data.map((ticket) => ticket.uuid)).toEqual(['latest'])
  expect(store.conversations.fetching).toBe(false)
})

test('a refresh waits for load more so that its newly loaded range is retained', async () => {
  const store = useConversationStore()
  await loadPages(store, undefined, 1)
  let finish
  mocks.api.getSidebarFolderConversations
    .mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
    .mockResolvedValueOnce({ data: { data: { results: tickets(0, 50), page: 1, total: 150, total_pages: 3 } } })
  const next = store.fetchNextConversations()
  const refresh = store.fetchFirstPageConversations()
  finish(response(tickets(25, 25), 2, 150))
  await next
  await refresh
  expect(store.conversations.data).toHaveLength(50)
  expect(store.conversations.page).toBe(2)
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenLastCalledWith(
    'customer', '0', '11', expect.objectContaining({ page_size: 50 })
  )
})

test('an explicit reset discards the saved position and loaded pages', async () => {
  const store = useConversationStore()
  await loadPages(store)
  store.listPositions.set(store.listKey, { scrollTop: 6500 })
  const revision = store.listRevision
  store.resetConversations()
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(response(tickets(0, 25), 1, 150))
  await store.fetchConversationsList(true, 'folder:customer:0:11')
  expect(store.listPositions.has(store.listKey)).toBe(false)
  expect(store.listRevision).toBe(revision + 1)
  expect(store.conversations.page).toBe(1)
  expect(store.conversations.data).toHaveLength(25)
})

test('large loaded windows respect the server page-size limit and are replaced atomically', async () => {
  const store = useConversationStore()
  await loadPages(store, undefined, 21)
  let finish, started
  const ready = new Promise((resolve) => { started = resolve })
  mocks.api.getSidebarFolderConversations
    .mockResolvedValueOnce({ data: { data: { results: tickets(1, 500), page: 1, total: 550, total_pages: 2 } } })
    .mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; started() }))
  const pending = store.fetchFirstPageConversations()
  await ready
  expect(store.conversations.data[0].uuid).toBe('ticket-0')
  finish({ data: { data: { results: tickets(501, 49), page: 2, total: 550, total_pages: 2 } } })
  await pending
  expect(store.conversations.page).toBe(21)
  expect(store.conversations.data).toHaveLength(525)
  expect(store.conversations.data[0].uuid).toBe('ticket-1')
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenLastCalledWith(
    'customer', '0', '11', expect.objectContaining({ page: 2, page_size: 500 })
  )
})

test('incoming message updates reorder loaded rows without resetting pages or fetching the list', async () => {
  const store = useConversationStore()
  mocks.api.getAllConversations = vi.fn().mockResolvedValue(response(tickets(0, 25), 1, 150))
  await store.fetchConversationsList(false, 'all')
  for (let page = 2; page <= 4; page++) {
    mocks.api.getAllConversations.mockResolvedValueOnce(response(tickets((page - 1) * 25, 25), page, 150))
    await store.fetchNextConversations()
  }
  store.handleConvPush({ uuid: 'ticket-65', last_message_at: '2026-11-09T12:00:00Z', last_message: 'new' })
  expect(store.conversationsList[0].uuid).toBe('ticket-65')
  expect(store.conversations.page).toBe(4)
  expect(store.conversations.data).toHaveLength(100)
  expect(mocks.api.getAllConversations).toHaveBeenCalledTimes(4)
})

test('closing a ticket in an open list removes it while keeping the other loaded pages', async () => {
  const store = useConversationStore()
  mocks.api.getAllConversations = vi.fn().mockResolvedValue(response(tickets(0, 25), 1, 150))
  await store.fetchConversationsList(false, 'all')
  for (let page = 2; page <= 4; page++) {
    mocks.api.getAllConversations.mockResolvedValueOnce(response(tickets((page - 1) * 25, 25), page, 150))
    await store.fetchNextConversations()
  }
  store.mergeConversationUpdate({ uuid: 'ticket-65', status: 'Closed' })
  expect(store.conversationsList.some((ticket) => ticket.uuid === 'ticket-65')).toBe(false)
  expect(store.conversationsList.some((ticket) => ticket.uuid === 'ticket-90')).toBe(true)
  expect(store.conversations.page).toBe(4)
  expect(store.conversations.hasMore).toBe(true)
  expect(mocks.api.getAllConversations).toHaveBeenCalledTimes(4)
})

test('a slow refresh cannot undo a newer incoming message or status update', async () => {
  const store = useConversationStore()
  mocks.api.getAllConversations = vi.fn().mockResolvedValue(response(tickets(0, 25), 1, 150))
  await store.fetchConversationsList(false, 'all')
  let finish
  mocks.api.getAllConversations.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
  const pending = store.fetchFirstPageConversations()
  store.handleConvPush({ uuid: 'ticket-15', last_message_at: '2026-11-09T12:00:00Z', last_message: 'new' })
  store.mergeConversationUpdate({ uuid: 'ticket-16', status: 'Closed' })
  finish(response(tickets(0, 25), 1, 150))
  await pending
  expect(store.conversationsList[0]).toMatchObject({ uuid: 'ticket-15', last_message: 'new' })
  expect(store.conversationsList.some((ticket) => ticket.uuid === 'ticket-16')).toBe(false)
  expect(mocks.api.getAllConversations).toHaveBeenCalledTimes(2)
})

test('Spam tag updates from another employee remove loaded rows and refresh totals without resetting pagination', async () => {
  const store = useConversationStore()
  mocks.api.getAllConversations = vi.fn().mockResolvedValue(response(tickets(0, 25), 1, 150))
  await store.fetchConversationsList(false, 'all')
  for (let page = 2; page <= 4; page++) {
    mocks.api.getAllConversations.mockResolvedValueOnce(response(tickets((page - 1) * 25, 25), page, 150))
    await store.fetchNextConversations()
  }
  mocks.api.getAllConversations.mockResolvedValue(response(tickets(0, 25), 1, 149))
  store.mergeConversationUpdate({ uuid: 'ticket-65', tags: ['🗑Spam', 'Rekla'], status: 'Closed' })
  expect(store.conversationsList.some((ticket) => ticket.uuid === 'ticket-65')).toBe(false)
  expect(store.conversations.page).toBe(4)
  expect(store.conversationsList.some((ticket) => ticket.uuid === 'ticket-90')).toBe(true)
  expect(mocks.emit).toHaveBeenCalledWith(expect.any(String), { model: 'inbox-list' })
})

test('Spam wins over custom folders immediately on WebSocket updates', async () => {
  const store = useConversationStore()
  mocks.api.getSidebarFolderConversations.mockResolvedValue(response([{ uuid: 'rekla', status: 'Open', tags: ['Rekla'] }]))
  await store.fetchConversationsList(false, 'folder:customer:0:11')
  mocks.api.getSidebarFolderConversations.mockResolvedValue(response([]))
  store.mergeConversationUpdate({ uuid: 'rekla', status: 'Closed', tags: ['Rekla', '🗑Spam'] })
  expect(store.conversationsList).toEqual([])
  expect(store.conversations.total).toBe(0)
})

test('an Open Spam push cannot enter a regular All list', async () => {
  const store = useConversationStore()
  mocks.api.getAllConversations = vi.fn().mockResolvedValue(response([]))
  await store.fetchConversationsList(false, 'all')
  store.handleConvPush({ uuid: 'spam', status: 'Open', tags: [{ id: 12, name: '🗑Spam' }] })
  expect(store.conversationsList).toEqual([])
})

test('removing Spam leaves the customer Spam view immediately, including originally Closed tickets', async () => {
  const store = useConversationStore()
  store.conversations.status = ''
  const filters = [{ model: 'conversations', field: 'tags', operator: 'contains', value: '[12]' }]
  mocks.api.getCustomerConversations = vi.fn().mockResolvedValue(response([{ uuid: 'spam', status: 'Closed', tags: ['🗑Spam', 'Rekla'] }]))
  await store.fetchConversationsList(false, 'customer', 0, filters)
  expect(store.conversationsList).toHaveLength(1)
  mocks.api.getCustomerConversations.mockResolvedValue(response([]))
  store.mergeConversationUpdate({ uuid: 'spam', tags: ['Rekla'] })
  expect(store.conversationsList).toEqual([])
  expect(store.conversations.total).toBe(0)
})

test('redistributed uses the authorized customer endpoint and server flags for active and snoozed work', async () => {
  const store = useConversationStore()
  store.setListStatus('', false)
  mocks.api.getCustomerConversations = vi.fn().mockResolvedValue(
    response([
      { uuid: 'external', status: 'Open', customer_redistributed_active: true },
      { uuid: 'sleeping', status: 'Snoozed', customer_redistributed_active: true }
    ])
  )
  await store.fetchConversationsList(false, 'customer-redistributed')
  expect(mocks.api.getCustomerConversations).toHaveBeenCalledWith(
    expect.objectContaining({ routing: 'redistributed', page_size: 25 })
  )
  expect(store.conversationsList.map((c) => c.uuid)).toEqual(['external', 'sleeping'])
  store.mergeConversationUpdate({
    uuid: 'external',
    status: 'Open',
    customer_redistributed_active: false,
    routing_changed: true
  })
  expect(store.conversationsList.map((c) => c.uuid)).toEqual(['sleeping'])
  expect(store.conversations.total).toBe(1)
})

test('routing removes external tickets from four loaded customer pages while preserving positions', async () => {
  const store = useConversationStore()
  mocks.api.getCustomerConversations = vi.fn()
  for (let page = 1; page <= 4; page++) {
    mocks.api.getCustomerConversations.mockResolvedValueOnce(
      response(tickets((page - 1) * 25, 25), page, 150)
    )
    await store.fetchConversationsList(false, 'customer', 0, [], 0, page)
  }
  const key = store.listKey
  store.listPositions.set(key, { scrollTop: 2200, candidates: [] })
  mocks.api.getCustomerConversations.mockResolvedValue(response(tickets(0, 25), 1, 149))
  store.mergeConversationUpdate({
    uuid: 'ticket-65',
    customer_redistributed_active: true,
    routing_changed: true
  })
  expect(store.conversationsList.some((c) => c.uuid === 'ticket-65')).toBe(false)
  expect(store.conversationsList.some((c) => c.uuid === 'ticket-90')).toBe(true)
  expect(store.conversations.page).toBe(4)
  expect(store.listPositions.get(key).scrollTop).toBe(2200)
  expect(mocks.emit).toHaveBeenCalledWith(expect.any(String), { model: 'inbox-list' })
})

test('Spam and completion override active redistributed lists immediately', async () => {
  const store = useConversationStore()
  store.setListStatus('', false)
  mocks.api.getCustomerConversations = vi.fn().mockResolvedValue(
    response([
      { uuid: 'closed', status: 'Open', customer_redistributed_active: true },
      { uuid: 'spam', status: 'Open', customer_redistributed_active: true }
    ])
  )
  await store.fetchConversationsList(false, 'customer-redistributed')
  store.mergeConversationUpdate({
    uuid: 'closed',
    status: 'Closed',
    customer_redistributed_active: false
  })
  store.mergeConversationUpdate({
    uuid: 'spam',
    status: 'Closed',
    customer_redistributed_active: true,
    tags: ['🗑Spam']
  })
  expect(store.conversationsList).toEqual([])
  expect(store.conversations.total).toBe(0)
})

test('routing changes trigger server evaluation of custom folders without resetting loaded pages', async () => {
  const store = useConversationStore()
  await loadPages(store)
  const revision = store.listRevision
  mocks.api.getSidebarFolderConversations.mockResolvedValue(response(tickets(0, 25)))
  store.mergeConversationUpdate({
    uuid: 'ticket-30',
    customer_redistributed_active: true,
    routing_changed: true
  })
  expect(store.conversations.page).toBe(4)
  expect(store.listRevision).toBe(revision)
  expect(mocks.emit).toHaveBeenCalledWith(expect.any(String), { model: 'inbox-list' })
})

test('redistributed unfiled uses server winner selection and preserves loaded pages and position during return', async () => {
  const store = useConversationStore()
  store.setListStatus('', false)
  const rows = (start, count) => tickets(start, count).map((row) => ({ ...row, customer_redistributed_active: true }))
  mocks.api.getCustomerConversations = vi.fn().mockResolvedValueOnce(response(rows(0, 25), 1, 60))
  await store.fetchConversationsList(false, 'customer-redistributed-unfiled')
  mocks.api.getCustomerConversations.mockResolvedValueOnce(response(rows(25, 25), 2, 60))
  await store.fetchNextConversations()
  expect(mocks.api.getCustomerConversations).toHaveBeenLastCalledWith(expect.objectContaining({ routing: 'redistributed', folder_mode: 'unfiled', page: 2 }))
  store.listPositions.set(store.listKey, { scrollTop: 1700, candidates: [] })
  mocks.api.getCustomerConversations.mockResolvedValue(response(rows(0, 25), 1, 59))
  store.mergeConversationUpdate({ uuid: 'ticket-30', customer_redistributed_active: false, routing_changed: true })
  expect(store.conversationsList.some((r) => r.uuid === 'ticket-30')).toBe(false)
  expect(store.conversationsList.some((r) => r.uuid === 'ticket-45')).toBe(true)
  expect(store.conversations.page).toBe(2)
  expect(store.listPositions.get(store.listKey).scrollTop).toBe(1700)
})

test('redistributed custom folder requests reuse the existing scoped endpoint', async () => {
  const store = useConversationStore()
  mocks.api.getSidebarFolderConversations.mockResolvedValue(response([{ uuid: 'external', status: 'Snoozed', customer_redistributed_active: true }]))
  await store.fetchConversationsList(false, 'folder:redistributed:0:11')
  expect(mocks.api.getSidebarFolderConversations).toHaveBeenCalledWith('redistributed', '0', '11', expect.not.objectContaining({ filters: expect.anything() }))
  expect(store.conversationsList).toHaveLength(1)
  store.mergeConversationUpdate({ uuid: 'external', tags: ['🗑Spam'], status: 'Closed' })
  expect(store.conversationsList).toEqual([])
})
