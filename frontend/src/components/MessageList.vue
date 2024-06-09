<template>
    <div>
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
import ContactMessageBubble from "./ContactMessageBubble.vue"
import ActivityMessageBubble from "./ActivityMessageBubble.vue"
import AgentMessageBubble from "./AgentMessageBubble.vue"

defineProps({
    messages: Array,
})

const isPrivateNote = (message) => {
    return message.type === "outgoing" && message.private
}
</script>
