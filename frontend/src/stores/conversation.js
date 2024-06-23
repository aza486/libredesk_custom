import { defineStore } from 'pinia'
import { computed, reactive } from "vue"
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import api from '@/api';

export const useConversationStore = defineStore('conversation', () => {
    // List of conversations
    const conversations = reactive({
        data: [],
        loading: false,
        page: 1,
        hasMore: true,
        errorMessage: ""
    })

    // Currently selected conversation.
    const conversation = reactive({
        data: null,
        participants: {},
        loading: false,
        errorMessage: ""
    })
    // Messages for the selected conversation.
    const messages = reactive({
        data: [],
        loading: false,
        errorMessage: ""
    })

    // Map to track seen msg UUIDs for deduplication
    let seenMsgUUIDs = new Map()
    let previousConvListType = ""
    let previousPreDefinedFilter = ""
    const { toast } = useToast()

    // Computed property to sort conversations by last_message_at
    const sortedConversations = computed(() => {
        if (!conversations.data) {
            return []
        }
        return [...conversations.data].sort(
            (a, b) => new Date(b.last_message_at) - new Date(a.last_message_at)
        );
    })

    // Computed property to sort message by created_at
    const sortedMessages = computed(() => {
        if (!messages.data) {
            return [];
        }
        return [...messages.data].sort(
            (a, b) => new Date(a.created_at) - new Date(b.created_at)
        );
    });

    const getContactFullName = (uuid) => {
        const conv = conversations.data.find(conv => conv.uuid === uuid);
        return conv ? `${conv.first_name} ${conv.last_name}` : '';
    }

    function markAsRead (uuid) {
        const index = conversations.data.findIndex(conv => conv.uuid === uuid);
        if (index !== -1) {
            conversations.data[index].unread_message_count = 0
        }
    }

    async function fetchConversation (uuid) {
        conversation.loading = true;
        try {
            const resp = await api.getConversation(uuid);
            conversation.data = resp.data.data
            markAsRead(uuid)
        } catch (error) {
            conversation.errorMessage = handleHTTPError(error).message;
        } finally {
            conversation.loading = false;
        }
    }

    async function fetchParticipants (uuid) {
        try {
            const resp = await api.getConversationParticipants(uuid);
            const participants = resp.data.data.reduce((acc, p) => {
                acc[p.uuid] = p;
                return acc;
            }, {});
            updateParticipants(participants);
        } catch (error) {
            console.error("Error fetching participants:", error);
        }
    }

    async function fetchMessages (uuid) {
        messages.loading = true;
        try {
            const resp = await api.getMessages(uuid);
            messages.data = resp.data.data
        } catch (error) {
            toast({
                title: 'Uh oh! Could not fetch messages, Please try again.',
                variant: 'destructive',
                description: handleHTTPError(error).message,
            });
            messages.data = []
        } finally {
            messages.loading = false;
        }
    }

    async function fetchMessage (uuid) {
        try {
            const resp = await api.getMessage(uuid);
            // Update messages lis  only if the msg uuid does not exist.
            if (resp.data.data && resp.data.data.length > 0) {
                resp.data.data.forEach(respMsg => {
                    if (!messages.data.some(e => e.uuid === respMsg.uuid)) {
                        messages.data.push(respMsg);
                    }
                });
            }
        } catch (error) {
            messages.errorMessage = handleHTTPError(error).message;
        }
    }

    function onFilterchange () {
        conversations.data = null
        conversations.page = 1
        conversations.hasMore = true
        seenMsgUUIDs.clear()
    }

    async function fetchConversations (type, preDefinedFilter) {
        conversations.loading = true;
        conversations.errorMessage = ""

        if (type !== previousConvListType || preDefinedFilter !== previousPreDefinedFilter) {
            onFilterchange();
            previousConvListType = type
            previousPreDefinedFilter = preDefinedFilter
        }

        try {
            let response;
            switch (type) {
                case CONVERSATION_LIST_TYPE.ASSIGNED:
                    response = await api.getAssignedConversations(conversations.page, preDefinedFilter);
                    break;
                case CONVERSATION_LIST_TYPE.UNASSIGNED:
                    response = await api.getUnassignedConversations(conversations.page, preDefinedFilter);
                    break;
                case CONVERSATION_LIST_TYPE.ALL:
                    response = await api.getAllConversations(conversations.page, preDefinedFilter);
                    break;
                default:
                    console.warn(`Invalid type ${type}`);
                    return;
            }

            // Merge new conversations if any
            if (response?.data?.data) {
                const newConversations = response.data.data.filter(conversation => {
                    if (!seenMsgUUIDs.has(conversation.uuid)) {
                        seenMsgUUIDs.set(conversation.uuid, true);
                        return true;
                    }
                    return false;
                });

                if (!conversations.data) {
                    conversations.data = [];
                }

                if (newConversations.length === 0) {
                    conversations.hasMore = false;
                }

                conversations.data.push(...newConversations);
            } else {
                conversations.hasMore = false;
            }
        } catch (error) {
            conversations.errorMessage = handleHTTPError(error).message;
        } finally {
            conversations.loading = false;
        }
    }

    // Increments the page and fetches the next set of conversations.
    function fetchNextConversations (type, preDefinedFilter) {
        conversations.page++
        fetchConversations(type, preDefinedFilter)
    }

    async function updatePriority (v) {
        try {
            await api.updatePriority(conversation.data.uuid, { "priority": v });
            fetchConversation(conversation.data.uuid)
        } catch (error) {
            // Pass.
        }
    }

    async function updateStatus (v) {
        try {
            await api.updateStatus(conversation.data.uuid, { "status": v });
            fetchConversation(conversation.data.uuid)
        } catch (error) {
            toast({
                title: 'Uh oh! Could not update status, Please try again.',
                variant: 'destructive',
                description: handleHTTPError(error).message,
            });
        }
    }

    async function upsertTags (v) {
        try {
            await api.upsertTags(conversation.data.uuid, v);
            fetchConversation(conversation.data.uuid)
        } catch (error) {
            toast({
                title: 'Uh oh! Could not add tags, Please try again.',
                variant: 'destructive',
                description: handleHTTPError(error).message,
            });
        }
    }

    async function updateAssignee (type, v) {
        try {
            await api.updateAssignee(conversation.data.uuid, type, v)
            fetchConversation(conversation.data.uuid)
        } catch (error) {
            // Pass.
        }
    }

    async function updateAssigneeLastSeen (uuid) {
        try {
            await api.updateAssigneeLastSeen(uuid);
        } catch (error) {
            // Pass.
        }
    }

    function updateParticipants (newParticipants) {
        conversation.participants = {
            ...conversation.participants,
            ...newParticipants
        };
    }

    // Websocket updates.
    function updateConversationList (msg) {
        const updatedConversation = conversations.data.find(c => c.uuid === msg.conversation_uuid);
        if (updatedConversation) {
            updatedConversation.last_message = msg.last_message;
            updatedConversation.last_message_at = msg.last_message_at;
            // Increase conversation unread msg count only if it's not open.
            if (updatedConversation.uuid !== conversation.data.uuid) {
                updatedConversation.unread_message_count += 1
            }
        }
    }
    function updateMessageList (msg) {
        // Check if this conversation is selected and then update messages list.
        if (conversation?.data?.uuid === msg.conversation_uuid) {
            // Fetch entire msg if the give msg does not exist in the msg list.
            if (!messages.data.some(message => message.uuid === msg.uuid)) {
                fetchParticipants(msg.conversation_uuid)
                fetchMessage(msg.uuid)
                updateAssigneeLastSeen(msg.conversation_uuid)
            }
        }
    }

    function updateMessageStatus (uuid, status) {
        const message = messages.data.find(m => m.uuid === uuid)
        if (message) {
            message.status = status
        }
    }

    function $reset () {
        // Reset conversations state
        conversations.data = []
        conversations.loading = false
        conversations.page = 1
        conversations.hasMore = true
        conversations.errorMessage = ""

        // Reset conversation state
        conversation.data = null
        conversation.participants = {}
        conversation.loading = false
        conversation.errorMessage = ""

        // Reset messages state
        messages.data = []
        messages.loading = false
        messages.errorMessage = ""
    }

    return { conversations, conversation, messages, sortedConversations, sortedMessages, getContactFullName, fetchParticipants, fetchNextConversations, updateMessageStatus, updateAssigneeLastSeen, updateMessageList, fetchConversation, fetchConversations, fetchMessages, upsertTags, updateAssignee, updatePriority, updateStatus, updateConversationList, $reset };
})
