// Exact names, not categories: Closed and Resolved both have category "resolved".
export const DEFAULT_STATUS_KEY = Object.freeze({
  Open: 'globals.terms.open',
  Snoozed: 'globals.terms.snoozed',
  Resolved: 'globals.terms.resolved',
  Closed: 'globals.terms.closed'
})

export function conversationStatusLabel(name, t) {
  if (!Object.hasOwn(DEFAULT_STATUS_KEY, name) || typeof t !== 'function') return name
  const key = DEFAULT_STATUS_KEY[name]
  const label = t(key)
  return typeof label === 'string' && label && label !== key ? label : name
}
