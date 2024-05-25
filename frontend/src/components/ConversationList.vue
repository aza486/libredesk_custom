<template>
    <div class="h-screen text-left">
        <Error :errorMessage="conversationStore.conversations.errorMessage"></Error>
        <div v-if="!conversationStore.conversations.loading">
            <ScrollArea>
                <div class="border-b flex items-center cursor-pointer p-2 flex-row"
                    v-for="conversation in conversationStore.conversations.data" :key="conversation.uuid"
                    @click="router.push('/conversations/' + conversation.uuid)">
                    <div>
                        <Avatar class="size-[55px]">
                            <AvatarImage :src=conversation.contact_avatar_url />
                            <AvatarFallback>
                                {{ conversation.contact_first_name.substring(0, 2).toUpperCase() }}
                            </AvatarFallback>
                        </Avatar>
                    </div>
                    <div class="ml-3 w-full">
                        <div class="flex justify-between">
                            <div>
                                <p class="text-base font-normal">
                                    {{ conversation.contact_first_name + ' ' + conversation.contact_last_name }}
                                </p>
                            </div>
                            <div>
                                <span class="text-xs text-muted-foreground">
                                    {{ format(conversation.updated_at, 'h:mm a') }}
                                </span>
                            </div>
                        </div>
                        <div>
                            <p class="text-gray-600 max-w-xs text-sm dark:text-white">
                                {{ conversation.last_message }}
                            </p>
                        </div>
                    </div>
                </div>
            </ScrollArea>
        </div>
        <div v-else>
            <div class="flex items-center gap-5 p-6 border-b" v-for="index in 10" :key="index">
                <Skeleton class="h-12 w-12 rounded-full" />
                <div class="space-y-2">
                    <Skeleton class="h-4 w-[250px]" />
                    <Skeleton class="h-4 w-[200px]" />
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useConversationStore } from '@/stores/conversation'
import { format } from 'date-fns'

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Error } from '@/components/ui/error'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'

// Stores, states.
const conversationStore = useConversationStore()
const router = useRouter()

// Functions, methods.
onMounted(() => {
    conversationStore.fetchConversations()
});

</script>
