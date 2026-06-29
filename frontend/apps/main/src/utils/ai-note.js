const AI_NOTE_MARKER = '🤖 KI Analyse'
const MESSAGE_UUID_LABEL = 'Message UUID'

const ANSWER_START = /<(?:b|strong)(?:\s[^>]*)?>\s*(?:Empfohlene Antwort|KI-Antwort):?\s*<\/(?:b|strong)>/i
const ANSWER_END = /<(?:b|strong)(?:\s[^>]*)?>\s*Empfohlene Aktion:?\s*<\/(?:b|strong)>/i
const LEADING_BREAKS = /^(?:\s*<br\s*\/?>\s*)+/i
const TRAILING_BREAKS = /(?:\s*<br\s*\/?>\s*)+$/i

export const extractAiSuggestedReply = (message) => {
  if (!message?.private || message.type !== 'outgoing') return null

  const content = message.content
  if (
    typeof content !== 'string' ||
    !content.includes(AI_NOTE_MARKER) ||
    !content.includes(MESSAGE_UUID_LABEL)
  ) {
    return null
  }

  const start = ANSWER_START.exec(content)
  if (!start) return null

  const afterStart = content.slice(start.index + start[0].length)
  const end = ANSWER_END.exec(afterStart)
  if (!end) return null

  const reply = afterStart
    .slice(0, end.index)
    .replace(LEADING_BREAKS, '')
    .replace(TRAILING_BREAKS, '')
    .trim()

  return reply || null
}
