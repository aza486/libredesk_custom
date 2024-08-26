import { defineStore } from 'pinia'
import { computed, reactive, onUnmounted } from 'vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'

export const useConversationStore = defineStore('conversation', () => {
  // List of conversations
  const conversations = reactive({
    data: [],
    loading: false,
    page: 1,
    hasMore: true,
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
    hasMore: true,
    errorMessage: ''
  })

  // Map to track seen msg UUIDs for deduplication
  let seenConversationUUIDs = new Map()
  let seenMessageUUIDs = new Set()
  let previousConvListType = ''
  let previousPreDefinedFilter = ''
  let reRenderInterval = setInterval(() => {
    conversations.data = [...conversations.data]
  }, 60000)
  const { toast } = useToast()
  const emitter = useEmitter()

  // Clear the reRenderInterval when the store is destroyed.
  onUnmounted(() => {
    clearInterval(reRenderInterval)
  })

  // Computed property to sort conversations by last_message_at
  const sortedConversations = computed(() => {
    if (!conversations.data) {
      return []
    }
    return [...conversations.data].sort(
      (a, b) => new Date(b.last_message_at) - new Date(a.last_message_at)
    )
  })

  // Computed property to sort messages by created_at
  const sortedMessages = computed(() => {
    if (!messages.data) {
      return []
    }
    return [...messages.data].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  })

  const getContactFullName = (uuid) => {
    if (conversations?.data) {
      const conv = conversations.data.find((conv) => conv.uuid === uuid)
      return conv ? `${conv.first_name} ${conv.last_name}` : ''
    }
  }

  function markAsRead (uuid) {
    const index = conversations.data.findIndex((conv) => conv.uuid === uuid)
    if (index !== -1) {
      conversations.data[index].unread_message_count = 0
    }
  }

  async function fetchConversation (uuid) {
    conversation.loading = true
    try {
      const resp = await api.getConversation(uuid)
      conversation.data = resp.data.data
      // mark this conversation as read.
      markAsRead(uuid)
      // reset messages state on new conversation fetch.
      resetMessages()
    } catch (error) {
      conversation.errorMessage = handleHTTPError(error).message
    } finally {
      conversation.loading = false
    }
  }

  async function fetchParticipants (uuid) {
    try {
      const resp = await api.getConversationParticipants(uuid)
      const participants = resp.data.data.reduce((acc, p) => {
        acc[p.uuid] = p
        return acc
      }, {})
      updateParticipants(participants)
    } catch (error) {
      console.error('Error fetching participants:', error)
    }
  }

  async function fetchMessages (uuid) {
    messages.loading = true
    try {
      const response = await api.getMessages(uuid, messages.page)
      const fetchedMessages = response.data?.data || []

      // Filter out messages that have already been seen
      const newMessages = fetchedMessages.filter((message) => {
        if (!seenMessageUUIDs.has(message.uuid)) {
          seenMessageUUIDs.add(message.uuid)
          return true
        }
        return false
      })

      if (newMessages.length === 0 && messages.page === 1) {
        messages.data = []
        return
      }

      if (newMessages.length === 0 && messages.page > 1) {
        messages.hasMore = false
      }

      // Add new messages to the messages data
      messages.data.unshift(...newMessages)
    } catch (error) {
      toast({
        title: 'Could not fetch messages, Please try again.',
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

  async function fetchMessage (uuid) {
    try {
      const response = await api.getMessage(uuid)
      if (response?.data?.data) {
        const message = response.data.data
        if (!messages.data.some((m) => m.uuid === message.uuid)) {
          messages.data.push(message)
        }
      }
    } catch (error) {
      messages.errorMessage = handleHTTPError(error).message
    }
  }

  function onFilterchange () {
    conversations.data = null
    conversations.page = 1
    conversations.hasMore = true
    seenConversationUUIDs.clear()
  }

  async function fetchConversations (type, preDefinedFilter) {
    conversations.loading = true
    conversations.errorMessage = ''

    if (type !== previousConvListType || preDefinedFilter !== previousPreDefinedFilter) {
      onFilterchange()
      previousConvListType = type
      previousPreDefinedFilter = preDefinedFilter
    }

    try {
      let response
      switch (type) {
        case CONVERSATION_LIST_TYPE.ASSIGNED:
          response = await api.getAssignedConversations(conversations.page, preDefinedFilter)
          break
        case CONVERSATION_LIST_TYPE.UNASSIGNED:
          response = await api.getTeamConversations(conversations.page, preDefinedFilter)
          break
        case CONVERSATION_LIST_TYPE.ALL:
          response = await api.getAllConversations(conversations.page, preDefinedFilter)
          break
        default:
          console.warn(`Invalid type ${type}`)
          return
      }

      // Merge new conversations if any
      if (response?.data?.data) {
        const newConversations = response.data.data.filter((conversation) => {
          if (!seenConversationUUIDs.has(conversation.uuid)) {
            seenConversationUUIDs.set(conversation.uuid, true)
            return true
          }
          return false
        })

        if (!conversations.data) {
          conversations.data = []
        }

        if (newConversations.length === 0) {
          conversations.hasMore = false
        }

        conversations.data.push(...newConversations)
      } else {
        conversations.hasMore = false
      }
    } catch (error) {
      conversations.errorMessage = handleHTTPError(error).message
      toast({
        title: 'Could not fetch conversations.',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      conversations.loading = false
    }
  }

  // Increments the page and fetches the next set of conversations.
  function fetchNextConversations (type, preDefinedFilter) {
    conversations.page++
    fetchConversations(type, preDefinedFilter)
  }

  async function updatePriority (v) {
    try {
      await api.updateConversationPriority(conversation.data.uuid, { priority: v })
    } catch (error) {
      // Pass.
    }
  }

  async function updateStatus (v) {
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: v })
    } catch (error) {
      toast({
        title: 'Uh oh! Could not update status, Please try again.',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function upsertTags (v) {
    try {
      await api.upsertTags(conversation.data.uuid, v)
    } catch (error) {
      toast({
        title: 'Uh oh! Could not add tags, Please try again.',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssignee (type, v) {
    try {
      await api.updateAssignee(conversation.data.uuid, type, v)
    } catch (error) {
      // Pass.
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
    const conv = conversations.data.find((c) => c.uuid === message.conversation_uuid)
    if (conv) {
      conv.last_message = message.content
      conv.last_message_at = message.created_at
      // Increment unread count only if conversation is not open.
      if (conv.uuid !== conversation?.data?.uuid) {
        conv.unread_message_count += 1
      }
    }
  }

  // Update message in a conversation.
  function updateConversationMessageList (message) {
    // Fetch entire message only if the convesation is open and the message is not present in the list.
    if (conversation?.data?.uuid === message.conversation_uuid) {
      if (!messages.data.some((msg) => msg.uuid === message.uuid)) {
        fetchParticipants(message.conversation_uuid)
        fetchMessage(message.uuid)
        updateAssigneeLastSeen(message.conversation_uuid)
        if (message.type === 'outgoing') {
          setTimeout(() => {
            emitter.emit('new-outgoing-message', { conversation_uuid: message.conversation_uuid })
          }, 50)
        }
      }
    }
  }

  function addNewConversation (conversation) {
    if (!conversationUUIDExists(conversation.uuid)) {
      conversations.data.push(conversation)
    }
  }

  function updateMessageProp (message) {
    // Update prop in list.
    const existingMessage = messages.data.find((m) => m.uuid === message.uuid)
    if (existingMessage) {
      existingMessage[message.prop] = message.value
    }
  }

  function updateConversationProp (conversation) {
    // Update prop if conversation is open.
    if (conversation?.data?.uuid === conversation.uuid) {
      conversation.data[conversation.prop] = conversation.value
    }

    // Update prop in list.
    const existingConversation = conversations?.data?.find((c) => c.uuid === conversation.uuid)
    if (existingConversation) {
      existingConversation[conversation.prop] = conversation.value
    }
  }

  function $reset () {
    // Reset conversations state
    conversations.data = []
    conversations.loading = false
    conversations.page = 1
    conversations.hasMore = true
    conversations.errorMessage = ''

    // Reset conversation state
    conversation.data = null
    conversation.participants = {}
    conversation.loading = false
    conversation.errorMessage = ''

    // Reset messages state
    resetMessages()
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
    fetchConversations,
    fetchMessages,
    upsertTags,
    updateAssignee,
    updatePriority,
    updateStatus,
    updateConversationLastMessage,
    $reset
  }
})
