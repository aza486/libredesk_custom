export const CONVERSATION_LIST_TYPE = {
  ASSIGNED: 'assigned',
  UNASSIGNED: 'unassigned',
  TEAM_UNASSIGNED: 'team_unassigned',
  VIEW: 'view',
  ALL: 'all'
}

export const CONVERSATION_VIEWS_INBOXES = {
  'assigned': 'My inbox',
  'unassigned': 'Unassigned',
  'all': 'All',
}

export const CONVERSATION_WS_ACTIONS = {
  SUB_LIST: 'conversations_list_sub',
  SET_CURRENT: 'conversation_set_current',
  UNSET_CURRENT: 'conversation_unset_current'
}

export const CONVERSATION_DEFAULT_STATUSES = {
  OPEN: 'Open',
  PENDING: 'Pending',
  RESOLVED: 'Resolved',
  CLOSED: 'Closed',
  SNOOZED: 'Snoozed',
}

export const CONVERSATION_DEFAULT_STATUSES_LIST = Object.values(CONVERSATION_DEFAULT_STATUSES);