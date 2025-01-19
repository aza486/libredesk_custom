<template>
  <div class="h-screen flex flex-col">
    <!-- Header -->
    <header class="border-b">
      <div class="flex items-center space-x-4 p-2">
        <SidebarTrigger class="h-4 w-4" />
        <span class="text-xl font-semibold text-gray-800">{{ title }}</span>
      </div>
    </header>

    <!-- Filters -->
    <div class="bg-white px-4 py-3 flex justify-between items-center">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            <div>
              <span class="mr-1">{{ conversationStore.conversations.total }}</span>
              <span>{{ conversationStore.getListStatus }}</span>
            </div>
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            v-for="status in conversationStore.statusOptions"
            :key="status.value"
            @click="handleStatusChange(status)"
          >
            {{ status.label }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            {{ conversationStore.getListSortField }}
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="handleSortChange('oldest')">Oldest</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">Newest</DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')"
            >Started first</DropdownMenuItem
          >
          <DropdownMenuItem @click="handleSortChange('started_last')"
            >Started last</DropdownMenuItem
          >
          <DropdownMenuItem @click="handleSortChange('waiting_longest')"
            >Waiting longest</DropdownMenuItem
          >
          <DropdownMenuItem @click="handleSortChange('next_sla_target')"
            >Next SLA target</DropdownMenuItem
          >
          <DropdownMenuItem @click="handleSortChange('priority_first')"
            >Priority first</DropdownMenuItem
          >
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Content -->
    <div class="flex-grow overflow-y-auto">
      <EmptyList
        v-if="!hasConversations && !hasErrored && !isLoading"
        key="empty"
        class="px-4 py-8"
        title="No conversations found"
        message="Try adjusting your filters"
        :icon="MessageCircleQuestion"
      />

      <!-- Empty State -->
      <TransitionGroup
        enter-active-class="transition-all duration-300 ease-in-out"
        enter-from-class="opacity-0 transform translate-y-4"
        enter-to-class="opacity-100 transform translate-y-0"
        leave-active-class="transition-all duration-300 ease-in-out"
        leave-from-class="opacity-100 transform translate-y-0"
        leave-to-class="opacity-0 transform translate-y-4"
      >
        <!-- Error State -->
        <EmptyList
          v-if="conversationStore.conversations.errorMessage"
          key="error"
          class="px-4 py-8"
          title="Could not fetch conversations"
          :message="conversationStore.conversations.errorMessage"
          :icon="MessageCircleWarning"
        />

        <!-- Conversation List -->
        <div v-else key="list" class="divide-y divide-gray-200">
          <ConversationListItem
            v-for="conversation in conversationStore.conversationsList"
            :key="conversation.uuid"
            :conversation="conversation"
            :currentConversation="conversationStore.current"
            :contactFullName="conversationStore.getContactFullName(conversation.uuid)"
            class="transition-colors duration-200 hover:bg-gray-50"
          />
        </div>

        <!-- Loading Skeleton -->
        <div v-if="isLoading" key="loading" class="space-y-4 p-4">
          <ConversationListItemSkeleton v-for="index in 10" :key="index" />
        </div>
      </TransitionGroup>

      <!-- Load More -->
      <div
        v-if="!hasErrored && (conversationStore.conversations.hasMore || hasConversations)"
        class="flex justify-center items-center p-5"
      >
        <Button
          v-if="conversationStore.conversations.hasMore"
          variant="outline"
          @click="loadNextPage"
          :disabled="isLoading"
          class="transition-all duration-200 ease-in-out transform hover:scale-105"
        >
          <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
          {{ isLoading ? 'Loading...' : 'Load more' }}
        </Button>
        <p v-else class="text-sm text-gray-500">All conversations loaded</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onUnmounted, ref } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { MessageCircleQuestion, MessageCircleWarning, ChevronDown, Loader2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { SidebarTrigger } from '@/components/ui/sidebar'
import EmptyList from '@/components/conversation/list/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversation/list/ConversationListItem.vue'
import { useRoute } from 'vue-router'
import ConversationListItemSkeleton from '@/components/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const route = useRoute()
let reFetchInterval = ref(null)

const title = computed(() => {
  const typeValue = route.meta?.type?.(route)
  return (
    (typeValue || route.meta?.title || '').charAt(0).toUpperCase() +
    (typeValue || route.meta?.title || '').slice(1)
  )
})

onUnmounted(() => {
  clearInterval(reFetchInterval.value)
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

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const isLoading = computed(() => conversationStore.conversations.loading)
</script>
