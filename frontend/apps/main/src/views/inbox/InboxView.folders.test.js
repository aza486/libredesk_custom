// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import InboxView from './InboxView.vue'
const mocks = vi.hoisted(() => ({ route: null, store: null }))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 101 }) }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@vueuse/core', async () => {
  const { ref } = await import('vue')
  return {
    useIntervalFn: () => ({ pause: () => {}, resume: () => {} }),
    useDocumentVisibility: () => ref('visible')
  }
})
vi.mock('@/features/conversation/ConversationPlaceholder.vue', () => ({
  default: { render: () => null }
}))
let app, host
async function flush() {
  for (let i = 0; i < 5; i++) await nextTick()
}
beforeEach(() => {
  mocks.route = reactive({
    name: 'inbox',
    params: { type: 'assigned' },
    query: { folder: '11', status: 'Open', scopeID: '999' }
  })
  mocks.store = {
    conversations: { initialized: false },
    setListStatus: vi.fn(),
    fetchConversationsList: vi.fn(),
    refreshConversationList: vi.fn()
  }
})
afterEach(() => {
  app?.unmount()
  host?.remove()
})
function mount() {
  host = document.createElement('div')
  app = createApp({ render: () => h(InboxView) })
  app.component('router-view', { render: () => null })
  app.mount(host)
}
test('assigned folder routes bind the logged-in employee and omit implicit status filters', async () => {
  mount()
  await flush()
  expect(mocks.store.setListStatus).toHaveBeenLastCalledWith('', false)
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(
    true,
    'folder:my_inbox:101:11'
  )
  mocks.route.query = { status: 'Open' }
  await flush()
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(true, 'assigned', 0, [])
})
test('visible and visible-internal remain separate routes and cannot select my_inbox folders', async () => {
  mocks.route.params.type = 'visible'
  mount()
  await flush()
  expect(mocks.store.fetchConversationsList).not.toHaveBeenCalled()
  mocks.route.query = {}
  await flush()
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(true, 'visible', 0, [])
  mocks.route.params.type = 'visible-internal'
  await flush()
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(
    true,
    'visible-internal',
    0,
    []
  )
})

test('redistributed folder routes bind their fixed global scope without a status filter', async () => {
  mocks.route.params.type = 'customer-redistributed'
  mount()
  await flush()
  expect(mocks.store.setListStatus).toHaveBeenLastCalledWith('', false)
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(true, 'folder:redistributed:0:11')
  mocks.route.query = {}
  await flush()
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(true, 'customer-redistributed', 0, [])
  mocks.route.params.type = 'customer-redistributed-unfiled'
  await flush()
  expect(mocks.store.fetchConversationsList).toHaveBeenLastCalledWith(true, 'customer-redistributed-unfiled', 0, [])
})
