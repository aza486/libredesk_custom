import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { subscribeConversationsList } from '@/websocket'
import api from '@/api'

export const useConversationStore = defineStore('conversation', () => {
  const MAX_CONV_LIST_PAGE_SIZE = 20
  const MAX_MESSAGE_LIST_PAGE_SIZE = 20
  const priorities = ref([])
  const statuses = ref([])

  const prioritiesForSelect = computed(() => {
    return priorities.value.map(p => ({ label: p.name, value: p.id }))
  })
  const statusesForSelect = computed(() => {
    return statuses.value.map(s => ({ label: s.name, value: s.id }))
  })

  const sortFieldMap = {
    oldest: {
      field: 'last_message_at',
      order: 'asc'
    },
    newest: {
      field: 'last_message_at',
      order: 'desc'
    },
    started_first: {
      field: 'created_at',
      order: 'asc'
    },
    started_last: {
      field: 'created_at',
      order: 'desc'
    },
    waiting_longest: {
      field: 'last_message_at',
      order: 'asc'
    },
    next_sla_target: {
      field: 'next_sla_deadline_at',
      order: 'asc'
    },
    priority_first: {
      field: 'priority_id',
      order: 'desc'
    }
  }

  const sortFieldLabels = {
    oldest: 'Oldest',
    newest: 'Newest',
    started_first: 'Started first',
    started_last: 'Started last',
    waiting_longest: 'Waiting longest',
    next_sla_target: 'Next SLA target',
    priority_first: 'Priority first'
  }

  const conversations = reactive({
    data: [],
    listType: null,
    status: 'Open',
    sortField: 'newest',
    listFilters: [],
    viewID: 0,
    teamID: 0,
    loading: false,
    page: 1,
    hasMore: false,
    errorMessage: ''
  })

  const conversation = reactive({
    data: null,
    participants: {},
    loading: false,
    errorMessage: ''
  })

  const messages = reactive({
    data: [],
    loading: false,
    page: 1,
    hasMore: false,
    errorMessage: ''
  })

  let seenConversationUUIDs = new Map()
  let seenMessageUUIDs = new Set()
  let reRenderInterval = setInterval(() => {
    conversations.data = [...conversations.data]
  }, 120000)
  const emitter = useEmitter()

  function clearListReRenderInterval () {
    clearInterval(reRenderInterval)
  }

  function setListStatus (status, fetch = true) {
    if (conversations.status === status) return
    conversations.status = status
    if (fetch) {
      resetConversations()
      reFetchConversationsList()
    }
  }

  function setListSortField (field) {
    if (conversations.sortField === field) return
    conversations.sortField = field
    resetConversations()
    reFetchConversationsList()
  }

  const getListSortField = computed(() => {
    return sortFieldLabels[conversations.sortField]
  })

  const getListStatus = computed(() => {
    return conversations.status
  })

  async function fetchStatuses () {
    if (statuses.value.length > 0) return
    try {
      const response = await api.getStatuses()
      statuses.value = response.data.data
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function fetchPriorities () {
    if (priorities.value.length > 0) return
    try {
      const response = await api.getPriorities()
      priorities.value = response.data.data
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  const conversationsList = computed(() => {
    if (!conversations.data) return []
    return conversations.data
  })

  const conversationMessages = computed(() => {
    if (!messages.data) return []
    return [...messages.data].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  })

  function markConversationAsRead (uuid) {
    const index = conversations.data.findIndex(conv => conv.uuid === uuid)
    if (index !== -1) {
      conversations.data[index].unread_message_count = 0
    }
  }

  const currentContactName = computed(() => {
    return conversation.data?.contact.first_name + ' ' + conversation.data?.contact.last_name
  })

  function getContactFullName (uuid) {
    if (conversations?.data) {
      const conv = conversations.data.find(conv => conv.uuid === uuid)
      return conv ? `${conv.contact.first_name} ${conv.contact.last_name}` : ''
    }
  }

  const current = computed(() => {
    return conversation.data
  })

  async function fetchConversation (uuid) {
    conversation.loading = true
    try {
      const resp = await api.getConversation(uuid)
      conversation.data = resp.data.data
      markConversationAsRead(uuid)
      resetMessages()
    } catch (error) {
      conversation.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: conversation.errorMessage
      })
    } finally {
      conversation.loading = false
    }
  }

  async function fetchParticipants (uuid) {
    try {
      const resp = await api.getConversationParticipants(uuid)
      const participants = resp.data.data.reduce((acc, p) => {
        acc[p.id] = p
        return acc
      }, {})
      updateParticipants(participants)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function fetchMessages (uuid) {
    messages.loading = true
    try {
      const response = await api.getConversationMessages(uuid, { page: messages.page, page_size: MAX_MESSAGE_LIST_PAGE_SIZE })
      const result = response.data?.data || {}
      const results = result.results || []
      const newMessages = results.filter(message => {
        if (!seenMessageUUIDs.has(message.uuid)) {
          seenMessageUUIDs.add(message.uuid)
          return true
        }
        return false
      })
      if (newMessages.length === 0 && messages.page === 1) messages.data = []
      if (result.total_pages <= messages.page) messages.hasMore = false
      else messages.hasMore = true
      messages.data.unshift(...newMessages)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      messages.data = []
      messages.hasMore = false
    } finally {
      messages.loading = false
    }
  }

  async function fetchNextMessages () {
    messages.page++
    fetchMessages(conversation.data.uuid)
  }

  async function fetchMessage (cuuid, uuid) {
    try {
      const response = await api.getConversationMessage(cuuid, uuid)
      if (response?.data?.data) {
        const newMsg = response.data.data
        if (!messages.data.some(m => m.uuid === newMsg.uuid)) {
          messages.data.push(newMsg)
        }
        return newMsg
      }
    } catch (error) {
      messages.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: messages.errorMessage
      })
    }
  }

  function fetchNextConversations () {
    conversations.page++
    fetchConversationsList(true, conversations.listType, conversations.teamID, conversations.listFilters)
  }

  function reFetchConversationsList (showLoader = true) {
    fetchConversationsList(showLoader, conversations.listType, conversations.teamID, conversations.listFilters, conversations.viewID)
  }

  async function fetchConversationsList (showLoader = true, listType = null, teamID = 0, filters = [], viewID = 0) {
    if (!listType) return
    if (conversations.listType !== listType || conversations.teamID !== teamID || conversations.viewID !== viewID) {
      resetConversations()
    }
    if (listType) conversations.listType = listType
    if (teamID) conversations.teamID = teamID
    if (viewID) conversations.viewID = viewID
    if (conversations.status) {
      filters = filters.filter(f => f.model !== 'conversation_statuses')
      filters.push({
        model: 'conversation_statuses',
        field: 'name',
        operator: 'equals',
        value: conversations.status
      })
    }
    if (filters) conversations.listFilters = filters
    subscribeConversationsList(listType, teamID)
    if (showLoader) conversations.loading = true
    try {
      conversations.errorMessage = ''
      const response = await makeConversationListRequest(listType, teamID, viewID, filters)
      processConversationListResponse(response)
    } catch (error) {
      conversations.errorMessage = handleHTTPError(error).message
    } finally {
      conversations.loading = false
    }
  }

  async function makeConversationListRequest (listType, teamID, viewID, filters) {
    filters = filters.length > 0 ? JSON.stringify(filters) : []
    switch (listType) {
      case CONVERSATION_LIST_TYPE.ASSIGNED:
        return await api.getAssignedConversations({
          page: conversations.page,
          page_size: MAX_CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.UNASSIGNED:
        return await api.getUnassignedConversations({
          page: conversations.page,
          page_size: MAX_CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.ALL:
        return await api.getAllConversations({
          page: conversations.page,
          page_size: MAX_CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
        return await api.getTeamUnassignedConversations(teamID, {
          page: conversations.page,
          page_size: MAX_CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order
        })
      case CONVERSATION_LIST_TYPE.VIEW:
        return await api.getViewConversations(viewID, {
          page: conversations.page,
          page_size: MAX_CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order
        })
      default:
        throw new Error('Invalid conversation list type: ' + listType)
    }
  }

  function processConversationListResponse (response) {
    const apiResponse = response.data.data
    const newConversations = apiResponse.results.filter(conversation => {
      if (!seenConversationUUIDs.has(conversation.uuid)) {
        seenConversationUUIDs.set(conversation.uuid, true)
        return true
      }
      return false
    })
    if (apiResponse.total_pages <= conversations.page) conversations.hasMore = false
    else conversations.hasMore = true
    if (!conversations.data) conversations.data = []
    conversations.data.push(...newConversations)
  }

  async function updatePriority (v) {
    try {
      await api.updateConversationPriority(conversation.data.uuid, { priority: v })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateStatus (v) {
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: v })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function snoozeConversation (snoozeDuration) {
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: CONVERSATION_DEFAULT_STATUSES.SNOOZED, snoozed_until: snoozeDuration })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }


  async function upsertTags (v) {
    try {
      await api.upsertTags(conversation.data.uuid, v)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssignee (type, v) {
    try {
      await api.updateAssignee(conversation.data.uuid, type, v)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssigneeLastSeen (uuid) {
    try {
      await api.updateAssigneeLastSeen(uuid)
    } catch (error) {
      // pass
    }
  }

  function updateParticipants (newParticipants) {
    conversation.participants = {
      ...conversation.participants,
      ...newParticipants
    }
  }

  function conversationUUIDExists (uuid) {
    return conversations.data?.find(c => c.uuid === uuid) ? true : false
  }

  function updateConversationLastMessage (message) {
    const listConversation = conversations.data.find(c => c.uuid === message.conversation_uuid)
    if (listConversation) {
      listConversation.last_message = message.content
      listConversation.last_message_at = message.created_at
      if (listConversation.uuid !== conversation?.data?.uuid) {
        listConversation.unread_message_count += 1
      }
    }
  }

  async function updateConversationMessageList (message) {
    if (conversation?.data?.uuid === message.conversation_uuid) {
      if (!messages.data.some(msg => msg.uuid === message.uuid)) {
        fetchParticipants(message.conversation_uuid)
        const fetchedMessage = await fetchMessage(message.conversation_uuid, message.uuid)
        updateAssigneeLastSeen(message.conversation_uuid)
        setTimeout(() => {
          emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
            conversation_uuid: message.conversation_uuid,
            message: fetchedMessage
          })
        }, 50)
      }
    }
  }

  function addNewConversation (conversation) {
    if (!conversationUUIDExists(conversation.uuid)) {
      conversations.data.push(conversation)
    }
  }

  function updateMessageProp (message) {
    const existingMessage = messages.data.find(m => m.uuid === message.uuid)
    if (existingMessage) {
      existingMessage[message.prop] = message.value
    }
  }

  function updateConversationProp (update) {
    if (conversation.data?.uuid === update.uuid) {
      conversation.data[update.prop] = update.value
    }
    const existingConversation = conversations?.data?.find(c => c.uuid === update.uuid)
    if (existingConversation) {
      existingConversation[update.prop] = update.value
    }
  }

  function resetCurrentConversation () {
    Object.assign(conversation, {
      data: null,
      participants: {},
      loading: false,
      errorMessage: ''
    })
  }

  function resetConversations () {
    conversations.data = []
    conversations.page = 1
    seenConversationUUIDs = new Map()
  }

  function resetMessages () {
    messages.data.length = 0
    Object.assign(messages, {
      loading: false,
      page: 1,
      hasMore: true,
      errorMessage: ''
    })
    seenMessageUUIDs = new Set()
  }

  return {
    conversations,
    conversation,
    messages,
    conversationsList,
    conversationMessages,
    current,
    currentContactName,
    clearListReRenderInterval,
    conversationUUIDExists,
    updateConversationProp,
    addNewConversation,
    getContactFullName,
    fetchParticipants,
    fetchNextMessages,
    fetchNextConversations,
    updateMessageProp,
    updateAssigneeLastSeen,
    updateConversationMessageList,
    snoozeConversation,
    fetchConversation,
    fetchConversationsList,
    fetchMessages,
    upsertTags,
    reFetchConversationsList,
    updateAssignee,
    updatePriority,
    updateStatus,
    updateConversationLastMessage,
    resetMessages,
    resetCurrentConversation,
    fetchStatuses,
    fetchPriorities,
    setListSortField,
    setListStatus,
    getListSortField,
    getListStatus,
    statuses,
    priorities,
    prioritiesForSelect,
    statusesForSelect
  }
})
