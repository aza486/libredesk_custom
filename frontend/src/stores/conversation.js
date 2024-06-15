import { defineStore } from 'pinia'
import { ref, computed } from "vue"
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api';

export const useConversationStore = defineStore('conversation', () => {
    // List of conversations
    const conversations = ref({
        data: [],
        loading: false,
        errorMessage: ""
    })

    // Currently selected convesation.
    const conversation = ref({
        data: null,
        participants: {},
        loading: false,
        errorMessage: ""
    })
    // Selected converation messages.
    const messages = ref({
        data: [],
        loading: false,
        errorMessage: ""
    })
    const { toast } = useToast()

    // Computed property to sort conversations by last_message_at
    const sortedConversations = computed(() => {
        if (!conversations.value.data) {
            return []
        }
        return [...conversations.value.data].sort(
            (a, b) => new Date(b.last_message_at) - new Date(a.last_message_at)
        );
    })

    // Computed property to sort message by created_at
    const sortedMessages = computed(() => {
        if (!messages.value.data) {
            return [];
        }
        return [...messages.value.data].sort(
            (a, b) => new Date(a.created_at) - new Date(b.created_at)
        );
    });

    function markAsRead (uuid) {
        const index = conversations.value.data.findIndex(conv => conv.uuid === uuid);
        if (index !== -1) {
            conversations.value.data[index].unread_message_count = 0
        }
    }

    async function fetchConversation (uuid) {
        fetchParticipants(uuid)
        conversation.value.loading = true;
        try {
            const resp = await api.getConversation(uuid);
            conversation.value.data = resp.data.data
            markAsRead(uuid)
        } catch (error) {
            conversation.value.errorMessage = handleHTTPError(error).message;
        } finally {
            conversation.value.loading = false;
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
        messages.value.loading = true;
        try {
            const resp = await api.getMessages(uuid);
            messages.value.data = resp.data.data
        } catch (error) {
            toast({
                title: 'Uh oh! Could not fetch messages, Please try again.',
                variant: 'destructive',
                description: handleHTTPError(error).message,
            });
            messages.value.data = []
        } finally {
            messages.value.loading = false;
        }
    }

    async function fetchMessage (uuid) {
        try {
            const resp = await api.getMessage(uuid);
            // Push only if the msg uuid doesn't exist already.
            if (resp.data.data && resp.data.data.length > 0) {
                resp.data.data.forEach(respMsg => {
                    if (!messages.value.data.some(e => e.uuid === respMsg.uuid)) {
                        messages.value.data.push(respMsg);
                    }
                });
            }
        } catch (error) {
            messages.value.errorMessage = handleHTTPError(error).message;
        }
    }

    async function fetchConversations () {
        conversations.value.loading = true;
        try {
            const resp = await api.getConversations();
            conversations.value.data = resp.data.data
        } catch (error) {
            conversations.value.errorMessage = handleHTTPError(error).message;
        } finally {
            conversations.value.loading = false;
        }
    }

    async function updatePriority (v) {
        try {
            await api.updatePriority(conversation.value.data.uuid, { "priority": v });
        } catch (error) {
            // Pass.
        }
    }

    async function updateStatus (v) {
        try {
            await api.updateStatus(conversation.value.data.uuid, { "status": v });
            conversation.value.data.status = v
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
            await api.upsertTags(conversation.value.data.uuid, v);
        } catch (error) {
            // Pass.
        }
    }

    async function updateAssignee (type, v) {
        try {
            await api.updateAssignee(conversation.value.data.uuid, type, v);
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

    // Action to update participants
    function updateParticipants (newParticipants) {
        conversation.value.participants = {
            ...conversation.value.participants,
            ...newParticipants
        };
    }


    // Websocket updates.
    function updateConversationList (msg) {
        const updatedConversation = conversations.value.data.find(c => c.uuid === msg.conversation_uuid);
        if (updatedConversation) {
            updatedConversation.last_message = msg.last_message;
            updatedConversation.last_message_at = msg.last_message_at;
            // Increase conversation unread msg count only if it's not open.
            if (updatedConversation.uuid !== conversation.value.data.uuid) {
                updatedConversation.unread_message_count += 1
            }
        }
    }
    function updateMessageList (msg) {
        // First check if this conversation is selected and then update messages list.
        if (conversation.value?.data?.uuid === msg.conversation_uuid) {
            // Fetch entire msg if the give msg does not exist in the msg list.
            if (!messages.value.data.some(message => message.uuid === msg.uuid)) {
                fetchParticipants(msg.conversation_uuid)
                fetchMessage(msg.uuid)
                updateAssigneeLastSeen(msg.conversation_uuid)
            }
        }
    }
    function updateMessageStatus (uuid, status) {
        const message = messages.value.data.find(m => m.uuid === uuid);
        if (message) {
            message.status = status
        }
    }

    function $reset () {
        conversations.value = { data: null, loading: false, errorMessage: "" };
        conversation.value = { data: null, loading: false, errorMessage: "" };
        messages.value = { data: null, loading: false, errorMessage: "" };
    }

    return { conversations, conversation, messages, sortedConversations, sortedMessages, updateMessageStatus, updateAssigneeLastSeen, updateMessageList, fetchConversation, fetchConversations, fetchMessages, upsertTags, updateAssignee, updatePriority, updateStatus, updateConversationList, $reset };
})
