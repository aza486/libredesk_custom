import { expect, test } from 'vitest'
import { prioritizeConversationTags, warningTagKind } from './warning-tags'

test.each([
  [
    ['Kundenticket', 'Intern'],
    ['Kundenticket', 'Intern']
  ],
  [
    ['Intern', 'Mensch erforderlich'],
    ['Mensch erforderlich', 'Intern']
  ],
  [
    ['Intern', 'Verarbeitungsfehler'],
    ['Verarbeitungsfehler', 'Intern']
  ],
  [
    ['😎 Mensch erforderlich', '🛑 Verarbeitungsfehler'],
    ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich']
  ],
  [
    ['Kundenticket', '😎 Mensch erforderlich', 'Intern', '🛑 Verarbeitungsfehler'],
    ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich', 'Kundenticket', 'Intern']
  ],
  [
    ['🛑 Verarbeitungsfehler', 'Intern', '😎 Mensch erforderlich', 'Support'],
    ['🛑 Verarbeitungsfehler', '😎 Mensch erforderlich', 'Intern', 'Support']
  ]
])('prioritizes warnings without changing source arrays: %j', (tags, expected) => {
  Object.freeze(tags)
  const original = [...tags]
  expect(prioritizeConversationTags(tags)).toEqual(expected)
  expect(tags).toEqual(original)
})

test('recognizes complete names with existing icons and spacing, not unrelated names', () => {
  expect(warningTagKind(' 🛑 VERARBEITUNGSFEHLER ')).toBe('error')
  expect(warningTagKind('⚠️ Mensch   erforderlich')).toBe('human')
  expect(warningTagKind('Kein Verarbeitungsfehler')).toBeNull()
  expect(warningTagKind('Mensch erforderlich erledigt')).toBeNull()
  expect(warningTagKind(null)).toBeNull()
})
