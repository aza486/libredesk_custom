// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import SidebarFolderForm from './SidebarFolderForm.vue'
const mocks = vi.hoisted(() => ({ create: vi.fn(), update: vi.fn() }))
vi.mock('@main/api', () => ({
  default: { createSidebarFolder: mocks.create, updateSidebarFolder: mocks.update }
}))
vi.mock('@main/composables/useConversationFilters', async () => {
  const { computed } = await import('vue')
  return {
    useConversationFilters: () => ({
      conversationsListFilters: computed(() => ({
        subject: { label: 'Betreff', type: 'text', operators: ['contains'] }
      }))
    })
  }
})
vi.mock('@shared-ui/components/ui/dialog', async () => {
  const { h } = await import('vue')
  const wrapper = {
    setup:
      (_, { slots }) =>
      () =>
        h('div', {}, slots.default?.())
  }
  return {
    Dialog: wrapper,
    DialogContent: wrapper,
    DialogHeader: wrapper,
    DialogTitle: wrapper,
    DialogDescription: wrapper,
    DialogFooter: wrapper
  }
})
vi.mock('@main/components/filter/FilterGroupBuilder.vue', async () => {
  const { h } = await import('vue')
  return {
    default: {
      emits: ['update:modelValue'],
      setup:
        (_, { emit }) =>
        () =>
          h(
            'button',
            {
              type: 'button',
              'data-set-filter': '',
              onClick: () =>
                emit('update:modelValue', {
                  logic: 'AND',
                  rules: [
                    {
                      logic: 'AND',
                      rules: [
                        {
                          model: 'conversations',
                          field: 'subject',
                          operator: 'contains',
                          value: 'Rekla'
                        }
                      ]
                    }
                  ]
                })
            },
            'Set filter'
          )
    }
  }
})
let app, host
async function flush() {
  for (let i = 0; i < 8; i++) await nextTick()
}
afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.clearAllMocks()
})

test('the existing dialog creates my_inbox folders with a fixed scope and default badge', async () => {
  mocks.create.mockResolvedValue({ data: { data: [] } })
  const props = reactive({
    open: false,
    scope: 'my_inbox',
    scopeId: 101,
    scopeName: 'Mein Posteingang',
    folderCount: 0
  })
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({ render: () => h(SidebarFolderForm, props) })
  app.mount(host)
  props.open = true
  await flush()
  const inputs = host.querySelectorAll('input')
  expect(inputs[1].readOnly).toBe(true)
  expect(inputs[1].value).toBe('Mein Posteingang')
  expect(host.querySelector('input[type="checkbox"]').checked).toBe(true)
  inputs[0].value = 'Reklamationen'
  inputs[0].dispatchEvent(new Event('input', { bubbles: true }))
  host.querySelector('[data-set-filter]').click()
  await flush()
  host.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flush()
  expect(mocks.create).toHaveBeenCalledWith(
    'my_inbox',
    101,
    expect.objectContaining({ name: 'Reklamationen', show_count: true, sort_order: 0 })
  )
})

test('priority can still be changed in the dialog without sidebar arrow controls', async () => {
  mocks.update.mockResolvedValue({ data: { data: [] } })
  const props = reactive({
    open: false,
    scope: 'team',
    scopeId: 7,
    scopeName: 'Team',
    folderCount: 3,
    folder: {
      id: 11,
      name: 'Rekla',
      sort_order: 2,
      show_count: false,
      filters: [{ model: 'conversations', field: 'subject', operator: 'contains', value: 'Rekla' }]
    }
  })
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({ render: () => h(SidebarFolderForm, props) })
  app.mount(host)
  props.open = true
  await flush()
  const position = host.querySelector('input[type="number"]')
  position.value = '1'
  position.dispatchEvent(new Event('input', { bubbles: true }))
  await flush()
  host.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flush()
  expect(mocks.update).toHaveBeenCalledWith(
    'team',
    7,
    11,
    expect.objectContaining({ sort_order: 0, show_count: false })
  )
})
