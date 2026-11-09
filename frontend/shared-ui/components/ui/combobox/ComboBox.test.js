// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import ComboBox from './ComboBox.vue'

const state = vi.hoisted(() => ({ search: null }))
vi.mock('../popover', async () => {
  const { h } = await import('vue')
  const wrapper = { setup: (_, { slots }) => () => h('div', slots.default?.()) }
  return { Popover: wrapper, PopoverTrigger: wrapper, PopoverContent: wrapper }
})
vi.mock('../command', async () => {
  const { h } = await import('vue')
  const wrapper = { setup: (_, { slots }) => () => h('div', slots.default?.()) }
  return {
    Command: {
      emits: ['update:searchTerm'],
      setup: (_, { slots, emit }) => {
        state.search = (value) => emit('update:searchTerm', value)
        return () => h('div', slots.default?.())
      }
    }, CommandInput: { render: () => null }, CommandEmpty: wrapper,
    CommandList: wrapper, CommandGroup: wrapper,
    CommandItem: {
      props: ['value'], emits: ['select'],
      setup: (props, { slots, emit }) => () => h('button', {
        'data-option': '', onClick: () => emit('select', { detail: { value: props.value } })
      }, slots.default?.())
    }
  }
})
let app, host
const mount = (items, selected = '') => {
  const value = ref(selected), select = vi.fn()
  host = document.createElement('div')
  app = createApp({ render: () => h(ComboBox, {
    items, modelValue: value.value, 'onUpdate:modelValue': (next) => { value.value = next }, onSelect: select
  }) })
  app.config.globalProperties.$t = (key) => key
  app.mount(host)
  return { value, select }
}
afterEach(() => { app?.unmount(); host?.remove() })

test('localized selection and search keep original emitted labels and ID values', async () => {
  const items = [{ label: 'Closed', displayLabel: 'Geschlossen', value: '2' },
    { label: 'Resolved', displayLabel: 'Gelöst', value: '3' },
    { label: 'Open Projects', displayLabel: 'Open Projects', value: '9' }]
  const before = structuredClone(items)
  const { value, select } = mount(items, '2')
  expect(host.querySelector('[role=combobox]').textContent).toContain('Geschlossen')
  state.search('gelöst')
  await nextTick()
  expect(host.querySelectorAll('[data-option]')).toHaveLength(1)
  host.querySelector('[data-option]').click()
  await nextTick()
  expect(value.value).toBe('3')
  expect(select).toHaveBeenLastCalledWith({ label: 'Resolved', value: '3' })
  expect(host.querySelector('[role=combobox]').textContent).toContain('Gelöst')
  state.search('Closed')
  await nextTick()
  expect(host.querySelector('[data-option]').textContent).toContain('Geschlossen')
  expect(items).toEqual(before)
})

test('generic names without displayLabel continue to render and emit unchanged', () => {
  const { select } = mount([{ label: 'Sales Team', value: '7' }], '7')
  expect(host.querySelector('[role=combobox]').textContent).toContain('Sales Team')
  host.querySelector('[data-option]').click()
  expect(select).toHaveBeenCalledWith({ label: 'Sales Team', value: '7' })
})


test('priority display text never becomes the emitted action label', async () => {
  const { value, select } = mount([{ label: 'high', displayLabel: 'Hoch', value: '3' },
    { label: 'medium', displayLabel: 'Mittel', value: '2' }], '3')
  expect(host.querySelector('[role=combobox]').textContent).toContain('Hoch')
  state.search('mittel')
  await nextTick()
  host.querySelector('[data-option]').click()
  expect(value.value).toBe('2')
  expect(select).toHaveBeenLastCalledWith({ label: 'medium', value: '2' })
})
