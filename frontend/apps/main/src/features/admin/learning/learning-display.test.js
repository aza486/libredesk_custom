import { expect, test } from 'vitest'
import {
  typeLabel,
  categoryLabel,
  statusLabel,
  publicationLabel,
  statusSummary,
  reasonLabel,
  formatLearningDate
} from './learning-display'
test.each([
  ['employee_fact', 'Unternehmensfakt'],
  ['factual_review', 'Faktenprüfung erforderlich'],
  ['response_example', 'Antwortbeispiel'],
  ['classification_example', 'Kategorie-Korrektur'],
  ['style_rule', 'Stilregel'],
  ['manual', 'Manuelles Firmenwissen']
])('type %s is translated without changing the stored code', (code, label) =>
  expect(typeLabel(code)).toBe(label)
)
test.each([
  ['bestellung', 'Bestellung'],
  ['bestellstatus', 'Bestellstatus'],
  ['versandstatus', 'Versandstatus'],
  ['rechnung', 'Rechnung'],
  ['produktfragen', 'Produktfragen'],
  ['firmenfragen', 'Unternehmensfragen'],
  ['reklamation', 'Reklamation'],
  ['spam', 'Spam'],
  ['human_needed', 'Menschliche Prüfung'],
  ['danksagung', 'Dankesnachricht']
])('category %s is translated', (code, label) => expect(categoryLabel(code)).toBe(label))
test.each([
  ['pending', 'Prüfung ausstehend'],
  ['skipped', 'Übersprungen'],
  ['prepared', 'Aufbereitet'],
  ['approved', 'Freigegeben'],
  ['rejected', 'Abgelehnt'],
  ['disabled', 'Deaktiviert']
])('review status %s is translated', (code, label) => expect(statusLabel(code)).toBe(label))
test.each([
  ['unpublished', 'Nicht veröffentlicht'],
  ['publishing', 'Veröffentlichung läuft'],
  ['published', 'Veröffentlicht'],
  ['failed', 'Veröffentlichung fehlgeschlagen']
])('publication %s stays distinct from review', (code, label) =>
  expect(publicationLabel(code)).toBe(label)
)
test('skipped feedback has no invented candidate state, type or category', () => {
  const item = {
    record_type: 'feedback',
    learning_status: 'skipped',
    quality_status: 'pending',
    decision: 'skipped',
    approved: false,
    active: false,
    publish_state: 'unpublished',
    candidate_count: 0,
    content: { original_categories: ['bestellung', 'danksagung', 'bestellung'] }
  }
  expect(statusSummary(item)).toEqual({
    feedback: 'Übersprungen',
    review: 'Noch kein Lernkandidat',
    publication: 'Nicht anwendbar',
    activation: 'Nicht anwendbar'
  })
  expect(statusSummary({ ...item, candidate_count: 1 }).review).toBe('Lernkandidat vorhanden')
})
test('real review decision, publication and activation cannot be conflated', () => {
  const item = {
    record_type: 'candidate',
    decision: 'approved',
    quality_status: 'pending',
    publish_state: 'publishing',
    approved: false,
    active: false
  }
  expect(statusSummary(item)).toMatchObject({
    review: 'Freigegeben',
    quality: 'Prüfung ausstehend',
    publication: 'Veröffentlichung läuft',
    activation: 'Für RAG nicht aktiviert'
  })
  expect(statusSummary({ ...item, review_exists: false }).review).toBe(
    'Noch keine manuelle Prüfung'
  )
  expect(statusSummary({ ...item, active: true, publish_state: 'published' }).activation).toBe(
    'Für RAG aktiviert'
  )
})
test('unknown values are not silently classified or exposed as labels', () => {
  expect(typeLabel('some_other_type')).toBe('Unbekannter Lern-Typ')
  expect(categoryLabel('special_unknown')).toBe('Unbekannte Kategorie')
  expect(categoryLabel('')).toBe('Nicht zugeordnet')
  expect(statusLabel('constructor')).toBe('Unbekannter Status')
  expect(publicationLabel('__proto__')).toBe('Unbekannter Veröffentlichungsstatus')
})
test.each([
  ['2026-10-30T10:20:00Z', '30.10.2026, 11:20 Uhr'],
  ['2026-10-30T11:20:00+01:00', '30.10.2026, 11:20 Uhr'],
  ['2026-07-30T09:20:00Z', '30.07.2026, 11:20 Uhr'],
  ['2026-10-30T10:20:00', '30.10.2026, 11:20 Uhr'],
  ['', 'Datum nicht verfügbar'],
  ['invalid', 'Datum nicht verfügbar']
])('German dates use Europe/Berlin for %s', (value, label) =>
  expect(formatLearningDate(value)).toBe(label)
)
test('skip reasons stay understandable without a preview or manual creation helper', () => {
  expect(reasonLabel('no_complete_grounded_learning_candidate')).toContain('belegter')
  expect(reasonLabel('no_valid_learning_candidate')).toBe('Kein geeigneter Lernkandidat erkannt.')
  expect(reasonLabel('new_unmapped_reason')).not.toContain('new_unmapped_reason')
})

test('system audit messages are translated while actual reviewer text remains unchanged', () => {
  expect(reasonLabel('Human review and 1536-dimensional embedding confirmed')).toBe(
    'Menschliche Prüfung und Suchvektor mit 1536 Dimensionen bestätigt.'
  )
  expect(reasonLabel('Quelle im Firmenhandbuch geprüft.')).toBe('Quelle im Firmenhandbuch geprüft.')
})
