<template>
  <div class="h-screen">
    <div class="flex justify-between px-2 py-2 border-b">
      <Tabs v-model:model-value="conversationType">
        <TabsList class="w-full flex justify-evenly">
          <TabsTrigger value="assigned" class="w-full"> Assigned </TabsTrigger>
          <TabsTrigger value="unassigned" class="w-full"> Unassigned </TabsTrigger>
          <TabsTrigger value="all" class="w-full"> All </TabsTrigger>
        </TabsList>
      </Tabs>

      <Popover>
        <PopoverTrigger as-child>
          <div class="flex items-center mr-2">
            <ListFilter size="20" class="mx-auto cursor-pointer"></ListFilter>
          </div>
        </PopoverTrigger>
        <PopoverContent class="w-52">
          <div>
            <Select @update:modelValue="handleFilterChange" v-model="predefinedFilter">
              <SelectTrigger>
                <SelectValue placeholder="Select a filter" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <!-- <SelectLabel>Status</SelectLabel> -->
                  <SelectItem value="status_all"> All </SelectItem>
                  <SelectItem value="status_open"> Open </SelectItem>
                  <SelectItem value="status_processing"> Processing </SelectItem>
                  <SelectItem value="status_spam"> Spam </SelectItem>
                  <SelectItem value="status_resolved"> Resolved </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </PopoverContent>
      </Popover>
    </div>

    <EmptyList v-if="emptyConversations" title="No conversation found" message="Try adjusting filters."></EmptyList>
    <EmptyList v-if="conversationStore.conversations.errorMessage" title="Something went wrong" :message="conversationStore.conversations.errorMessage"></EmptyList>

    <div class="h-screen overflow-y-scroll pb-[180px] flex flex-col">
      <ConversationListItem />

      <div v-if="conversationsLoading">
        <div class="flex items-center gap-5 p-6 border-b" v-for="index in 8" :key="index">
          <Skeleton class="h-12 w-12 rounded-full" />
          <div class="space-y-2">
            <Skeleton class="h-4 w-[250px]" />
            <Skeleton class="h-4 w-[200px]" />
          </div>
        </div>
      </div>

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
import { CONVERSATION_LIST_TYPE, CONVERSATION_PRE_DEFINED_FILTERS } from '@/constants/conversation'

import { ListFilter } from 'lucide-vue-next'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import EmptyList from '@/components/conversationlist/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversationlist/ConversationListItem.vue'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

const conversationStore = useConversationStore()
const predefinedFilter = ref(CONVERSATION_PRE_DEFINED_FILTERS.ALL)
const conversationType = ref(CONVERSATION_LIST_TYPE.ASSIGNED)
let listRefreshInterval = null

onMounted(() => {
  conversationStore.fetchConversations(conversationType.value, predefinedFilter.value)
  subscribeConversationsList(conversationType.value, predefinedFilter.value)
  // Refesh list every 1 minute to sync any missed changes.
  listRefreshInterval = setInterval(() => {
    conversationStore.fetchConversations(conversationType.value, predefinedFilter.value)
  }, 60000)
})

onUnmounted(() => {
  clearInterval(listRefreshInterval)
})

watch(conversationType, (newType) => {
  conversationStore.fetchConversations(newType, predefinedFilter.value)
  subscribeConversationsList(newType, predefinedFilter.value)
})

const handleFilterChange = (filter) => {
  predefinedFilter.value = filter
  conversationStore.fetchConversations(conversationType.value, filter)
  subscribeConversationsList(conversationType.value, predefinedFilter.value)
}

const loadNextPage = () => {
  conversationStore.fetchNextConversations(conversationType.value, predefinedFilter.value)
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
