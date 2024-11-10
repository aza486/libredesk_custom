<template>
  <div ref="threadEl" class="overflow-y-scroll relative h-full" @scroll="handleScroll">
    <div class="min-h-full relative pb-20">
      <div class="text-center mt-3" v-if="conversationStore.messages.hasMore && !conversationStore.messages.loading">
        <Button variant="ghost" @click="conversationStore.fetchNextMessages">
          <RefreshCw size="17" class="mr-2" />
          Load more
        </Button>
      </div>
      <div v-for="message in conversationStore.sortedMessages" :key="message.uuid"
        :class="message.type === 'activity' ? 'm-4' : 'm-6'">
        <div v-if="conversationStore.messages.loading">
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

    <!-- Sticky container for the scroll button -->
    <div v-show="!isAtBottom" class="sticky bottom-6 flex justify-end px-6">
      <div class="relative">
        <button @click="handleScrollToBottom" class="w-11 h-11 rounded-full flex items-center justify-center shadow">
          <ArrowDown size="20" />
        </button>
        <span v-if="unReadMessages > 0"
          class="absolute -top-1 -right-1 min-w-[20px] h-5 px-1.5 rounded-full bg-primary text-white text-xs font-medium flex items-center justify-center">
          {{ unReadMessages }}
        </span>
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
import { useUserStore } from '@/stores/user'
import { Button } from '@/components/ui/button'
import { RefreshCw, ArrowDown } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'

const conversationStore = useConversationStore()
const userStore = useUserStore()
const threadEl = ref(null)
const emitter = useEmitter()
const isAtBottom = ref(true)
const unReadMessages = ref(0)

const checkIfAtBottom = () => {
  const thread = threadEl.value
  if (thread) {
    const tolerance = 100
    const isBottom = thread.scrollHeight - thread.scrollTop - thread.clientHeight <= tolerance
    isAtBottom.value = isBottom
  }
}

const handleScroll = () => {
  checkIfAtBottom()
}

const handleScrollToBottom = () => {
  unReadMessages.value = 0
  scrollToBottom()
}

const scrollToBottom = () => {
  setTimeout(() => {
    console.log('scrolling..')
    const thread = threadEl.value
    if (thread) {
      thread.scrollTop = thread.scrollHeight
      checkIfAtBottom()
    }
  }, 50)
}

onMounted(() => {
  scrollToBottom()
  checkIfAtBottom()
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, (data) => {
    if (data.conversation_uuid === conversationStore.current.uuid) {
      if (data.message.sender_id === userStore.userID) {
        scrollToBottom()
      } else if (!isAtBottom.value) {
        unReadMessages.value++
      }
    }
  })
})

// On conversation change scroll to the bottom
watch(
  () => conversationStore.current.uuid,
  () => {
    unReadMessages.value = 0
    scrollToBottom()
  }
)

const isPrivateNote = (message) => {
  return message.type === 'outgoing' && message.private
}
</script>