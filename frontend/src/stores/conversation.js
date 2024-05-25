import { defineStore } from 'pinia'
import { ref } from "vue"
import { handleHTTPError } from '@/utils/http'
import api from '@/api';

export const useConversationStore = defineStore('conversation', () => {
    const conversations = ref({
        data: null,
        loading: false,
        errorMessage: ""
    })
    const conversation = ref({
        data: null,
        loading: false,
        errorMessage: ""
    })
    const messages = ref({
        data: null,
        loading: false,
        errorMessage: ""
    })

    async function fetchConversation (uuid) {
        conversation.value.loading = true;
        try {
            const resp = await api.getConversation(uuid);
            conversation.value.data = resp.data.data
        } catch (error) {
            conversation.value.errorMessage = handleHTTPError(error).message;
        } finally {
            conversation.value.loading = false;
        }
    }

    async function fetchMessages (uuid) {
        messages.value.loading = true;
        try {
            const resp = await api.getMessages(uuid);
            messages.value.data = resp.data.data
        } catch (error) {
            messages.value.errorMessage = handleHTTPError(error).message;
        } finally {
            messages.value.loading = false;
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
            fetchConversation(conversation.value.data.uuid)
        } catch (error) {
            // Pass.
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

    function $reset () {
        conversations.value = { data: null, loading: false, errorMessage: "" };
        conversation.value = { data: null, loading: false, errorMessage: "" };
        messages.value = { data: null, loading: false, errorMessage: "" };
    }

    return { conversations, conversation, messages, fetchConversation, fetchConversations, fetchMessages, upsertTags, updateAssignee, updatePriority, updateStatus, $reset };
})
