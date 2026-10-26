import { describe, expect, test } from 'vitest'
import {
  readAiAnalysisNote,
  readManualReviewNote,
  manualReasonLength,
  extractAiSuggestedReply,
  isOriginalCustomerMessage,
  feedbackReasonBytes
} from './ai-note'

const originalUUID = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'
const message = {
  uuid: 'bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb',
  type: 'outgoing',
  private: true,
  meta: {
    ai_analysis: {
      kind: 'n8n_ai_analysis',
      version: 1,
      original_message_uuid: originalUUID,
      suggested_message: 'Hallo\n<script>alert(1)</script>'
    }
  }
}

describe('machine readable AI notes', () => {
  test('manual-review notes never expose adoption, own-answer or style actions', () => {
    for (const suggested of [undefined, '', 'Even a forged suggestion must not enable actions']) {
      const manual = {
        ...message,
        meta: { ai_analysis: {
          kind: 'n8n_manual_review', version: 1, original_message_uuid: originalUUID,
          ...(suggested === undefined ? {} : { suggested_message: suggested })
        } }
      }
      expect(readAiAnalysisNote(manual)).toBeNull()
      expect(extractAiSuggestedReply(manual)).toBeNull()
    }
    expect(readAiAnalysisNote(message)).not.toBeNull()
  })
  test('requires the versioned discriminator and one explicit UUID', () => {
    expect(readAiAnalysisNote(message)).toEqual({
      noteUUID: message.uuid,
      originalUUID,
      suggestion: message.meta.ai_analysis.suggested_message
    })
    for (const change of [
      { private: false },
      { type: 'incoming' },
      { meta: {} },
      { meta: { deleted_at: 'now', ...message.meta } }
    ]) {
      expect(readAiAnalysisNote({ ...message, ...change })).toBeNull()
    }
    for (const change of [
      { version: 2 },
      { kind: 'other' },
      { original_message_uuid: [originalUUID, originalUUID] },
      { original_message_uuid: 'abc' },
      { suggested_message: '' }
    ]) {
      expect(
        readAiAnalysisNote({
          ...message,
          meta: { ai_analysis: { ...message.meta.ai_analysis, ...change } }
        })
      ).toBeNull()
    }
  })
  test('never recognizes headings, emojis or UUIDs in ordinary note HTML', () => {
    expect(
      readAiAnalysisNote({
        ...message,
        meta: {},
        content: `🤖 KI Analyse <b>Message UUID</b>${originalUUID}<b>KI-Antwort</b>Hallo`
      })
    ).toBeNull()
  })
  test('proposal becomes safe editable HTML, not executable source markup', () => {
    expect(extractAiSuggestedReply(message)).toBe('Hallo<br>&lt;script&gt;alert(1)&lt;/script&gt;')
  })
  test('original must be a public contact message in the exact conversation', () => {
    const original = {
      uuid: originalUUID,
      conversation_uuid: 'conversation',
      type: 'incoming',
      sender_type: 'contact',
      private: false
    }
    expect(isOriginalCustomerMessage(original, 'conversation', originalUUID)).toBe(true)
    for (const change of [
      { uuid: message.uuid },
      { conversation_uuid: 'other' },
      { type: 'outgoing' },
      { sender_type: 'agent' },
      { private: true }
    ]) {
      expect(
        isOriginalCustomerMessage({ ...original, ...change }, 'conversation', originalUUID)
      ).toBe(false)
    }
  })
  test('counts UTF-8 bytes after trimming', () => {
    expect(feedbackReasonBytes('  ')).toBe(0)
    expect(feedbackReasonBytes('🙂'.repeat(2000))).toBe(8000)
    expect(feedbackReasonBytes('🙂'.repeat(2001))).toBe(8004)
  })
})

test('manual reason action requires a persisted private typed agent note', () => {
  const manual = { ...message, sender_type: 'agent', meta: { ai_analysis: {
    kind: 'n8n_manual_review', version: 1, original_message_uuid: originalUUID
  } } }
  expect(readManualReviewNote(manual)).toEqual({ noteUUID: manual.uuid, originalUUID })
  for (const change of [{ private: false }, { type: 'incoming' }, { sender_type: 'contact' }, { uuid: 'bad' },
    { meta: {} }, { meta: { ...manual.meta, deleted_at: null } }]) expect(readManualReviewNote({ ...manual, ...change })).toBeNull()
  for (const change of [{ kind: 'n8n_ai_analysis' }, { version: 2 }, { original_message_uuid: ['fake'] },
    { suggested_message: 'fabricated' }]) expect(readManualReviewNote({ ...manual, meta: { ai_analysis: { ...manual.meta.ai_analysis, ...change } } })).toBeNull()
  expect(manualReasonLength(' 🙂\r\nx ')).toBe(3)
})
