// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import SidebarFolders from './SidebarFolders.vue'
const mocks = vi.hoisted(() => ({
  api: {},
  route: null,
  push: vi.fn(),
  refresh: vi.fn(),
  handlers: new Map()
}))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push, replace: mocks.push })
}))
vi.mock('@main/stores/conversation', () => ({
  useConversationStore: () => ({ refreshConversationListForRealtimeEvent: mocks.refresh })
}))
vi.mock('@main/composables/useEmitter', () => ({
  useEmitter: () => ({
    on: (event, cb) => mocks.handlers.set(event, cb),
    off: (event) => mocks.handlers.delete(event),
    emit: (event) => mocks.handlers.get(event)?.()
  })
}))
vi.mock('@shared-ui/components/ui/sidebar', async () => {
  const { h } = await import('vue')
  return {
    SidebarMenuSubItem: {
      setup:
        (_, { slots }) =>
        () =>
          h('li', {}, slots.default?.())
    },
    SidebarMenuButton: {
      setup:
        (_, { slots }) =>
        () =>
          h('button', {}, slots.default?.())
    }
  }
})
vi.mock('./SidebarFolderForm.vue', () => ({ default: { render: () => null } }))
const definitions = [
  { id: 11, name: 'Alle Reklas', sort_order: 0, show_count: true },
  { id: 12, name: 'Ohne Badge', sort_order: 1, show_count: false }
]
let app, host, props
async function flush() {
  for (let i = 0; i < 8; i++) await nextTick()
}
beforeEach(async () => {
  vi.clearAllMocks()
  mocks.handlers.clear()
  mocks.route = reactive({ params: { teamID: '7' }, query: {} })
  mocks.api.getSidebarFolders = vi.fn().mockResolvedValue({ data: { data: definitions } })
  mocks.api.getSidebarFolderCounts = vi
    .fn()
    .mockResolvedValue({ data: { data: { 11: 125, 12: 7 } } })
  mocks.api.reorderSidebarFolders = vi
    .fn()
    .mockResolvedValue({ data: { data: [...definitions].reverse() } })
  host = document.createElement('ul')
  document.body.append(host)
  props = reactive({ scope: 'team', scopeId: 7, scopeName: 'Team' })
  app = createApp({ render: () => h(SidebarFolders, props) })
  app.mount(host)
  await flush()
})
afterEach(() => {
  app.unmount()
  host.remove()
})
test('counts once per scope, shows the exact total, and places FolderPen immediately before the optional badge', () => {
  expect(mocks.api.getSidebarFolders).toHaveBeenCalledWith('team', 7)
  expect(mocks.api.getSidebarFolderCounts).toHaveBeenCalledTimes(1)
  const edit = host.querySelector('[aria-label="Alle Reklas bearbeiten"]')
  expect(edit.previousElementSibling.textContent).toContain('Alle Reklas')
  expect(edit.nextElementSibling.textContent).toBe('125')
  expect(host.querySelector('[aria-label="Ohne Badge bearbeiten"]').nextElementSibling).toBeNull()
})
test('navigates within the fixed scope without adding a status condition', async () => {
  host.querySelector('li button').click()
  await flush()
  expect(mocks.push).toHaveBeenCalledWith({
    name: 'team-inbox',
    params: { teamID: 7 },
    query: { folder: '11', folderName: 'Alle Reklas' }
  })
})
test('keyboard reordering submits the full shared permutation', async () => {
  host
    .querySelector('li button')
    .dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', altKey: true, bubbles: true }))
  await flush()
  expect(mocks.api.reorderSidebarFolders).toHaveBeenCalledWith('team', 7, [12, 11])
  expect(mocks.refresh).toHaveBeenCalled()
})
test('a deleted active folder returns to its standard scope', async () => {
  mocks.route.query = { folder: '11' }
  mocks.api.getSidebarFolders.mockResolvedValue({ data: { data: [] } })
  mocks.api.getSidebarFolderCounts.mockResolvedValue({ data: { data: {} } })
  await mocks.handlers.get('sidebar-folders-refresh')()
  await flush()
  expect(host.querySelector('[aria-label="Alle Reklas bearbeiten"]')).toBeNull()
  expect(mocks.push).toHaveBeenCalledWith({
    name: 'team-inbox',
    params: { teamID: 7 },
    query: { status: 'Open' }
  })
})

test('folder rows have no leading icon or visible sorting controls, including after focus', async () => {
  const row = host.querySelector('li')
  const navigate = row.querySelector('button')
  expect(navigate.querySelector('svg')).toBeNull()
  const edit = row.querySelector('[aria-label="Alle Reklas bearbeiten"]')
  expect(edit.querySelector('svg.lucide-folder-pen')).not.toBeNull()
  navigate.focus()
  await flush()
  expect(row.querySelectorAll('button')).toHaveLength(2)
  expect(row.querySelector('.lucide-arrow-up, .lucide-arrow-down')).toBeNull()
  expect(row.textContent).not.toMatch(/nach oben|nach unten/)
})

test('drag and drop still changes the shared priority order', async () => {
  const rows = host.querySelectorAll('li[draggable="true"]')
  rows[0].dispatchEvent(new Event('dragstart', { bubbles: true }))
  rows[1].dispatchEvent(new Event('drop', { bubbles: true, cancelable: true }))
  await flush()
  expect(mocks.api.reorderSidebarFolders).toHaveBeenCalledWith('team', 7, [12, 11])
})

test('my_inbox loads the employee area and navigates to the existing assigned route', async () => {
  props.scope = 'my_inbox'
  props.scopeId = 101
  props.scopeName = 'Mein Posteingang'
  mocks.route.params = { type: 'assigned' }
  await flush()
  await mocks.handlers.get('sidebar-folders-refresh')()
  host.querySelector('li button').click()
  await flush()
  expect(mocks.api.getSidebarFolders).toHaveBeenLastCalledWith('my_inbox', 101)
  expect(mocks.api.getSidebarFolderCounts).toHaveBeenLastCalledWith('my_inbox', 101)
  expect(mocks.push).toHaveBeenLastCalledWith({
    name: 'inbox',
    params: { type: 'assigned' },
    query: { folder: '11', folderName: 'Alle Reklas' }
  })
})
