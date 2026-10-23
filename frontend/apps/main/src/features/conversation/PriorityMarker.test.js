// @vitest-environment jsdom
import { expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import PriorityMarker from './PriorityMarker.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: () => 'Priorität' }) }))

test('priority bars follow existing names and react to updates without guessing missing levels', async () => {
  const props = reactive({ priority: '' })
  const host = document.createElement('div')
  const app = createApp({
    render: () => h(TooltipProvider, null, { default: () => h(PriorityMarker, props) })
  })
  app.mount(host)
  try {
    expect(host.querySelector('svg')).toBeNull()
    for (const [priority, active, color] of [
      ['Low', 1, 'text-muted-foreground'],
      ['Medium', 2, 'text-warning-600'],
      ['High', 3, 'text-destructive'],
      ['  HIGH  ', 3, 'text-destructive']
    ]) {
      props.priority = priority
      await nextTick()
      const icon = host.querySelector('svg')
      expect(icon.getAttribute('aria-label')).toBe(`Priorität: ${priority.trim()}`)
      expect(icon.classList.contains(color)).toBe(true)
      expect(icon.querySelectorAll('rect.fill-current')).toHaveLength(active)
      expect(icon.querySelectorAll('rect.fill-border')).toHaveLength(3 - active)
      expect([...icon.querySelectorAll('rect')].map((bar) => bar.getAttribute('height'))).toEqual([
        '4',
        '8',
        '12'
      ])
    }
    props.priority = 'Custom priority'
    await nextTick()
    expect(host.querySelectorAll('rect')).toHaveLength(0)
    expect(host.querySelector('circle')).not.toBeNull()
    props.priority = null
    await nextTick()
    expect(host.querySelector('svg')).toBeNull()
  } finally {
    app.unmount()
  }
})
