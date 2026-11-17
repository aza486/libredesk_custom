import { expect, test } from 'vitest'
import { appendTranscript } from './voice-text'

test('appends to current live text without replacing it', () => {
  expect(appendTranscript('Manual edit', ' spoken words ')).toBe('Manual edit spoken words')
  expect(appendTranscript('Manual edit\n', 'spoken')).toBe('Manual edit\nspoken')
  expect(appendTranscript('', 'Hello')).toBe('Hello')
})
test('checks code points and UTF-8 bytes without splitting or truncating', () => {
  expect(appendTranscript('', '😀', 1, 4)).toBe('😀')
  expect(appendTranscript('', '😀', 1, 3)).toBeNull()
  expect(appendTranscript('a', 'b', 2)).toBeNull()
  expect(appendTranscript('a'.repeat(7998), 'ä', Infinity, 8000)).toBeNull()
  expect(appendTranscript('a'.repeat(7997), 'ä', Infinity, 8000)).toHaveLength(7999)
})
