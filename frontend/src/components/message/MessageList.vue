<template>
    <div ref="threadEl" class="overflow-y-scroll">
        <div class="text-center mt-3" v-if="conversationStore.messages.hasMore">
            <Button variant="secondary" @click="conversationStore.fetchNextMessages">Fetch more</Button>
        </div>
        <div v-for="message in conversationStore.sortedMessages" :key="message.uuid" :class="message.type === 'activity' ? 'm-4' : 'm-6'">
            <div v-if="!message.private">
                <ContactMessageBubble :message="message" v-if="message.type === 'incoming'" />
                <AgentMessageBubble :message="message" v-if="message.type === 'outgoing'" />
            </div>
            <div v-else-if="isPrivateNote(message)">
                <AgentMessageBubble :message="message" v-if="message.type === 'outgoing'" />
            </div>
            <div v-else-if="message.type === 'activity'">
                <ActivityMessageBubble :message="message" />
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import ContactMessageBubble from "./ContactMessageBubble.vue";
import ActivityMessageBubble from "./ActivityMessageBubble.vue";
import AgentMessageBubble from "./AgentMessageBubble.vue";
import { useConversationStore } from '@/stores/conversation'
import { Button } from '@/components/ui/button';

const conversationStore = useConversationStore()
const threadEl = ref(null);


const scrollToBottom = () => {
    const thread = threadEl.value;
    if (thread) {
        thread.scrollTop = thread.scrollHeight;
    }
};

onMounted(() => {
    setTimeout(() => {
        scrollToBottom();
    }, 0);
});


const isPrivateNote = (message) => {
    return message.type === "outgoing" && message.private;
};
</script>
