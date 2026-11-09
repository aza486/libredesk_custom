// Database names use title case; existing route filters use lower case.
// Match only known spellings, preserving custom names and original values.
const PRIORITY_KEYS = Object.freeze({
  Low: 'globals.terms.low', low: 'globals.terms.low',
  Medium: 'globals.terms.medium', medium: 'globals.terms.medium',
  High: 'globals.terms.high', high: 'globals.terms.high'
})

export function conversationPriorityLabel(name, t) {
  if (!Object.hasOwn(PRIORITY_KEYS, name) || typeof t !== 'function') return name
  const key = PRIORITY_KEYS[name]
  const label = t(key)
  return typeof label === 'string' && label && label !== key ? label : name
}
