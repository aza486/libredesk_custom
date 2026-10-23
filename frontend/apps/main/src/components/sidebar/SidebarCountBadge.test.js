// @vitest-environment jsdom
import { expect, test } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import SidebarCountBadge from './SidebarCountBadge.vue'

test('sidebar badge preserves visibility and 9+ threshold during count updates', async () => {
  const props = reactive({ count: 0 })
  const host = document.createElement('div')
  const app = createApp({ render: () => h(SidebarCountBadge, props) })
  app.mount(host)
  try {
    expect(host.textContent).toBe('')
    for (const [count, label] of [
      [1, '1'],
      [9, '9'],
      [10, '9+'],
      [125, '9+']
    ]) {
      props.count = count
      await nextTick()
      expect(host.textContent.trim()).toBe(label)
      expect(host.firstElementChild.classList.contains('ml-auto')).toBe(true)
      expect(host.firstElementChild.classList.contains('bg-background')).toBe(true)
      expect(host.firstElementChild.classList.contains('bg-primary')).toBe(false)
    }
    props.count = 0
    await nextTick()
    expect(host.firstElementChild).toBeNull()
  } finally {
    app.unmount()
  }
})
