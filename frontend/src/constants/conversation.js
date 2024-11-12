export const CONVERSATION_LIST_TYPE = {
  ASSIGNED: 'assigned',
  UNASSIGNED: 'unassigned',
  ALL: 'all'
}

export const CONVERSATION_WS_ACTIONS = {
  SUB_LIST: 'conversations_list_sub',
  SET_CURRENT: 'conversation_set_current',
  UNSET_CURRENT: 'conversation_unset_current'
}

export const CONVERSATION_DEFAULT_STATUSES = [
  'Open',
  'Pending',
  'Resolved',
  'Closed'
]