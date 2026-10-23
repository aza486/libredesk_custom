// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import ConversationTagOverflow from './ConversationTagOverflow.vue'

let observer, frames, width, widths, app, host

beforeEach(() => {
  frames = new Map()
  let frameID = 0
  width = 200
  widths = { A: 30, B: 40, C: 50 }
  vi.stubGlobal('requestAnimationFrame', (fn) => {
    frames.set(++frameID, fn)
    return frameID
  })
  vi.stubGlobal('cancelAnimationFrame', (id) => frames.delete(id))
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(callback) {
        this.callback = callback
        observer = this
      }
      observe = vi.fn()
      disconnect = vi.fn()
    }
  )
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function () {
    const measured = this.hasAttribute('data-conversation-tags')
      ? width
      : this.hasAttribute('data-overflow-measure')
        ? 16
        : widths[this.textContent.trim()] || 0
    return { width: measured, height: 20, x: 0, y: 0, top: 0, left: 0, right: measured, bottom: 20 }
  })
})

afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

async function flush() {
  await nextTick()
  await nextTick()
  const work = [...frames.values()]
  frames.clear()
  work.forEach((fn) => fn())
  await nextTick()
}

async function mount(tags) {
  const props = reactive({ tags })
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({
    render: () => h(TooltipProvider, null, { default: () => h(ConversationTagOverflow, props) })
  })
  app.mount(host)
  host.querySelector('[data-conversation-tags]').firstElementChild.style.columnGap = '4px'
  await flush()
  return props
}

test('responds to container resizing without a viewport change and coalesces observer callbacks', async () => {
  await mount(['A', 'B', 'C'])
  const root = host.querySelector('[data-conversation-tags]')
  for (const [available, expected] of [
    [200, 3],
    [94, 2],
    [50, 1],
    [30, 0],
    [500, 3],
    [94, 2],
    [500, 3]
  ]) {
    width = available
    for (let i = 0; i < 5; i++) observer.callback([{ target: root }])
    expect(frames.size).toBe(1)
    await flush()
    expect(host.querySelectorAll('[data-visible-tag]')).toHaveLength(expected)
    const overflow = host.querySelector('[data-tag-overflow]')
    expect(Boolean(overflow)).toBe(expected < 3)
    if (overflow)
      expect(overflow.getAttribute('aria-label')).toBe(['A', 'B', 'C'].slice(expected).join(', '))
  }
})

test('updates changed tags and font measurements and stops observing on unmount', async () => {
  const props = await mount([])
  expect(host.querySelector('[data-tag-overflow]')).toBeNull()
  props.tags = ['A']
  await flush()
  expect(host.querySelectorAll('[data-visible-tag]')).toHaveLength(1)
  props.tags = ['A', 'B', 'C']
  await flush()
  widths.A = 500
  observer.callback([{ target: host.querySelector('[data-tag-measure]').parentElement }])
  await flush()
  expect(host.querySelectorAll('[data-visible-tag]')).toHaveLength(0)
  const indicator = host.querySelector('[data-tag-overflow]')
  const click = new MouseEvent('click', { bubbles: true, cancelable: true })
  const parentClick = vi.fn()
  host.addEventListener('click', parentClick)
  indicator.dispatchEvent(click)
  expect(click.defaultPrevented).toBe(true)
  expect(parentClick).not.toHaveBeenCalled()
  expect(indicator.getAttribute('tabindex')).toBe('0')
  observer.callback([{ target: host.firstElementChild }])
  app.unmount()
  app = null
  expect(observer.disconnect).toHaveBeenCalledOnce()
  expect(frames.size).toBe(0)
})

test('fits prioritized warning borders, preserves icon labels and exposes hidden warnings', async () => {
  const tags = ['A', '😎 Mensch erforderlich', 'B', '🛑 Verarbeitungsfehler']
  widths['🛑 Verarbeitungsfehler'] = 100
  widths['😎 Mensch erforderlich'] = 90
  const props = await mount(tags)
  const root = host.querySelector('[data-conversation-tags]')
  for (const [available, expected] of [
    [400, ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich', 'A', 'B']],
    [214, ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich']],
    [120, ['🛑 Verarbeitungsfehler']],
    [70, []],
    [400, ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich', 'A', 'B']]
  ]) {
    width = available
    observer.callback([{ target: root }])
    await flush()
    expect(
      [...host.querySelectorAll('[data-visible-tag]')].map((el) => el.textContent.trim())
    ).toEqual(expected)
    expect(props.tags).toEqual(tags)
    const indicator = host.querySelector('[data-tag-overflow]')
    if (expected.length < tags.length) {
      expect(indicator.getAttribute('aria-label')).toBe(
        ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich', 'A', 'B']
          .slice(expected.length)
          .join(', ')
      )
    } else expect(indicator).toBeNull()
  }
  for (const selector of ['[data-visible-tag]', '[data-tag-measure]']) {
    const badges = [...host.querySelectorAll(selector)]
    expect(badges[0].classList.contains('border-red-500')).toBe(true)
    expect(badges[1].classList.contains('border-yellow-500')).toBe(true)
    expect(badges[2].classList.contains('border-transparent')).toBe(true)
    expect(badges[3].classList.contains('border-transparent')).toBe(true)
  }
})
