// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useConversationStore } from './conversation'
import { reportConversationAccessDenied } from '../utils/conversation-access'
const mocks = vi.hoisted(() => ({ api: {}, emit: vi.fn(), route: {}, replace: vi.fn() }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: mocks.emit }) }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ roles: [], user: { id: 1 } }) }))
vi.mock('@main/stores/notification', () => ({ useNotificationStore: () => ({}) }))
vi.mock('@main/websocket', () => ({
  subscribeToConversation: vi.fn(),
  sendTypingIndicator: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/i18n', () => ({
  getI18n: () => ({ global: { t: () => 'Du hast keinen Zugriff mehr auf diese Konversation.' } })
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ currentRoute: { value: mocks.route }, replace: mocks.replace })
}))
const uuid = 'cf3525dd-c148-41b6-9070-5dcc2848d737'
const detail = (id = uuid) => ({
  data: { data: { uuid: id, subject: 'secret', custom_attributes: {} } }
})
const history = {
  data: {
    data: {
      results: [{ uuid: 'message', content: 'private-note', conversation_uuid: uuid }],
      page: 1,
      total_pages: 1
    }
  }
}
const denied = (status = 403) => ({ response: { status, data: { message: 'Permission denied' } } })
const deferred = () => {
  let resolve
  const promise = new Promise((r) => {
    resolve = r
  })
  return { promise, resolve }
}
let pinia
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  vi.clearAllMocks()
  mocks.route = {
    params: { uuid, type: 'visible' },
    path: `/inboxes/visible/conversation/${uuid}`,
    query: { status: 'Open' }
  }
  Object.assign(mocks.api, {
    getConversation: vi.fn().mockResolvedValue(detail()),
    getConversationMessages: vi.fn().mockResolvedValue(history),
    updateAssigneeLastSeen: vi.fn().mockResolvedValue({}),
    getConversationMessage: vi
      .fn()
      .mockResolvedValue({ data: { data: history.data.data.results[0] } })
  })
})
afterEach(() => pinia._e.stop())
test.each([403, 404])(
  'direct UUID / hard reload with HTTP %s clears details and closes the route',
  async (status) => {
    mocks.api.getConversation.mockRejectedValue(denied(status))
    const store = useConversationStore()
    await store.fetchConversation(uuid)
    expect(store.current).toEqual({})
    expect(store.isConversationOpen).toBe(false)
    expect(mocks.replace).toHaveBeenCalledWith({
      path: '/inboxes/visible',
      query: { status: 'Open' }
    })
    expect(mocks.emit).toHaveBeenCalledWith(
      'show-toast',
      expect.objectContaining({
        description: 'Du hast keinen Zugriff mehr auf diese Konversation.'
      })
    )
  }
)
test('cached reopen denial purges details, messages, drafts and selection while preserving pages/scroll', async () => {
  const store = useConversationStore()
  await store.fetchConversation(uuid)
  await store.fetchMessages(uuid)
  store.setDraft(uuid, 'reply', { content: 'secret-draft' })
  store.conversations.data = [{ uuid, subject: 'secret' }, { uuid: 'other' }]
  store.conversations.page = 3
  store.listPositions.set('kept', { scrollTop: 900 })
  store.toggleSelect(uuid)
  mocks.api.getConversation.mockRejectedValue(denied())
  await store.fetchConversation(uuid)
  expect(store.current).toEqual({})
  expect(store.messages.data.getAllPagesMessages(uuid)).toEqual([])
  expect(store.getDraft(uuid, 'reply')).toBeUndefined()
  expect(store.selectedCount).toBe(0)
  expect(store.conversations.data.map((row) => row.uuid)).toEqual(['other'])
  expect(store.conversations.page).toBe(3)
  expect(store.listPositions.get('kept')).toEqual({ scrollTop: 900 })
})
test('revocation clears an open ticket and ignores later message/property patches', async () => {
  const store = useConversationStore()
  await store.fetchConversation(uuid)
  await store.fetchMessages(uuid)
  store.mergeConversationUpdate({ uuid, access_revoked: true })
  mocks.api.getConversation.mockRejectedValue(denied())
  store.mergeMessageUpdate({ conversation_uuid: uuid, uuid: 'message', content: 'later-secret' })
  store.mergeConversationUpdate({ uuid, subject: 'later-secret' })
  await Promise.resolve()
  expect(store.current).toEqual({})
  expect(store.messages.data.getAllPagesMessages(uuid)).toEqual([])
})
test('late detail, history and individual-message responses cannot restore revoked data', async () => {
  const store = useConversationStore(),
    d = deferred(),
    h = deferred(),
    m = deferred()
  mocks.api.getConversation.mockReturnValueOnce(d.promise)
  mocks.api.getConversationMessages.mockReturnValueOnce(h.promise)
  mocks.api.getConversationMessage.mockReturnValueOnce(m.promise)
  const fetchingDetail = store.fetchConversation(uuid),
    fetchingHistory = store.fetchMessages(uuid)
  store.conversation.data = { uuid }
  const fetchingMessage = store.updateConversationMessage({
    conversation_uuid: uuid,
    uuid: 'new-message',
    type: 'incoming'
  })
  store.mergeConversationUpdate({ uuid, access_revoked: true })
  d.resolve(detail())
  h.resolve(history)
  m.resolve({ data: { data: { uuid: 'new-message', content: 'secret' } } })
  await Promise.all([fetchingDetail, fetchingHistory, fetchingMessage])
  expect(store.current).toEqual({})
  expect(store.messages.data.getAllPagesMessages(uuid)).toEqual([])
})
test('older navigation cannot replace a newly opened authorized ticket', async () => {
  const store = useConversationStore(),
    old = deferred()
  mocks.api.getConversation
    .mockReturnValueOnce(old.promise)
    .mockResolvedValueOnce(detail('current'))
  const first = store.fetchConversation(uuid)
  await store.fetchConversation('current')
  old.resolve(detail())
  await first
  expect(store.current.uuid).toBe('current')
})
test('new authorized detail check after reassignment restores access and history', async () => {
  const store = useConversationStore()
  store.revokeConversationAccess(uuid)
  await store.fetchConversation(uuid)
  await store.fetchMessages(uuid)
  expect(store.current.uuid).toBe(uuid)
  expect(store.messages.data.getAllPagesMessages(uuid)).toHaveLength(1)
})
test('component-owned denial rechecks access; action permission alone does not revoke a readable ticket', async () => {
  const store = useConversationStore()
  await store.fetchConversation(uuid)
  reportConversationAccessDenied({
    ...denied(),
    config: { url: `/api/v1/conversations/${uuid}/status` }
  })
  await Promise.resolve()
  expect(store.current.uuid).toBe(uuid)
  mocks.api.getConversation.mockRejectedValue(denied())
  reportConversationAccessDenied({
    ...denied(),
    config: { url: `/api/v1/conversations/${uuid}/messages` }
  })
  await Promise.resolve()
  await Promise.resolve()
  expect(store.isConversationOpen).toBe(false)
})
test('intercepted direct detail denial invalidates earlier pending requests', async () => {
  const store = useConversationStore(),
    pending = deferred()
  mocks.api.getConversation.mockReturnValueOnce(pending.promise)
  const fetching = store.fetchConversation(uuid)
  reportConversationAccessDenied({ ...denied(), config: { url: `/api/v1/conversations/${uuid}` } })
  pending.resolve(detail())
  await fetching
  expect(store.isConversationOpen).toBe(false)
})

test('a late list response cannot restore a revoked row; a new authorized server result can', async () => {
  const store = useConversationStore()
  const pending = deferred()
  const page = {
    data: { data: { results: [{ uuid, subject: 'secret' }], page: 1, total: 1, total_pages: 1 } }
  }
  mocks.api.getSidebarFolderConversations = vi.fn().mockReturnValueOnce(pending.promise)
  const fetching = store.fetchConversationsList(false, 'folder:customer:0:11')
  store.revokeConversationAccess(uuid)
  pending.resolve(page)
  await fetching
  expect(store.conversations.data).toEqual([])
  mocks.api.getSidebarFolderConversations.mockResolvedValueOnce(page)
  await store.fetchFirstPageConversations()
  expect(store.conversations.data.map((row) => row.uuid)).toEqual([uuid])
})
