import { defineStore } from 'pinia'
import { computed, reactive, ref, watch, watchEffect, onScopeDispose } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { TYPING_RECEIVE_TIMEOUT } from '@shared-ui/composables/useTypingIndicator.js'
import { deepMerge } from '@shared-ui/utils/object.js'
import { computeRecipientsFromMessage } from '../utils/email-recipients'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { subscribeToConversation, sendTypingIndicator, subscribeListReplace } from '@main/websocket'
import { playNotificationSound } from '@shared-ui/composables/useNotificationSound'
import MessageCache from '../utils/conversation-message-cache'
import { getI18n } from '../i18n'
import { conversationStatusLabel } from '../utils/conversation-status'
import { conversationPriorityLabel } from '../utils/conversation-priority'
import {
  CONVERSATION_LIST_TYPE,
  CONVERSATION_DEFAULT_STATUSES,
  TAG_ACTION
} from '@/constants/conversation'
import { useThrottleFn } from '@vueuse/core'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { delayedLoading } from '@/utils/delayed-loading'
import api from '../api'
import { onConversationAccessDenied } from '../utils/conversation-access'

export const useConversationStore = defineStore('conversation', () => {
  const CONV_LIST_PAGE_SIZE = 25
  const MESSAGE_LIST_PAGE_SIZE = 30
  const priorities = ref([])
  const statuses = ref([])
  const currentTo = ref([])
  const currentBCC = ref([])
  const currentCC = ref([])
  const macros = ref({})
  const drafts = ref(new Map())
  // In-memory, resets on reload.
  const selectedDraftType = ref(new Map())
  let resolveDraftsReady
  const draftsReady = new Promise((resolve) => {
    resolveDraftsReady = resolve
  })
  const userStore = useUserStore()
  const notificationStore = useNotificationStore()
  const router = useRouter()
  const isViewingConversation = (uuid) => router.currentRoute.value.params.uuid === uuid

  const selectedUUIDs = ref(new Set())

  const priorityOptions = computed(() => {
    return priorities.value.map((p) => ({
      label: p.name,
      value: p.id,
      displayLabel: conversationPriorityLabel(p.name, getI18n()?.global?.t)
    }))
  })
  const statusOptions = computed(() => {
    return statuses.value.map((s) => ({
      label: s.name,
      value: s.id,
      displayLabel: conversationStatusLabel(s.name, getI18n()?.global?.t)
    }))
  })
  const statusOptionsNoSnooze = computed(() =>
    statuses.value
      .filter((s) => s.name !== CONVERSATION_DEFAULT_STATUSES.SNOOZED)
      .map((s) => ({
        label: s.name,
        value: s.id,
        displayLabel: conversationStatusLabel(s.name, getI18n()?.global?.t)
      }))
  )

  let lastClickedUUID = null

  const selectedCount = computed(() => selectedUUIDs.value.size)
  const allSelected = computed(() => {
    const list = conversationsList.value
    return list.length > 0 && selectedUUIDs.value.size === list.length
  })

  function toggleSelect(uuid, shiftKey = false) {
    const next = new Set(selectedUUIDs.value)

    if (shiftKey && lastClickedUUID && lastClickedUUID !== uuid) {
      const list = conversationsList.value
      const lastIdx = list.findIndex((c) => c.uuid === lastClickedUUID)
      const curIdx = list.findIndex((c) => c.uuid === uuid)
      if (lastIdx !== -1 && curIdx !== -1) {
        const start = Math.min(lastIdx, curIdx)
        const end = Math.max(lastIdx, curIdx)
        for (let i = start; i <= end; i++) {
          next.add(list[i].uuid)
        }
      }
    } else {
      if (next.has(uuid)) next.delete(uuid)
      else next.add(uuid)
    }

    lastClickedUUID = uuid
    selectedUUIDs.value = next
  }

  function selectAll() {
    selectedUUIDs.value = new Set(conversationsList.value.map((c) => c.uuid))
  }

  function clearSelection() {
    selectedUUIDs.value = new Set()
    lastClickedUUID = null
  }

  function isSelected(uuid) {
    return selectedUUIDs.value.has(uuid)
  }

  // TODO: Move to constants.
  const sortFieldMap = {
    newest_customer_reply: {
      model: 'customer_reply', field: 'last_customer_reply_at', order: 'desc'
    },
    oldest_customer_reply: {
      model: 'customer_reply', field: 'last_customer_reply_at', order: 'asc'
    },
    oldest: {
      model: 'conversations',
      field: 'last_message_at',
      order: 'asc'
    },
    newest: {
      model: 'conversations',
      field: 'last_message_at',
      order: 'desc'
    },
    started_first: {
      model: 'conversations',
      field: 'created_at',
      order: 'asc'
    },
    started_last: {
      model: 'conversations',
      field: 'created_at',
      order: 'desc'
    },
    waiting_longest: {
      model: 'conversations',
      field: 'waiting_since',
      order: 'asc'
    },
    next_sla_target: {
      model: 'conversations',
      field: 'next_sla_deadline_at',
      order: 'asc'
    },
    priority_first: {
      model: 'conversations',
      field: 'priority_id',
      order: 'desc'
    }
  }

  const sortFieldI18nKeys = {
    newest_customer_reply: 'conversation.sort.newestCustomerReply',
    oldest_customer_reply: 'conversation.sort.oldestCustomerReply',
    oldest: 'conversation.sort.oldestActivity',
    newest: 'conversation.sort.newestActivity',
    started_first: 'conversation.sort.startedFirst',
    started_last: 'conversation.sort.startedLast',
    waiting_longest: 'conversation.sort.waitingLongest',
    next_sla_target: 'conversation.sort.nextSLATarget',
    priority_first: 'conversation.sort.priorityFirst'
  }

  let typingTimeout = null
  const typingByUUID = reactive({})
  const typingTimeoutsByUUID = new Map()

  const conversations = reactive({
    data: [],
    listType: null,
    status: 'Open',
    sortField: 'newest',
    listFilters: [],
    viewID: 0,
    teamID: 0,
    inboxID: 0,
    loading: false,
    fetching: false,
    initialized: false,
    page: 1,
    hasMore: false,
    total: 0,
    errorMessage: ''
  })

  // Session-local list snapshots. Detail navigation never changes this identity.
  const listKey = ref('')
  const listRevision = ref(0)
  const listPositions = new Map()
  const listCache = new Map()
  let listRequestSeq = 0
  let activeListRequest
  let bypassListCache = false
  let listEventPatches = new Map()
  let listEventInserts = new Set()

  function retainListEvent(uuid, payload) {
    if (!conversations.fetching) return
    const patch = listEventPatches.get(uuid) || {}
    deepMerge(patch, payload)
    listEventPatches.set(uuid, patch)
  }

  const conversation = reactive({
    data: null,
    loading: false,
    errorMessage: '',
    isTyping: false
  })

  const messages = reactive({
    data: new MessageCache(),
    loading: false,
    fetching: false,
    page: 1,
    // To trigger reactivity on the messages cache, simpler than making MessageCache reactive.
    version: 0
  })

  // Convos whose message cache is stale; drained lazily by fetchMessages on next open.
  let staleConversationUUIDs = new Set()
  const CONVERSATION_CACHE_MAX = 50
  const conversationDataCache = new Map()

  function cacheConversationData(uuid, data) {
    if (!uuid || !data) return
    if (conversationDataCache.has(uuid)) conversationDataCache.delete(uuid)
    conversationDataCache.set(uuid, data)
    while (conversationDataCache.size > CONVERSATION_CACHE_MAX) {
      const oldest = conversationDataCache.keys().next().value
      if (oldest === conversation.data?.uuid) break
      conversationDataCache.delete(oldest)
    }
  }
  // Bumped on resetConversations() so in-flight requests can drop stale responses.
  let contextSeq = 0
  const emitter = useEmitter()

  const incrementMessageVersion = () => setTimeout(() => messages.version++, 0)

  function setListStatus(status, fetch = true) {
    conversations.status = status
    if (fetch) {
      resetConversations()
      reFetchConversationsList()
    }
  }

  const getListStatus = computed(() => {
    return conversations.status
  })

  function setListSortField(field) {
    if (conversations.sortField === field) return
    conversations.sortField = field
    resetConversations()
    reFetchConversationsList()
  }

  const getListSortField = computed(() => {
    const i18n = getI18n()
    const t = i18n?.global?.t || ((key) => key.split('.').pop())
    return t(sortFieldI18nKeys[conversations.sortField])
  })

  async function fetchStatuses() {
    if (statuses.value.length > 0) return
    try {
      const response = await api.getStatuses()
      statuses.value = response.data.data.map((status) => ({
        ...status,
        id: status.id.toString()
      }))
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function fetchPriorities() {
    if (priorities.value.length > 0) return
    try {
      const response = await api.getPriorities()
      priorities.value = response.data.data.map((priority) => ({
        ...priority,
        id: priority.id.toString()
      }))
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  function isAssignedToMe(conv) {
    const me = Number(userStore.userID)
    const ids = [...(conv.assigned_user_ids || []), conv.assigned_user_id]
    return ids.some((id) => Number(id) === me)
  }

  function isPersonalInboxOwner(conv, userID = userStore.userID) {
    const ownerIDs = conv.inbox_owner_user_ids?.length
      ? conv.inbox_owner_user_ids
      : [conv.inbox_owner_user_id]
    return ownerIDs.some((id) => Number(id) === Number(userID))
  }

  function isSpamConversation(conv) {
    return (Array.isArray(conv?.tags) ? conv.tags : []).some((tag) =>
      typeof tag === 'object' && tag !== null
        ? Number(tag.id) === 12 || tag.name === '🗑Spam'
        : String(tag) === '12' || tag === '🗑Spam'
    )
  }

  function hasSpamViewFilter() {
    const filters = conversations.listFilters
    if (!Array.isArray(filters) || filters.length !== 1) return false
    const [filter] = filters
    if (filter.model !== 'conversations' || filter.field !== 'tags' || filter.operator !== 'contains') return false
    try {
      const values = JSON.parse(filter.value)
      return Array.isArray(values) && values.length === 1 && values[0] === 12
    } catch {
      return false
    }
  }

  function matchesAssignmentScope(conv) {
    const spamView = !conversations.listType?.startsWith('folder:') && hasSpamViewFilter()
    if (isSpamConversation(conv) !== spamView) return false
    // Folder responses already use the full authorized server-side base list.
    // In particular, an explicitly shared personal ticket can belong to assigned
    // even when this employee is not an owner of that mailbox.
    if (conversations.listType?.startsWith('folder:')) return null
    if ([CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED, CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED_UNFILED].includes(conversations.listType)) {
      return conv.customer_redistributed_active === true
    }
    if (!spamView && [CONVERSATION_LIST_TYPE.CUSTOMER, CONVERSATION_LIST_TYPE.CUSTOMER_HIGH].includes(conversations.listType) && conv.customer_redistributed_active === true) return false


    if (conv.custom_attributes?.access_mode === 'personal') {
      if (
        [
          CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED,
        CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED_UNFILED,
        CONVERSATION_LIST_TYPE.CUSTOMER,
          CONVERSATION_LIST_TYPE.CUSTOMER_HIGH,
          CONVERSATION_LIST_TYPE.SERVICE_MAILS,
          CONVERSATION_LIST_TYPE.UNASSIGNED,
          CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
          CONVERSATION_LIST_TYPE.CREATED
        ].includes(conversations.listType)
      )
        return false
      if (!userStore.roles.includes('Admin') && !isPersonalInboxOwner(conv)) return false
    }
    switch (conversations.listType) {
      case CONVERSATION_LIST_TYPE.PERSONAL:
      case CONVERSATION_LIST_TYPE.PERSONAL_HIGH:
        if (
          !(
            conv.inbox_access_mode === 'personal' &&
            isPersonalInboxOwner(conv) &&
            Number(conv.inbox_id) === Number(conversations.inboxID)
          )
        ) {
          return false
        }
        if (hasSpamViewFilter()) {
          return isSpamConversation(conv) &&
            (conversations.listType !== CONVERSATION_LIST_TYPE.PERSONAL_HIGH || conv.priority === 'High')
        }
        return (
          !isSpamConversation(conv) &&
          (conversations.listType !== CONVERSATION_LIST_TYPE.PERSONAL_HIGH || conv.priority === 'High')
        )
      case CONVERSATION_LIST_TYPE.ASSIGNED:
      case CONVERSATION_LIST_TYPE.ASSIGNED_HIGH:
        return (
          isAssignedToMe(conv) &&
          !(
            conv.inbox_access_mode === 'personal' &&
            isPersonalInboxOwner(conv)
          ) &&
          (conversations.listType !== CONVERSATION_LIST_TYPE.ASSIGNED_HIGH ||
            conv.priority === 'High')
        )
      case CONVERSATION_LIST_TYPE.UNASSIGNED:
        return !conv.assigned_user_id && !conv.assigned_team_id
      case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
        return (
          Number(conv.assigned_team_id) === Number(conversations.teamID) && !conv.assigned_user_id
        )
      default:
        return null
    }
  }

  function belongsToList(conv) {
    const matched = matchesAssignmentScope(conv)
    return matched === null ? true : matched
  }

  const conversationsList = computed(() => {
    if (!conversations.data) return []
    let filteredConversations = conversations.data
    if (conversations.status !== '') {
      filteredConversations = filteredConversations.filter(
        (conv) => conv.status === conversations.status
      )
    }
    filteredConversations = filteredConversations.filter(belongsToList)

    return [...filteredConversations].sort((a, b) => {
      const field = sortFieldMap[conversations.sortField]?.field
      const customerReplySort = field === 'last_customer_reply_at'
      const direction = sortFieldMap[conversations.sortField]?.order === 'asc' ? 1 : -1
      const tieBreak = () => customerReplySort ? direction * (a.id - b.id) : 0
      if (!a[field] && !b[field]) return tieBreak()
      if (!a[field]) return 1 // null goes last
      if (!b[field]) return -1
      const order = sortFieldMap[conversations.sortField]?.order
      const difference = order === 'asc'
        ? new Date(a[field]) - new Date(b[field])
        : new Date(b[field]) - new Date(a[field])
      return difference || tieBreak()
    })
  })

  const currentConversationHasMoreMessages = computed(() => {
    return messages.data.hasMore(conversation.data?.uuid)
  })

  const conversationMessages = computed(() => {
    return messages.data.getAllPagesMessages(conversation.data?.uuid)
  })

  function markConversationAsRead(uuid) {
    const index = conversations.data.findIndex((conv) => conv.uuid === uuid)
    if (index !== -1) {
      setTimeout(() => {
        if (conversations.data?.[index]) {
          conversations.data[index].unread_message_count = 0
        }
      }, 3000)
    }
  }

  async function markAsUnread(uuid) {
    try {
      await api.markConversationAsUnread(uuid)
      const index = conversations.data.findIndex((conv) => conv.uuid === uuid)
      if (index !== -1) {
        conversations.data[index].unread_message_count = 1
      }
    } catch (err) {
      handleHTTPError(err)
    }
  }

  function incrementUnread(uuid) {
    const row = conversations.data.find((c) => c.uuid === uuid)
    if (!row) return
    row.unread_message_count = Math.min((row.unread_message_count || 0) + 1, 10)
  }

  const currentContactName = computed(() => {
    if (!conversation.data?.contact) return ''
    return conversation.data?.contact.first_name + ' ' + conversation.data?.contact.last_name
  })

  function getContactFullName(uuid) {
    if (conversations?.data) {
      const conv = conversations.data.find((conv) => conv.uuid === uuid)
      return conv ? `${conv.contact.first_name} ${conv.contact.last_name}` : ''
    }
  }

  const current = computed(() => {
    return conversation.data || {}
  })

  const isConversationOpen = computed(() => {
    return Object.keys(conversation.data || {}).length > 0
  })

  watchEffect(async () => {
    const _ = messages.version // eslint-disable-line no-unused-vars
    const conv = conversation.data
    const msgData = messages.data
    const inboxEmail = conv?.inbox_mail

    if (conv?.inbox_channel === 'livechat') {
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    if (!conv || !msgData || !inboxEmail) return

    // Skip automated messages (auto-replies, CSAT) so the prefill reflects the last human-driven recipients.
    const latestMessage = msgData.getLatestMessage(conv.uuid, ['incoming', 'outgoing'], true, true)
    if (!latestMessage) {
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    const { to, cc, bcc } = computeRecipientsFromMessage(
      latestMessage,
      conv.contact?.email || '',
      inboxEmail,
      conv?.inbox_reply_to || ''
    )
    currentTo.value = to
    currentCC.value = cc
    currentBCC.value = bcc
  })

  function resetTypingState() {
    conversation.isTyping = false
    if (typingTimeout) {
      clearTimeout(typingTimeout)
      typingTimeout = null
    }
  }

  const accessVersions = new Map()
  const revokedUUIDs = new Set()
  let accessSequence = 0
  let detailRequest = 0
  const accessVersion = (uuid) => accessVersions.get(uuid) || 0
  const deniedRead = (error) => [403, 404].includes(error?.response?.status)

  function revokeConversationAccess(uuid) {
    if (!uuid) return
    const alreadyRevoked = revokedUUIDs.has(uuid)
    revokedUUIDs.add(uuid)
    accessVersions.set(uuid, ++accessSequence)
    conversationDataCache.delete(uuid)
    messages.data.purgeConversation(uuid)
    staleConversationUUIDs.delete(uuid)
    selectedUUIDs.value.delete(uuid)
    pendingNotificationUUIDs.delete(uuid)
    selectedDraftType.value.delete(uuid)
    notificationStore.removeConversationNotifications?.(uuid)
    for (const key of drafts.value.keys()) {
      if (key.startsWith(`${uuid}::`)) drafts.value.delete(key)
    }
    // Drop only this row, retaining loaded pages and stored scroll positions.
    const prune = (entry) => {
      const before = entry.data?.length || 0
      entry.data = entry.data?.filter((row) => row.uuid !== uuid) || []
      if (entry.data.length < before) entry.total = Math.max(0, entry.total - 1)
    }
    prune(conversations)
    for (const entry of listCache.values()) prune(entry)
    listEventPatches.delete(uuid)
    listEventInserts.delete(uuid)
    delete typingByUUID[uuid]
    clearTimeout(typingTimeoutsByUUID.get(uuid))
    typingTimeoutsByUUID.delete(uuid)
    if (conversation.data?.uuid === uuid || isViewingConversation(uuid)) {
      conversation.data = null
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      resetTypingState()
      conversation.loading = false
      messages.loading = false
      messages.fetching = false
      conversation.errorMessage =
        getI18n()?.global?.t('conversation.accessRevoked') ||
        'Du hast keinen Zugriff mehr auf diese Konversation.'
      if (!alreadyRevoked) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive', description: conversation.errorMessage
        })
      }
      const route = router.currentRoute.value
      if (route.params.uuid === uuid && route.path) {
        const parentPath = route.path.replace(/\/conversation\/[^/]+\/?$/, '')
        router.replace({ path: parentPath, query: route.query })
      }
    }
    incrementMessageVersion()
  }

  const stopAccessListener = onConversationAccessDenied((uuid, directDetail) => {
    if (directDetail) revokeConversationAccess(uuid)
    else silentRefetchConversation(uuid)
  })
  onScopeDispose(stopAccessListener)

  async function fetchConversation(uuid) {
    const request = ++detailRequest
    const version = accessVersion(uuid)
    conversation.errorMessage = ''
    const cached = revokedUUIDs.has(uuid) ? null : conversationDataCache.get(uuid)
    if (cached) conversation.data = cached
    const guard = delayedLoading(conversation, 'loading')
    try {
      const resp = await api.getConversation(uuid)
      if (version !== accessVersion(uuid) || request !== detailRequest) return
      const restored = revokedUUIDs.delete(uuid)
      notificationStore.restoreConversationNotifications?.(uuid)
      conversation.data = resp.data.data
      resetTypingState()
      subscribeToConversation(uuid)
      cacheConversationData(uuid, conversation.data)
      if (restored) await fetchMessages(uuid)
    } catch (error) {
      if (version !== accessVersion(uuid)) return
      if (deniedRead(error)) revokeConversationAccess(uuid)
      else {
        conversation.errorMessage = handleHTTPError(error).message
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive', description: conversation.errorMessage
        })
      }
    } finally {
      guard.release()
    }
  }

  async function silentRefetchConversation(uuid) {
    const version = accessVersion(uuid)
    try {
      const resp = await api.getConversation(uuid)
      if (version !== accessVersion(uuid)) return
      const wasRevoked = revokedUUIDs.delete(uuid)
      notificationStore.restoreConversationNotifications?.(uuid)
      if (conversation.data?.uuid === uuid) {
        deepMerge(conversation.data, resp.data.data)
        cacheConversationData(uuid, conversation.data)
      }
      if (wasRevoked) refreshConversationListForRealtimeEvent()
    } catch (error) {
      if (version !== accessVersion(uuid)) return
      if (deniedRead(error)) revokeConversationAccess(uuid)
      else console.warn('silent conversation refetch failed', error)
    }
  }

  async function refreshCurrentConversation(uuid) {
    await Promise.all([silentRefetchConversation(uuid), fetchMessages(uuid)])
  }

  async function fetchMessages(uuid, fetchNextPage = false) {
    if (revokedUUIDs.has(uuid)) return
    const version = accessVersion(uuid)
    if (staleConversationUUIDs.has(uuid) && messages.data.hasConversation(uuid)) {
      try {
        const response = await api.getConversationMessages(uuid, {
          page: 1,
          page_size: MESSAGE_LIST_PAGE_SIZE
        })
        if (version !== accessVersion(uuid)) return
        const newMessages = response.data?.data?.results || []
        let lastAdded = null
        for (const m of newMessages) {
          if (!messages.data.hasMessage(uuid, m.uuid)) {
            messages.data.addMessage(uuid, m)
            lastAdded = m
          }
        }
        staleConversationUUIDs.delete(uuid)
        if (lastAdded) {
          incrementMessageVersion()
          setTimeout(() => {
            if (version !== accessVersion(uuid)) return
            emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
              conversation_uuid: uuid,
              message: lastAdded
            })
          }, 100)
        }
      } catch (error) {
        if (version !== accessVersion(uuid)) return
        if (deniedRead(error)) {
          revokeConversationAccess(uuid)
          return
        }
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }

    if (!fetchNextPage && messages.data.getAllPagesMessages(uuid).length > 0) {
      markConversationAsRead(uuid)
      return
    }

    const guard = fetchNextPage ? null : delayedLoading(messages, 'loading')
    messages.fetching = true
    const page = messages.data.getLastFetchedPage(uuid) + 1
    try {
      const response = await api.getConversationMessages(uuid, {
        page,
        page_size: MESSAGE_LIST_PAGE_SIZE
      })
      if (version !== accessVersion(uuid)) return
      const result = response.data?.data || {}
      markConversationAsRead(uuid)
      messages.data.addMessages(uuid, result.results || [], result.page, result.total_pages)
      incrementMessageVersion()
    } catch (error) {
      if (version !== accessVersion(uuid)) return
      if (deniedRead(error)) {
        revokeConversationAccess(uuid)
        return
      }
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      if (guard) guard.release()
      messages.fetching = false
    }
  }

  async function fetchNextMessages() {
    return fetchMessages(conversation.data.uuid, true)
  }

  async function fetchMessage(conversationUUID, messageUUID) {
    if (revokedUUIDs.has(conversationUUID)) return
    const version = accessVersion(conversationUUID)
    try {
      const response = await api.getConversationMessage(conversationUUID, messageUUID)
      if (version !== accessVersion(conversationUUID)) return
      if (response?.data?.data) {
        const newMsg = response.data.data
        messages.data.addMessage(conversationUUID, newMsg)
        incrementMessageVersion()
        return newMsg
      }
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function deleteMessage(conversationUUID, messageUUID) {
    try {
      const resp = await api.deleteMessage(conversationUUID, messageUUID)
      const deletedText = resp.data.data.content
      const existing = messages.data
        .getAllPagesMessages(conversationUUID)
        .find((m) => m.uuid === messageUUID)
      messages.data.updateMessage(conversationUUID, messageUUID, {
        content: deletedText,
        text_content: deletedText,
        meta: { ...(existing?.meta || {}), deleted_at: new Date().toISOString() }
      })
      incrementMessageVersion()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  function fetchNextConversations() {
    if (conversations.fetching || !conversations.hasMore) return
    return fetchConversationsList(
      false,
      conversations.listType,
      conversations.teamID,
      conversations.listFilters,
      conversations.viewID,
      conversations.page + 1,
      conversations.inboxID
    )
  }

  function reFetchConversationsList(showLoader = true) {
    fetchConversationsList(
      showLoader,
      conversations.listType,
      conversations.teamID,
      conversations.listFilters,
      conversations.viewID,
      conversations.page,
      conversations.inboxID
    )
  }

  async function fetchFirstPageConversations() {
    const seq = contextSeq
    if (conversations.fetching && activeListRequest) await activeListRequest
    if (seq !== contextSeq) return
    return fetchConversationsList(
      false,
      conversations.listType,
      conversations.teamID,
      conversations.listFilters,
      conversations.viewID,
      1,
      conversations.inboxID,
      true
    )
  }

  function fetchConversationsList(...args) {
    activeListRequest = loadConversationsList(...args)
    return activeListRequest
  }

  async function loadConversationsList(
    showLoader = true,
    listType = null,
    teamID = 0,
    filters = [],
    viewID = 0,
    page = 0,
    inboxID = 0,
    refresh = false
  ) {
    if (!listType) return
    filters = [...filters]
    if (listType.startsWith('folder:')) conversations.status = ''
    if (conversations.status && !listType.startsWith('folder:')) {
      filters = filters.filter((f) => f.model !== 'conversation_statuses')
      filters.push({
        model: 'conversation_statuses',
        field: 'name',
        operator: 'equals',
        value: conversations.status
      })
    }
    const key = JSON.stringify([
      userStore.userID, listType, String(teamID), String(inboxID), String(viewID),
      conversations.status, conversations.sortField, filters
    ])
    const changed = key !== listKey.value
    if (changed) {
      if (listKey.value && conversations.initialized && !conversations.errorMessage) {
        listCache.set(listKey.value, {
          data: conversations.data, page: conversations.page,
          hasMore: conversations.hasMore, total: conversations.total
        })
      }
      resetConversations(false)
      listKey.value = key
      const cached = bypassListCache ? null : listCache.get(key)
      if (cached) {
        Object.assign(conversations, cached, { initialized: true })
        refresh = true
      }
    } else if (page === 0 && conversations.fetching && !refresh) {
      return activeListRequest
    } else if (page === 0 && conversations.initialized && !conversations.errorMessage && !refresh) {
      return
    }
    bypassListCache = false
    conversations.listType = listType
    conversations.inboxID = inboxID
    conversations.teamID = teamID
    conversations.viewID = viewID
    conversations.listFilters = filters
    // Keep mounted rows during updates; only an empty list needs skeletons.
    conversations.loading = showLoader && !conversations.data.length
    conversations.fetching = true
    if (page === 0) page = conversations.page
    const seq = contextSeq
    const accessSnapshot = accessSequence
    listEventPatches = new Map()
    listEventInserts = new Set()
    const requestSeq = ++listRequestSeq
    const isStale = () => seq !== contextSeq || requestSeq !== listRequestSeq
    const loadedPages = conversations.page
    try {
      conversations.errorMessage = ''
      if (refresh) {
        // Validate the entire loaded window on the server, including custom /
        // exclusive filters. One request up to the backend's 500-row limit;
        // larger windows are assembled atomically before replacing any rows.
        const pageSize = Math.min(loadedPages * CONV_LIST_PAGE_SIZE, 500)
        const rows = []
        let total = 0
        for (let batch = 1; rows.length < loadedPages * CONV_LIST_PAGE_SIZE; batch++) {
          const response = await makeConversationListRequest(
            listType, teamID, viewID, filters, batch, inboxID, pageSize
          )
          if (isStale()) return
          const result = response.data.data
          rows.push(...result.results)
          total = result.total
          if (batch >= result.total_pages || !result.results.length) break
        }
        if (isStale()) return
        // Events received after this request started are newer than its snapshot.
        // Preserve newly inserted, locally eligible rows as well as field patches.
        const returned = new Set(rows.map((row) => row.uuid))
        for (const row of conversations.data) {
          if (listEventInserts.has(row.uuid) && !returned.has(row.uuid)) rows.unshift(row)
        }
        conversations.data = []
        processConversationListResponse({ data: { data: {
          results: rows, page: loadedPages, total,
          total_pages: Math.ceil(total / CONV_LIST_PAGE_SIZE)
        } } }, accessSnapshot)
      } else {
        const response = await makeConversationListRequest(
          listType, teamID, viewID, filters, page, inboxID
        )
        if (isStale()) return
        processConversationListResponse(response, accessSnapshot)
      }
    } catch (error) {
      if (isStale()) return
      if ([401, 403, 404].includes(error?.response?.status)) {
        listCache.delete(key)
        conversations.data = []
        conversations.total = 0
        conversations.hasMore = false
      }
      if (
        [CONVERSATION_LIST_TYPE.PERSONAL, CONVERSATION_LIST_TYPE.PERSONAL_HIGH].includes(
          listType
        ) &&
        (error?.response?.status === 403 || error?.response?.status === 404)
      ) {
        const route = router.currentRoute.value
        if (route?.name !== 'inbox' || route.params?.type !== 'assigned') {
          router.replace({ name: 'inbox', params: { type: 'assigned' }, query: { status: 'Open' } })
        }
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          description: handleHTTPError(error).message
        })
      }
      if (conversations.data.length === 0) {
        conversations.errorMessage = handleHTTPError(error).message
        conversations.total = 0
      }
    } finally {
      if (!isStale()) {
        conversations.loading = false
        conversations.initialized = true
        conversations.fetching = false
      }
    }
  }

  async function makeConversationListRequest(listType, teamID, viewID, filters, page, inboxID, pageSize = CONV_LIST_PAGE_SIZE) {
    if (listType.startsWith('folder:')) {
      const [, scope, scopeID, folderID] = listType.split(':')
      return api.getSidebarFolderConversations(scope, scopeID, folderID, {
        page,
        page_size: pageSize,
        order_by: `${sortFieldMap[conversations.sortField].model}.${sortFieldMap[conversations.sortField].field}`,
        order: sortFieldMap[conversations.sortField].order
      })
    }

    filters = filters.length > 0 ? JSON.stringify(filters) : []
    if ([CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED, CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED_UNFILED].includes(listType)) {
      return api.getCustomerConversations({ page, page_size: pageSize, filters, routing: 'redistributed',
        ...(listType === CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED_UNFILED && {folder_mode:'unfiled'}),
        order_by: `${sortFieldMap[conversations.sortField].model}.${sortFieldMap[conversations.sortField].field}`,
        order: sortFieldMap[conversations.sortField].order })
    }

    switch (listType) {
      case CONVERSATION_LIST_TYPE.PERSONAL:
      case CONVERSATION_LIST_TYPE.PERSONAL_HIGH:
        return await api.getPersonalConversations(inboxID, {
          page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters,
          ...(listType === CONVERSATION_LIST_TYPE.PERSONAL_HIGH && { priority: 'high' })
        })

      case CONVERSATION_LIST_TYPE.ASSIGNED:
      case CONVERSATION_LIST_TYPE.ASSIGNED_HIGH:
        return await api.getAssignedConversations({
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters,
          ...(listType === CONVERSATION_LIST_TYPE.ASSIGNED_HIGH && { priority: 'high' })
        })
      case CONVERSATION_LIST_TYPE.UNASSIGNED:
        return await api.getUnassignedConversations({
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.ALL:
        return await api.getAllConversations({
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
        return await api.getTeamUnassignedConversations(teamID, {
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.TEAM_ALL:
      case CONVERSATION_LIST_TYPE.TEAM_HIGH:
        return await api.getTeamConversations(teamID, {
          page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters,
          ...(listType === CONVERSATION_LIST_TYPE.TEAM_HIGH && { priority: 'high' })
        })
      case CONVERSATION_LIST_TYPE.VIEW:
        return await api.getViewConversations(viewID, {
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order
        })
      case CONVERSATION_LIST_TYPE.MENTIONED:
        return await api.getMentionedConversations({
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })

      case CONVERSATION_LIST_TYPE.VISIBLE:
        return await api.getVisibleConversations({
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })

      case CONVERSATION_LIST_TYPE.VISIBLE_INTERNAL:
        return await api.getVisibleInternalConversations({
          page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })

      case CONVERSATION_LIST_TYPE.CUSTOMER:
      case CONVERSATION_LIST_TYPE.CUSTOMER_HIGH:
        return await api.getCustomerConversations({
          page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters,
          ...(listType === CONVERSATION_LIST_TYPE.CUSTOMER_HIGH && { priority: 'high' })
        })

      case CONVERSATION_LIST_TYPE.SERVICE_MAILS:
        return await api.getServiceMailConversations({
          page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })

      case CONVERSATION_LIST_TYPE.CREATED:
        return await api.getCreatedConversations({
          page: page,
          page_size: pageSize,
          order_by:
            sortFieldMap[conversations.sortField].model +
            '.' +
            sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })

      default:
        throw new Error('Invalid conversation list type: ' + listType)
    }
  }

  function trimListToCurrentPage() {
    const maxLen = conversations.page * CONV_LIST_PAGE_SIZE
    if (conversations.data.length > maxLen) {
      conversations.data.splice(maxLen)
    }
  }

  function mergeIntoList(uuid, payload) {
    const existing = conversations.data?.find((c) => c.uuid === uuid)
    if (existing) deepMerge(existing, payload)
    return existing
  }

  function processConversationListResponse(response, accessSnapshot) {
    const apiResponse = response.data.data
    const newConversations = []
    const seen = new Set()
    for (const conv of apiResponse.results) {
      if (seen.has(conv.uuid)) continue
      if (revokedUUIDs.has(conv.uuid)) {
        if (accessVersion(conv.uuid) > accessSnapshot) continue
        revokedUUIDs.delete(conv.uuid)
        notificationStore.restoreConversationNotifications?.(conv.uuid)
      }
      seen.add(conv.uuid)
      const newer = listEventPatches.get(conv.uuid)
      if (newer) deepMerge(conv, newer)
      if (!mergeIntoList(conv.uuid, conv)) newConversations.push(conv)
    }
    conversations.page = Math.max(conversations.page, apiResponse.page)
    conversations.hasMore = apiResponse.total_pages > conversations.page
    if (!conversations.data) conversations.data = []
    if (apiResponse.page === 1) {
      conversations.data.unshift(...newConversations)
    } else {
      conversations.data.push(...newConversations)
    }
    conversations.total = apiResponse.total

    trimListToCurrentPage()

    // Re-check document.hidden in case the user returned while the refresh was in flight.
    if (pendingNotificationUUIDs.size > 0) {
      let shouldPlay = false
      for (const uuid of pendingNotificationUUIDs) {
        if (isConversationInList(uuid)) {
          shouldPlay = true
        }
      }
      pendingNotificationUUIDs.clear()
      if (shouldPlay && document.hidden) {
        playNotificationSound()
      }
    }
  }

  async function updatePriority(v) {
    try {
      await api.updateConversationPriority(conversation.data.uuid, { priority: v })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateStatus(v) {
    if (!conversation.data) return
    const previous = conversation.data.status
    conversation.data.status = v
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: v })
      notificationStore.markAssignmentAsReadForConversation(conversation.data.uuid)
    } catch (error) {
      if (conversation.data) conversation.data.status = previous
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function snoozeConversation(snoozeDuration) {
    try {
      await api.updateConversationStatus(conversation.data.uuid, {
        status: CONVERSATION_DEFAULT_STATUSES.SNOOZED,
        snoozed_until: snoozeDuration
      })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  function applyTagsLocally(uuid, action, tags) {
    const targets = []
    const listConv = conversations.data?.find((c) => c.uuid === uuid)
    if (listConv) targets.push(listConv)
    if (conversation.data?.uuid === uuid) targets.push(conversation.data)

    for (const conv of targets) {
      if (!Array.isArray(conv.tags)) conv.tags = []
      if (action === TAG_ACTION.ADD) {
        for (const t of tags) {
          if (!conv.tags.includes(t)) conv.tags.push(t)
        }
      } else if (action === TAG_ACTION.SET) {
        conv.tags = [...tags]
      } else if (action === TAG_ACTION.REMOVE) {
        conv.tags = conv.tags.filter((t) => !tags.includes(t))
      }
    }
  }

  async function updateConversationTags(uuid, action, tags) {
    const source =
      conversation.data?.uuid === uuid
        ? conversation.data
        : conversations.data?.find((c) => c.uuid === uuid)
    const previous = source ? [...(source.tags || [])] : null
    applyTagsLocally(uuid, action, tags)
    try {
      await api.upsertTags(uuid, { action, tags })
    } catch (error) {
      if (previous) applyTagsLocally(uuid, TAG_ACTION.SET, previous)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      throw error
    }
  }

  async function updateAssignee(type, v) {
    try {
      await api.updateAssignee(conversation.data.uuid, type, v)
      if (conversation.data.customer_ticket) await silentRefetchConversation(conversation.data.uuid)
      else conversation.data[`assigned_${type}_id`] = v.assignee_id
      refreshConversationListForRealtimeEvent()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function removeAssignee(type) {
    try {
      await api.removeAssignee(conversation.data.uuid, type)
      if (conversation.data.customer_ticket) await silentRefetchConversation(conversation.data.uuid)
      else conversation.data[`assigned_${type}_id`] = null
      refreshConversationListForRealtimeEvent()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssigneeLastSeen(uuid) {
    if (revokedUUIDs.has(uuid)) return
    if (!isViewingConversation(uuid)) return
    markConversationAsRead(uuid)
    api.updateAssigneeLastSeen(uuid).catch(() => { })
  }

  function isConversationInList(uuid) {
    return Boolean(conversations.data?.find((c) => c.uuid === uuid))
  }

  const pendingNotificationUUIDs = new Set()

  function addPendingNotification(uuid) {
    pendingNotificationUUIDs.add(uuid)
  }

  // trailing=true: fires one final refresh after a burst so the list converges to latest state.
  const throttledFetchFirstPage = useThrottleFn(fetchFirstPageConversations, 60000, true)
  // A list cannot always determine from a WebSocket payload whether it belongs to
  // a visibility- or filter-based inbox. Re-fetch those lists promptly from the
  // backend, while combining bursts of incoming events into one follow-up request.
  const throttledRealtimeFetchFirstPage = useThrottleFn(fetchFirstPageConversations, 1000, true)

  function refreshConversationList() {
    throttledFetchFirstPage()
  }

  function refreshConversationListForRealtimeEvent() {
    throttledRealtimeFetchFirstPage()
  }

  function updateConversationLastMessage(uuid, message) {
    const conv = conversations.data?.find((c) => c.uuid === uuid)
    if (!conv) return
    const update = {}
    const timestamp = Date.parse(message.created_at)
    if (timestamp >= (Date.parse(conv.last_message_at) || 0)) {
      Object.assign(update, {
        last_message: message.text_content || message.content || getMediaPreview(message.attachments),
        last_message_at: message.created_at,
        last_message_sender: message.sender_type
      })
    }
    if (
      message.type === 'incoming' &&
      message.sender_type === 'contact' &&
      message.private === false &&
      timestamp >= (Date.parse(conv.last_customer_reply_at) || 0)
    ) {
      update.last_customer_reply_at = message.created_at
    }
    retainListEvent(uuid, update)
    Object.assign(conv, update)
  }

  async function updateConversationMessage(message) {
    if (revokedUUIDs.has(message.conversation_uuid)) return
    const version = accessVersion(message.conversation_uuid)
    if (conversation.data?.uuid !== message.conversation_uuid) {
      // Lazy invalidation: refresh the cache when the user next opens this convo, not on every WS event.
      if (messages.data.hasConversation(message.conversation_uuid)) {
        staleConversationUUIDs.add(message.conversation_uuid)
      }
      return
    }

    if (!messages.data.hasMessage(message.conversation_uuid, message.uuid)) {
      const echoId = message.echo_id
      if (echoId && messages.data.hasMessage(message.conversation_uuid, echoId)) {
        messages.data.updateMessage(message.conversation_uuid, echoId, { uuid: message.uuid })
        incrementMessageVersion()
        updateAssigneeLastSeen(message.conversation_uuid)
        return
      }

      if (message.type === 'activity') {
        const activityMessage = {
          uuid: message.uuid,
          conversation_uuid: message.conversation_uuid,
          type: 'activity',
          content: message.preview,
          created_at: message.created_at,
          updated_at: message.created_at,
          sender_type: message.sender_type
        }
        messages.data.addMessage(message.conversation_uuid, activityMessage)
        incrementMessageVersion()
        setTimeout(() => {
          if (version !== accessVersion(message.conversation_uuid)) return
          emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
            conversation_uuid: message.conversation_uuid,
            message: activityMessage
          })
        }, 100)
        if (!document.hidden) {
          updateAssigneeLastSeen(message.conversation_uuid)
        }
        return
      }

      const fetchedMessage = await fetchMessage(message.conversation_uuid, message.uuid)
      if (version !== accessVersion(message.conversation_uuid)) return
      if (fetchedMessage) {
        updateConversationLastMessage(message.conversation_uuid, fetchedMessage)
        setTimeout(() => {
          if (version !== accessVersion(message.conversation_uuid)) return
          emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
            conversation_uuid: message.conversation_uuid,
            message: fetchedMessage
          })
        }, 100)
      }

      if (!document.hidden) {
        updateAssigneeLastSeen(message.conversation_uuid)
      }
    }
  }

  function addPendingMessage(
    conversationUUID,
    content,
    isPrivate,
    author,
    attachments = [],
    textContent = '',
    meta = {}
  ) {
    const pendingMessage = {
      uuid: `pending-${Date.now()}`,
      type: 'outgoing',
      status: 'pending',
      content,
      text_content: textContent,
      content_type: 'html',
      private: isPrivate,
      sender_type: 'agent',
      sender_id: author.id,
      conversation_uuid: conversationUUID,
      created_at: new Date().toISOString(),
      author,
      attachments: attachments.map((a) => ({
        uuid: a.uuid,
        name: a.filename || a.name,
        size: a.size,
        content_type: a.content_type,
        url: a.url,
        disposition: a.disposition
      })),
      meta
    }
    messages.data.addMessage(conversationUUID, pendingMessage)
    incrementMessageVersion()
    setTimeout(() => {
      emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
        conversation_uuid: conversationUUID,
        message: pendingMessage
      })
    }, 0)

    // Safety net: auto-remove after 10 seconds if still pending.
    const tempId = pendingMessage.uuid
    setTimeout(() => {
      if (messages.data.hasMessage(conversationUUID, tempId)) {
        messages.data.removeMessage(conversationUUID, tempId)
        incrementMessageVersion()
      }
    }, 10000)

    return pendingMessage.uuid
  }

  function replacePendingMessage(conversationUUID, tempUUID, realMessage) {
    if (messages.data.hasMessage(conversationUUID, realMessage.uuid)) {
      messages.data.removeMessage(conversationUUID, tempUUID)
    } else {
      messages.data.updateMessage(conversationUUID, tempUUID, realMessage)
    }
    incrementMessageVersion()
  }

  function removePendingMessage(conversationUUID, tempUUID) {
    messages.data.removeMessage(conversationUUID, tempUUID)
    incrementMessageVersion()
  }

  function addNewConversation(conversation) {
    if (!isConversationInList(conversation.uuid)) {
      refreshConversationList()
    }
  }

  function mergeMessageUpdate(data) {
    if (revokedUUIDs.has(data.conversation_uuid)) return
    const { conversation_uuid, uuid, ...fields } = data
    if (!messages.data.hasMessage(conversation_uuid, uuid)) return
    messages.data.updateMessage(conversation_uuid, uuid, fields)
    incrementMessageVersion()
  }

  function hasExclusiveScope() {
    return (
      conversations.listType?.startsWith('folder:') ||
      [
        CONVERSATION_LIST_TYPE.ASSIGNED,
        CONVERSATION_LIST_TYPE.ASSIGNED_HIGH,
        CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED,
        CONVERSATION_LIST_TYPE.CUSTOMER_REDISTRIBUTED_UNFILED,
        CONVERSATION_LIST_TYPE.CUSTOMER,
        CONVERSATION_LIST_TYPE.CUSTOMER_HIGH,
        CONVERSATION_LIST_TYPE.TEAM_ALL,
        CONVERSATION_LIST_TYPE.TEAM_HIGH,
        CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
        CONVERSATION_LIST_TYPE.PERSONAL,
        CONVERSATION_LIST_TYPE.PERSONAL_HIGH
      ].includes(conversations.listType)
    )
  }

  function canPushInsert(conv) {
    if (isSpamConversation(conv) || hasSpamViewFilter() || hasExclusiveScope()) return false

    // Tag and custom filters are evaluated by the backend, including Spam.
    if (
      [CONVERSATION_LIST_TYPE.PERSONAL, CONVERSATION_LIST_TYPE.PERSONAL_HIGH].includes(
        conversations.listType
      ) &&
      conversations.listFilters.some((filter) => filter.model !== 'conversation_statuses')
    )
      return false
    const matched = matchesAssignmentScope(conv)
    if (matched !== null) return matched
    return conversations.listType === CONVERSATION_LIST_TYPE.ALL
  }

  function handleConvPush(payload) {
    if (!payload || !payload.uuid) return
    if (revokedUUIDs.has(payload.uuid)) {
      silentRefetchConversation(payload.uuid)
      return
    }
    retainListEvent(payload.uuid, payload)
    const existing = mergeIntoList(payload.uuid, payload)
    if (existing) {
      if (conversation.data?.uuid === payload.uuid) {
        deepMerge(conversation.data, payload)
      }
      const scopeMatch = matchesAssignmentScope(existing)
      const hasServerOnlyFilters =
        [CONVERSATION_LIST_TYPE.PERSONAL, CONVERSATION_LIST_TYPE.PERSONAL_HIGH].includes(
          conversations.listType
        ) && conversations.listFilters.some((filter) => filter.model !== 'conversation_statuses')
      if (hasServerOnlyFilters || hasExclusiveScope() || Object.hasOwn(payload, 'tags')) {
        refreshConversationListForRealtimeEvent()
        return
      }
      if (
        scopeMatch === false ||
        (conversations.status !== '' && existing.status !== conversations.status)
      ) {
        conversations.data = conversations.data.filter((item) => item.uuid !== payload.uuid)
        conversations.total = Math.max(0, conversations.total - 1)
        refreshConversationListForRealtimeEvent()
      }
      return
    }
    if (!canPushInsert(payload)) {
      refreshConversationListForRealtimeEvent()
      return
    }
    if (conversations.status !== '' && payload.status !== conversations.status) return
    if (!conversations.data) conversations.data = []
    conversations.data.unshift(payload)
    if (conversations.fetching) listEventInserts.add(payload.uuid)
    conversations.total += 1
  }

  function mergeConversationUpdate(update) {
    if (update.service_emails_changed) {
      emitter.emit('service-emails-changed')
      emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'inbox-list' })
      refreshConversationListForRealtimeEvent()
      return
    }
    if (update.access_revoked) {
      revokeConversationAccess(update.uuid)
      return
    }
    if (revokedUUIDs.has(update.uuid)) {
      silentRefetchConversation(update.uuid)
      return
    }
    retainListEvent(update.uuid, update)
    if (conversation.data?.uuid === update.uuid) {
      deepMerge(conversation.data, update)
      if (update.routing_changed || Object.hasOwn(update, 'tags')) {
        conversation.data.can_return_to_support = false
        silentRefetchConversation(update.uuid)
      }
    }
    const existing = mergeIntoList(update.uuid, update)
    if (Object.hasOwn(update, 'tags') || Object.hasOwn(update, 'customer_redistributed_active') || update.routing_changed) {
      emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'inbox-list' })
      if (existing && matchesAssignmentScope(existing) === false) {
        conversations.data = conversations.data.filter((item) => item.uuid !== update.uuid)
        conversations.total = Math.max(0, conversations.total - 1)
      }
      refreshConversationListForRealtimeEvent()
    }
  }

  function mergeContactUpdate(update) {
    const { contact_id, ...fields } = update
    if (conversation.data?.contact_id === contact_id) {
      if (!conversation.data.contact) conversation.data.contact = {}
      deepMerge(conversation.data.contact, fields)
    }
    conversations?.data?.forEach((c) => {
      if (c.contact_id === contact_id) {
        if (!c.contact) c.contact = {}
        deepMerge(c.contact, fields)
      }
    })
  }

  function resetConversations(clearPosition = true) {
    if (clearPosition) {
      bypassListCache = true
      listCache.delete(listKey.value)
      listPositions.delete(listKey.value)
      listRevision.value++
    }
    conversations.loading = false
    conversations.fetching = false
    conversations.data = []
    conversations.page = 1
    conversations.initialized = false
    conversations.hasMore = false
    conversations.total = 0
    contextSeq++
    pendingNotificationUUIDs.clear()
    clearSelection()
  }

  function setMacro(macro, context) {
    macros.value[context] = macro
  }

  function setMacroActions(actions, context) {
    if (!macros.value[context]) {
      macros.value[context] = {}
    }
    macros.value[context].actions = actions
  }

  function getMacro(context) {
    return macros.value[context] || {}
  }

  function removeMacroAction(action, context) {
    if (!macros.value[context]) return
    macros.value[context].actions = macros.value[context].actions.filter(
      (a) => a.type !== action.type
    )
  }

  function resetMacro(context) {
    macros.value = { ...macros.value, [context]: {} }
  }

  function updateTypingStatus(typingData) {
    const { conversation_uuid: uuid, is_typing } = typingData

    if (conversation.data?.uuid === uuid) {
      if (typingTimeout) {
        clearTimeout(typingTimeout)
        typingTimeout = null
      }
      conversation.isTyping = is_typing
      if (is_typing) {
        typingTimeout = setTimeout(() => {
          conversation.isTyping = false
          typingTimeout = null
        }, TYPING_RECEIVE_TIMEOUT)
      }
    }

    const prev = typingTimeoutsByUUID.get(uuid)
    if (prev) clearTimeout(prev)
    if (is_typing) {
      typingByUUID[uuid] = true
      typingTimeoutsByUUID.set(
        uuid,
        setTimeout(() => {
          delete typingByUUID[uuid]
          typingTimeoutsByUUID.delete(uuid)
        }, TYPING_RECEIVE_TIMEOUT)
      )
    } else {
      delete typingByUUID[uuid]
      typingTimeoutsByUUID.delete(uuid)
    }
  }

  function sendTyping(isTyping, otherAttributes = {}) {
    if (conversation.data?.uuid) {
      sendTypingIndicator(conversation.data.uuid, isTyping, otherAttributes.isPrivateMessage)
    }
  }

  function draftMapKey(uuid, type) {
    return `${uuid}::${type}`
  }

  async function fetchAllDrafts() {
    try {
      const resp = await api.getAllDrafts()
      const newDrafts = new Map()
      if (resp.data?.data) {
        for (const draft of resp.data.data) {
          if (!revokedUUIDs.has(draft.conversation_uuid)) newDrafts.set(draftMapKey(draft.conversation_uuid, draft.type), draft)
        }
      }
      drafts.value = newDrafts
    } catch (e) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(e).message
      })
    } finally {
      resolveDraftsReady()
    }
  }

  function getDraft(uuid, type) {
    return drafts.value.get(draftMapKey(uuid, type))
  }

  function setDraft(uuid, type, draft) {
    if (revokedUUIDs.has(uuid)) return
    drafts.value.set(draftMapKey(uuid, type), draft)
    drafts.value = new Map(drafts.value)
  }

  function removeDraft(uuid, type) {
    drafts.value.delete(draftMapKey(uuid, type))
    drafts.value = new Map(drafts.value)
  }

  function hasDraft(uuid, type) {
    return drafts.value.has(draftMapKey(uuid, type))
  }

  function conversationHasDraft(uuid) {
    return hasDraft(uuid, 'reply') || hasDraft(uuid, 'private_note')
  }

  function setSelectedDraftType(uuid, type) {
    selectedDraftType.value.set(uuid, type)
    selectedDraftType.value = new Map(selectedDraftType.value)
  }

  function resolveDraftType(uuid) {
    const last = selectedDraftType.value.get(uuid)
    if (last && hasDraft(uuid, last)) return last
    if (hasDraft(uuid, 'reply')) return 'reply'
    if (hasDraft(uuid, 'private_note')) return 'private_note'
    return last || 'reply'
  }

  function conversationDraftPreview(uuid) {
    return getDraft(uuid, resolveDraftType(uuid))
  }

  function getMediaPreview(attachments) {
    if (!attachments?.length) return ''
    const contentType = attachments[0].content_type || ''
    const i18n = getI18n()
    const t = i18n?.global?.t || ((key) => key.split('.').pop())

    if (contentType.startsWith('image/')) return t('globals.terms.image')
    if (contentType.startsWith('video/')) return t('globals.terms.video')
    if (contentType.startsWith('audio/')) return t('globals.terms.audio')
    return t('globals.terms.file')
  }

  // On new conversation uuids, subscribere user to those conversations.
  watch(
    () =>
      conversations.data
        ?.map((c) => c.uuid)
        .sort()
        .join(',') ?? '',
    () => subscribeListReplace(conversations.data?.map((c) => c.uuid) || [])
  )

  return {
    macros,
    conversations,
    listKey,
    listRevision,
    listPositions,
    conversation,
    messages,
    conversationsList,
    conversationMessages,
    currentConversationHasMoreMessages,
    isConversationOpen,
    current,
    currentContactName,
    currentTo,
    currentBCC,
    currentCC,
    isConversationInList,
    addPendingNotification,
    mergeConversationUpdate,
    revokeConversationAccess,
    handleConvPush,
    mergeContactUpdate,
    addNewConversation,
    getContactFullName,
    fetchNextMessages,
    fetchNextConversations,
    mergeMessageUpdate,
    updateAssigneeLastSeen,
    markAsUnread,
    incrementUnread,
    updateConversationMessage,
    snoozeConversation,
    fetchConversation,
    fetchConversationsList,
    fetchMessages,
    updateConversationTags,
    refreshCurrentConversation,
    updateAssignee,
    updatePriority,
    updateStatus,
    refreshConversationList,
    refreshConversationListForRealtimeEvent,
    resetConversations,
    updateConversationLastMessage,
    fetchFirstPageConversations,
    fetchStatuses,
    fetchPriorities,
    setListSortField,
    setListStatus,
    removeMacroAction,
    getMacro,
    setMacro,
    resetMacro,
    setMacroActions,
    removeAssignee,
    getListSortField,
    getListStatus,
    statuses,
    priorities,
    priorityOptions,
    statusOptionsNoSnooze,
    statusOptions,
    updateTypingStatus,
    typingByUUID,
    sendTyping,
    drafts,
    draftsReady,
    fetchAllDrafts,
    getDraft,
    setDraft,
    removeDraft,
    hasDraft,
    deleteMessage,
    conversationHasDraft,
    conversationDraftPreview,
    getMediaPreview,
    setSelectedDraftType,
    resolveDraftType,
    addPendingMessage,
    replacePendingMessage,
    removePendingMessage,
    selectedUUIDs,
    selectedCount,
    allSelected,
    toggleSelect,
    selectAll,
    clearSelection,
    isSelected
  }
})
