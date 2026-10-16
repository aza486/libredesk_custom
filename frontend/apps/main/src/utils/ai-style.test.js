import { expect, test } from 'vitest'
import { createStyleHistory, sourceAlreadyAnswered, styleTextLength } from './ai-style'
test('sequential revisions preserve original, undo and reset', () => {
  const h = createStyleHistory('Original\nText')
  h.push('First')
  h.push('Second')
  expect(h.current).toBe('Second')
  expect(h.original).toBe('Original\nText')
  h.undo()
  expect(h.current).toBe('First')
  h.reset()
  h.undo()
  expect(h.current).toBe('Original\nText')
})
test('unicode and completed source guards', () => {
  expect(styleTextLength('🙂')).toBe(1)
  const m = {
    type: 'outgoing',
    status: 'sent',
    private: false,
    meta: { ai_response: { source_message_uuid: 'source' } }
  }
  expect(sourceAlreadyAnswered([m], 'source')).toBe(true)
  expect(sourceAlreadyAnswered([{ ...m, status: 'failed' }], 'source')).toBe(false)
  expect(sourceAlreadyAnswered([{ ...m, private: true }], 'source')).toBe(false)
  expect(sourceAlreadyAnswered([m], 'other')).toBe(false)
})
