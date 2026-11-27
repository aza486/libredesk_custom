// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import TemplateAttachments from './TemplateAttachments.vue'
import api from '@main/api'
vi.mock('@main/api', () => ({
  default: { uploadMessageTemplateAttachment: vi.fn(), discardMessageTemplateAttachment: vi.fn() }
}))
let app, root, files, component
const flush = async () => {
  for (let i = 0; i < 15; i++) await nextTick()
}
const file = {
  id: 1,
  uuid: 'one',
  filename: 'one.pdf',
  content_type: 'application/pdf',
  size: 32,
  url: '/uploads/one'
}
const mount = async (initial = []) => {
  files = ref(initial)
  root = document.createElement('div')
  document.body.append(root)
  api.discardMessageTemplateAttachment.mockResolvedValue({ data: { data: true } })
  app = createApp({
    setup: () => () =>
      h(TemplateAttachments, {
        ref: (v) => {
          component = v
        },
        modelValue: files.value,
        'onUpdate:modelValue': (v) => {
          files.value = v
        },
        conversationUUID: 'ticket'
      })
  })
  app.mount(root)
  await flush()
}
const select = async (...selected) => {
  const input = root.querySelector('input')
  Object.defineProperty(input, 'files', { value: selected, configurable: true })
  input.dispatchEvent(new Event('change', { bubbles: true }))
  await flush()
}
afterEach(async () => {
  app?.unmount()
  root?.remove()
  await flush()
  vi.resetAllMocks()
})
test('uploads several files with progress and discards only staged files', async () => {
  api.uploadMessageTemplateAttachment.mockImplementation(async (_file, _context, options) => {
    options.onUploadProgress({ loaded: 5, total: 10 })
    return { data: { data: file } }
  })
  await mount()
  await select(new File(['pdf'], 'one.pdf', { type: 'application/pdf' }))
  expect(files.value).toEqual([file])
  expect(root.textContent).toContain('one.pdf')
  root.querySelector('[aria-label="one.pdf entfernen"]').click()
  await flush()
  expect(files.value).toEqual([])
  expect(api.discardMessageTemplateAttachment).toHaveBeenCalledWith('one', 'ticket')
})
test('removing a saved attachment changes only the local form and saving commits staged files', async () => {
  await mount([file])
  root.querySelector('[aria-label="one.pdf entfernen"]').click()
  await flush()
  expect(api.discardMessageTemplateAttachment).not.toHaveBeenCalled()
  api.uploadMessageTemplateAttachment.mockResolvedValue({ data: { data: file } })
  await select(new File(['pdf'], 'one.pdf'))
  component.markSaved([file])
  app.unmount()
  app = null
  expect(api.discardMessageTemplateAttachment).not.toHaveBeenCalled()
})
test('shows failed uploads and cancels pending work without losing saved files', async () => {
  api.uploadMessageTemplateAttachment.mockRejectedValueOnce({
    response: { data: { message: 'Upload fehlgeschlagen' } }
  })
  await mount([file])
  await select(new File(['bad'], 'bad.txt'))
  expect(root.textContent).toContain('Upload fehlgeschlagen')
  expect(files.value).toEqual([file])
  let resolve
  api.uploadMessageTemplateAttachment.mockImplementation(
    (_file, _context, options) =>
      new Promise((r) => {
        resolve = r
        expect(options.signal.aborted).toBe(false)
      })
  )
  await select(new File(['slow'], 'slow.pdf'))
  expect(root.querySelector('progress')).toBeTruthy()
  component.resetUploads()
  resolve({ data: { data: { ...file, id: 2, uuid: 'late' } } })
  await flush()
  expect(files.value).toEqual([file])
  expect(api.discardMessageTemplateAttachment).toHaveBeenCalledWith('late', 'ticket')
})

test('canceling the remaining batch keeps already completed staged files usable', async () => {
  let resolve
  api.uploadMessageTemplateAttachment
    .mockResolvedValueOnce({ data: { data: file } })
    .mockImplementationOnce(
      () =>
        new Promise((r) => {
          resolve = r
        })
    )
  await mount()
  await select(new File(['done'], 'one.pdf'), new File(['waiting'], 'two.pdf'))
  expect(files.value).toEqual([file])
  const cancel = [...root.querySelectorAll('button')].find((button) =>
    button.textContent.includes('Uploads abbrechen')
  )
  cancel.click()
  await flush()
  resolve({ data: { data: { ...file, id: 2, uuid: 'late' } } })
  await flush()
  expect(files.value).toEqual([file])
  expect(api.discardMessageTemplateAttachment).not.toHaveBeenCalledWith('one', 'ticket')
})
test('reports every failed file in a multiple selection', async () => {
  api.uploadMessageTemplateAttachment.mockRejectedValue({
    response: { data: { message: 'Upload fehlgeschlagen' } }
  })
  await mount()
  await select(new File(['one'], 'one.pdf'), new File(['two'], 'two.pdf'))
  expect(root.textContent).toContain('one.pdf: Upload fehlgeschlagen')
  expect(root.textContent).toContain('two.pdf: Upload fehlgeschlagen')
})

test('starts two uploads concurrently and retains both responses arriving in the same tick', async () => {
  const resolvers = []
  api.uploadMessageTemplateAttachment.mockImplementation(
    () => new Promise((resolve) => resolvers.push(resolve))
  )
  await mount()
  await select(new File(['one'], 'one.pdf'), new File(['two'], 'two.pdf'))
  expect(api.uploadMessageTemplateAttachment).toHaveBeenCalledTimes(2)
  resolvers[0]({ data: { data: file } })
  resolvers[1]({ data: { data: { ...file, id: 2, uuid: 'two', filename: 'two.pdf' } } })
  await flush()
  expect(files.value.map((item) => item.uuid)).toEqual(['one', 'two'])
})
