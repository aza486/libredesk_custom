<template>
  <div class="relative" v-if="conversationStore.messages.data">

    <!-- Header -->
    <div class="px-4 border-b h-[47px] flex items-center justify-between shadow shadow-gray-100">
      <div class="flex items-center space-x-3 text-sm">
        <div class="font-semibold">
          {{ conversationStore.current.subject }}
        </div>
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <Badge variant="primary">
              {{ conversationStore.current.status }}
            </Badge>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem v-for="status in statuses" :key="status.name" @click="handleUpdateStatus(status.name)">
              {{ status.name }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Messages & reply box -->
    <div class="flex flex-col h-screen" v-auto-animate>
      <MessageList class="flex-1" />
      <ReplyBox class="h-max mb-12" />
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { useConversationStore } from '@/stores/conversation'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import MessageList from '@/components/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import api from '@/api'

const conversationStore = useConversationStore()
const statuses = ref([])

onMounted(() => {
  getStatuses()
})

const getStatuses = async () => {
  const resp = await api.getStatuses()
  statuses.value = resp.data.data
}

const handleUpdateStatus = (status) => {
  conversationStore.updateStatus(status)
}
</script>
