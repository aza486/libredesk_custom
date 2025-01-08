<template>
  <div class="h-screen flex flex-col">
    <div class="flex justify-start items-center p-3 w-full space-x-4 border-b">
      <SidebarTrigger class="cursor-pointer w-5 h-5" />
      <span class="text-xl font-semibold">{{title}}</span>
    </div>

    <div class="flex justify-between px-2 py-2 w-full">
      <DropdownMenu>
        <DropdownMenuTrigger class="cursor-pointer">
          <Button variant="ghost">
            {{ conversationStore.getListStatus }}
            <ChevronDown class="w-4 h-4 ml-2" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem v-for="status in conversationStore.statusesForSelect" :key="status.value"
            @click="handleStatusChange(status)">
            {{ status.label }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <DropdownMenu>
        <DropdownMenuTrigger class="cursor-pointer">
          <Button variant="ghost">
            {{ conversationStore.getListSortField }}
            <ChevronDown class="w-4 h-4 ml-2" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="handleSortChange('oldest')">Oldest</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">Newest</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')">Started first</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_last')">Started last</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('waiting_longest')">Waiting longest</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('next_sla_target')">Next SLA target</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('priority_first')">Priority first</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Empty -->
    <EmptyList class="px-4" v-if="!hasConversations && !hasErrored && !isLoading" title="No conversations found"
      message="Try adjusting filters." :icon="MessageCircleQuestion"></EmptyList>

    <!-- List -->
    <div class="flex-grow overflow-y-auto">
      <EmptyList class="px-4" v-if="conversationStore.conversations.errorMessage" title="Could not fetch conversations"
        :message="conversationStore.conversations.errorMessage" :icon="MessageCircleWarning"></EmptyList>

      <!-- Items -->
      <div v-else>
        <div class="space-y-5 px-2">
          <ConversationListItem class="mt-2" :conversation="conversation"
            :currentConversation="conversationStore.current" v-for="conversation in conversationStore.conversationsList"
            :key="conversation.uuid" :contactFullName="conversationStore.getContactFullName(conversation.uuid)" />
        </div>
      </div>

      <!-- skeleton -->
      <div v-if="isLoading">
        <ConversationListItemSkeleton v-for="index in 10" :key="index" />
      </div>

      <!-- Load more -->
      <div class="flex justify-center items-center p-5 relative" v-if="!hasErrored">
        <div v-if="conversationStore.conversations.hasMore">
          <Button variant="link" @click="loadNextPage">
            <p v-if="!isLoading">Load more</p>
          </Button>
        </div>
        <div v-else-if="!conversationStore.conversations.hasMore && hasConversations">
          All conversations loaded!
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { onMounted, computed, onUnmounted } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { MessageCircleQuestion, MessageCircleWarning, ChevronDown } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { SidebarTrigger } from '@/components/ui/sidebar'
import EmptyList from '@/components/conversation/list/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversation/list/ConversationListItem.vue'
import { useRoute } from 'vue-router'
import ConversationListItemSkeleton from '@/components/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
let reFetchInterval = null

// Re-fetch conversations list every 30 seconds for any missed updates.
// FIXME: Figure out a better way to handle this.
onMounted(() => {
  reFetchInterval = setInterval(() => {
    conversationStore.reFetchConversationsList(false)
  }, 30000)
})

const route = useRoute()
const title = computed(() => route.meta.title || '')

onUnmounted(() => {
  clearInterval(reFetchInterval)
  conversationStore.clearListReRenderInterval()
})

const handleStatusChange = (status) => {
  conversationStore.setListStatus(status.label)
}

const handleSortChange = (order) => {
  conversationStore.setListSortField(order)
}

const loadNextPage = () => {
  conversationStore.fetchNextConversations()
}

const hasConversations = computed(() => {
  return conversationStore.conversationsList.length !== 0
})

const hasErrored = computed(() => {
  return conversationStore.conversations.errorMessage ? true : false
})

const isLoading = computed(() => {
  return conversationStore.conversations.loading
})
</script>
