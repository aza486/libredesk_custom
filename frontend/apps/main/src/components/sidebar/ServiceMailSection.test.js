// @vitest-environment jsdom
import { beforeEach, afterEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import ServiceMailSection from './ServiceMailSection.vue'
const mocks = vi.hoisted(() => ({ push: vi.fn(), open: null, route: null, user: null }))
vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push })
}))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key) =>
      ({
        'serviceEmails.title': 'Service-Mails',
        'serviceEmails.all': 'Alle Service-Mails'
      })[key] || key
  })
}))
vi.mock('@vueuse/core', async () => {
  const { ref } = await import('vue')
  return { useStorage: (_, value) => (mocks.open = ref(value)) }
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
vi.mock('@main/stores/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('./ServiceEmailManager.vue', () => ({
  default: { render: () => h('div', { 'data-manager': true }) }
}))
vi.mock('@shared-ui/components/ui/dialog', async () => {
  const { h } = await import('vue')
  const component = {
    setup:
      (_, { slots }) =>
      () =>
        h('div', {}, slots.default?.())
  }
  return {
    Dialog: component,
    DialogContent: component,
    DialogHeader: component,
    DialogTitle: component,
    DialogDescription: component
  }
})
let app, host, props
beforeEach(() => {
  vi.clearAllMocks()
  mocks.user = reactive({ roles: ['Kundensupport'], can: () => false })
  mocks.route = reactive({ params: { type: 'service-mails' }, query: {} })
  props = reactive({ count: 125 })
  host = document.createElement('nav')
  app = createApp({ render: () => h(ServiceMailSection, props) })
  app.mount(host)
})
afterEach(() => {
  app.unmount()
  document.documentElement.classList.remove('dark')
})
test('shows the total once in the expanded all row and preserves navigation', async () => {
  const main = host.querySelector('section > li')
  expect(main).not.toBeNull()
  expect(main.firstElementChild.textContent).toContain('Service-Mails')
  expect(main.firstElementChild.querySelector('.tabular-nums')).toBeNull()
  expect(main.querySelector('ul .tabular-nums').textContent).toBe('125')
  expect(main.querySelector('ul [data-scope="service_mail"]').dataset.scopeId).toBe('0')
  const all = [...main.querySelectorAll('button')].find((b) =>
    b.textContent.includes('Alle Service-Mails')
  )
  all.click()
  expect(mocks.push).toHaveBeenCalledWith({
    name: 'inbox',
    params: { type: 'service-mails' },
    query: {}
  })
  props.count = 126
  await nextTick()
  expect(main.querySelector('ul .tabular-nums').textContent).toBe('126')
  expect(main.querySelectorAll('.tabular-nums')).toHaveLength(1)
})
test('keeps shared theme tokens and navigation structure in light and dark mode', async () => {
  for (const dark of [false, true]) {
    document.documentElement.classList.toggle('dark', dark)
    await nextTick()
    const badge = host.querySelector('.tabular-nums')
    expect(badge.classList.contains('bg-background')).toBe(true)
    expect(badge.classList.contains('shrink-0')).toBe(true)
    expect(host.querySelector('ul button span').classList.contains('truncate')).toBe(true)
    mocks.open.value = false
    await nextTick()
    expect(host.querySelector('.bg-success')).not.toBeNull()
    expect(host.querySelector('.tabular-nums')).toBeNull()
    mocks.open.value = true
    await nextTick()
    expect(host.querySelector('section > li ul')).not.toBeNull()
    expect(host.querySelector('[data-scope="service_mail"]')).not.toBeNull()
  }
})

test('reading service mails does not grant address management', async () => {
  expect(
    [...host.querySelectorAll('button')].some((b) => b.textContent.includes('serviceEmails.manage'))
  ).toBe(false)
  mocks.user.can = (permission) => permission === 'service_emails:manage'
  await nextTick()
  const manage = [...host.querySelectorAll('button')].find((b) =>
    b.textContent.includes('serviceEmails.manage')
  )
  expect(manage).toBeDefined()
  expect(
    [...host.querySelectorAll('button')].some((b) =>
      b.textContent.includes('serviceEmails.addAddress')
    )
  ).toBe(false)
  manage.click()
  await nextTick()
  expect(host.querySelector('[data-manager]')).not.toBeNull()
})

test('collapsed uses the existing green dot for positive counts and no numerical badge', async () => {
  mocks.open.value = false
  await nextTick()
  const header = host.querySelector('section > li').firstElementChild
  expect(header.querySelector('.tabular-nums')).toBeNull()
  expect(host.querySelector('.tabular-nums')).toBeNull()
  const dot = header.querySelector('.bg-success')
  expect(dot).not.toBeNull()
  expect(dot.getAttribute('aria-hidden')).toBe('true')
  expect(dot.classList.contains('shrink-0')).toBe(true)
  expect(header.querySelector('svg.ml-auto')).not.toBeNull()
  props.count = 0
  await nextTick()
  expect(header.querySelector('.bg-success')).toBeNull()
  props.count = 193
  mocks.open.value = true
  await nextTick()
  expect(header.querySelector('.bg-success')).toBeNull()
  expect(host.querySelectorAll('.tabular-nums')).toHaveLength(1)
  expect(host.querySelector('ul .tabular-nums').textContent).toBe('193')
})
