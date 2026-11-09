// @vitest-environment jsdom
import { afterEach, expect, test } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import de from '../../../../../../../i18n/de-DE.json'
import en from '../../../../../../../i18n/en-US.json'
import ActivityMessageBubble from './ActivityMessageBubble.vue'

let app, host
const render = (message) => {
  const i18n = createI18n({ legacy: false, locale: 'de-DE', fallbackLocale: 'en-US',
    messages: { 'de-DE': de, 'en-US': en } })
  host = document.createElement('div')
  app = createApp({ render: () => h(ActivityMessageBubble, { message }) })
  app.use(i18n)
  app.mount(host)
  return i18n
}
afterEach(() => { app?.unmount(); host?.remove() })
const system = (content, type) => ({
  type: 'activity', private: true, sender_type: 'agent', content_type: 'text',
  content, updated_at: '2026-11-09T10:00:00Z',
  ...(type ? { meta: { activity_type: type } } : {})
})

test('renders the active locale and keeps the stored content while switching languages', async () => {
  const message = reactive(system('Daniel marked the conversation as Closed', 'status_change'))
  const i18n = render(message)
  expect(host.textContent).toContain('Daniel hat die Konversation als Geschlossen markiert')
  expect(host.textContent).toContain('09.11.2026')
  i18n.global.locale.value = 'en-US'
  await nextTick()
  expect(host.textContent).toContain(message.content)
  expect(message.content).toBe('Daniel marked the conversation as Closed')
})

test('historical events render in German and unknown/ambiguous formats keep their original text', async () => {
  const message = reactive(system('Daniel added tag Rekla'))
  render(message)
  expect(host.textContent).toContain('Daniel hat das Tag Rekla hinzugefügt')
  for (const content of ['Assigned to Support team by Daniel', 'Custom event', 'Daniel added tag A added tag B']) {
    message.content = content
    await nextTick()
    expect(host.textContent).toContain(content)
  }
})

test('dynamic names are escaped as text and normal/private messages never change', async () => {
  const message = reactive(system('Zoë added tag <script>alert(1)</script>', 'tag_added'))
  render(message)
  expect(host.textContent).toContain('Zoë hat das Tag <script>alert(1)</script> hinzugefügt')
  expect(host.querySelector('script')).toBeNull()
  message.content = 'Daniel added tag Rekla'
  message.type = 'outgoing'
  await nextTick()
  expect(host.textContent).toContain(message.content)
  expect(host.textContent).not.toContain('hinzugefügt')
})
