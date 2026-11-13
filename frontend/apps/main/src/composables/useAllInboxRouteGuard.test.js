import { afterEach, expect, test, vi } from 'vitest'
import { effectScope, nextTick, reactive } from 'vue'
import { useAllInboxRouteGuard } from './useAllInboxRouteGuard'

let scope
afterEach(() => scope?.stop())

function startGuard(path = '/inboxes/all/conversation/ticket-uuid') {
  const user = reactive({ userID: null, roles: [] })
  const route = reactive({ path, fullPath: `${path}?scrollTo=message-uuid` })
  const router = { replace: vi.fn() }
  scope = effectScope()
  scope.run(() => useAllInboxRouteGuard(user, route, router))
  return { user, route, router }
}

test('preserves an admin deep link while the profile loads and after it arrives', async () => {
  const { user, route, router } = startGuard()
  expect(router.replace).not.toHaveBeenCalled()
  Object.assign(user, { userID: 1, roles: ['Admin'] })
  await nextTick()
  expect(router.replace).not.toHaveBeenCalled()
  expect(route.fullPath).toBe('/inboxes/all/conversation/ticket-uuid?scrollTo=message-uuid')
})

test('redirects a non-admin as soon as the authenticated profile arrives', async () => {
  const { user, router } = startGuard()
  expect(router.replace).not.toHaveBeenCalled()
  Object.assign(user, { userID: 101, roles: ['Agent'] })
  await nextTick()
  expect(router.replace).toHaveBeenCalledExactlyOnceWith('/inboxes/assigned')
})

test('rechecks a revoked admin role without needing a route change', async () => {
  const { user, router } = startGuard()
  Object.assign(user, { userID: 1, roles: ['Admin'] })
  await nextTick()
  user.roles.splice(0, 1, 'Agent')
  await nextTick()
  expect(router.replace).toHaveBeenCalledExactlyOnceWith('/inboxes/assigned')
})

test('retains other inbox routes for non-admin users and rechecks navigation', async () => {
  const { user, route, router } = startGuard('/inboxes/visible/conversation/ticket-uuid')
  Object.assign(user, { userID: 101, roles: ['Agent'] })
  await nextTick()
  expect(router.replace).not.toHaveBeenCalled()
  route.path = '/inboxes/all'
  route.fullPath = '/inboxes/all?status=Open'
  await nextTick()
  expect(router.replace).toHaveBeenCalledExactlyOnceWith('/inboxes/assigned')
})
