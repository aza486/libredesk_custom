// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import ReplySourceSelector from './ReplySourceSelector.vue'
const { getSources } = vi.hoisted(() => ({ getSources: vi.fn() }))
vi.mock('@main/api', () => ({ default: { getReplySources: getSources } }))
vi.mock('@shared-ui/components/ui/select', () => {
  const wrap = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
  return {
    Select: defineComponent({
      props: ['modelValue', 'disabled'],
      emits: ['update:modelValue'],
      setup(p, { slots, emit }) {
        return () =>
          h('div', [
            h(
              'button',
              {
                'data-pick': '',
                disabled: p.disabled,
                onClick: () => emit('update:modelValue', 'older')
              },
              'Pick older'
            ),
            slots.default?.()
          ])
      }
    }),
    SelectContent: wrap,
    SelectItem: wrap,
    SelectTrigger: wrap,
    SelectValue: wrap
  }
})
let app, root, props, updates
const flush = async () => {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
function response(results, total = results.length, has_more = false) {
  return { data: { data: { results, total, has_more } } }
}
async function mount(options = {}) {
  props = reactive({
    conversationUuid: 'conversation',
    modelValue: '',
    lockedSource: '',
    ...options
  })
  updates = []
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      h(ReplySourceSelector, {
        ...props,
        'onUpdate:modelValue': (v) => {
          updates.push(v)
          props.modelValue = v
        }
      })
  })
  app.mount(root)
  await flush()
}
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.resetAllMocks()
})
test('multiple candidates never select latest; employee can choose older', async () => {
  getSources.mockResolvedValue(response([{ uuid: 'latest' }, { uuid: 'older' }]))
  await mount()
  expect(updates).toEqual([])
  expect(root.textContent).toContain('Mehrere Kundenmails')
  root.querySelector('[data-pick]').click()
  await flush()
  expect(updates).toEqual(['older'])
})
test('single candidate is selected without erasing an existing draft association', async () => {
  getSources.mockResolvedValue(response([{ uuid: 'only' }]))
  await mount({ modelValue: 'saved-source' })
  expect(updates).toEqual([])
  props.conversationUuid = 'next'
  props.modelValue = ''
  await flush()
  expect(updates).toEqual(['only'])
})
test('locked AI source cannot change and pagination does not infer latest', async () => {
  getSources
    .mockResolvedValueOnce(response([{ uuid: 'latest' }], 101, true))
    .mockResolvedValueOnce(response([{ uuid: 'older' }], 101, false))
  await mount({ lockedSource: 'original-ai-source' })
  expect(root.querySelector('[data-pick]').disabled).toBe(true)
  const more = [...root.querySelectorAll('button')].find(
    (b) => b.textContent === 'Weitere Kundenmails laden'
  )
  more.click()
  await flush()
  expect(getSources.mock.calls[1][1].page).toBe(2)
  expect(updates).toEqual([])
})
test('switching ignores stale responses and preserves saved association', async () => {
  let resolve
  getSources
    .mockImplementationOnce(() => new Promise((r) => (resolve = r)))
    .mockResolvedValue(response([{ uuid: 'new-source' }]))
  await mount({ modelValue: 'draft-source' })
  props.conversationUuid = 'next'
  await flush()
  resolve(response([{ uuid: 'stale-source' }]))
  await flush()
  expect(updates).toEqual([])
  expect(props.modelValue).toBe('draft-source')
})
