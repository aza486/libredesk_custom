// @vitest-environment jsdom
import { afterEach, beforeEach, test, expect, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import AILearning from './AILearning.vue'
const mocks = vi.hoisted(() => ({
  emit: vi.fn(),
  api: { getLearningItems: vi.fn(), getLearningItem: vi.fn(), reviewLearningItem: vi.fn() }
}))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: mocks.emit }) }))
vi.mock('@main/api', () => ({ default: mocks.api }))
vi.mock('@main/stores/user', () => ({
  useUserStore: () => ({ userID: 42, hasAdminRole: true, can: () => true })
}))
vi.mock('@main/layouts/admin/AdminSplitLayout.vue', () => ({
  default: defineComponent({
    setup(_, { slots }) {
      return () => h('main', slots.content?.())
    }
  })
}))
vi.mock('@main/components/layout/LoadingOverlay.vue', () => ({
  default: defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
}))
vi.mock('@shared-ui/components/ui/dialog', () => {
  const Wrapper = defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
  return {
    Dialog: defineComponent({
      props: ['open'],
      setup(p, { slots }) {
        return () => (p.open ? h('section', slots.default?.()) : null)
      }
    }),
    DialogContent: Wrapper,
    DialogHeader: Wrapper,
    DialogTitle: Wrapper,
    DialogDescription: Wrapper
  }
})
let root, app
const fixture = () => ({
  id: 'candidate',
  record_type: 'candidate',
  review_exists: true,
  review_ready: true,
  context: {
    customer_question: 'Vollständige Anfrage von Kunde Beispiel',
    human_answer: 'Tatsächlich versendeter Text'
  },
  ai_suggestion: 'Anonymisierter ursprünglicher KI-Vorschlag',
  created_at: '2026-10-30T10:20:00Z',
  revision: 1,
  content: {
    source_type: 'employee_fact',
    category: 'firmenfragen',
    topic: '<img src=x onerror=alert(1)>',
    original_categories: ['bestellung'],
    rule: 'Das Büro ist samstags geschlossen.',
    evidence: 'Das Büro ist samstags geschlossen.',
    limits: 'Nur Büro',
    source: 'human_answer'
  },
  quality_status: 'pending',
  decision: 'pending',
  approved: false,
  active: false,
  publish_state: 'unpublished',
  feedback_reason: 'Kürzer antworten',
  rationale: 'Beleg prüfen'
})
const response = (data) => ({ data: { data } })
const flush = async () => {
  for (let n = 0; n < 15; n++) {
    await Promise.resolve()
    await nextTick()
  }
}
const click = (text) =>
  [...root.querySelectorAll('button')].find((b) => b.textContent === text).click()
beforeEach(async () => {
  vi.resetAllMocks()
  mocks.api.getLearningItems.mockResolvedValue(response({ items: [fixture()], has_more: false }))
  mocks.api.getLearningItem.mockResolvedValue(response(fixture()))
  mocks.api.reviewLearningItem.mockResolvedValue(response({ id: 'candidate' }))
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(AILearning)
  app.mount(root)
  await flush()
})
afterEach(() => {
  app.unmount()
  root.remove()
})
test('two tabs and proposal cards replace manual forms, customer text remains plaintext', async () => {
  expect(root.querySelector('img')).toBeNull()
  expect(root.querySelector('table')).toBeNull()
  expect(root.textContent).toContain('Zur Prüfung')
  expect(root.textContent).not.toContain('Übersprungen')
  click('Vorschlag prüfen')
  await flush()
  for (const heading of [
    'Was hat die KI erkannt?',
    'Was möchte die KI lernen?',
    'Warum muss das geprüft werden?',
    'Kontext'
  ])
    expect(root.textContent).toContain(heading)
  expect(root.textContent).toContain('Vollständige Anfrage von Kunde Beispiel')
  expect(root.textContent).toContain('Tatsächlich versendeter Text')
  expect(root.querySelector('textarea, input, select')).toBeNull()
  expect(root.querySelector('img')).toBeNull()
})
test('learn is green, sends only decision and revision, toast waits for API', async () => {
  click('Vorschlag prüfen')
  await flush()
  let resolve
  mocks.api.reviewLearningItem.mockReturnValue(
    new Promise((ok) => {
      resolve = ok
    })
  )
  const button = [...root.querySelectorAll('button')].find((b) => b.textContent === 'Lernen')
  expect(button.className).toContain('bg-green-600')
  button.click()
  await flush()
  expect(mocks.api.reviewLearningItem).toHaveBeenCalledWith('candidate', {
    action: 'learn',
    revision: 1
  })
  expect(mocks.emit).not.toHaveBeenCalled()
  resolve(response({ id: 'candidate' }))
  await flush()
  expect(mocks.emit).toHaveBeenCalledTimes(1)
})
test('discard has no extra required field and clears selection after confirmation', async () => {
  click('Vorschlag prüfen')
  await flush()
  click('Verwerfen')
  await flush()
  expect(mocks.api.reviewLearningItem).toHaveBeenCalledWith('candidate', {
    action: 'discard',
    revision: 1
  })
})
test.each([409, 503])(
  'API %s retains proposal and displays no successful decision',
  async (status) => {
    click('Vorschlag prüfen')
    await flush()
    mocks.api.reviewLearningItem.mockRejectedValue({ response: { status } })
    click('Lernen')
    await flush()
    expect(root.querySelector('[role="alert"]')).toBeTruthy()
    expect(mocks.emit).not.toHaveBeenCalled()
    expect(root.textContent).toContain('Was möchte die KI lernen?')
  }
)
test('history and incomplete legacy candidates expose no manual forms or learn button', async () => {
  mocks.api.getLearningItem.mockResolvedValue(response({ ...fixture(), review_ready: false }))
  click('Verlauf')
  await flush()
  click('Ansehen')
  await flush()
  expect(root.textContent).toContain('Keine offene Entscheidung')
  expect([...root.querySelectorAll('button')].some((b) => b.textContent === 'Lernen')).toBe(false)
  expect(root.querySelector('textarea, input, select')).toBeNull()
})
