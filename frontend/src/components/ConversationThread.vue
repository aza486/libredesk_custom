<template>
    <div class="h-screen" v-if="conversationStore.messages.data">
        <!-- Header -->
        <div class="h-12 px-4 box relative">
            <div class="flex flex-row justify-between items-center pt-1">
                <div class="flex h-5 items-center space-x-4 text-sm">
                    <Tooltip>
                        <TooltipTrigger>#{{ conversationStore.conversation.data.reference_number }}
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Reference number</p>
                        </TooltipContent>
                    </Tooltip>
                    <Separator orientation="vertical" />
                    <Tooltip>
                        <TooltipTrigger>
                            <Badge :variant="getBadgeVariant">{{ conversationStore.conversation.data.status }}
                            </Badge>
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Status</p>
                        </TooltipContent>
                    </Tooltip>
                </div>
                <div>
                    <DropdownMenu>
                        <DropdownMenuTrigger>
                            <Icon icon="lucide:ellipsis-vertical" class="mt-2 size-6"></Icon>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent>
                            <DropdownMenuItem @click="handleUpdateStatus('Open')">
                                <span>Open</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem @click="handleUpdateStatus('Processing')">
                                <span>Processing</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem @click="handleUpdateStatus('Spam')">
                                <span>Mark as spam</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem @click="handleUpdateStatus('Resolved')">
                                <span>Resolve</span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>
        </div>
        <!-- Body -->
        <Error class="sticky" :error-message="conversationStore.messages.errorMessage"></Error>
        <div class="flex flex-col h-screen scroll-y" v-if="conversationStore.messages.data">
            <!-- Messages -->
            <div class="break-word text-wrap overflow-y-scroll h-full">
                <MessageList :messages="conversationStore.sortedMessages" class="flex-1 bg-[#f8f9fa41]" />
            </div>
            <ReplyBox />
        </div>
    </div>
</template>

<script setup>
import { computed } from 'vue';
import { useConversationStore } from '@/stores/conversation'

import { Separator } from '@/components/ui/separator'
import { Error } from '@/components/ui/error'
import { Badge } from '@/components/ui/badge'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger
} from '@/components/ui/tooltip'
import MessageList from './MessageList.vue'
import ReplyBox from "./ReplyBox.vue"
import { Icon } from '@iconify/vue'

const conversationStore = useConversationStore()

const getBadgeVariant = computed(() => {
    return conversationStore.conversation.data?.status == "Spam" ? "destructive" : "success"
})

const handleUpdateStatus = (status) => {
    conversationStore.updateStatus(status)
}
</script>