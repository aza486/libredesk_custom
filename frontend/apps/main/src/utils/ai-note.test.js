import { describe, expect, test } from 'vitest'
import { extractAiSuggestedReply } from './ai-note'

const aiNote = {
  type: 'outgoing',
  private: true,
  content:
    '🤖 KI Analyse<br><br><b>Kategorie: </b><br>Unbekannt<br><br>' +
    '<b>Empfohlene Antwort: </b><br><br>Hallo Daniel,<br><br>gerne helfen wir weiter.<br><br>' +
    '<b>Empfohlene Aktion: </b><br>-<br><br>---<br><b>Message UUID: </b><br>4e743584'
}

describe('extractAiSuggestedReply', () => {
  test('extracts the HTML reply from an AI private note', () => {
    expect(extractAiSuggestedReply(aiNote)).toBe(
      'Hallo Daniel,<br><br>gerne helfen wir weiter.'
    )
  })

  test('supports the KI-Antwort label and strong tags', () => {
    expect(
      extractAiSuggestedReply({
        ...aiNote,
        content:
          '🤖 KI Analyse<br><strong>KI-Antwort:</strong><br>Antwort' +
          '<br><strong>Empfohlene Aktion:</strong><br>-<br><b>Message UUID:</b><br>abc'
      })
    ).toBe('Antwort')
  })

  test('ignores regular private notes', () => {
    expect(
      extractAiSuggestedReply({
        type: 'outgoing',
        private: true,
        content: '<b>Empfohlene Antwort:</b><br>Keine KI-Notiz'
      })
    ).toBeNull()
  })

  test('ignores non-private messages', () => {
    expect(extractAiSuggestedReply({ ...aiNote, private: false })).toBeNull()
  })

  test('ignores malformed AI notes without an action boundary', () => {
    expect(
      extractAiSuggestedReply({
        ...aiNote,
        content: '🤖 KI Analyse<br><b>Empfohlene Antwort:</b><br>Antwort<br>Message UUID: abc'
      })
    ).toBeNull()
  })
})
