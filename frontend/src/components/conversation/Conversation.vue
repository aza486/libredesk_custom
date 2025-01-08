<template>
  <div v-if="conversationStore.messages.data">
    <!-- Header -->
    <div class="p-3 border-b flex items-center justify-between">
      <div class="flex items-center space-x-3 text-sm">
        <div class="font-medium">
          {{ conversationStore.current.subject }}
        </div>
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <div class="flex items-center space-x-1 cursor-pointer bg-primary px-2 py-1 rounded-md text-sm">
              <GalleryVerticalEnd size="14" class="text-secondary" />
              <span class="text-secondary font-medium">{{ conversationStore.current.status }}</span>
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem v-for="status in conversationStore.statusesForSelect" :key="status.value"
              @click="handleUpdateStatus(status.label)">
              {{ status.label }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Messages & reply box -->
    <div class="flex flex-col h-[calc(100vh-theme(spacing.10))]">
      <MessageList class="flex-1 overflow-y-auto" />
      <div class="sticky bottom-0 bg-white">
        <ReplyBox class="h-max" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { useConversationStore } from '@/stores/conversation'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import {
  GalleryVerticalEnd,
} from 'lucide-vue-next'
import MessageList from '@/components/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { useEmitter } from '@/composables/useEmitter'

const conversationStore = useConversationStore()
const emitter = useEmitter()

const handleUpdateStatus = (status) => {
  if (status === CONVERSATION_DEFAULT_STATUSES.SNOOZED) {
    emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, 'snooze')
    return
  }
  conversationStore.updateStatus(status)
}
</script>
