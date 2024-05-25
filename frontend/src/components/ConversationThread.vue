<template>
    <div class="h-screen" v-if="conversationStore.conversation.data">
        <div class="h-14 border-b px-4">
            <div class="flex flex-row justify-between items-center pt-2">
                <div class="flex h-5 items-center space-x-4 text-sm">
                    <TooltipProvider :delay-duration=200>
                        <Tooltip>
                            <TooltipTrigger>#{{ conversationStore.conversation.data.reference_number }}
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Reference number</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                    <Separator orientation="vertical" />
                    <TooltipProvider :delay-duration=200>
                        <Tooltip>
                            <TooltipTrigger>
                                <Badge :variant="getBadgeVariant">{{ conversationStore.conversation.data.status }}
                                </Badge>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Status</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                    <Separator orientation="vertical" />
                    <TooltipProvider :delay-duration=200>
                        <Tooltip>
                            <TooltipTrigger>
                                <Badge variant="default">{{ conversationStore.conversation.data.priority }}
                                </Badge>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Priority</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
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
        <div class="flex flex-col h-screen">
            <Error :error-message="conversationStore.messages.errorMessage"></Error>
            <ScrollArea v-if="conversationStore.messages.data" class="flex-1">
                <MessageBubble v-for="message in conversationStore.messages.data" :key="message.uuid" :message="message"
                    :conversation="conversationStore.conversation.data" />
            </ScrollArea>
            <TextEditor @send="sendMessage" :identifier="conversationStore.conversation.data.uuid"
                :canned-responses="cannedResponsesStore.responses"></TextEditor>
        </div>
    </div>
</template>

<script setup>
import { computed, onMounted } from 'vue';
import { useConversationStore } from '@/stores/conversation'
import { useCannedResponses } from '@/stores/canned_responses'

import { Separator } from '@/components/ui/separator'
import { Error } from '@/components/ui/error'
import { Badge } from '@/components/ui/badge'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger
} from '@/components/ui/tooltip'
import MessageBubble from './MessageBubble.vue'
import TextEditor from './TextEditor.vue'
import { Icon } from '@iconify/vue'

// Store, state.
const conversationStore = useConversationStore()
const cannedResponsesStore = useCannedResponses()

// Functions, methods.
const sendMessage = (message) => {
    // TODO: Create msg.
    console.log(message)
}

const getBadgeVariant = computed(() => {
    return conversationStore.conversation.data.status == "Spam" ? "destructive" : "default"
})

const handleUpdateStatus = (status) => {
    conversationStore.updateStatus(status)
}

onMounted(() => {
    cannedResponsesStore.fetchAll()
})
</script>