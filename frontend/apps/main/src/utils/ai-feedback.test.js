import { describe, expect, test, vi } from 'vitest'
import { createAIFeedback } from './ai-feedback'
import { syncServerDraft } from './draft-sync'

const originalUUID = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'
const outgoingUUID = 'bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb'
const note = {
  originalUUID,
  noteUUID: 'cccccccc-cccc-4ccc-cccc-cccccccccccc',
  suggestion: 'AI text'
}
function setup() {
  const map = new Map()
  const drafts = {
    getDraft: (uuid, type) => map.get(`${uuid}:${type}`),
    setDraft: (uuid, type, draft) => map.set(`${uuid}:${type}`, draft),
    removeDraft: (uuid, type) => map.delete(`${uuid}:${type}`)
  }
  const message = {
    uuid: outgoingUUID,
    conversation_uuid: 'conversation',
    sender_id: 42,
    sender_type: 'agent',
    private: false,
    type: 'outgoing',
    status: 'pending',
    meta: { ai_feedback_original_message_uuid: originalUUID, echo_id: 'echo-1' }
  }
  const original = {
    uuid: originalUUID,
    conversation_uuid: 'conversation',
    sender_type: 'contact',
    private: false,
    type: 'incoming'
  }
  const api = {
    saveDraft: vi.fn().mockResolvedValue({}),
    deleteDraft: vi.fn().mockResolvedValue({}),
    getConversationMessage: vi.fn(async (uuid, id) => ({
      data: { data: id === originalUUID ? original : message }
    })),
    getConversationMessages: vi.fn(async () => ({
      data: { data: { results: [message], total_pages: 1 } }
    })),
    saveAIFeedback: vi
      .fn()
      .mockResolvedValue({ data: { data: { id: outgoingUUID, committed: true } } })
  }
  let owner = 42
  const options = { api, drafts, userID: () => owner }
  const feedback = createAIFeedback(options)
  const queue = async (reason = '') => {
    await feedback.arm('conversation', note, reason)
    expect(await feedback.prepare('conversation', 'echo-1')).toBe(originalUUID)
    await feedback.queued('conversation', message)
  }
  return {
    api,
    drafts,
    message,
    original,
    options,
    feedback,
    queue,
    changeOwner: (id) => {
      owner = id
    }
  }
}

describe('feedback lifecycle with the existing API', () => {
  test.each(['pending', 'failed'])('no feedback for %s', async (status) => {
    const s = setup()
    await s.queue()
    s.message.status = status
    await s.feedback.check('conversation')
    expect(s.api.saveAIFeedback).not.toHaveBeenCalled()
    expect(s.feedback.records.conversation.phase).toBe(status === 'failed' ? 'failed' : 'waiting')
  })
  test('persists intent before queueing and submits only a server-confirmed sent reply', async () => {
    const s = setup()
    await s.queue('  Incorrect AI  ')
    expect(s.drafts.getDraft('conversation', 'reply').meta.ai_feedback.reason).toBe('Incorrect AI')
    await s.feedback.check('conversation')
    expect(s.api.saveAIFeedback).not.toHaveBeenCalled()
    s.message.status = 'sent'
    await s.feedback.check('conversation')
    expect(s.api.saveAIFeedback).toHaveBeenCalledWith('conversation', outgoingUUID, {
      feedback_reason: 'Incorrect AI'
    })
    expect(s.feedback.records.conversation.phase).toBe('saved')
    expect(s.feedback.records.conversation.reason).toBe('')
    expect(s.drafts.getDraft('conversation', 'reply')).toBeUndefined()
    await s.feedback.check('conversation', true)
    expect(s.api.saveAIFeedback).toHaveBeenCalledTimes(1)
  })
  test('failed feedback preserves frozen reason, retry has only the feedback API available', async () => {
    const s = setup()
    await s.queue('reason')
    s.message.status = 'sent'
    s.api.saveAIFeedback.mockRejectedValueOnce(new Error('502'))
    await s.feedback.check('conversation')
    expect(s.feedback.records.conversation.phase).toBe('error')
    expect(s.feedback.records.conversation.reason).toBe('reason')
    await s.feedback.check('conversation')
    expect(s.api.saveAIFeedback).toHaveBeenCalledTimes(1)
    await s.feedback.check('conversation', true)
    expect(s.api.saveAIFeedback).toHaveBeenNthCalledWith(2, 'conversation', outgoingUUID, {
      feedback_reason: 'reason'
    })
    expect(s.api.sendMessage).toBeUndefined()
  })
  test('double clicks do not duplicate feedback calls', async () => {
    const s = setup()
    await s.queue()
    s.message.status = 'sent'
    await Promise.all([s.feedback.check('conversation'), s.feedback.check('conversation', true)])
    expect(s.api.saveAIFeedback).toHaveBeenCalledTimes(1)
  })
  test('rejects unconfirmed receipts and keeps the reason', async () => {
    const s = setup()
    await s.queue('reason')
    s.message.status = 'sent'
    s.api.saveAIFeedback.mockResolvedValue({
      data: { data: { id: outgoingUUID, committed: false } }
    })
    await s.feedback.check('conversation')
    expect(s.feedback.records.conversation.phase).toBe('error')
    expect(s.feedback.records.conversation.reason).toBe('reason')
  })
  test('page reload restores operation from the authenticated server draft', async () => {
    const s = setup()
    await s.queue('persistent reason')
    const restored = createAIFeedback(s.options)
    restored.hydrate('conversation')
    expect(restored.records.conversation.outgoingUUID).toBe(outgoingUUID)
    s.message.status = 'sent'
    await restored.check('conversation')
    expect(s.api.saveAIFeedback).toHaveBeenCalledWith('conversation', outgoingUUID, {
      feedback_reason: 'persistent reason'
    })
  })
  test('lost queue response is recovered via the persisted echo ID without sending again', async () => {
    const s = setup()
    await s.feedback.arm('conversation', note, 'reason')
    await s.feedback.prepare('conversation', 'echo-1')
    const restored = createAIFeedback(s.options)
    restored.hydrate('conversation')
    s.message.status = 'sent'
    await restored.check('conversation')
    expect(s.api.saveAIFeedback).toHaveBeenCalledTimes(1)
  })
  test('missing or ambiguous echo references produce no feedback', async () => {
    for (const results of [[], null]) {
      const s = setup()
      await s.feedback.arm('conversation', note, 'reason')
      await s.feedback.prepare('conversation', 'echo-1')
      s.api.getConversationMessages.mockResolvedValue({
        data: { data: { results: results || [s.message, s.message], total_pages: 1 } }
      })
      await s.feedback.check('conversation')
      expect(s.api.saveAIFeedback).not.toHaveBeenCalled()
    }
  })
  test('foreign reply or original cannot cause feedback', async () => {
    for (const target of ['message', 'original']) {
      const s = setup()
      await s.queue()
      s.message.status = 'sent'
      s[target].conversation_uuid = 'other'
      await s.feedback.check('conversation')
      expect(s.api.saveAIFeedback).not.toHaveBeenCalled()
    }
  })
  test('cancel preserves an existing response draft', async () => {
    const s = setup()
    s.drafts.setDraft('conversation', 'reply', { content: 'Existing response', meta: {} })
    await s.feedback.arm('conversation', note, 'reason')
    await s.feedback.cancel('conversation')
    expect(s.drafts.getDraft('conversation', 'reply').content).toBe('Existing response')
  })
  test('oversized reason and another employee cannot resume', async () => {
    const s = setup()
    await expect(s.feedback.arm('conversation', note, '🙂'.repeat(2001))).rejects.toThrow()
    await s.queue()
    s.changeOwner(99)
    const restored = createAIFeedback(s.options)
    restored.hydrate('conversation')
    expect(restored.records.conversation).toBeUndefined()
    await s.feedback.check('conversation')
    expect(s.api.saveAIFeedback).not.toHaveBeenCalled()
  })
  test('draft persistence must succeed before a reply can be queued', async () => {
    const s = setup()
    await s.feedback.arm('conversation', note, '')
    s.api.saveDraft.mockRejectedValue(new Error('offline'))
    await expect(s.feedback.prepare('conversation', 'echo-1')).rejects.toThrow()
    expect(s.feedback.records.conversation.phase).toBe('composing')
  })
  test('serialized autosaves read newest metadata instead of resurrecting old feedback', async () => {
    const s = setup()
    s.drafts.setDraft('conversation', 'reply', {
      content: 'draft',
      meta: { ai_feedback: { reason: 'old' } }
    })
    const first = syncServerDraft(s.api, s.drafts, 'conversation', 'reply')
    s.drafts.setDraft('conversation', 'reply', { content: 'draft', meta: {} })
    const second = syncServerDraft(s.api, s.drafts, 'conversation', 'reply')
    await Promise.all([first, second])
    for (const call of s.api.saveDraft.mock.calls) expect(call[2].meta.ai_feedback).toBeUndefined()
  })
})

test('does not infer uniqueness from an incomplete recovery search', async () => {
  const s = setup()
  await s.feedback.arm('conversation', note, 'reason')
  await s.feedback.prepare('conversation', 'echo-1')
  s.api.getConversationMessages.mockResolvedValue({
    data: { data: { results: [s.message], total_pages: 11 } }
  })
  s.message.status = 'sent'
  await s.feedback.check('conversation')
  expect(s.api.saveAIFeedback).not.toHaveBeenCalled()
})
