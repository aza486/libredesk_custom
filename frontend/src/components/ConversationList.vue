<template>
    <div class="h-screen">
        <div v-if="!conversationStore.conversations.loading">
            <Error :errorMessage="conversationStore.conversations.errorMessage"
                v-if="conversationStore.conversations.errorMessage"></Error>
            <div v-else>
                <div class="relative mx-auto my-2 px-3">
                    <Input id="search" type="text" placeholder="Reference num, email." class="pl-10 bg-[#F0F2F5]" />
                    <span class="absolute start-2 inset-y-0 flex items-center justify-center px-2">
                        <Search class="size-6 text-muted-foreground" />
                    </span>
                </div>

                <div class="h-screen overflow-y-scroll pb-32">
                    <div class="flex items-center cursor-pointer flex-row hover:bg-slate-50"
                        :class="{ 'bg-slate-50': conversation.uuid === conversationStore.conversation.data?.uuid }"
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
                        <div class="ml-3 w-full border-b pb-3">
                            <div class="flex justify-between pt-2 pr-3">
                                <div>
                                    <p class="text-xs text-gray-600 flex gap-x-1">
                                        <Mail size="12" />
                                        {{ conversation.inbox_name }}
                                    </p>
                                    <p class="text-base font-normal">
                                        {{ conversation.first_name + ' ' + conversation.last_name }}
                                    </p>
                                </div>
                                <div>
                                    <span class="text-sm text-muted-foreground" v-if="conversation.last_message_at">
                                        {{ format(conversation.last_message_at, 'h:mm a') }}
                                    </span>
                                </div>
                            </div>
                            <div class="pt-2 pr-3">
                                <div class="flex justify-between">
                                    <p class="text-gray-800 max-w-xs text-sm dark:text-white text-ellipsis">
                                        {{ conversation.last_message }}
                                    </p>
                                    <div class="flex items-center justify-center bg-green-500 h-5 w-5 rounded-full" v-if="conversation.unread_message_count > 0">
                                        <span class="text-white text-xs font-extrabold">
                                            {{ conversation.unread_message_count }}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
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
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Search, Mail } from 'lucide-vue-next';

const conversationStore = useConversationStore()
const router = useRouter()

onMounted(() => {
    conversationStore.fetchConversations()
});

</script>