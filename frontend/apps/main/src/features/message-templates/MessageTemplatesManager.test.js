// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import MessageTemplatesManager from './MessageTemplatesManager.vue'
import api from '@main/api'
vi.mock('@main/api', () => ({
  default: {
    listMessageTemplates: vi.fn(),
    getMessageTemplate: vi.fn(),
    createMessageTemplate: vi.fn(),
    updateMessageTemplate: vi.fn(),
    deleteMessageTemplate: vi.fn()
  }
}))
vi.mock('./MessageTemplateEditor.vue', () => ({
  default: defineComponent({
    props: ['modelValue'],
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('textarea', {
          'aria-label': 'Vorlageninhalt',
          value: props.modelValue,
          onInput: (e) => emit('update:modelValue', e.target.value)
        })
    }
  })
}))
let app, root, open
const item = {
  uuid: 'one',
  name: 'Gelangensbestätigung',
  content: '<p>Original</p>',
  scope: 'personal',
  revision: 1
}
const flush = async () => {
  for (let i = 0; i < 20; i++) await nextTick()
}
const button = (label) =>
  [...document.querySelectorAll('button')].find(
    (b) => b.textContent.trim() === label || b.getAttribute('aria-label') === label
  )
const click = async (label) => {
  const target = button(label)
  expect(target, label).toBeTruthy()
  target.click()
  await flush()
}
const input = async (selector, value) => {
  const element = document.querySelector(selector)
  element.value = value
  element.dispatchEvent(new Event('input', { bubbles: true }))
  await flush()
}
beforeEach(async () => {
  vi.resetAllMocks()
  api.listMessageTemplates.mockResolvedValue({
    data: { data: { templates: [item], can_customer_support: true } }
  })
  api.getMessageTemplate.mockResolvedValue({ data: { data: item } })
  root = document.createElement('div')
  document.body.append(root)
  open = ref(true)
  app = createApp({
    setup: () => () =>
      h(MessageTemplatesManager, {
        open: open.value,
        identity: 'employee-1',
        conversationUUID: 'conversation',
        'onUpdate:open': (v) => {
          open.value = v
        }
      })
  })
  app.mount(root)
  await flush()
})
afterEach(async () => {
  app.unmount()
  root.remove()
  await flush()
})
test('warns on closing and switching templates with unsaved changes', async () => {
  await click('Gelangensbestätigung')
  await input('#message-template-name', 'Lokale Änderung')
  await click('Vorlagenverwaltung schließen')
  expect(open.value).toBe(true)
  expect(document.body.textContent).toContain('Ungespeicherte Änderungen verwerfen?')
  // Use the confirmation cancel button, not the form cancel button.
  document.querySelector('[role="alertdialog"] button').click()
  await flush()
  expect(document.querySelector('#message-template-name').value).toBe('Lokale Änderung')
  await click('Neue Vorlage')
  await click('Verwerfen')
  expect(document.querySelector('#message-template-name').value).toBe('')
  expect(open.value).toBe(true)
})
test('creates, saves sanitized response, confirms delete and sends revision', async () => {
  api.createMessageTemplate.mockResolvedValue({ data: { data: item } })
  api.deleteMessageTemplate.mockResolvedValue({ data: { data: true } })
  await click('Neue Vorlage')
  await input('#message-template-name', item.name)
  await input('textarea', item.content)
  document
    .querySelector('form')
    .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flush()
  expect(api.createMessageTemplate).toHaveBeenCalledWith(
    { name: item.name, content: item.content, scope: 'personal', revision: 0, attachment_ids: [] },
    'conversation'
  )
  expect(document.body.textContent).toContain('Vorlage gespeichert.')
  await click('Löschen')
  expect(api.deleteMessageTemplate).not.toHaveBeenCalled()
  document.querySelector('[role="alertdialog"] button:last-child').click()
  await flush()
  expect(api.deleteMessageTemplate).toHaveBeenCalledWith('one', 1, 'conversation')
})
test('shows conflict without discarding local changes', async () => {
  api.updateMessageTemplate.mockRejectedValue({
    response: { status: 409, data: { message: 'Zwischenzeitlich geändert' } }
  })
  await click('Gelangensbestätigung')
  await input('#message-template-name', 'Mein neuer Name')
  document
    .querySelector('form')
    .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flush()
  expect(document.querySelector('#message-template-name').value).toBe('Mein neuer Name')
  expect(document.body.textContent).toContain('Zwischenzeitlich geändert')
})

test('keeps unsaved changes on a temporary refresh failure and clears them after denied access', async () => {
  await click('Gelangensbestätigung')
  await input('#message-template-name', 'Lokaler Entwurf')
  api.listMessageTemplates.mockRejectedValueOnce(new Error('Offline'))
  window.dispatchEvent(new Event('focus'))
  await flush()
  expect(document.querySelector('#message-template-name').value).toBe('Lokaler Entwurf')
  api.listMessageTemplates.mockRejectedValueOnce({
    response: { status: 403, data: { message: 'Zugriff entzogen' } }
  })
  window.dispatchEvent(new Event('focus'))
  await flush()
  expect(document.querySelector('#message-template-name')).toBeNull()
  expect(document.body.textContent).not.toContain('Gelangensbestätigung')
})
