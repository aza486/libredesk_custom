// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import { useFormRecipient } from './useFormRecipient'
let app, host
const flush = async () => { await Promise.resolve(); await nextTick(); await Promise.resolve() }
afterEach(() => { app?.unmount(); host?.remove(); vi.useRealTimers() })
function mount(initial, load = vi.fn()) {
  vi.useFakeTimers()
  const conversation = ref(initial)
  const recipients = [ref('bestellung@pixelfotoexpress.de'), ref('cc@example.com'), ref('bcc@example.com')]
  let state
  app = createApp(defineComponent({ setup() {
    state = useFormRecipient(conversation, recipients, load)
    return () => h('div')
  } }))
  host = document.createElement('div'); document.body.append(host); app.mount(host)
  return { conversation, recipients, state, load }
}
test('resolved form overrides old drafts and excludes CC/BCC, including subsequent changes', async () => {
  const f = mount({ uuid: 'a', reply_recipient: { managed: true, blocked: false, email: 'customer@example.com' } })
  expect(f.recipients.map(r => r.value)).toEqual(['customer@example.com', '', ''])
  f.recipients[0].value = 'bestellung@pixelfotoexpress.de'
  f.recipients[1].value = 'inherited@example.com'
  await flush()
  expect(f.recipients.map(r => r.value)).toEqual(['customer@example.com', '', ''])
})
test('delayed ingestion is rechecked and outage clears the recipient', async () => {
  const load = vi.fn().mockResolvedValueOnce({ managed: true, blocked: false, email: 'late@example.com' }).mockRejectedValueOnce(new Error('offline'))
  const f = mount({ uuid: 'a', reply_recipient: { managed: true, blocked: true, email: '' } }, load)
  expect(f.recipients.map(r => r.value)).toEqual(['', '', ''])
  await vi.advanceTimersByTimeAsync(5000)
  expect(f.recipients[0].value).toBe('late@example.com')
  await vi.advanceTimersByTimeAsync(5000)
  expect(f.state.value.blocked).toBe(true)
  expect(f.recipients[0].value).toBe('')
})
test('a late response from another conversation is discarded', async () => {
  let resolve
  const f = mount({ uuid: 'a', reply_recipient: { managed: true, blocked: true } }, () => new Promise(r => { resolve = r }))
  await vi.advanceTimersByTimeAsync(5000)
  f.conversation.value = { uuid: 'b', reply_recipient: { managed: true, email: 'b@example.com' } }
  await flush()
  resolve({ managed: true, email: 'a@example.com' }); await flush()
  expect(f.recipients[0].value).toBe('b@example.com')
})
test('ordinary mail recipients stay editable without polling', async () => {
  const f = mount({ uuid: 'normal', reply_recipient: { managed: false } })
  f.recipients[0].value = 'manual@example.com'
  await vi.advanceTimersByTimeAsync(15000)
  expect(f.recipients.map(r => r.value)).toEqual(['manual@example.com', 'cc@example.com', 'bcc@example.com'])
  expect(f.load).not.toHaveBeenCalled()
})
