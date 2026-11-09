// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import NotificationBell from './NotificationBell.vue'

const mocks = vi.hoisted(() => ({ store: null, mobile: false }))
vi.mock('@main/stores/notification', () => ({ useNotificationStore: () => mocks.store }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@shared-ui/composables', () => ({ useIsMobile: () => mocks.mobile }))
vi.mock('./NotificationPanel.vue', () => ({ default: { render: () => null } }))
vi.mock('@shared-ui/components/ui/popover', () => {
  const wrapper = {
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.())
  }
  return { Popover: wrapper, PopoverTrigger: wrapper, PopoverContent: wrapper }
})
vi.mock('@shared-ui/components/ui/sidebar', () => ({
  SidebarMenuButton: {
    setup:
      (_, { slots }) =>
      () =>
        h('button', slots.default?.())
  }
}))

let app

afterEach(() => app?.unmount())

test.each([false, true])(
  'shows exact notification counts without a fixed width (mobile: %s) and still loads stats',
  async (mobile) => {
    mocks.mobile = mobile
    mocks.store = reactive({ unreadCount: 0, fetchStats: vi.fn() })
    const host = document.createElement('div')
    app = createApp({ render: () => h(NotificationBell) })
    app.mount(host)
    expect(mocks.store.fetchStats).toHaveBeenCalledOnce()
    expect(host.querySelector('[data-notification-count]')).toBeNull()
    for (const count of [9, 10, 27, 125, 1024]) {
      mocks.store.unreadCount = count
      await nextTick()
      const badge = host.querySelector('[data-notification-count]')
      expect(badge.textContent.trim()).toBe(String(count))
      expect(badge.classList.contains('min-w-3.5')).toBe(true)
      expect(badge.classList.contains('size-3.5')).toBe(false)
      expect(badge.classList.contains('absolute')).toBe(false)
      expect(badge.classList.contains('md:absolute')).toBe(true)
      if (mobile)
        expect(badge.previousElementSibling.textContent).toBe('globals.terms.notification')
    }
  }
)
