// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import AIStyleDialog from './AIStyleDialog.vue'
const { adjust } = vi.hoisted(() => ({ adjust: vi.fn() }))
vi.mock('@main/api', () => ({ default: { adjustAIStyle: adjust } }))
vi.mock('@shared-ui/components/ui/dialog', () => {
  const wrapper = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
  return {
    Dialog: defineComponent({
      props: ['open'],
      setup(p, { slots }) {
        return () => (p.open ? h('div', slots.default?.()) : null)
      }
    }),
    DialogContent: wrapper,
    DialogHeader: wrapper,
    DialogTitle: wrapper,
    DialogDescription: wrapper,
    DialogFooter: wrapper
  }
})
vi.mock('@shared-ui/components/ui/select', () => {
  const wrapper = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
  return {
    Select: wrapper,
    SelectContent: wrapper,
    SelectTrigger: wrapper,
    SelectValue: wrapper,
    SelectItem: wrapper
  }
})
let app, root
const props = reactive({
  open: false,
  conversationUUID: 'conversation',
  note: { noteUUID: 'note', suggestion: 'Original\n<script>unsafe</script>' }
})
const adopted = vi.fn()
const flush = async () => {
  for (let i = 0; i < 8; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
async function mount() {
  adjust.mockReset()
  adopted.mockReset()
  props.open = false
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      h(AIStyleDialog, { ...props, 'onUpdate:open': (v) => (props.open = v), onAdopt: adopted })
  })
  app.mount(root)
  props.open = true
  await flush()
}
function click(text) {
  const b = [...root.querySelectorAll('button')].find((b) => b.textContent.trim() === text)
  expect(b).toBeTruthy()
  b.click()
}
const result = (text) => ({
  data: { success: true, request_id: 'request', adapted_text: text, profile_source: 'preset' }
})
afterEach(() => {
  app?.unmount()
  root?.remove()
})
test('iterations, safe preview, compare, undo, reset and discard', async () => {
  await mount()
  expect(root.querySelector('script')).toBeNull()
  adjust.mockResolvedValueOnce(result('First')).mockResolvedValueOnce(result('Second'))
  click('Anpassen')
  await flush()
  click('Anpassen')
  await flush()
  expect(adjust.mock.calls[1][1]).toMatchObject({
    original_text: props.note.suggestion,
    current_text: 'First',
    style: 'mine'
  })
  expect(root.querySelector('[aria-label="Antwortvorschau"]').textContent).toBe('Second')
  click('Original')
  await flush()
  expect(root.querySelector('[aria-label="Antwortvorschau"]').textContent).toBe(
    props.note.suggestion
  )
  click('Angepasst')
  click('Letzte Änderung rückgängig')
  await flush()
  expect(root.querySelector('[aria-label="Antwortvorschau"]').textContent).toBe('First')
  click('Original wiederherstellen')
  await flush()
  expect(root.querySelector('[aria-label="Antwortvorschau"]').textContent).toBe(
    props.note.suggestion
  )
  click('Verwerfen')
  await flush()
  expect(adopted).not.toHaveBeenCalled()
})
test('failure retains instructions, retry and adopt emit only selected text', async () => {
  await mount()
  const input = root.querySelector('textarea')
  input.value = 'Bitte kurz'
  input.dispatchEvent(new Event('input'))
  await flush()
  adjust.mockRejectedValueOnce(new Error('Timeout')).mockResolvedValueOnce(result('Adapted'))
  click('Anpassen')
  await flush()
  expect(input.value).toBe('Bitte kurz')
  expect(root.querySelector('[role="alert"]')).toBeTruthy()
  click('Anpassen')
  await flush()
  click('Übernehmen')
  await flush()
  expect(adopted).toHaveBeenCalledWith({ note: props.note, text: 'Adapted' })
  expect(adjust.mock.calls[1][1].instructions).toBe('Bitte kurz')
})
test('closing cancels stale requests and guards parallel clicks', async () => {
  await mount()
  let resolve
  adjust.mockImplementation(() => new Promise((r) => (resolve = r)))
  click('Anpassen')
  click('Anpassen')
  await flush()
  expect(adjust).toHaveBeenCalledTimes(1)
  const signal = adjust.mock.calls[0][2]
  click('Verwerfen')
  await flush()
  expect(signal.aborted).toBe(true)
  props.open = true
  await flush()
  resolve(result('Stale'))
  await flush()
  expect(root.querySelector('[aria-label="Antwortvorschau"]').textContent).toBe(
    props.note.suggestion
  )
  expect(adopted).not.toHaveBeenCalled()
})
