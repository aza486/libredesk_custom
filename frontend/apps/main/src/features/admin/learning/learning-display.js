// Presentation only: database codes and the review/publisher contract stay unchanged.
export const typeLabels = {
  employee_fact: 'Unternehmensfakt',
  factual_review: 'Faktenprüfung erforderlich',
  response_example: 'Antwortbeispiel',
  classification_example: 'Kategorie-Korrektur',
  style_rule: 'Stilregel',
  manual: 'Manuelles Firmenwissen'
}
export const categoryLabels = {
  bestellung: 'Bestellung',
  bestellstatus: 'Bestellstatus',
  versandstatus: 'Versandstatus',
  rechnung: 'Rechnung',
  produktfragen: 'Produktfragen',
  firmenfragen: 'Unternehmensfragen',
  reklamation: 'Reklamation',
  spam: 'Spam',
  human_needed: 'Menschliche Prüfung',
  danksagung: 'Dankesnachricht'
}
export const statusLabels = {
  pending: 'Prüfung ausstehend',
  skipped: 'Übersprungen',
  prepared: 'Aufbereitet',
  approved: 'Freigegeben',
  rejected: 'Abgelehnt',
  disabled: 'Deaktiviert',
  processing: 'Wird aufbereitet',
  new: 'Neu',
  failed: 'Verarbeitung fehlgeschlagen',
  blocked: 'Blockiert',
  unverified: 'Nicht geprüft',
  employee_attested: 'Vom Mitarbeiter belegt',
  human_verified: 'Fachlich durch Menschen geprüft',
  not_applicable: 'Nicht anwendbar'
}
export const publicationLabels = {
  unpublished: 'Nicht veröffentlicht',
  publishing: 'Veröffentlichung läuft',
  published: 'Veröffentlicht',
  failed: 'Veröffentlichung fehlgeschlagen',
  blocked: 'Veröffentlichung blockiert',
  new: 'Nicht veröffentlicht'
}
const reasons = {
  'Human review and 1536-dimensional embedding confirmed':
    'Menschliche Prüfung und Suchvektor mit 1536 Dimensionen bestätigt.',
  no_complete_grounded_learning_candidate:
    'Kein vollständiger, belegter und sicher wiederverwendbarer Lernvorschlag.',
  no_valid_learning_candidate: 'Kein geeigneter Lernkandidat erkannt.',
  no_meaningful_difference_or_feedback:
    'Kein verwertbarer Unterschied oder zusätzlicher Lernhinweis.',
  operational_case_specific_learning_blocked:
    'Fallbezogene Aktion oder Zusage: keine allgemeine Lernregel.',
  privacy_review_needed: 'Datenschutzprüfung erforderlich.',
  missing_anonymized_question: 'Keine sicher anonymisierte Kundenfrage vorhanden.',
  unsupported_response_source: 'Die Antwortquelle ist für diesen Lernvorgang nicht geeignet.',
  manual_resolution_not_human_needed:
    'Der Eingang erfüllt die Voraussetzungen für manuelles Lernfeedback nicht.',
  publication_validation_failed:
    'Der Lerninhalt oder das Embedding hat die Veröffentlichungsprüfung nicht bestanden.',
  embedding_or_database_unavailable:
    'Embedding-Dienst oder Datenbank vorübergehend nicht verfügbar.',
  invalid_retrieval_text_length: 'Der Suchtext hat keine zulässige Länge.',
  possibly_identifying_retrieval_text: 'Der Suchtext könnte personenbezogene Angaben enthalten.',
  invalid_style_example: 'Das Antwortbeispiel erfüllt die Stilvorgaben nicht.',
  unanchored_employee_fact: 'Der vorgeschlagene Fakt ist nicht ausreichend durch ein Zitat belegt.',
  possibly_identifying_employee_fact: 'Der Fakt könnte personenbezogene Angaben enthalten.',
  invalid_corrected_categories: 'Die Kategorie-Korrektur ist nicht gültig.',
  unsupported_type: 'Dieser Lern-Typ kann nicht automatisch veröffentlicht werden.',
  case_specific_operation: 'Fallbezogene Aktion',
  individual_customer_commitment: 'Individuelle Zusage',
  binding_time_or_service_promise: 'Verbindliche Zeit- oder Leistungszusage',
  financial_or_after_sales_commitment: 'Finanzielle Zusage oder Zusage zur Nachbearbeitung',
  short_acceptance_of_customer_request: 'Pauschale Zustimmung zu einer konkreten Kundenanfrage'
}
const label = (mapping, value, fallback) =>
  Object.hasOwn(mapping, value) ? mapping[value] : fallback
export const typeLabel = (value) =>
  label(typeLabels, value, value ? 'Unbekannter Lern-Typ' : 'Lern-Typ nicht angegeben')
export const categoryLabel = (value) =>
  label(categoryLabels, value, value ? 'Unbekannte Kategorie' : 'Nicht zugeordnet')
export const statusLabel = (value) =>
  label(statusLabels, value, value ? 'Unbekannter Status' : 'Nicht angegeben')
export const publicationLabel = (value) =>
  label(publicationLabels, value, value ? 'Unbekannter Veröffentlichungsstatus' : 'Nicht angegeben')
export const isFeedback = (item) =>
  item?.record_type === 'feedback' ||
  (item?.record_type !== 'candidate' && item?.decision === 'skipped')
export function statusSummary(item) {
  if (isFeedback(item))
    return {
      feedback: statusLabel(item.learning_status || 'skipped'),
      review:
        item.candidate_count > 0
          ? 'Lernkandidat vorhanden'
          : item.candidate_count === 0
            ? 'Noch kein Lernkandidat'
            : 'Lernkandidatenstatus nicht verfügbar',
      publication: 'Nicht anwendbar',
      activation: 'Nicht anwendbar'
    }
  return {
    review:
      item.review_exists === false ? 'Noch keine manuelle Prüfung' : statusLabel(item.decision),
    quality: statusLabel(item.quality_status),
    publication: publicationLabel(item.publish_state),
    activation:
      item.active === true
        ? 'Für RAG aktiviert'
        : item.active === false
          ? 'Für RAG nicht aktiviert'
          : 'Aktivierungsstatus nicht verfügbar'
  }
}
export function reasonLabel(value) {
  if (!value) return 'Kein zusätzlicher Grund hinterlegt.'
  if (Object.hasOwn(reasons, value)) return reasons[value]
  const [base, details] = value.split(':', 2)
  if (Object.hasOwn(reasons, base))
    return (
      reasons[base] +
      (details
        ? ' ' +
          details
            .split(',')
            .map((code) => label(reasons, code, 'Weiterer technischer Prüfgrund'))
            .join('; ')
        : '')
    )
  if (/^[a-z0-9_,:]+$/.test(value))
    return 'Ein zusätzlicher technischer Prüfgrund liegt vor. Siehe Technische Details.'
  return value // Authorized display data; render only as text.
}
export function formatLearningDate(value) {
  if (typeof value !== 'string' || !value.trim()) return 'Datum nicht verfügbar'
  let iso = value.trim().replace(' ', 'T')
  if (!/(?:Z|[+-]\d{2}:?\d{2})$/i.test(iso)) iso += 'Z' // Existing PostgreSQL UTC timestamps lack an offset.
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return 'Datum nicht verfügbar'
  const parts = new Intl.DateTimeFormat('de-DE', {
    timeZone: 'Europe/Berlin',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23'
  }).formatToParts(date)
  const get = (name) => parts.find((p) => p.type === name)?.value
  return `${get('day')}.${get('month')}.${get('year')}, ${get('hour')}:${get('minute')} Uhr`
}
export const auditActionLabel = (action) =>
  label(
    {
      create: 'Lernkandidat erstellt',
      derive: 'Wissenskandidat abgeleitet',
      edit: 'Inhalt bearbeitet',
      approve: 'Fachlich freigegeben',
      reject: 'Abgelehnt',
      deactivate: 'Deaktiviert',
      retry: 'Veröffentlichung erneut eingeplant',
      publish: 'Veröffentlicht'
    },
    action,
    'Prüfvorgang'
  )
