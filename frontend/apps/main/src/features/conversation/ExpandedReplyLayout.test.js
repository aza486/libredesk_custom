// @vitest-environment jsdom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive, ref, shallowRef, Teleport } from 'vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import ExpandedReplyLayout from './ExpandedReplyLayout.vue'
import ReplyBoxContent from './ReplyBoxContent.vue'

const state = vi.hoisted(() => ({
  store: null,
  mobile: null,
  mounts: 0,
  send: null,
  sendStatus: null
}))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => state.store }))
vi.mock('@main/stores/macro', () => ({ useMacroStore: () => ({ setCurrentView: () => {} }) }))
vi.mock('@main/stores/users', () => ({ useUsersStore: () => ({ users: [] }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: () => {} }) }))
vi.mock('@shared-ui/composables', () => ({ useIsMobile: () => state.mobile }))
vi.mock('@main/composables/useIsComposerCramped', () => ({
  useIsComposerCramped: () => state.mobile
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@main/components/editor/ConversationEditor.vue', () => ({
  default: defineComponent({
    props: ['htmlContent'],
    emits: ['update:htmlContent', 'update:textContent'],
    setup(props, { emit, expose }) {
      state.mounts++
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
vi.mock('./ReplyBoxMenuBar.vue', () => ({
  default: defineComponent({
    props: ['handleSend', 'handleSendAndSetStatus', 'enableSend', 'outgoingTemplates'],
    setup: (props) => () =>
      h('footer', [
        h(
          'button',
          { 'data-send': '', disabled: !props.enableSend, onClick: props.handleSend },
          'Send'
        ),
        h(
          'button',
          { 'data-status-send': '', onClick: () => props.handleSendAndSetStatus('Resolved') },
          'Send and resolve'
        ),
        h('span', { 'data-templates': '' }, props.outgoingTemplates?.[0]?.name)
      ])
  })
}))
vi.mock('./MacroActionsPreview.vue', () => ({ default: { render: () => null } }))
vi.mock('./message/attachment/ReplyBoxAttachmentPreview.vue', () => ({
  default: { render: () => null }
}))

let app, root, draft, target, open, contextActive
const flush = async () => {
  for (let i = 0; i < 8; i++) await nextTick()
}
const clickTab = async (value) => {
  const button =
    root.querySelector(`[role="tab"][data-state][value="${value}"]`) ||
    [...root.querySelectorAll('[role="tab"]')].filter((el) => el.textContent === value).at(-1)
  expect(button).toBeTruthy()
  button.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }))
  button.click()
  await flush()
}
const panel = (value) =>
  [...root.querySelectorAll('[role="tabpanel"]')].find((el) => el.id.endsWith(`content-${value}`))

beforeEach(async () => {
  state.mobile = ref(false)
  state.mounts = 0
  state.send = vi.fn()
  state.sendStatus = vi.fn()
  state.store = reactive({
    current: {
      subject: 'Very long subject '.repeat(40),
      status: 'Open',
      priority: 'High',
      inbox_channel: 'email',
      contact: { email: 'long'.repeat(50) + '@example.com' }
    },
    currentContactName: 'Customer name',
    statusOptions: [{ value: 'open', label: 'Open' }],
    getMacro: () => null,
    updateStatus: vi.fn()
  })
  draft = reactive({
    htmlContent: 'Prepared answer',
    textContent: 'Prepared answer',
    messageType: 'reply',
    to: state.store.current.contact.email,
    cc: 'cc@example.com',
    bcc: 'bcc@example.com',
    showBcc: true
  })
  target = shallowRef(null)
  open = ref(true)
  contextActive = ref(false)
  const history = defineComponent({
    setup: () => () =>
      h('div', { 'data-history': '', style: 'overflow-y:auto' }, 'Existing history')
  })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      h(TooltipProvider, null, {
        default: () => [
          h(Teleport, { to: target.value || 'body', disabled: !target.value }, [h(history)]),
          open.value &&
            h(
              ExpandedReplyLayout,
              {
                privateNote: draft.messageType === 'private_note',
                contextActive: contextActive.value,
                onHistoryTarget: (element) => {
                  target.value = element
                },
                onRestore: () => {
                  open.value = false
                }
              },
              {
                default: () =>
                  h(ReplyBoxContent, {
                    ...draft,
                    isFullscreen: true,
                    hideFullscreenToggle: true,
                    aiPrompts: [],
                    isSending: false,
                    uploadingFiles: [],
                    uploadedFiles: [],
                    outgoingTemplates: [{ id: 1, name: 'Signature' }],
                    ...Object.fromEntries(
                      Object.keys(draft).map((key) => [
                        `onUpdate:${key}`,
                        (value) => {
                          draft[key] = value
                        }
                      ])
                    ),
                    onSend: state.send,
                    onSendAndSetStatus: state.sendStatus
                  })
              }
            )
        ]
      })
  })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await flush()
})
afterEach(() => {
  app?.unmount()
  root?.remove()
})

test('desktop shows both panes, actual metadata, existing priority bars and one restore button', () => {
  expect(panel('history').style.display).not.toBe('none')
  expect(panel('history').hidden).toBe(false)
  expect(panel('reply').style.display).not.toBe('none')
  expect(root.querySelector('header').textContent).toContain(state.store.current.subject)
  expect(root.querySelector('header').textContent).toContain(state.store.current.contact.email)
  expect(root.querySelectorAll('svg[role="img"] rect')).toHaveLength(3)
  expect(root.querySelectorAll('[aria-label="replyBox.restoreEditor"]')).toHaveLength(1)
  expect(root.querySelector('[data-templates]').textContent).toBe('Signature')
})

test('mobile switches mounted panes and preserves edited draft, recipients and history position through resize', async () => {
  const editor = root.querySelector('[data-editor]')
  const history = root.querySelector('[data-history]')
  history.scrollTop = 127
  editor.value = 'Edited answer'
  editor.dispatchEvent(new Event('input', { bubbles: true }))
  state.mobile.value = true
  await flush()
  expect(panel('history').style.display).toBe('none')
  await clickTab('replyBox.history')
  expect(panel('reply').style.display).toBe('none')
  expect(panel('history').style.display).not.toBe('none')
  await clickTab('globals.terms.reply')
  expect(root.querySelector('[data-editor]')).toBe(editor)
  expect(editor.value).toBe('Edited answer')
  expect(history.scrollTop).toBe(127)
  expect(draft.to).toBe(state.store.current.contact.email)
  expect(draft.cc).toBe('cc@example.com')
  expect(draft.bcc).toBe('bcc@example.com')
  state.mobile.value = false
  await flush()
  expect(panel('history').style.display).not.toBe('none')
  expect(panel('reply').style.display).not.toBe('none')
  expect(state.mounts).toBe(1)
})

test('reply/private-note tabs and both send paths still work inside the expanded editor', async () => {
  await clickTab('globals.terms.privateNote')
  expect(draft.messageType).toBe('private_note')
  expect(root.querySelector('input')).toBeNull()
  await clickTab('globals.terms.reply')
  expect(draft.messageType).toBe('reply')
  root.querySelector('[data-send]').click()
  await flush()
  expect(state.send).toHaveBeenCalledTimes(1)
  root.querySelector('[data-status-send]').click()
  await flush()
  expect(state.sendStatus).toHaveBeenCalledWith('Resolved')
})

test('restoring returns the same history element and clears its popup target', async () => {
  const history = root.querySelector('[data-history]')
  expect(target.value.contains(history)).toBe(true)
  root.querySelector('[aria-label="replyBox.restoreEditor"]').click()
  await flush()
  expect(target.value).toBeNull()
  expect(root.querySelector('[data-history]')).toBe(history)
  expect(root.contains(history)).toBe(true)
})


test('subject row and its shared lower border span the entire header above both panes', () => {
  const header = root.querySelector('header')
  const subjectRow = header.children[1]
  expect(subjectRow.classList.contains('w-full')).toBe(true)
  expect(subjectRow.className).not.toContain('md:w-1/2')
  expect(header.classList.contains('border-b')).toBe(true)
  expect(header.contains(panel('history'))).toBe(false)
  expect(panel('history').classList.contains('md:border-r')).toBe(true)
})


test('context navigation hides and restores the existing history without remounting the editor', async () => {
  const history = root.querySelector('[data-history]')
  const editor = root.querySelector('[data-editor]')
  history.scrollTop = 250
  contextActive.value = true
  await flush()
  expect(history.parentElement.style.display).toBe('none')
  expect(root.querySelector('[data-editor]')).toBe(editor)
  contextActive.value = false
  await flush()
  expect(root.querySelector('[data-history]')).toBe(history)
  expect(history.parentElement.style.display).not.toBe('none')
  expect(history.scrollTop).toBe(250)
  expect(editor.value).toBe('Prepared answer')
  expect(state.mounts).toBe(1)
})
