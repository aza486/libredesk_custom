// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive, ref } from 'vue'
import { useConversationListPosition } from './useConversationListPosition'

let app, host, viewport
const ids = Array.from({ length: 100 }, (_, i) => `ticket-${i}`)

function mount(existing) {
  const store =
    existing ||
    reactive({
      listKey: 'folder-a',
      listRevision: 0,
      listPositions: new Map(),
      ids: [...ids],
      selected: 'ticket-65',
      version: 0
    })
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({
    setup() {
      const container = ref(null)
      const { saveListPosition } = useConversationListPosition(container, store)
      return () =>
        h(
          'div',
          {
            ref: (el) => {
              container.value = el
              if (!el) return
              viewport = el
              Object.defineProperty(el, 'clientHeight', { configurable: true, value: 1000 })
              el.getBoundingClientRect = () => ({ top: 10 })
            },
            onScroll: saveListPosition,
            'data-version': store.version
          },
          store.ids.map((id) =>
            h('div', {
              key: id,
              'data-conversation-id': id,
              'data-selected': store.selected === id,
              ref: (el) => {
                if (!el) return
                el.getBoundingClientRect = () => {
                  const index = Array.from(el.parentElement.children).indexOf(el)
                  const top = 10 + index * 100 - el.parentElement.scrollTop
                  return { top, bottom: top + 100 }
                }
              }
            })
          )
        )
    }
  })
  app.mount(host)
  return store
}

function scroll(top = 6525) {
  viewport.scrollTop = top
  viewport.dispatchEvent(new Event('scroll'))
}

const offset = (id) =>
  viewport.querySelector(`[data-conversation-id="${id}"]`).getBoundingClientRect().top - 10

afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.unstubAllGlobals()
})

test('opening and switching conversations preserves position in a multi-page list', async () => {
  const store = mount()
  scroll()
  store.selected = 'ticket-66'
  await nextTick()
  expect(viewport.scrollTop).toBe(6525)
  expect(offset('ticket-65')).toBe(-25)
  expect(viewport.children).toHaveLength(100)
  store.selected = 'ticket-67'
  await nextTick()
  expect(viewport.scrollTop).toBe(6525)
})

test('removing the closed anchor keeps the nearest surviving neighbour at its old offset', async () => {
  const store = mount()
  scroll()
  expect(offset('ticket-64')).toBe(-125)
  store.ids = store.ids.filter((id) => id !== 'ticket-65')
  await nextTick()
  expect(offset('ticket-66')).toBe(75)
  expect(viewport.scrollTop).toBe(6425)
})

test('removing a row above the viewport compensates for its lost height', async () => {
  const store = mount()
  scroll()
  store.ids = store.ids.filter((id) => id !== 'ticket-5')
  await nextTick()
  expect(offset('ticket-65')).toBe(-25)
  expect(viewport.scrollTop).toBe(6425)
})

test('a new incoming message can reorder a ticket to the top without moving the viewport anchor', async () => {
  const store = mount()
  scroll()
  store.ids = ['ticket-90', ...store.ids.filter((id) => id !== 'ticket-90')]
  await nextTick()
  expect(store.ids[0]).toBe('ticket-90')
  expect(offset('ticket-65')).toBe(-25)
  expect(viewport.scrollTop).toBe(6625)
})

test('folder switches restore independent anchors after rendering their own rows', async () => {
  const store = mount()
  scroll()
  store.listKey = 'folder-b'
  store.ids = Array.from({ length: 30 }, (_, i) => `other-${i}`)
  await nextTick()
  expect(viewport.scrollTop).toBe(0)
  scroll(525)
  store.listKey = 'folder-a'
  store.ids = [...ids]
  await nextTick()
  expect(viewport.scrollTop).toBe(6525)
  expect(offset('ticket-65')).toBe(-25)
  store.listKey = 'folder-b'
  store.ids = Array.from({ length: 30 }, (_, i) => `other-${i}`)
  await nextTick()
  expect(viewport.scrollTop).toBe(525)
})

test('unmount and remount restore the saved anchor', async () => {
  const store = mount()
  scroll()
  app.unmount()
  host.remove()
  mount(store)
  await nextTick()
  expect(viewport.scrollTop).toBe(6525)
})

test('a deliberate filter change starts at the top and an explicit reset clears the position', async () => {
  const store = mount()
  scroll()
  store.listKey = 'folder-a:search=new'
  store.ids = ids.slice(0, 30)
  await nextTick()
  expect(viewport.scrollTop).toBe(0)
  scroll(525)
  store.listPositions.delete(store.listKey)
  store.listRevision++
  store.ids = [...store.ids]
  await nextTick()
  expect(viewport.scrollTop).toBe(0)
})

test('scrollTop is used when every previously loaded anchor is gone', async () => {
  const store = mount()
  scroll()
  store.ids = ids.map((id) => `replacement-${id}`)
  await nextTick()
  expect(viewport.scrollTop).toBe(6525)
})


test('a hidden mobile viewport restores when it becomes visible again', async () => {
  let resized
  const disconnect = vi.fn()
  vi.stubGlobal('ResizeObserver', class {
    constructor(callback) { resized = callback }
    observe() {}
    disconnect = disconnect
  })
  const store = mount()
  scroll()
  Object.defineProperty(viewport, 'clientHeight', { configurable: true, value: 0 })
  store.ids = store.ids.filter((id) => id !== 'ticket-5')
  await nextTick()
  viewport.scrollTop = 0
  Object.defineProperty(viewport, 'clientHeight', { configurable: true, value: 1000 })
  resized()
  expect(offset('ticket-65')).toBe(-25)
  expect(viewport.scrollTop).toBe(6425)
  app.unmount()
  expect(disconnect).toHaveBeenCalledOnce()
  app = null
})
