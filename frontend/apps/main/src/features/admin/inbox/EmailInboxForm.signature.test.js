// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import EmailInboxForm from './EmailInboxForm.vue'

const mocks = vi.hoisted(() => ({ getInboxSignatures: vi.fn() }))
vi.mock('@/api', () => ({ default: mocks }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@main/stores/users', () => ({ useUsersStore: () => ({ users: [], fetchUsers: vi.fn() }) }))
vi.mock('@/stores/appSettings', () => ({ useAppSettingsStore: () => ({ settings: {} }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: key => key }) }))
// Exercise the actual vee-validate form and its selection handler. The existing
// searchable combobox emits the selected {label,value} through its select event.
vi.mock('@main/components/combobox/UserMultiSelect.vue', () => ({ default: defineComponent({ setup: () => () => h('div') }) }))
vi.mock('@main/components/combobox/SelectCombobox.vue', () => ({ default: defineComponent({
  props: ['modelValue', 'items'], emits: ['select'],
  setup(props, { emit }) { return () => h('select', {
    'data-signature': '', value: props.modelValue,
    onChange: event => emit('select', props.items.find(item => item.value === event.target.value))
  }, props.items.map(item => h('option', { value: item.value }, item.label))) }
}) }))
let app, host
const flush = async () => {
  await new Promise(resolve => setTimeout(resolve, 20))
  for (let i = 0; i < 15; i++) { await Promise.resolve(); await nextTick() }
}
afterEach(() => { app?.unmount(); host?.remove(); vi.clearAllMocks() })

test.each(['public', 'personal'])('loads, saves and clears the signature in the existing %s inbox form', async accessMode => {
  mocks.getInboxSignatures.mockResolvedValue({ data: { data: [{ id: 11, name: 'Support' }, { id: 22, name: 'Personal' }] } })
  const submit = vi.fn()
  const values = {
    access_mode: accessMode, owner_user_id: accessMode === 'personal' ? 101 : null,
    owner_user_ids: accessMode === 'personal' ? [101] : [],
    default_signature_id: 11, name: 'Support inbox', from: 'support@example.com',
    auth_type: 'password',
    imap: { host: 'imap.example.com', port: 993, mailbox: 'INBOX', username: 'support', password: 'secret', tls_type: 'tls', read_interval: '5m', scan_inbox_since: '48h' },
    smtp: { host: 'smtp.example.com', port: 587, username: 'support', password: 'secret', max_conns: 10, max_msg_retries: 3, idle_timeout: '25s', pool_wait_timeout: '120s', auth_protocol: 'login', tls_type: 'starttls' }
  }
  host = document.createElement('div'); document.body.append(host)
  app = createApp(EmailInboxForm, { initialValues: values, submitForm: submit })
  app.config.globalProperties.$t = key => key
  app.mount(host); await flush()
  expect(mocks.getInboxSignatures).toHaveBeenCalledOnce()
  const select = host.querySelector('[data-signature]')
  expect(select.value).toBe('11')
  expect(select.textContent).toContain('Support')
  select.value = '22'; select.dispatchEvent(new Event('change')); await flush()
  host.querySelector('form').dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }))
  await flush()
  expect(submit).toHaveBeenLastCalledWith(expect.objectContaining({ default_signature_id: 22, name: values.name, from: values.from, access_mode: accessMode }))
  select.value = 'none'; select.dispatchEvent(new Event('change')); await flush()
  host.querySelector('form').dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }))
  await flush()
  expect(submit).toHaveBeenLastCalledWith(expect.objectContaining({ default_signature_id: null }))
  mocks.getInboxSignatures.mockResolvedValue({ data: { data: [{ id: 11, name: 'Renamed' }, { id: 33, name: 'New signature' }] } })
  window.dispatchEvent(new Event('focus')); await flush()
  expect(select.textContent).toContain('Renamed')
  expect(select.textContent).toContain('New signature')
  expect(select.value).toBe('none')
})
