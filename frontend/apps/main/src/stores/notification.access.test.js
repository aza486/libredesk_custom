// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useNotificationStore } from './notification'
const mocks = vi.hoisted(() => ({ api: {} }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
let pinia
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => pinia._e.stop())
test('revocation purges loaded previews and late responses until access is validated again', async () => {
  const store = useNotificationStore()
  store.addNotification({ id: 1, conversation_uuid: 'revoked', body: 'secret' })
  store.addNotification({ id: 2, conversation_uuid: 'kept' })
  let finish
  mocks.api.getNotifications = vi.fn(
    () =>
      new Promise((resolve) => {
        finish = resolve
      })
  )
  const fetching = store.fetchNotifications()
  store.removeConversationNotifications('revoked')
  expect(store.notifications.map((n) => n.id)).toEqual([2])
  expect(store.unreadCount).toBe(1)
  finish({ data: { data: [{ id: 1, conversation_uuid: 'revoked', body: 'secret' }] } })
  await fetching
  expect(store.notifications).toEqual([])
  store.addNotification({ id: 3, conversation_uuid: 'revoked' })
  expect(store.notifications).toEqual([])
  store.restoreConversationNotifications('revoked')
  store.addNotification({ id: 4, conversation_uuid: 'revoked' })
  expect(store.notifications.map((n) => n.id)).toEqual([4])
})
