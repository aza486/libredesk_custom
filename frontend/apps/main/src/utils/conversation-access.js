// Ticket read denials are observed centrally, including component-owned requests.
const listeners = new Set()
export function onConversationAccessDenied(listener) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}
export function reportConversationAccessDenied(error) {
  if (![403, 404].includes(error?.response?.status)) return
  const path = error.config?.url || ''
  const match = path.match(/^\/api\/v1\/conversations\/([0-9a-f-]{36})(?:\/(.*))?$/i)
  if (!match) return
  for (const listener of listeners) listener(match[1], !match[2])
}
