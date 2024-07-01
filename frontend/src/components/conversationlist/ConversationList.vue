<template>
    <div class="h-screen">
        <div class="px-3 pb-2 border-b-2 rounded-b-lg shadow-md">
            <div class="flex justify-between mt-3">
                <h3 class="scroll-m-20 text-2xl font-medium flex gap-x-2">
                    Conversations
                </h3>
            </div>

            <!-- Search -->
            <!-- <div class="relative mx-auto my-3">
                <Input id="search" type="text" placeholder="Search message or reference number"
                    class="pl-10 bg-[#F0F2F5]" />
                <span class="absolute start-1 inset-y-0 flex items-center justify-center px-2">
                    <Search class="size-6 text-muted-foreground" />
                </span>
            </div> -->

            <div class="flex justify-between mt-5">
                <Tabs v-model:model-value="conversationType">
                    <TabsList class="w-full flex justify-evenly">
                        <TabsTrigger value="assigned" class="w-full">
                            Assigned
                        </TabsTrigger>
                        <TabsTrigger value="unassigned" class="w-full">
                            Unassigned
                        </TabsTrigger>
                        <TabsTrigger value="all" class="w-full">
                            All
                        </TabsTrigger>
                    </TabsList>
                </Tabs>
                <div class="space-x-2">
                    <div class="w-[8rem]">
                        <Select @update:modelValue="handleFilterChange" v-model="predefinedFilter">
                            <SelectTrigger>
                                <SelectValue placeholder="Select a filter" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectGroup>
                                    <!-- <SelectLabel>Status</SelectLabel> -->
                                    <SelectItem value="status_all">
                                        All
                                    </SelectItem>
                                    <SelectItem value="status_open">
                                        Open
                                    </SelectItem>
                                    <SelectItem value="status_processing">
                                        Processing
                                    </SelectItem>
                                    <SelectItem value="status_spam">
                                        Spam
                                    </SelectItem>
                                    <SelectItem value="status_resolved">
                                        Resolved
                                    </SelectItem>
                                </SelectGroup>
                            </SelectContent>
                        </Select>
                    </div>
                </div>
            </div>
        </div>

        <Error :errorMessage="conversationStore.conversations.errorMessage"></Error>
        <EmptyList v-if="emptyConversations"></EmptyList>

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
                <div v-else-if="everythingLoaded">
                    All conversations loaded 😎
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { onMounted, ref, watch, computed, onUnmounted } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { subscribeConversations } from "@/websocket.js"
import { CONVERSATION_LIST_TYPE, CONVERSATION_PRE_DEFINED_FILTERS } from '@/constants/conversation'

import { Error } from '@/components/ui/error'
import { Skeleton } from '@/components/ui/skeleton'
import {
    Tabs,
    TabsList,
    TabsTrigger,
} from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import EmptyList from '@/components/conversationlist/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversationlist/ConversationListItem.vue'


const conversationStore = useConversationStore()
const predefinedFilter = ref(CONVERSATION_PRE_DEFINED_FILTERS.ALL)
const conversationType = ref(CONVERSATION_LIST_TYPE.ASSIGNED)
let listRefreshInterval = null

onMounted(() => {
    conversationStore.fetchConversations(conversationType.value, predefinedFilter.value)
    subscribeConversations(conversationType.value, predefinedFilter.value)
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
    subscribeConversations(newType, predefinedFilter.value)
});

const handleFilterChange = (filter) => {
    predefinedFilter.value = filter
    conversationStore.fetchConversations(conversationType.value, filter)
    subscribeConversations(conversationType.value, predefinedFilter.value)
}

const loadNextPage = () => {
    conversationStore.fetchNextConversations(conversationType.value, predefinedFilter.value)
};

const hasConversations = computed(() => {
    return conversationStore.sortedConversations.length !== 0 && !conversationStore.conversations.errorMessage && !conversationStore.conversations.loading
})

const emptyConversations = computed(() => {
    return conversationStore.sortedConversations.length === 0 && !conversationStore.conversations.errorMessage && !conversationStore.conversations.loading
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
