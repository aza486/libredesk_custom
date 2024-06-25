<template>
    <!-- <div v-if="cannedResponsesStore.responses.length === 0" class="w-full drop-shadow-sm overflow-hidden p-2 border-t">
        <ul class="space-y-2 max-h-96">
            <li v-for="(response, index) in filteredCannedResponses" :key="response.id"
                @click="selectResponse(response.content)" class="cursor-pointer rounded p-1 hover:bg-secondary"
                :class="{ 'bg-secondary': cannedResponseIndex === index }" :ref="el => cannedResponseRefItems.push(el)">
                <span class="font-semibold">{{ response.title }}</span> - {{ response.content }}
            </li>
        </ul>
    </div> -->
    <TextEditor @send="sendMessage" :conversationuuid="conversationStore.conversation.data.uuid" class="mb-[40px]" />
</template>

<script setup>
import { onMounted } from "vue";
import api from '@/api';

import TextEditor from './TextEditor.vue'
import { useConversationStore } from '@/stores/conversation'
import { useCannedResponses } from '@/stores/canned_responses'

const conversationStore = useConversationStore()
const cannedResponsesStore = useCannedResponses()

onMounted(() => {
    cannedResponsesStore.fetchAll()
})

const sendMessage = (message) => {
    api.sendMessage(conversationStore.conversation.data.uuid, {
        private: message.private,
        message: message.html,
        attachments: JSON.stringify(message.attachments),
    })
    api.updateAssigneeLastSeen(conversationStore.conversation.data.uuid)
}

</script>