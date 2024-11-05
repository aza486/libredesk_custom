import { defineStore } from 'pinia'
import { computed, reactive, watch, toRefs } from 'vue'
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useStorage } from '@vueuse/core'
import { subscribeConversationsList } from '@/websocket'
import api from '@/api'

export const useConversationStore = defineStore('conversation', () => {
  const conversationsListType = useStorage('conversation_list_type', CONVERSATION_LIST_TYPE.ASSIGNED)
  const conversationListFilters = useStorage('conversation_list_filters', [])

  // List of conversations
  const conversations = reactive({
    data: [],
    loading: false,
    type: conversationsListType,
    filters: conversationListFilters,
    page: 1,
    hasMore: false,
    errorMessage: ''
  })

  // Currently selected conversation.
  const conversation = reactive({
    data: null,
    participants: {},
    loading: false,
    errorMessage: ''
  })

  // Messages for the selected conversation.
  const messages = reactive({
    data: [],
    loading: false,
    page: 1,
    hasMore: false,
    errorMessage: ''
  })

  const { type: conversatonType } = toRefs(conversations);

  // Set type on tab change.
  watch(conversatonType, (type) => {
    setConversationList(type)
  })

  // Map to track seen msg UUIDs for deduplication
  let seenConversationUUIDs = new Map()
  let seenMessageUUIDs = new Set()
  let reRenderInterval = setInterval(() => {
    conversations.data = [...conversations.data]
  }, 120000)
  const emitter = useEmitter()

  // Clears the re-render interval
  function clearListReRenderInterval () {
    clearInterval(reRenderInterval)
  }

  // Sort conversations by last_message_at
  const sortedConversations = computed(() => {
    if (!conversations.data) {
      return []
    }
    return [...conversations.data].sort(
      (a, b) => new Date(b.last_message_at) - new Date(a.last_message_at)
    )
  })

  // Sort messages by created_at
  const sortedMessages = computed(() => {
    if (!messages.data) {
      return []
    }
    return [...messages.data].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  })

  function markConversationAsRead (uuid) {
    const index = conversations.data.findIndex((conv) => conv.uuid === uuid)
    if (index !== -1) {
      conversations.data[index].unread_message_count = 0
    }
  }

  const currentContactName = computed(() => {
    return conversation.data?.contact.first_name + " " + conversation.data?.contact.last_name
  })

  const getContactFullName = (uuid) => {
    if (conversations?.data) {
      const conv = conversations.data.find((conv) => conv.uuid === uuid)
      return conv ? `${conv.contact.first_name} ${conv.contact.last_name}` : ''
    }
  }

  // Returns the current conversation
  const current = computed(() => {
    return conversation.data
  })

  // Fetches conversation by uuid.
  async function fetchConversation (uuid) {
    conversation.loading = true
    try {
      const resp = await api.getConversation(uuid)
      conversation.data = resp.data.data
      // Mark this conversation as read.
      markConversationAsRead(uuid)
      // Reset messages state.
      resetMessages()
    } catch (error) {
      conversation.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Something went wrong',
        variant: 'destructive',
        description: conversation.errorMessage
      })
    } finally {
      conversation.loading = false
    }
  }

  // Fetches participants of conversation by uuid.
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
        title: 'Something went wrong',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Fetches messages of a conversation.
  async function fetchMessages (uuid) {
    messages.loading = true
    try {
      const response = await api.getConversationMessages(uuid, messages.page)
      const result = response.data?.data || {}
      const results = result.results || []
      // Filter out messages already seen.
      const newMessages = results.filter((message) => {
        if (!seenMessageUUIDs.has(message.uuid)) {
          seenMessageUUIDs.add(message.uuid)
          return true
        }
        return false
      })
      if (newMessages.length === 0 && messages.page === 1) messages.data = []
      if (result.total_pages <= messages.page)
        messages.hasMore = false
      else
        messages.hasMore = true
      messages.data.unshift(...newMessages)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Something went wrong',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      messages.data = []
      messages.hasMore = false
    } finally {
      messages.loading = false
    }
  }

  // Fetches next page of messages by incrementing the page number.
  async function fetchNextMessages () {
    messages.page++
    fetchMessages(conversation.data.uuid)
  }

  // Fetches a specific message of conversation
  async function fetchMessage (cuuid, uuid) {
    try {
      const response = await api.getConversationMessage(cuuid, uuid)
      if (response?.data?.data) {
        const newMsg = response.data.data
        if (!messages.data.some((m) => m.uuid === newMsg.uuid)) {
          messages.data.push(newMsg)
        }
        return newMsg
      }
    } catch (error) {
      messages.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Could not fetch message',
        variant: 'destructive',
        description: messages.errorMessage
      })
    }
  }

  function setConversationList (type) {
    resetConversations()
    conversations.type = type
    subscribeConversationsList(type)
    fetchConversationsList(true)
  }

  function setConversationListFilters (filters) {
    resetConversations()
    conversations.filters = filters
    fetchConversationsList(true)
  }

  async function fetchConversationsList (showLoader = true) {
    if (showLoader) conversations.loading = true

    try {
      conversations.errorMessage = ''
      let response = null

      switch (conversations.type) {
        case CONVERSATION_LIST_TYPE.ASSIGNED:
          response = await api.getAssignedConversations({
            page: conversations.page,
            filters: conversations.filters ? JSON.stringify(conversations.filters) : '[]',
          })
          break
        case CONVERSATION_LIST_TYPE.UNASSIGNED:
          response = await api.getUnassignedConversations({
            page: conversations.page,
            filters: conversations.filters ? JSON.stringify(conversations.filters) : '[]',
          })
          break
        case CONVERSATION_LIST_TYPE.ALL:
          response = await api.getAllConversations({
            page: conversations.page,
            filters: conversations.filters ? JSON.stringify(conversations.filters) : '[]',
          })
          break
        default:
          return
      }
      const apiResponse = response.data.data
      const newConversations = apiResponse.results.filter((conversation) => {
        if (!seenConversationUUIDs.has(conversation.uuid)) {
          seenConversationUUIDs.set(conversation.uuid, true)
          return true
        }
        return false
      })
      if (apiResponse.total_pages <= conversations.page)
        conversations.hasMore = false
      else
        conversations.hasMore = true
      if (!conversations.data) conversations.data = []
      conversations.data.push(...newConversations)
    } catch (error) {
      conversations.errorMessage = handleHTTPError(error).message
    } finally {
      conversations.loading = false
    }
  }


  // Increments the page and fetches conversations
  function fetchNextConversations () {
    conversations.page++
    fetchConversationsList(true)
  }

  async function updatePriority (v) {
    try {
      await api.updateConversationPriority(conversation.data.uuid, { priority: v })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Could not update priority',
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
        title: 'Could not update status',
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
        title: 'Could not add tags',
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
        title: 'Could not update assignee',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssigneeLastSeen (uuid) {
    try {
      await api.updateAssigneeLastSeen(uuid)
    } catch (error) {
      // Pass.
    }
  }

  function updateParticipants (newParticipants) {
    conversation.participants = {
      ...conversation.participants,
      ...newParticipants
    }
  }

  function conversationUUIDExists (uuid) {
    return conversations.data?.find((c) => c.uuid === uuid) ? true : false
  }

  /**** Websocket updates ****/

  // Update the last message for a conversation.
  function updateConversationLastMessage (message) {
    const listConversation = conversations.data.find((c) => c.uuid === message.conversation_uuid)
    if (listConversation) {
      listConversation.last_message = message.content
      listConversation.last_message_at = message.created_at
      // Increment unread count only if conversation is not open.
      if (listConversation.uuid !== conversation?.data?.uuid) {
        listConversation.unread_message_count += 1
      }
    }
  }

  // Adds a new message to conversation.
  async function updateConversationMessageList (message) {
    // Fetch entire message only if the convesation is open and the message is not present in the list.
    if (conversation?.data?.uuid === message.conversation_uuid) {
      if (!messages.data.some((msg) => msg.uuid === message.uuid)) {
        fetchParticipants(message.conversation_uuid)
        const fetchedMessage = await fetchMessage(message.conversation_uuid, message.uuid)
        updateAssigneeLastSeen(message.conversation_uuid)
        setTimeout(() => {
          emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, { conversation_uuid: message.conversation_uuid, message: fetchedMessage })
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
    const existingMessage = messages.data.find((m) => m.uuid === message.uuid)
    if (existingMessage) {
      existingMessage[message.prop] = message.value
    }
  }

  function updateConversationProp (update) {
    // Update prop in open conversation.
    if (conversation.data?.uuid === update.uuid) {
      conversation.data[update.prop] = update.value
    }
    // Update prop in conversation list.
    const existingConversation = conversations?.data?.find((c) => c.uuid === update.uuid)
    if (existingConversation) {
      existingConversation[conversation.prop] = conversation.value
    }
  }

  function resetConversations () {
    conversations.data = []
    conversations.loading = false
    conversations.page = 1
    conversations.hasMore = true
    conversations.errorMessage = ''
    seenConversationUUIDs.clear()
  }

  function resetCurrentConversation () {
    conversation.data = null
    conversation.participants = {}
    conversation.loading = false
    conversation.errorMessage = ''
  }

  function resetMessages () {
    messages.data = []
    messages.loading = false
    messages.page = 1
    messages.hasMore = true
    messages.errorMessage = ''
    seenMessageUUIDs = new Set()
  }

  return {
    conversations,
    conversation,
    messages,
    sortedConversations,
    sortedMessages,
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
    fetchConversation,
    fetchConversationsList,
    fetchMessages,
    setConversationList,
    setConversationListFilters,
    upsertTags,
    updateAssignee,
    updatePriority,
    updateStatus,
    updateConversationLastMessage,
    resetMessages,
    resetCurrentConversation,
  }
})