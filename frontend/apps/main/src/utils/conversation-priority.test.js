import { expect, test } from 'vitest'
import { createI18n } from 'vue-i18n'
import de from '../../../../../i18n/de-DE.json'
import { conversationPriorityLabel } from './conversation-priority'
import { conversationActivityLabel } from './conversation-activity'

const t = createI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } }).global.t

test.each([
  ['high', 'Hoch'], ['High', 'Hoch'], ['medium', 'Mittel'], ['Medium', 'Mittel'],
  ['low', 'Niedrig'], ['Low', 'Niedrig'], ['urgent', 'urgent'], ['Urgent', 'Urgent'],
  ['HIGH', 'HIGH'], ['High priority customer', 'High priority customer'],
  [' High ', ' High '], ['toString', 'toString'], ['__proto__', '__proto__']
])('priority %s displays as %s without changing the value', (value, label) => {
  expect(conversationPriorityLabel(value, t)).toBe(label)
  const message = { type: 'activity', sender_type: 'agent', private: true, content_type: 'text',
    content: `Daniel set priority to ${value}`, meta: { activity_type: 'priority_change' } }
  const original = message.content
  // Noncanonical outer whitespace is rejected by the existing event formatter.
  const expected = value.trim() === value ? `Daniel hat die Priorität auf ${label} gesetzt` : original
  expect(conversationActivityLabel(message, t)).toBe(expected)
  expect(message.content).toBe(original)
  if (value.trim() === value) expect(conversationActivityLabel({ ...message, meta: null }, t)).toBe(expected)
})

test('missing priority translation keeps original names', () => {
  expect(conversationPriorityLabel('High')).toBe('High')
  expect(conversationPriorityLabel('High', (key) => key)).toBe('High')
})
