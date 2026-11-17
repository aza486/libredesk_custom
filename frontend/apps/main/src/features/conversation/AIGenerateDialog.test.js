// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import AIGenerateDialog from './AIGenerateDialog.vue'
const { generate } = vi.hoisted(() => ({ generate: vi.fn() }))
vi.mock('@main/api', () => ({ default: { aiGenerateReply: generate } }))
vi.mock('./AIActionPanel.vue', () => ({
  default: defineComponent({
    props: ['open'],
    setup(p, { slots }) {
      return () => (p.open ? h('section', [slots.default?.(), slots.footer?.()]) : null)
    }
  })
}))
let app, root, props, adopt
const flush = async () => {
  for (let i = 0; i < 8; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
async function mount(draft = '') {
  generate.mockReset()
  adopt = vi.fn()
  props = reactive({
    open: true,
    conversationUUID: 'ticket-a',
    draftHTML: draft,
    draftText: draft ? 'existing legacy instruction' : '',
    editorMode: 'private_note'
  })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      h(AIGenerateDialog, { ...props, 'onUpdate:open': (v) => (props.open = v), onAdopt: adopt })
  })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await flush()
}
const click = async (key) => {
  const b = [...root.querySelectorAll('button')].find((b) => b.textContent.trim() === key)
  expect(b).toBeTruthy()
  b.click()
  await flush()
}
const instruction = async (text) => {
  root.querySelector('textarea').value = text
  root.querySelector('textarea').dispatchEvent(new Event('input'))
  await flush()
}
afterEach(() => {
  app?.unmount()
  root?.remove()
})
test('empty instruction retains legacy payload and requires explicit adoption', async () => {
  await mount('<p>Draft</p>')
  generate.mockResolvedValue({ data: { data: '<p>Reply</p>' } })
  await click('aiGenerate.create')
  expect(generate.mock.calls[0][0]).toEqual({
    conversation_uuid: 'ticket-a',
    instruction: 'existing legacy instruction'
  })
  expect(props.draftHTML).toBe('<p>Draft</p>')
  expect(adopt).not.toHaveBeenCalled()
  await click('aiGenerate.adopt')
  expect(adopt).toHaveBeenCalledExactlyOnceWith({
    uuid: 'ticket-a',
    html: '<p>Draft</p>',
    editorMode: 'private_note',
    resultHTML: '<p>Reply</p>'
  })
})
test('new instruction without draft generates with current thread UUID', async () => {
  await mount()
  await instruction('Thank the customer')
  generate.mockResolvedValue({ data: { data: '<p>Thanks</p>' } })
  await click('aiGenerate.create')
  expect(generate.mock.calls[0][0]).toEqual({
    conversation_uuid: 'ticket-a',
    instruction: 'Thank the customer',
    mode: 'generate'
  })
})
test.each(['Correct spelling only', 'Translate into English'])(
  'draft edit uses same API and preview for %s',
  async (text) => {
    await mount('<p>Hallo <strong>Welt</strong></p>')
    await instruction(text)
    generate.mockResolvedValue({ data: { data: '<p>Edited</p>' } })
    await click('aiGenerate.apply')
    expect(generate.mock.calls[0][0]).toEqual({
      conversation_uuid: 'ticket-a',
      instruction: text,
      mode: 'edit',
      draft_html: props.draftHTML
    })
    await click('aiGenerate.discard')
    expect(adopt).not.toHaveBeenCalled()
    expect(props.draftHTML).toContain('Hallo')
  }
)
test('draft can explicitly request a new suggestion', async () => {
  await mount('<p>Draft</p>')
  await instruction('New reply')
  const radio = root.querySelector('input[value="generate"]')
  radio.checked = true
  radio.dispatchEvent(new Event('change'))
  await flush()
  generate.mockResolvedValue({ data: { data: '<p>New</p>' } })
  await click('aiGenerate.create')
  expect(generate.mock.calls[0][0]).toEqual({
    conversation_uuid: 'ticket-a',
    instruction: 'New reply',
    mode: 'generate'
  })
})
test('manual edits during generation block adoption even when text is later restored', async () => {
  await mount('<p>Draft</p>')
  let done
  generate.mockReturnValue(new Promise((resolve) => (done = resolve)))
  await click('aiGenerate.create')
  props.draftHTML = '<p>Manual edit</p>'
  await flush()
  props.draftHTML = '<p>Draft</p>'
  await flush()
  done({ data: { data: '<p>Late</p>' } })
  await flush()
  await click('aiGenerate.adopt')
  expect(adopt).not.toHaveBeenCalled()
  expect(root.textContent).toContain('aiGenerate.changed')
})
test('ticket switching A-B-A discards stale generation and prevents parallel requests', async () => {
  await mount()
  let done
  generate.mockReturnValue(new Promise((resolve) => (done = resolve)))
  await click('aiGenerate.create')
  await click('aiGenerate.processing')
  expect(generate).toHaveBeenCalledTimes(1)
  props.conversationUUID = 'ticket-b'
  await flush()
  props.conversationUUID = 'ticket-a'
  await flush()
  done({ data: { data: '<p>Late</p>' } })
  await flush()
  expect(root.textContent).not.toContain('Late')
  expect(adopt).not.toHaveBeenCalled()
  expect(generate.mock.calls[0][1].aborted).toBe(true)
})
test('error keeps instructions and draft; unsafe preview is plain text', async () => {
  await mount('<p>Original</p>')
  await instruction('Rewrite')
  generate.mockRejectedValueOnce(new Error('network'))
  await click('aiGenerate.apply')
  expect(root.textContent).toContain('aiGenerate.failed')
  expect(root.querySelector('textarea').value).toBe('Rewrite')
  generate.mockResolvedValueOnce({
    data: { data: '<p>Preview</p><img src="x" onerror="alert(1)">' }
  })
  await click('aiGenerate.apply')
  expect(root.querySelector('img')).toBeNull()
  expect(adopt).not.toHaveBeenCalled()
})
