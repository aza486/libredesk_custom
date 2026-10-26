export const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

// Deliberately no fallback to headings, emojis or UUIDs found in HTML.
export function readAiAnalysisNote(message) {
  const note = message?.meta?.ai_analysis
  if (
    message?.private !== true ||
    message.type !== 'outgoing' ||
    message.meta?.deleted_at ||
    typeof message.uuid !== 'string' ||
    !UUID_PATTERN.test(message.uuid) ||
    note?.kind !== 'n8n_ai_analysis' ||
    note.version !== 1 ||
    typeof note.original_message_uuid !== 'string' ||
    !UUID_PATTERN.test(note.original_message_uuid) ||
    typeof note.suggested_message !== 'string' ||
    !note.suggested_message.trim() ||
    note.suggested_message.length > 65536
  )
    return null
  return {
    noteUUID: message.uuid,
    originalUUID: note.original_message_uuid,
    suggestion: note.suggested_message
  }
}

export function isOriginalCustomerMessage(message, conversationUUID, originalUUID) {
  return (
    message?.uuid === originalUUID &&
    message.conversation_uuid === conversationUUID &&
    message.type === 'incoming' &&
    message.sender_type === 'contact' &&
    !message.private
  )
}

export function suggestionToHTML(text) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
    .replace(/\r?\n/g, '<br>')
}

export const extractAiSuggestedReply = (message) => {
  const note = readAiAnalysisNote(message)
  return note ? suggestionToHTML(note.suggestion) : null
}
export const feedbackReasonBytes = (reason) => new TextEncoder().encode(reason.trim()).length

// Only persisted typed metadata, never HTML, exposes manual annotation actions.
export function readManualReviewNote(message) {
  const note = message?.meta?.ai_analysis
  if (message?.private !== true || message.type !== 'outgoing' || message.sender_type !== 'agent' ||
      Object.hasOwn(message.meta || {}, 'deleted_at') ||
      typeof message.uuid !== 'string' || !UUID_PATTERN.test(message.uuid) || note?.kind !== 'n8n_manual_review' || note.version !== 1 ||
      typeof note.original_message_uuid !== 'string' || !UUID_PATTERN.test(note.original_message_uuid) ||
      (note.suggested_message !== undefined && note.suggested_message !== '')) return null
  return { noteUUID: message.uuid, originalUUID: note.original_message_uuid }
}
export const manualReasonLength = (reason) => [...reason.trim().replace(/\r\n?/g, '\n')].length
