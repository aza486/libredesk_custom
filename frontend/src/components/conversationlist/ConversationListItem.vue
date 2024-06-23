<template>
    <div class="flex items-center cursor-pointer flex-row hover:bg-slate-50"
        :class="{ 'bg-slate-100': conversation.uuid === conversationStore.conversation.data?.uuid }"
        v-for="conversation in conversationStore.sortedConversations" :key="conversation.uuid"
        @click="router.push('/conversations/' + conversation.uuid)">
        <div class="pl-3">
            <Avatar class="size-[45px]">
                <AvatarImage :src=conversation.avatar_url v-if="conversation.avatar_url" />
                <AvatarFallback>
                    {{ conversation.first_name.substring(0, 2).toUpperCase() }}
                </AvatarFallback>
            </Avatar>
        </div>
        <div class="ml-3 w-full border-b pb-2">
            <div class="flex justify-between pt-2 pr-3">
                <div>
                    <p class="text-xs text-gray-600 flex gap-x-1">
                        <Mail size="12" />
                        {{ conversation.inbox_name }}
                    </p>
                    <p class="text-base font-normal">
                        {{ conversationStore.getContactFullName (conversation.uuid)}}
                    </p>
                </div>
                <div>
                    <span class="text-sm text-muted-foreground" v-if="conversation.last_message_at">
                        {{ formatTime(conversation.last_message_at) }}
                    </span>
                </div>
            </div>
            <div class="pt-2 pr-3">
                <div class="flex justify-between">
                    <p class="text-gray-800 max-w-xs text-sm dark:text-white text-ellipsis">
                        {{ conversation.last_message }}
                    </p>
                    <div class="flex items-center justify-center bg-green-500 rounded-full w-[20px] h-[20px]"
                        v-if="conversation.unread_message_count > 0">
                        <span class="text-white text-xs font-extrabold">
                            {{ conversation.unread_message_count }}
                        </span>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useConversationStore } from '@/stores/conversation'
import { formatTime } from '@/utils/datetime'

import { Mail } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

const router = useRouter()
const conversationStore = useConversationStore()
</script>