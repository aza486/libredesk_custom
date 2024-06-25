<template>
    <div class="h-screen">
        <div class="px-3 pb-2 border-b-2 rounded-b-lg shadow-md">
            <div class="flex justify-between mt-3">
                <h3 class="scroll-m-20 text-2xl font-semibold tracking-tight flex gap-x-2">
                    Conversations
                </h3>
                <div class="w-[8rem]">
                    <Select @update:modelValue="handleFilterChange" v-model="predefinedFilter">
                        <SelectTrigger>
                            <SelectValue placeholder="Select a filter" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectLabel>Status</SelectLabel>
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
                    <Popover>
                        <PopoverTrigger>
                            <button>
                                <ListFilter :size="30" class="p-2 bg-slate-100 rounded-sm" :stroke-width="1.9" />
                            </button>
                        </PopoverTrigger>
                        <PopoverContent class="flex flex-col gap-3 w-full">
                            Work in progress.
                        </PopoverContent>
                    </Popover>
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


            <div class="flex justify-center items-center mt-6 relative"
                v-if="conversationStore.conversations.hasMore && !hasErrored && hasConversations">
                <Button variant="link" @click="loadNextPage">
                    <Spinner v-if="conversationStore.conversations.loading" />
                    <p v-else>Load more...</p>
                </Button>
            </div>
        </div>
    </div>
</template>

<script setup>
import { onMounted, ref, watch, computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_LIST_TYPE, CONVERSATION_PRE_DEFINED_FILTERS } from '@/constants/conversation'

import { Error } from '@/components/ui/error'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Search, ListFilter } from 'lucide-vue-next'
import {
    Tabs,
    TabsList,
    TabsTrigger,
} from '@/components/ui/tabs'
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectLabel,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import EmptyList from '@/components/conversationlist/ConversationEmptyList.vue'
import ConversationListItem from '@/components/conversationlist/ConversationListItem.vue'


const conversationStore = useConversationStore()
const predefinedFilter = ref(CONVERSATION_PRE_DEFINED_FILTERS.STATUS_OPEN)
const conversationType = ref(CONVERSATION_LIST_TYPE.ASSIGNED)

onMounted(() => {
    conversationStore.fetchConversations(conversationType.value, predefinedFilter.value)
})

watch(conversationType, (newType) => {
    conversationStore.fetchConversations(newType, predefinedFilter.value)
});

const hasConversations = computed(() => {
    return conversationStore.sortedConversations.length !== 0 && !conversationStore.conversations.errorMessage && !conversationStore.conversations.loading
})

const emptyConversations = computed(() => {
    return conversationStore.sortedConversations.length === 0 && !conversationStore.conversations.errorMessage && !conversationStore.conversations.loading
})

const hasErrored = computed(() => {
    return conversationStore.conversations.errorMessage ? true : false
})

const conversationsLoading = computed(() => {
    return conversationStore.conversations.loading
})

const handleFilterChange = (filter) => {
    predefinedFilter.value = filter
    conversationStore.fetchConversations(conversationType.value, filter)
}

const loadNextPage = () => {
    conversationStore.fetchNextConversations(conversationType.value, predefinedFilter.value)
};

</script>
