<template>
  <div class="relative" v-if="conversationStore.messages.data">

    <!-- Header -->
    <div class="px-4 border-b h-[47px] flex items-center justify-between shadow shadow-gray-100">
      <div class="flex items-center space-x-3 text-sm">
        <div class="font-bold">
          {{ conversationStore.current.subject }}
        </div>
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <Badge :variant="getBadgeVariant">
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
    <!-- Header end -->

    <!-- Messages & reply box -->
    <div class="flex flex-col h-screen">
      <MessageList class="flex-1" />
      <ReplyBox class="h-max mb-12" />
    </div>
    <!-- Messages & reply box end -->

  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
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

const getBadgeVariant = computed(() => {
  return conversationStore.current?.status == 'Spam' ? 'destructive' : 'primary'
})

const handleUpdateStatus = (status) => {
  conversationStore.updateStatus(status)
}
</script>
