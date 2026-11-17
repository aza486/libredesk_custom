// Append without replacing live manual edits. Reject a whole transcript exceeding
// the limit rather than silently dropping words or splitting Unicode characters.
export function appendTranscript(current, transcript, maxChars = Infinity, maxBytes = Infinity) {
  const text = String(transcript).trim()
  const next = current + (current && !/\s$/.test(current) ? ' ' : '') + text
  if ([...next].length > maxChars || new TextEncoder().encode(next).length > maxBytes) return null
  return next
}
