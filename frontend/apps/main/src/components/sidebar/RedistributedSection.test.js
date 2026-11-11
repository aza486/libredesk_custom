// @vitest-environment jsdom
import { beforeEach, afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import RedistributedSection from './RedistributedSection.vue'
const mocks = vi.hoisted(() => ({ push: vi.fn(), route: null }))
vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push })
}))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key) =>
      ({
        'conversation.routing.redistributed': 'Umverteilt',
        'conversation.routing.allRedistributed': 'Alle umverteilten'
      })[key] || key
  })
}))
vi.mock('@vueuse/core', async () => {
  const { ref } = await import('vue')
  return { useStorage: (_, value) => ref(value) }
})
vi.mock('@shared-ui/components/ui/collapsible', async () => {
  const { h } = await import('vue')
  const component = (tag) => ({
    setup:
      (_, { slots }) =>
      () =>
        h(tag, {}, slots.default?.())
  })
  return {
    Collapsible: component('section'),
    CollapsibleContent: component('div'),
    CollapsibleTrigger: component('div')
  }
})
vi.mock('@shared-ui/components/ui/sidebar', async () => {
  const { h } = await import('vue')
  const component = (tag) => ({
    setup:
      (_, { slots }) =>
      () =>
        h(tag, {}, slots.default?.())
  })
  return {
    SidebarMenuItem: component('li'),
    SidebarMenuButton: component('button'),
    SidebarMenuSub: component('ul'),
    SidebarMenuSubItem: component('li')
  }
})
vi.mock('./SidebarFolders.vue', () => ({
  default: {
    props: ['scope', 'scopeId'],
    setup: (props) => () =>
      h('li', { 'data-scope': props.scope, 'data-scope-id': props.scopeId }, 'Folders')
  }
}))
let app, host, props
beforeEach(() => {
  vi.clearAllMocks()
  mocks.route = reactive({ params: { type: 'customer-redistributed' }, query: {} })
  props = reactive({ count: 125 })
  host = document.createElement('nav')
  app = createApp({ render: () => h(RedistributedSection, props) })
  app.mount(host)
})
afterEach(() => {
  app.unmount()
  document.documentElement.classList.remove('dark')
})
test('renders an independent main area with its total outside the exclusive folder subtree', async () => {
  const main = host.querySelector('section > li')
  expect(main).not.toBeNull()
  expect(main.firstElementChild.textContent).toContain('Umverteilt')
  expect(main.firstElementChild.textContent).toContain('125')
  expect(main.querySelector('ul [data-scope="redistributed"]').dataset.scopeId).toBe('0')
  const all = [...main.querySelectorAll('button')].find((b) =>
    b.textContent.includes('Alle umverteilten')
  )
  all.click()
  expect(mocks.push).toHaveBeenCalledWith({
    name: 'inbox',
    params: { type: 'customer-redistributed' },
    query: {}
  })
  props.count = 126
  await nextTick()
  expect(main.firstElementChild.textContent).toContain('126')
})
test('keeps shared theme tokens and navigation structure in light and dark mode', async () => {
  for (const dark of [false, true]) {
    document.documentElement.classList.toggle('dark', dark)
    await nextTick()
    const badge = host.querySelector('.tabular-nums')
    expect(badge.classList.contains('bg-background')).toBe(true)
    expect(host.querySelector('section > li ul')).not.toBeNull()
    expect(host.querySelector('[data-scope="redistributed"]')).not.toBeNull()
  }
})
