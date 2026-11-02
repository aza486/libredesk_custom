// @vitest-environment jsdom
import { beforeEach, afterEach, test, expect, vi } from 'vitest'
import { createApp, h, reactive, nextTick } from 'vue'
import { useLearningReview } from './useLearningReview'
const api = vi.hoisted(() => ({
  getLearningItems: vi.fn(),
  getLearningItem: vi.fn(),
  reviewLearningItem: vi.fn()
}))
vi.mock('@main/api', () => ({ default: api }))
let app, root, review, user
const item = () => ({
  id: 'candidate',
  revision: 1,
  decision: 'pending',
  active: false,
  approved: false,
  publish_state: 'unpublished',
  review_ready: true,
  content: { topic: 'Bürozeiten' }
})
const response = (data) => ({ data: { data } })
const flush = async () => {
  for (let n = 0; n < 12; n++) {
    await Promise.resolve()
    await nextTick()
  }
}
beforeEach(async () => {
  vi.resetAllMocks()
  api.getLearningItems.mockResolvedValue(response({ items: [item()], has_more: false }))
  api.getLearningItem.mockResolvedValue(response(item()))
  api.reviewLearningItem.mockResolvedValue(response({ id: 'candidate' }))
  user = reactive({ userID: 42, hasAdminRole: true, can: () => false })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    setup() {
      review = useLearningReview(user)
      return () => h('div')
    }
  })
  app.mount(root)
  await flush()
})
afterEach(() => {
  app.unmount()
  root.remove()
})
test('pending list and detail use the authenticated API and preserve explicit publication flags', async () => {
  expect(api.getLearningItems).toHaveBeenCalledWith({
    tab: 'pending',
    type: '',
    status: '',
    page: 1
  })
  await review.open(item())
  expect(review.selected.value.approved).toBe(false)
  expect(review.selected.value.publish_state).toBe('unpublished')
  expect(review.selected.value.content.topic).toBe('Bürozeiten')
})
test('normal employees cannot read or mutate, dedicated reviewers can', async () => {
  user.hasAdminRole = false
  user.userID++
  await flush()
  expect(review.items.value).toEqual([])
  await review.open(item())
  expect(review.selected.value).toBeNull()
  expect(api.getLearningItem).not.toHaveBeenCalled()
  user.can = (p) => p === 'ai_learning:manage'
  user.userID++
  await flush()
  expect(review.items.value).toHaveLength(1)
})
test('only two decisions send revision without flags, content or required reason', async () => {
  await review.open(item())
  let resolve
  api.reviewLearningItem.mockReturnValue(
    new Promise((ok) => {
      resolve = ok
    })
  )
  const saving = review.act('learn')
  await review.act('learn')
  expect(api.reviewLearningItem).toHaveBeenCalledTimes(1)
  expect(api.reviewLearningItem).toHaveBeenCalledWith('candidate', { action: 'learn', revision: 1 })
  expect(review.selected.value.active).toBe(false)
  resolve(response({ id: 'candidate' }))
  await saving
})
test('discard uses the same versioned path and obsolete actions cannot be sent', async () => {
  await review.open(item())
  for (const action of ['edit', 'create', 'derive', 'approve', 'retry']) await review.act(action)
  expect(api.reviewLearningItem).not.toHaveBeenCalled()
  await review.act('discard')
  expect(api.reviewLearningItem).toHaveBeenCalledWith('candidate', {
    action: 'discard',
    revision: 1
  })
})
test('legacy incomplete candidate and history cannot be learned', async () => {
  api.getLearningItem.mockResolvedValue(response({ ...item(), review_ready: false }))
  await review.open(item())
  await review.act('learn')
  expect(api.reviewLearningItem).not.toHaveBeenCalled()
})
test('conflict and network failure do not show successful approval', async () => {
  await review.open(item())
  api.reviewLearningItem.mockRejectedValue({ response: { status: 409 } })
  await review.act('learn')
  expect(review.error.value).toContain('geändert')
  expect(review.selected.value.active).toBe(false)
  api.reviewLearningItem.mockRejectedValue(new Error('network'))
  await review.act('learn')
  expect(review.error.value).toContain('nicht gespeichert')
})
test('pagination and filters are server-side and old list/detail responses cannot cross employee changes', async () => {
  review.page.value = 2
  await flush()
  expect(api.getLearningItems.mock.lastCall[0].page).toBe(2)
  review.type.value = 'employee_fact'
  await flush()
  expect(api.getLearningItems.mock.lastCall[0]).toMatchObject({ type: 'employee_fact', page: 1 })
  let resolve
  api.getLearningItem.mockReturnValue(
    new Promise((ok) => {
      resolve = ok
    })
  )
  const opening = review.open(item())
  user.userID++
  await flush()
  resolve(response(item()))
  await opening
  expect(review.selected.value).toBeNull()
})
