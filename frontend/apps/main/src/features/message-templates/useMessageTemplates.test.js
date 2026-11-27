// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import { useMessageTemplates } from './useMessageTemplates'
import api from '@main/api'
vi.mock('@main/api', () => ({ default: { listMessageTemplates: vi.fn() } }))
let app, state, identity, context
function mount() {
  identity = ref('employee-1')
  context = ref('public-ticket')
  const root = document.createElement('div')
  app = createApp({
    setup() {
      state = useMessageTemplates(
        () => context.value,
        () => identity.value
      )
      return () => h('div')
    }
  })
  app.mount(root)
}
afterEach(() => {
  app?.unmount()
  vi.resetAllMocks()
})
test('keeps names without content and clears data on identity and permission changes', async () => {
  api.listMessageTemplates.mockResolvedValue({
    data: {
      data: {
        templates: [{ uuid: '1', name: 'Name', content: 'Secret' }],
        can_customer_support: true
      }
    }
  })
  mount()
  await state.load()
  expect(state.templates.value).toEqual([{ uuid: '1', name: 'Name' }])
  identity.value = 'employee-2'
  await nextTick()
  expect(state.templates.value).toEqual([])
  expect(state.canSupport.value).toBe(false)
  await state.load()
  identity.value = 'employee-2-revoked'
  await nextTick()
  expect(state.templates.value).toEqual([])
})
test('ignores late responses from old user or inbox and clears revoked responses', async () => {
  let resolve
  api.listMessageTemplates.mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r
      })
  )
  mount()
  const loading = state.load()
  context.value = 'private-ticket'
  await nextTick()
  resolve({
    data: { data: { templates: [{ uuid: '1', name: 'Hidden' }], can_customer_support: true } }
  })
  await loading
  expect(state.templates.value).toEqual([])
  api.listMessageTemplates.mockRejectedValue({
    response: { status: 403, data: { message: 'Rechte entzogen' } }
  })
  await state.load()
  expect(state.error.value).toBe('Rechte entzogen')
  expect(state.templates.value).toEqual([])
})
