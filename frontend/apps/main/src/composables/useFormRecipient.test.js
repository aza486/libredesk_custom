// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import { useFormRecipient } from './useFormRecipient'
let app, host
const flush = async () => { await Promise.resolve(); await nextTick(); await Promise.resolve() }
afterEach(() => { app?.unmount(); host?.remove(); vi.useRealTimers() })
const form = (email = '', blocked = !email) => ({ uuid: 'a', reply_recipient: { managed: true, blocked, email } })
function mount(initial, load = vi.fn(), saved = null) {
  vi.useFakeTimers()
  const conversation = ref(initial)
  const recipients = [ref('bestellung@pixelfotoexpress.de'), ref('unchecked@example.com'), ref('unchecked-bcc@example.com')]
  const draft = ref(saved)
  let manager
  app = createApp(defineComponent({ setup() {
    manager = useFormRecipient(conversation, recipients, load, { draft })
    return () => h('div')
  } }))
  host = document.createElement('div'); document.body.append(host); app.mount(host)
  return { conversation, recipients, ...manager, draft, load }
}
test('prefills only the form TO and excludes inherited CC/BCC once', () => {
  const f = mount(form('customer@example.com'))
  expect(f.recipients.map(r => r.value)).toEqual(['customer@example.com', '', ''])
  expect(f.draft.value).toBeNull()
})
for (const value of ['corrected@example.com', 'another@example.com', 'one@example.com, two@example.com', '']) {
  test(`preserves deliberate TO change ${JSON.stringify(value)} during refresh and outage`, async () => {
    const f = mount(form('typo@example.com'), vi.fn().mockResolvedValue({ managed: true, email: 'new@example.com' }))
    f.recipients[0].value = value
    f.recipients[1].value = 'cc@example.com, second@example.com'
    f.recipients[2].value = 'bcc@example.com'
    await vi.advanceTimersByTimeAsync(5000)
    f.conversation.value = form('', true); await flush()
    expect(f.recipients.map(r => r.value)).toEqual([value, 'cc@example.com, second@example.com', 'bcc@example.com'])
    expect(f.draft.value.edited).toEqual({ to: true, cc: true, bcc: true })
  })
}
test('late form data fills untouched TO without overwriting manually entered CC/BCC', async () => {
  const f = mount(form(), vi.fn().mockResolvedValue({ managed: true, email: 'late@example.com' }))
  expect(f.recipients.map(r => r.value)).toEqual(['', '', ''])
  f.recipients[1].value = 'cc@example.com'
  await vi.advanceTimersByTimeAsync(5000)
  expect(f.recipients.map(r => r.value)).toEqual(['late@example.com', 'cc@example.com', ''])
})
test('explicit input on an already empty TO prevents a later automatic fill', async () => {
  const f = mount(form(), vi.fn().mockResolvedValue({ managed: true, email: 'late@example.com' }))
  f.markEdited('to')
  await vi.advanceTimersByTimeAsync(5000)
  expect(f.recipients[0].value).toBe('')
  expect(f.draft.value.edited.to).toBe(true)
})
test('saved recipient edits and deliberately empty fields survive hydration and background updates', async () => {
  const saved = { to: '', cc: 'saved@example.com', bcc: 'hidden@example.com', edited: { to: true, cc: true, bcc: true } }
  const f = mount(form('form@example.com'), vi.fn().mockResolvedValue({ managed: true, email: 'late@example.com' }), saved)
  expect(f.recipients.map(r => r.value)).toEqual(['', 'saved@example.com', 'hidden@example.com'])
  await vi.advanceTimersByTimeAsync(5000)
  expect(f.recipients[0].value).toBe('')
})
test('late draft hydration preserves edits made while loading', () => {
  const f = mount(form('form@example.com'))
  f.recipients[0].value = 'manual@example.com'
  f.draft.value = { to: 'old@example.com', cc: 'saved@example.com', bcc: '', edited: { to: true, cc: true, bcc: false } }
  expect(f.recipients[0].value).toBe('manual@example.com')
  expect(f.recipients[1].value).toBe('saved@example.com')
})
test('outage keeps a shown suggestion; missing initial data never uses the technical address', async () => {
  const f = mount(form('customer@example.com'), vi.fn().mockRejectedValue(new Error('offline')))
  await vi.advanceTimersByTimeAsync(5000)
  expect(f.state.value.blocked).toBe(true)
  expect(f.recipients[0].value).toBe('customer@example.com')
  f.draft.value = null
  f.conversation.value = { ...form(), uuid: 'b' }; await flush()
  expect(f.recipients[0].value).toBe('')
})
test('late response from a previous conversation cannot replace current recipients', async () => {
  let resolve
  const f = mount(form(), () => new Promise(r => { resolve = r }))
  await vi.advanceTimersByTimeAsync(5000)
  f.conversation.value = { ...form('b@example.com'), uuid: 'b' }; await flush()
  resolve({ managed: true, email: 'a@example.com' }); await flush()
  expect(f.recipients[0].value).toBe('b@example.com')
})
test('ordinary email defaults and editing remain independent of the form integration', async () => {
  const f = mount({ uuid: 'normal', reply_recipient: { managed: false } })
  f.recipients[0].value = 'manual@example.com'
  await vi.advanceTimersByTimeAsync(15000)
  expect(f.recipients[0].value).toBe('manual@example.com')
  expect(f.load).not.toHaveBeenCalled()
  expect(f.draft.value).toBeNull()
})
