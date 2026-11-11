// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, nextTick, reactive } from 'vue'
import ProfileEditView from './ProfileEditView.vue'

const mocks = vi.hoisted(() => ({ updateCurrentUser: vi.fn(), emit: vi.fn(), store: null }))
vi.mock('../../../api', () => ({ default: { updateCurrentUser: mocks.updateCurrentUser } }))
vi.mock('../../../stores/user', () => ({ useUserStore: () => mocks.store }))
vi.mock('../../../composables/useEmitter', () => ({ useEmitter: () => ({ emit: mocks.emit }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: key => key }) }))
vi.mock('@shared-ui/components/ui/avatar', () => ({ AvatarUpload: { template: '<div />' } }))
let app, host
const flush = async () => { for (let i = 0; i < 12; i++) { await Promise.resolve(); await nextTick() } }
const mount = async phone => {
  mocks.store = reactive({ user: { id: 102, work_phone_number: phone }, setCurrentUser: user => { mocks.store.user = user } })
  mocks.updateCurrentUser.mockImplementation(async data => ({ data: { data: { id: 102, work_phone_number: data.get('work_phone_number')?.trim() || null } } }))
  host = document.createElement('div'); document.body.append(host)
  app = createApp(ProfileEditView)
  app.config.globalProperties.$t = key => key
  app.mount(host); await flush()
}
const edit = async value => {
  const input = host.querySelector('#work-phone')
  input.value = value; input.dispatchEvent(new Event('input', { bubbles: true })); await flush()
}
afterEach(() => { app?.unmount(); host?.remove(); vi.clearAllMocks() })

test('placeholder stays empty; saves, restores, changes and clears full phone numbers', async () => {
  await mount(null)
  const input = host.querySelector('#work-phone')
  const save = host.querySelector('button')
  expect(input.value).toBe('')
  expect(input.placeholder).toBe('0351 40424-0')
  expect(input.type).toBe('tel')
  expect(input.maxLength).toBe(64)
  expect(save.disabled).toBe(true)
  for (const value of ['0351 40424-23', '+49 (351) 40424-23', '']) {
    await edit(value)
    expect(save.disabled).toBe(false)
    save.click(); await flush()
    const data = mocks.updateCurrentUser.mock.lastCall[0]
    expect(data.get('work_phone_number')).toBe(value)
    expect(data.has('new_password')).toBe(false)
    expect(data.has('first_name')).toBe(false)
    expect(mocks.store.user.work_phone_number).toBe(value || null)
    expect(save.disabled).toBe(true)
  }
})

test('opens with saved phone and preserves input on server rejection', async () => {
  await mount('0351 40424-23')
  expect(host.querySelector('#work-phone').value).toBe('0351 40424-23')
  await edit('<b>123</b>')
  mocks.updateCurrentUser.mockRejectedValueOnce(new Error('invalid phone'))
  host.querySelector('button').click(); await flush()
  expect(host.querySelector('#work-phone').value).toBe('<b>123</b>')
  expect(mocks.store.user.work_phone_number).toBe('0351 40424-23')
  expect(mocks.emit).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ variant: 'destructive' }))
})

test('loads profile asynchronously without replacing edited input', async () => {
  await mount(undefined)
  mocks.store.user.work_phone_number = '030 12345678'; await flush()
  expect(host.querySelector('#work-phone').value).toBe('030 12345678')
  await edit('0351 40424-99')
  mocks.store.user.work_phone_number = '030 87654321'; await flush()
  expect(host.querySelector('#work-phone').value).toBe('0351 40424-99')
})
