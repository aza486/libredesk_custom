// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, ref, nextTick } from 'vue'
import { useDraftManager } from './useDraftManager'
const mocks = vi.hoisted(() => ({
  store: null,
  api: { saveDraft: vi.fn().mockResolvedValue({}), deleteDraft: vi.fn().mockResolvedValue({}) }
}))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => mocks.store }))
vi.mock('@main/api', () => ({ default: mocks.api }))
let app, host
const flush = async () => {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
    await nextTick()
  }
}
afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.clearAllMocks()
})
test('clearing a queued response retains the server-side pending feedback record', async () => {
  const map = new Map()
  const record = { version: 1, phase: 'waiting', reason: 'Keep this reason' }
  map.set('conversation:reply', { content: 'sent content', meta: { ai_feedback: record } })
  mocks.store = {
    draftsReady: Promise.resolve(),
    macros: {},
    getMacro: () => ({}),
    getDraft: (uuid, type) => map.get(`${uuid}:${type}`),
    setDraft: (uuid, type, draft) => map.set(`${uuid}:${type}`, draft),
    removeDraft: (uuid, type) => map.delete(`${uuid}:${type}`)
  }
  let manager
  app = createApp(
    defineComponent({
      setup() {
        manager = useDraftManager(ref('conversation'), ref('reply'))
        return () => h('div')
      }
    })
  )
  host = document.createElement('div')
  document.body.append(host)
  app.mount(host)
  await flush()
  expect(manager.htmlContent.value).toBe('sent content')
  manager.clearDraft()
  await flush()
  expect(manager.htmlContent.value).toBe('')
  expect(map.get('conversation:reply')).toEqual({ content: '', meta: { ai_feedback: record } })
  expect(mocks.api.deleteDraft).not.toHaveBeenCalled()
  expect(mocks.api.saveDraft).toHaveBeenCalledWith('conversation', 'reply', {
    content: '',
    meta: { ai_feedback: record }
  })
})

test('draft hydration and saving retain only provenance metadata, clear removes it', async () => {
  const map = new Map()
  const provenance = { response_source: 'ai_adapted', source_message_uuid: 'source', note_uuid: 'note' }
  map.set('conversation:reply', { content: 'Adapted', meta: { ai_response: provenance } })
  mocks.store = {
    draftsReady: Promise.resolve(), macros: {}, getMacro: () => ({}),
    getDraft: (uuid, type) => map.get(`${uuid}:${type}`),
    setDraft: (uuid, type, draft) => map.set(`${uuid}:${type}`, draft),
    removeDraft: (uuid, type) => map.delete(`${uuid}:${type}`)
  }
  let manager
  app = createApp(defineComponent({ setup() { manager = useDraftManager(ref('conversation'), ref('reply')); return () => h('div') } }))
  host = document.createElement('div'); document.body.append(host); app.mount(host); await flush()
  expect(manager.aiResponse.value).toEqual(provenance)
  manager.clearDraft(); await flush()
  expect(manager.aiResponse.value).toBeNull()
  expect(map.has('conversation:reply')).toBe(false)
})
