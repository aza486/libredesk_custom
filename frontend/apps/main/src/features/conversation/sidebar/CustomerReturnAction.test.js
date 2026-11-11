// @vitest-environment jsdom
import { beforeEach, afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import CustomerReturnAction from './CustomerReturnAction.vue'
const mocks = vi.hoisted(() => ({ store: null, can: vi.fn(), api: {}, emit: vi.fn() }))
vi.mock('@/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ can: mocks.can }) }))
vi.mock('@/api', () => ({ default: mocks.api }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit: mocks.emit }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@shared-ui/utils/http', () => ({
  handleHTTPError: (error) => ({ message: error.message })
}))
let app, host
const settle = async () => {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
function mount(props = {}) {
  host = document.createElement('div')
  app = createApp({ render: () => h(CustomerReturnAction, props) })
  app.mount(host)
}
beforeEach(() => {
  vi.clearAllMocks()
  mocks.store = reactive({
    current: {
      uuid: 'ticket',
      can_return_to_support: true,
      customer_ticket: true,
      customer_redistributed: true,
      tags: ['🦽Kundenticket'],
      assigned_team_id: 202
    },
    refreshConversationListForRealtimeEvent: vi.fn()
  })
  mocks.can.mockReturnValue(true)
  mocks.api.returnCustomerToSupport = vi.fn().mockResolvedValue({})
  mocks.api.getConversation = vi.fn().mockResolvedValue({
    data: {
      data: {
        uuid: 'ticket',
        customer_ticket: true,
        customer_redistributed: false,
        can_return_to_support: false,
        status: 'Open',
        assigned_team_id: null,
        assigned_user_ids: []
      }
    }
  })
})
afterEach(() => {
  app?.unmount()
  app = null
})
test('uses server-authorized return capability for an external employee without generic write rights', async () => {
  mocks.can.mockReturnValue(false)
  mount()
  expect(host.querySelector('button')).not.toBeNull()
  mocks.store.current.tags.push('🗑Spam')
  await nextTick()
  expect(host.querySelector('button')).toBeNull()
  mocks.store.current.tags.pop()
  mocks.store.current.can_return_to_support = false
  await nextTick()
  expect(host.querySelector('button')).toBeNull()
})
test('waits for pending assignment, prevents duplicate clicks, refreshes state and shows success after commit', async () => {
  let finish
  const beforeReturn = vi.fn(
    () =>
      new Promise((resolve) => {
        finish = resolve
      })
  )
  const close = vi.fn()
  mount({ beforeReturn, close })
  host.querySelector('button').click()
  await settle()
  expect(mocks.api.returnCustomerToSupport).not.toHaveBeenCalled()
  expect(host.querySelector('button').disabled).toBe(true)
  host.querySelector('button').click()
  finish()
  await settle()
  expect(mocks.api.returnCustomerToSupport).toHaveBeenCalledExactlyOnceWith('ticket')
  expect(mocks.store.current).toMatchObject({
    status: 'Open',
    assigned_team_id: null,
    assigned_user_ids: []
  })
  expect(close).toHaveBeenCalledOnce()
  expect(mocks.emit).toHaveBeenCalledWith(expect.any(String), {
    description: 'conversation.routing.returnSuccess'
  })
  expect(mocks.store.refreshConversationListForRealtimeEvent).toHaveBeenCalledOnce()
  expect(host.querySelector('button')).toBeNull()
})
test('a failed return has no success toast or optimistic state change', async () => {
  mocks.api.returnCustomerToSupport.mockRejectedValue(new Error('Denied'))
  const close = vi.fn()
  mount({ close })
  host.querySelector('button').click()
  await settle()
  expect(close).not.toHaveBeenCalled()
  expect(mocks.api.getConversation).not.toHaveBeenCalled()
  expect(mocks.store.current.assigned_team_id).toBe(202)
  expect(mocks.emit).toHaveBeenCalledExactlyOnceWith(expect.any(String), {
    variant: 'destructive',
    description: 'Denied'
  })
  expect(host.querySelector('button').disabled).toBe(false)
})
test('navigation during a request does not overwrite another conversation', async () => {
  let finish
  mocks.api.returnCustomerToSupport.mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      })
  )
  mount()
  host.querySelector('button').click()
  await settle()
  mocks.store.current = { uuid: 'other', status: 'Closed' }
  finish()
  await settle()
  expect(mocks.store.current).toEqual({ uuid: 'other', status: 'Closed' })
})
