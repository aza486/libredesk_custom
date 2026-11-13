// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { initI18n } from '@main/i18n'
import en from '../../../../../i18n/en-US.json'
import Harness from './__tests__/fixtures/FullscreenLightboxHarness.vue'
import ImageLightbox from './ImageLightbox.vue'

// Keep real Radix dialogs, Teleports, lightbox, attachment preview and split layout.
// Editor rendering is covered in the browser with the real Tiptap component.
vi.mock('@main/components/editor/ConversationEditor.vue', () => ({
  default: defineComponent({
    props: ['htmlContent'],
    emits: ['update:htmlContent', 'update:textContent'],
    setup(props, { emit, expose }) {
      expose({ focus: () => {} })
      return () =>
        h('textarea', {
          'data-editor': '',
          value: props.htmlContent,
          onInput: (event) => {
            emit('update:htmlContent', event.target.value)
            emit('update:textContent', event.target.value)
          }
        })
    }
  })
}))
vi.mock('@main/features/conversation/ReplyBoxMenuBar.vue', () => ({
  default: { render: () => null }
}))

let app, host
const flush = async () => {
  for (let i = 0; i < 12; i++) await nextTick()
  await new Promise((resolve) => setTimeout(resolve, 25))
}
const find = (selector) => document.querySelector(selector)
const pointer = async (element) => {
  element.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, button: 0 }))
  element.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }))
  element.click()
  await flush()
}
const key = async (name) => {
  document.activeElement.dispatchEvent(
    new KeyboardEvent('keydown', { key: name, bubbles: true, cancelable: true })
  )
  await flush()
}
const lightbox = () => find('.vel-modal')
const fullscreen = () => find('.lightbox-fullscreen-harness')
const currentImage = () => lightbox()?.querySelector('img[style]')?.getAttribute('src')
const openImage = async (index = 0) => {
  await pointer(document.querySelectorAll('[data-test=history] img')[index].parentElement)
  expect(lightbox()).toBeTruthy()
}

beforeEach(async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { render: () => null } }]
  })
  await router.push('/')
  host = document.createElement('div')
  document.body.append(host)
  app = createApp(Harness)
  app
    .use(createPinia())
    .use(router)
    .use(initI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': en } }))
  app.config.globalProperties.emitter = { emit: vi.fn() }
  app.mount(host)
  await flush()
})
afterEach(async () => {
  app?.unmount()
  host?.remove()
  await flush()
})

test('X closes only the lightbox, retains draft/note/history and allows reopen then restore', async () => {
  await pointer(find('[data-test=expand]'))
  const dialog = fullscreen()
  const editor = find('[data-editor]')
  editor.value = 'Unsent regression draft'
  editor.dispatchEvent(new Event('input', { bubbles: true }))
  const note = [...document.querySelectorAll('[role=tab]')].find(
    (el) => el.textContent === 'Private note'
  )
  await pointer(note)
  const history = find('[data-test=history]')
  history.scrollTop = 103
  await openImage()
  await pointer(find('.btn__close'))
  expect(fullscreen()).toBe(dialog)
  expect(lightbox()).toBeNull()
  expect(find('[data-editor]')).toBe(editor)
  expect(editor.value).toBe('Unsent regression draft')
  expect(find('[data-test=mode-state]').textContent).toBe('private_note')
  expect(find('[data-test=history]')).toBe(history)
  expect(history.scrollTop).toBe(103)
  await openImage(1)
  await pointer(find('.btn__close'))
  await pointer(find('[aria-label="Restore editor"]'))
  expect(fullscreen()).toBeNull()
  expect(find('[data-test=draft-state]').textContent).toBe('Unsent regression draft')
  expect(find('[data-test=mode-state]').textContent).toBe('private_note')
})

test('Escape dismisses only the top dialog; the next Escape retains normal fullscreen behavior', async () => {
  await pointer(find('[data-test=expand]'))
  const dialog = fullscreen()
  await openImage()
  await key('Escape')
  expect(lightbox()).toBeNull()
  expect(fullscreen()).toBe(dialog)
  await key('Escape')
  expect(fullscreen()).toBeNull()
})

test('pointer and keyboard image navigation keep the fullscreen and editor mounted, with one step per key', async () => {
  await pointer(find('[data-test=expand]'))
  const dialog = fullscreen()
  const editor = find('[data-editor]')
  await openImage()
  const first = currentImage()
  await pointer(find('.btn__next'))
  const second = currentImage()
  expect(second).not.toBe(first)
  await pointer(find('.btn__prev'))
  expect(currentImage()).toBe(first)
  await key('ArrowRight')
  expect(currentImage()).toBe(second)
  await key('ArrowLeft')
  expect(currentImage()).toBe(first)
  expect(fullscreen()).toBe(dialog)
  expect(find('[data-editor]')).toBe(editor)
})

test('normal attachment lightbox still navigates, closes on mask and reopens', async () => {
  await openImage(1)
  const second = currentImage()
  await key('ArrowRight')
  expect(currentImage()).not.toBe(second)
  await pointer(lightbox())
  expect(lightbox()).toBeNull()
  await openImage(1)
  expect(currentImage()).toBe(second)
  await key('Escape')
  expect(lightbox()).toBeNull()
  expect(fullscreen()).toBeNull()
})

test('initially open lightbox respects startIndex and unmount leaves no keyboard handler or body lock', async () => {
  app.unmount()
  const before = document.body.style.overflow
  const onChange = vi.fn()
  app = createApp({
    render: () =>
      h(ImageLightbox, {
        modelValue: true,
        startIndex: 1,
        images: [{ url: 'first' }, { url: 'second' }],
        'onUpdate:modelValue': onChange
      })
  })
  app.use(initI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': en } }))
  app.mount(host)
  await flush()
  expect(currentImage()).toBe('second')
  app.unmount()
  app = null
  await flush()
  const event = new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true, cancelable: true })
  document.dispatchEvent(event)
  expect(event.defaultPrevented).toBe(false)
  expect(onChange).not.toHaveBeenCalled()
  expect(document.body.style.overflow).toBe(before)
})
