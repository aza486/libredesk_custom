// @vitest-environment jsdom
import { expect, test } from 'vitest'
import { createI18n } from 'vue-i18n'
import de from '../../../../../../i18n/de-DE.json'
import { createApp, h, nextTick, reactive } from 'vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import PriorityMarker from './PriorityMarker.vue'


test.each(['light', 'dark'])('German priority labels preserve bars and colours in %s mode', async (theme) => {
  document.documentElement.classList.toggle('dark', theme === 'dark')
  const props = reactive({ priority: '' })
  const host = document.createElement('div')
  const app = createApp({
    render: () => h(TooltipProvider, null, { default: () => h(PriorityMarker, props) })
  })
  app.use(createI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } }))
  app.mount(host)
  try {
    expect(host.querySelector('svg')).toBeNull()
    for (const [priority, active, color] of [
      ['low', 1, 'text-muted-foreground'],
      ['Low', 1, 'text-muted-foreground'],
      ['medium', 2, 'text-warning-600'],
      ['Medium', 2, 'text-warning-600'],
      ['high', 3, 'text-destructive'],
      ['High', 3, 'text-destructive'],
      ['  HIGH  ', 3, 'text-destructive']
    ]) {
      props.priority = priority
      await nextTick()
      const icon = host.querySelector('svg')
      expect(icon.getAttribute('aria-label')).toBe(`Priorität: ${({ low: 'Niedrig', Low: 'Niedrig', medium: 'Mittel', Medium: 'Mittel', high: 'Hoch', High: 'Hoch' })[priority] ?? priority}`)
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
    document.documentElement.classList.remove('dark')
    app.unmount()
  }
})
