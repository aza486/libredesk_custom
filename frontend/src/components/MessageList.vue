<template>
    <div ref="threadEl">
        <div v-for="message in messages" :key="message.uuid" :class="message.type === 'activity' ? 'm-4' : 'm-6'">
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
import { ref, onMounted, nextTick, watch } from 'vue';

import ContactMessageBubble from "./ContactMessageBubble.vue"
import ActivityMessageBubble from "./ActivityMessageBubble.vue"
import AgentMessageBubble from "./AgentMessageBubble.vue"

const props = defineProps({
    messages: Array,
})
const threadEl = ref(null)

watch(() => props.messages, () => {
    scrollToBottom()
});

onMounted(() => {
    scrollToBottom()
})

const scrollToBottom = () => {
    nextTick(() => {
        if (threadEl.value) {
            console.log("SCROLLING !!!", threadEl.value.scrollHeight)
            threadEl.value.scrollTop = threadEl.value.scrollHeight;
        }
    })
};


const isPrivateNote = (message) => {
    return message.type === "outgoing" && message.private
}
</script>
