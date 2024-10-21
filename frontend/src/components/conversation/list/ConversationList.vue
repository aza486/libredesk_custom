<template>
  <div class="h-screen flex flex-col">
    <!-- Filters -->
    <div class="shrink-0">
      <ConversationListFilters @updateFilters="handleUpdateFilters" />
    </div>

    <!-- Empty list -->
    <EmptyList v-if="emptyConversations" title="No conversations found" message="Try adjusting filters."
      :icon="MessageCircleQuestion"></EmptyList>

    <!-- List -->
    <div class="flex-grow overflow-y-auto">
      <!-- Items -->
      <div>
        <ConversationListItem :conversation="conversation" :currentConversation="conversationStore.current"
          v-for="conversation in conversationStore.sortedConversations" :key="conversation.uuid" :contactFullName="conversationStore.getContactFullName(conversation.uuid)" />
      </div>

      <!-- List skeleton -->
      <div v-if="conversationsLoading">
        <ConversationListItemSkeleton v-for="index in 10" :key="index" />
      </div>

      <!-- Load more -->
      <div class="flex justify-center items-center p-5">
        <div v-if="conversationStore.conversations.hasMore && !hasErrored && hasConversations">
          <Button variant="link" @click="loadNextPage">
            <Spinner v-if="conversationStore.conversations.loading" />
            <p v-else>Load more</p>
          </Button>
        </div>
        <div v-else-if="!conversationStore.conversations.hasMore && conversationStore.sortedConversations.length > 0">All conversations loaded!</div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { onMounted, computed, onUnmounted } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { MessageCircleQuestion } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import Spinner from '@/components/ui/spinner/Spinner.vue'
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
  conversationStore.setConversationListFilters(filters)
}

const hasConversations = computed(() => {
  return (
    conversationStore.sortedConversations.length !== 0 &&
    !conversationStore.conversations.errorMessage &&
    !conversationStore.conversations.loading
  )
})

const emptyConversations = computed(() => {
  return (
    conversationStore.sortedConversations.length === 0 &&
    !conversationStore.conversations.errorMessage &&
    !conversationStore.conversations.loading
  )
})

const hasErrored = computed(() => {
  return conversationStore.conversations.errorMessage ? true : false
})

const conversationsLoading = computed(() => {
  return conversationStore.conversations.loading
})
</script>
