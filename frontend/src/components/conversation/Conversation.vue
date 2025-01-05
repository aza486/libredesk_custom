<template>
  <div class="relative" v-if="conversationStore.messages.data">

    <!-- Header -->
    <div class="px-4 border-b h-[44px] flex items-center justify-between">
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
import { Button } from '@/components/ui/button'
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
