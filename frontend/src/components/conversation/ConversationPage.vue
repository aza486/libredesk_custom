<template>
  <div class="relative" v-if="conversationStore.messages.data">
    <!-- Header -->
    <div class="px-4 border-b h-[47px] flex items-center justify-between">
      <div class="flex items-center space-x-3 text-sm">
        <div class="font-bold">
          {{ conversationStore.getContactFullName(conversationStore.conversation.data.uuid) }}
        </div>
        <Tooltip>
          <TooltipTrigger>
            <Badge :variant="getBadgeVariant">
              {{ conversationStore.conversation.data.status }}
            </Badge>
          </TooltipTrigger>
          <TooltipContent>
            <p>Status</p>
          </TooltipContent>
        </Tooltip>
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <Icon icon="lucide:ellipsis-vertical" class="mt-2 size-6"></Icon>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              v-for="status in statuses"
              :key="status.name"
              @click="handleUpdateStatus(status.name)"
            >
              {{ status.name }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
    <Error class="sticky" :error-message="conversationStore.messages.errorMessage"></Error>
    <div class="flex flex-col h-screen">
      <!-- flex-1-->
      <MessageList class="flex-1" />
      <ReplyBox class="h-max mb-12" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { Error } from '@/components/ui/error'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import MessageList from '@/components/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import api from '@/api'
import { Icon } from '@iconify/vue'

const conversationStore = useConversationStore()
const statuses = ref([])

onMounted(() => {
  getStatuses()
})

const getStatuses = async () => {
  const resp = await api.getStatuses()
  statuses.value = resp.data.data
}

const getBadgeVariant = computed(() => {
  return conversationStore.conversation.data?.status == 'Spam' ? 'destructive' : 'primary'
})

const handleUpdateStatus = (status) => {
  conversationStore.updateStatus(status)
}
</script>
