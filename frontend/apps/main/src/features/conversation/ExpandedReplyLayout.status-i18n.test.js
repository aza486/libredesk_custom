// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import de from '../../../../../../i18n/de-DE.json'
import ExpandedReplyLayout from './ExpandedReplyLayout.vue'

const state = vi.hoisted(() => ({ store: null, emit: vi.fn() }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => state.store }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: state.emit }) }))
vi.mock('@shared-ui/composables', async () => {
  const { ref } = await import('vue')
  return { useIsMobile: () => ref(false) }
})
// Render menu slots directly to exercise real click handlers independently of portals.
vi.mock('@shared-ui/components/ui/dropdown-menu', async () => {
  const { h } = await import('vue')
  const wrapper = { setup: (_, { slots }) => () => h('div', slots.default?.()) }
  return {
    DropdownMenu: wrapper, DropdownMenuContent: wrapper, DropdownMenuTrigger: wrapper,
    DropdownMenuItem: { setup: (_, { slots }) => () => h('button', { 'data-status-option': '' }, slots.default?.()) }
  }
})
vi.mock('./PriorityMarker.vue', () => ({ default: { render: () => null } }))
let app, host
const render = () => {
  state.emit.mockClear()
  state.store = reactive({
    current: { status: 'Closed', subject: 'Open Projects', priority: 'high' }, currentContactName: 'Daniel',
    statusOptions: [
      { value: '1', label: 'Open', displayLabel: 'Offen' },
      { value: '2', label: 'Closed', displayLabel: 'Geschlossen' },
      { value: '3', label: 'Resolved', displayLabel: de['globals.terms.resolved'] },
      { value: '4', label: 'Snoozed', displayLabel: 'Schlummernd' },
      { value: '9', label: 'Open Projects', displayLabel: 'Open Projects' }
    ], updateStatus: vi.fn()
  })
  host = document.createElement('div')
  app = createApp({ render: () => h(ExpandedReplyLayout) })
  app.use(createI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } }))
  app.mount(host)
}
afterEach(() => { app?.unmount(); host?.remove() })

test('German labels send original names; Snoozed still opens its command', () => {
  render()
  const buttons = [...host.querySelectorAll('[data-status-option]')]
  expect(buttons.map((button) => button.textContent.trim())).toEqual(['Offen', 'Geschlossen', de['globals.terms.resolved'], 'Schlummernd', 'Open Projects'])
  for (const [index, original] of [[0, 'Open'], [1, 'Closed'], [2, 'Resolved'], [4, 'Open Projects']]) {
    buttons[index].click()
    expect(state.store.updateStatus).toHaveBeenLastCalledWith(original)
  }
  const calls = state.store.updateStatus.mock.calls.length
  buttons[3].click()
  expect(state.store.updateStatus).toHaveBeenCalledTimes(calls)
  expect(state.emit).toHaveBeenCalledWith(expect.any(String), { command: 'snooze', open: true })
})

test('header distinguishes Closed/Resolved and preserves custom names and subjects', async () => {
  render()
  expect(host.querySelector('header').textContent).toContain('Geschlossen')
  expect(host.querySelector('header').textContent).toContain('Hoch')
  expect(host.querySelector('header').textContent).toContain('Open Projects')
  state.store.current.status = 'Resolved'
  await nextTick()
  expect(host.querySelector('header button').textContent).toContain(de['globals.terms.resolved'])
  state.store.current.status = 'Closed Projects'
  await nextTick()
  expect(host.querySelector('header button').textContent).toContain('Closed Projects')
})
