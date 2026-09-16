export const CONVERSATION_LIST_TYPE = {
  ASSIGNED: 'assigned',
  UNASSIGNED: 'unassigned',
  TEAM_UNASSIGNED: 'team_unassigned',
  VIEW: 'view',
  ALL: 'all',
  MENTIONED: 'mentioned',
  VISIBLE: 'visible',
  VISIBLE_INTERNAL: 'visible-internal',
  CUSTOMER: 'customer',
  CUSTOMER_HIGH: 'customer-high',
  CREATED: 'created'
}

export const CONVERSATION_DEFAULT_STATUSES = {
  OPEN: 'Open',
  SNOOZED: 'Snoozed',
  RESOLVED: 'Resolved',
  CLOSED: 'Closed'
}

export const CONVERSATION_DEFAULT_STATUSES_LIST = Object.values(CONVERSATION_DEFAULT_STATUSES)

export const MACRO_CONTEXT = {
  REPLY: 'reply',
  NEW_CONVERSATION: 'new-conversation'
}

export const TAG_ACTION = {
  ADD: 'add_tags',
  SET: 'set_tags',
  REMOVE: 'remove_tags'
}
