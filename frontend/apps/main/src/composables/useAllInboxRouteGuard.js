import { watch } from 'vue'

// The user profile is loaded asynchronously on a full page navigation. An empty
// initial roles array must not be mistaken for an authenticated non-admin user.
export function useAllInboxRouteGuard(userStore, route, router) {
  return watch(
    [() => route.fullPath, () => userStore.userID, () => userStore.roles.includes('Admin')],
    () => {
      if (
        userStore.userID &&
        route.path.startsWith('/inboxes/all') &&
        !userStore.roles.includes('Admin')
      ) {
        router.replace('/inboxes/assigned')
      }
    },
    { immediate: true }
  )
}
