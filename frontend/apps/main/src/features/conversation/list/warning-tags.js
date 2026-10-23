// Conversation list responses contain tag names (including their emoji), not
// stable tag IDs. Match the complete normalized name; keep the original label.
export function warningTagKind(tag) {
  if (typeof tag !== 'string') return null
  const name = tag
    .replace(/^[^\p{L}\p{N}]+/u, '')
    .replace(/\s+/g, '')
    .toLowerCase()
  if (name === 'verarbeitungsfehler') return 'error'
  if (name === 'menscherforderlich') return 'human'
  return null
}

export function prioritizeConversationTags(tags) {
  const errors = [],
    humans = [],
    others = []
  for (const tag of tags) {
    const kind = warningTagKind(tag)
    if (kind === 'error') errors.push(tag)
    else if (kind === 'human') humans.push(tag)
    else others.push(tag)
  }
  return [...errors, ...humans, ...others]
}
