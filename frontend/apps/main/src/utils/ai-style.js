// Dialog-only history; never persisted alongside customer drafts.
export function createStyleHistory(original) {
  return {
    original,
    versions: [original],
    get current() {
      return this.versions.at(-1)
    },
    push(text) {
      this.versions.push(text)
    },
    undo() {
      if (this.versions.length > 1) this.versions.pop()
    },
    reset() {
      this.versions = [this.original]
    }
  }
}
export const styleTextLength = (text) => Array.from(text).length
export function sourceAlreadyAnswered(messages, sourceUUID) {
  return messages.some(
    (m) =>
      !m.private &&
      m.type === 'outgoing' &&
      ['pending', 'sent'].includes(m.status) &&
      (m.meta?.ai_response?.source_message_uuid === sourceUUID ||
        m.meta?.ai_feedback_original_message_uuid === sourceUUID)
  )
}
