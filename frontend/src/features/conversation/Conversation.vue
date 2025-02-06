<template>
  <div class="flex flex-col w-full">
    <div>
      <!-- Header -->
      <div class="p-2 border-b flex items-center justify-between">
        <div class="flex items-center space-x-3 pr-5">
          {{ conversationStore.currentContactName }}
        </div>
        <div class="flex items-center space-x-2">
          <div>
            <DropdownMenu>
              <DropdownMenuTrigger>
                <div
                  class="flex items-center space-x-1 cursor-pointer bg-primary px-2 py-1 rounded-md text-sm"
                >
                  <span
                    class="text-secondary font-medium inline-block"
                    v-if="conversationStore.current?.status"
                  >
                    {{ conversationStore.current?.status }}
                  </span>
                  <span v-else class="text-secondary font-medium inline-block"> Loading... </span>
                </div>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem
                  v-for="status in conversationStore.statusOptions"
                  :key="status.value"
                  @click="handleUpdateStatus(status.label)"
                >
                  {{ status.label }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </div>

    <!-- Messages & reply box -->
    <div>
      <div class="flex flex-col h-[calc(100vh-theme(spacing.10))]">
        <MessageList class="flex-1 overflow-y-auto" />
        <div class="sticky bottom-0 bg-white">
          <ReplyBox class="h-max" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import MessageList from '@/features/conversation/message/MessageList.vue'
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
