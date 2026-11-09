import { conversationStatusLabel } from './conversation-status'
import { conversationPriorityLabel } from './conversation-priority'

// These are the exact formats produced by getMessageActivityContent in
// internal/conversation/message.go. There are no independent actor/value fields
// in the API yet, so ambiguous text must remain untouched.
const EVENTS = Object.freeze({
  assigned_user_change: { prefix: 'Assigned to ', separator: ' by ', key: 'conversation.activity.assignedUser', fields: ['value', 'actor'] },
  assigned_team_change: { prefix: 'Assigned to ', separator: ' team by ', key: 'conversation.activity.assignedTeam', fields: ['value', 'actor'] },
  self_assign: { suffix: ' self-assigned this conversation', key: 'conversation.activity.selfAssigned', fields: ['actor'] },
  priority_change: { separator: ' set priority to ', key: 'conversation.activity.priorityChanged', fields: ['actor', 'value'] },
  status_change: { separator: ' marked the conversation as ', key: 'conversation.activity.statusChanged', fields: ['actor', 'value'] },
  tag_added: { separator: ' added tag ', key: 'conversation.activity.tagAdded', fields: ['actor', 'value'] },
  tag_removed: { separator: ' removed tag ', key: 'conversation.activity.tagRemoved', fields: ['actor', 'value'] },
  sla_set: { separator: ' set ', suffix: ' SLA policy', key: 'conversation.activity.slaSet', fields: ['actor', 'value'] },
  participant_added: { suffix: ' joined the conversation', key: 'conversation.activity.participantJoined', fields: ['actor'] }
})

function parseEvent(content, event) {
  let text = content
  if (event.prefix) {
    if (!text.startsWith(event.prefix)) return null
    text = text.slice(event.prefix.length)
  }
  if (event.suffix) {
    if (!text.endsWith(event.suffix)) return null
    text = text.slice(0, -event.suffix.length)
  }
  let values = [text]
  if (event.separator) {
    const index = text.indexOf(event.separator)
    // Reject repeated separators, including overlapping ones. No greedy or
    // speculative split of an employee name, team, tag or policy name.
    if (index < 0 || text.indexOf(event.separator, index + 1) !== -1) return null
    values = [text.slice(0, index), text.slice(index + event.separator.length)]
  }
  if (values.some((value) => !value || value.trim() !== value)) return null
  return Object.fromEntries(event.fields.map((field, index) => [field, values[index]]))
}

export function conversationActivityLabel(message, t) {
  const original = message?.content
  // type=activity is the backend's dedicated system-message discriminator.
  // Real API messages additionally carry private/content_type; WS previews omit
  // these fields. Conflicting metadata always fails closed.
  if (message?.type !== 'activity' || message.sender_type !== 'agent' ||
      (message.private !== undefined && message.private !== true) ||
      (message.content_type !== undefined && message.content_type !== 'text') ||
      typeof original !== 'string' || !original || original.length > 8192 ||
      // eslint-disable-next-line no-control-regex -- Intentionally reject control characters in system-event text.
      /[\u0000-\u001f\u007f\u2028\u2029]/.test(original) || typeof t !== 'function') return original

  const meta = message.meta
  if (meta != null && (typeof meta !== 'object' || Array.isArray(meta))) return original
  let matches
  if (meta && Object.hasOwn(meta, 'activity_type')) {
    const type = meta.activity_type
    if (typeof type !== 'string' || !Object.hasOwn(EVENTS, type)) return original
    const event = EVENTS[type]
    const params = parseEvent(original, event)
    matches = params ? [{ event, params, type }] : []
  } else {
    // Old messages and WS previews may have no activity_type. Require exactly
    // one possible format. In particular, "Assigned to X team by Y" also fits
    // the user-assignment format, so it stays original without a trusted type.
    matches = Object.entries(EVENTS).flatMap(([type, event]) => {
      const params = parseEvent(original, event)
      return params ? [{ event, params, type }] : []
    })
  }
  if (matches.length !== 1) return original
  const { event, params, type } = matches[0]
  if (type === 'status_change') params.value = conversationStatusLabel(params.value, t)
  if (type === 'priority_change') params.value = conversationPriorityLabel(params.value, t)
  const translated = t(event.key, params)
  // A missing catalog entry must never replace a useful historical message
  // with a visible translation key.
  return typeof translated === 'string' && translated && translated !== event.key ? translated : original
}
