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

test('a server refresh removes tickets moved out of a folder and resets pagination to the refreshed first page', async () => {
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
  expect(store.conversations.page).toBe(1)
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
