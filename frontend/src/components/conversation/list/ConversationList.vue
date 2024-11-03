<template>
  <div class="h-screen flex flex-col">
    <!-- Filters -->
    <div class="shrink-0">
      <ConversationListFilters @updateFilters="handleUpdateFilters" />
    </div>

    <!-- Empty list -->
    <EmptyList class="px-4" v-if="!hasConversations && !hasErrored && !isLoading" title="No conversations found"
      message="Try adjusting filters." :icon="MessageCircleQuestion"></EmptyList>


    <!-- List -->
    <div class="flex-grow overflow-y-auto">
      <EmptyList class="px-4" v-if="conversationStore.conversations.errorMessage" title="Could not fetch conversations"
        :message="conversationStore.conversations.errorMessage" :icon="MessageCircleWarning"></EmptyList>

      <!-- Items -->
      <div v-else>
        <ConversationListItem :conversation="conversation" :currentConversation="conversationStore.current"
          v-for="conversation in conversationStore.sortedConversations" :key="conversation.uuid"
          :contactFullName="conversationStore.getContactFullName(conversation.uuid)" />
      </div>

      <!-- List skeleton -->
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
import { MessageCircleQuestion, MessageCircleWarning } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import EmptyList from '@/components/conversation/list/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/components/conversation/list/ConversationListItemSkeleton.vue'
import ConversationListFilters from '@/components/conversation/list/ConversationListFilters.vue'

const conversationStore = useConversationStore()
let listRefreshInterval = null

onMounted(() => {
  conversationStore.fetchConversationsList()
  // Refresh list every min.
  listRefreshInterval = setInterval(() => {
    conversationStore.fetchConversationsList(false)
  }, 60000)
})

onUnmounted(() => {
  clearInterval(listRefreshInterval)
  conversationStore.clearListReRenderInterval()
})

const loadNextPage = () => {
  conversationStore.fetchNextConversations()
}

const handleUpdateFilters = (filters) => {
  console.log("setting ", filters)
  conversationStore.setConversationListFilters(filters)
}

const hasConversations = computed(() => {
  return conversationStore.sortedConversations.length !== 0
})

const hasErrored = computed(() => {
  return conversationStore.conversations.errorMessage ? true : false
})

const isLoading = computed(() => {
  return conversationStore.conversations.loading
})
</script>
