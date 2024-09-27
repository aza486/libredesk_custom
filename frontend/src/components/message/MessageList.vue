<template>
  <div ref="threadEl" class="overflow-y-scroll">
    <div class="text-center mt-3" v-if="conversationStore.messages.hasMore && !conversationStore.messages.loading">
      <Button variant="ghost" @click="conversationStore.fetchNextMessages">
        <RefreshCw size="17" class="mr-2" />
        Load more
      </Button>
    </div>
    <div v-for="message in conversationStore.sortedMessages" :key="message.uuid"
      :class="message.type === 'activity' ? 'm-4' : 'm-6'">
      <div v-if=conversationStore.messages.loading>
        <MessagesSkeleton></MessagesSkeleton>
      </div>
      <div v-else>
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
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import ContactMessageBubble from './ContactMessageBubble.vue'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import AgentMessageBubble from './AgentMessageBubble.vue'
import MessagesSkeleton from './MessagesSkeleton.vue'
import { useConversationStore } from '@/stores/conversation'
import { Button } from '@/components/ui/button'
import { RefreshCw } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'

const conversationStore = useConversationStore()
const threadEl = ref(null)
const emitter = useEmitter()

const scrollToBottom = () => {
  setTimeout(() => {
    const thread = threadEl.value
    if (thread) {
      thread.scrollTop = thread.scrollHeight
    }
  }, 0)
}

onMounted(() => {
  scrollToBottom()
  // On new outgoing message to the current conversation, scroll to the bottom.
  emitter.on('new-outgoing-message', (data) => {
    if (data.conversation_uuid === conversationStore.conversation.data.uuid) {
      scrollToBottom()
    }
  })
})

// On conversation change scroll to the bottom
watch(
  () => conversationStore.conversation.data,
  () => {
    scrollToBottom()
  }
)

const isPrivateNote = (message) => {
  return message.type === 'outgoing' && message.private
}
</script>
