// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import mitt from 'mitt'
import ServiceEmailManager from './ServiceEmailManager.vue'

const mocks = vi.hoisted(() => ({ api: {}, refresh: vi.fn() }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => ({ refreshConversationListForRealtimeEvent: mocks.refresh }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key, args) => `${key}${args?.address ? ` ${args.address}` : ''}` }) }))
vi.mock('@shared-ui/utils/http', () => ({ handleHTTPError: (error) => ({ message: error.message }) }))
vi.mock('@shared-ui/components/ui/input', () => ({ Input: {
  props: ['modelValue'], emits: ['update:modelValue'],
  setup: (props, { emit }) => () => h('input', { value: props.modelValue, onInput: (e) => emit('update:modelValue', e.target.value) })
} }))

let app, host, emitter
async function flush() { for (let i = 0; i < 8; i++) await nextTick() }
beforeEach(async () => {
  vi.clearAllMocks()
  mocks.api.getServiceEmailAddresses = vi.fn().mockResolvedValue({ data: { data: ['existing@example.com'] } })
  mocks.api.addServiceEmailAddress = vi.fn().mockResolvedValue({ data: { data: true } })
  mocks.api.removeServiceEmailAddress = vi.fn().mockResolvedValue({ data: { data: true } })
  emitter = mitt()
  host = document.createElement('div')
  app = createApp(ServiceEmailManager)
  app.config.globalProperties.emitter = emitter
  app.mount(host)
  await flush()
})
afterEach(() => app.unmount())

test('uses the existing admin API and only refreshes after confirmed persistence', async () => {
  expect(host.textContent).toContain('existing@example.com')
  let resolve
  mocks.api.addServiceEmailAddress.mockImplementation(() => new Promise((done) => { resolve = done }))
  host.querySelector('input').value = ' new@example.com '
  host.querySelector('input').dispatchEvent(new Event('input'))
  host.querySelector('form').dispatchEvent(new Event('submit', { cancelable: true }))
  await flush()
  expect(mocks.api.addServiceEmailAddress).toHaveBeenCalledWith('new@example.com')
  expect(mocks.refresh).not.toHaveBeenCalled()
  expect(mocks.api.getServiceEmailAddresses).toHaveBeenCalledTimes(1)
  resolve({ data: { data: true } })
  await flush()
  expect(mocks.api.getServiceEmailAddresses).toHaveBeenCalledTimes(2)
  expect(mocks.refresh).toHaveBeenCalledTimes(1)
  expect(host.querySelector('input').value).toBe('')
})

test('requires confirmation before removal and keeps old messages outside management', async () => {
  host.querySelector('[aria-label="serviceEmails.removeAddress existing@example.com"]').click()
  await flush()
  expect(mocks.api.removeServiceEmailAddress).not.toHaveBeenCalled()
  const confirmation = host.querySelector('[role="alert"]')
  expect(confirmation.textContent).toContain('serviceEmails.confirmRemove existing@example.com')
  confirmation.querySelector('button').click()
  await flush()
  expect(mocks.api.removeServiceEmailAddress).toHaveBeenCalledWith('existing@example.com')
  expect(mocks.refresh).toHaveBeenCalledTimes(1)
})

test('shows save errors without reporting success or clearing the input', async () => {
  mocks.api.addServiceEmailAddress.mockRejectedValue(new Error('Duplicate address'))
  host.querySelector('input').value = 'existing@example.com'
  host.querySelector('input').dispatchEvent(new Event('input'))
  host.querySelector('form').dispatchEvent(new Event('submit', { cancelable: true }))
  await flush()
  expect(host.querySelector('[role="alert"]').textContent).toBe('Duplicate address')
  expect(host.querySelector('input').value).toBe('existing@example.com')
  expect(mocks.refresh).not.toHaveBeenCalled()
})

test('reloads after authorized websocket invalidation and unsubscribes on unmount', async () => {
  mocks.api.getServiceEmailAddresses.mockResolvedValue({ data: { data: ['remote@example.com'] } })
  emitter.emit('service-emails-changed')
  await flush()
  expect(host.textContent).toContain('remote@example.com')
  expect(host.textContent).not.toContain('existing@example.com')
  expect(emitter.all.get('service-emails-changed')).toHaveLength(1)
  app.unmount()
  expect(emitter.all.get('service-emails-changed')).toHaveLength(0)
})
