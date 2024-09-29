<template>
  <div class="h-screen">

    <!-- Filters -->
    <ConversationListFilters v-model:type="conversationType" v-model:filter="conversationFilter" :handleFilterChange="handleFilterChange">
    </ConversationListFilters>

    <!-- Error / Empty list -->
    <EmptyList v-if="emptyConversations" title="No conversation found" message="Try adjusting filters."
      :icon="MessageCircleQuestion"></EmptyList>
    <EmptyList v-if="conversationStore.conversations.errorMessage" title="Could not fetch conversations"
      :message="conversationStore.conversations.errorMessage" :icon="MessageCircleWarning"></EmptyList>

    <div class="h-screen overflow-y-scroll pb-[180px] flex flex-col">
      <!-- List skeleton -->
      <div v-if="conversationsLoading">
        <ConversationListItemSkeleton v-for="index in 8" :key="index"></ConversationListItemSkeleton>
      </div>

      <!-- Item -->
      <ConversationListItem />

      <!-- Load more  -->
      <div class="flex justify-center items-center mt-5 relative">
        <div v-if="conversationStore.conversations.hasMore && !hasErrored && hasConversations">
          <Button variant="link" @click="loadNextPage">
            <Spinner v-if="conversationStore.conversations.loading" />
            <p v-else>Load more...</p>
          </Button>
        </div>
        <div v-else-if="everythingLoaded">All conversations loaded!</div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { onMounted, ref, watch, computed, onUnmounted } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { subscribeConversationsList } from '@/websocket.js'
import { CONVERSATION_LIST_TYPE, CONVERSATION_FILTERS } from '@/constants/conversation'
import { MessageCircleWarning, MessageCircleQuestion } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import EmptyList from '@/components/conversation/list/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/components/conversation/list/ConversationListItemSkeleton.vue'
import ConversationListFilters from '@/components/conversation/list/ConversationListFilters.vue'

const conversationStore = useConversationStore()
const conversationFilter = ref(CONVERSATION_FILTERS.ALL)
const conversationType = ref(CONVERSATION_LIST_TYPE.ASSIGNED)
let listRefreshInterval = null

onMounted(() => {
  conversationStore.fetchConversations(conversationType.value, conversationFilter.value)
  subscribeConversationsList(conversationType.value, conversationFilter.value)
  // Refresh list every min.
  listRefreshInterval = setInterval(() => {
    conversationStore.fetchConversations(conversationType.value, conversationFilter.value)
  }, 60000)
})

onUnmounted(() => {
  clearInterval(listRefreshInterval)
})

watch(conversationType, (newType) => {
  conversationStore.fetchConversations(newType, conversationFilter.value)
  subscribeConversationsList(newType, conversationFilter.value)
})

const handleFilterChange = (filter) => {
  conversationFilter.value = filter
  conversationStore.fetchConversations(conversationType.value, filter)
  subscribeConversationsList(conversationType.value, conversationFilter.value)
}

const loadNextPage = () => {
  conversationStore.fetchNextConversations(conversationType.value, conversationFilter.value)
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

const everythingLoaded = computed(() => {
  return !conversationStore.conversations.errorMessage && !emptyConversations.value
})

const conversationsLoading = computed(() => {
  return conversationStore.conversations.loading
})
</script>
