<template>
    <div class="flex flex-col items-start">
        <div class="pl-[47px] mb-1">
            <p class="text-muted-foreground text-sm">
                {{ getFullName }}
            </p>
        </div>
        <div class="flex flex-row gap-2">
            <Avatar class="cursor-pointer">
                <AvatarImage :src=getAvatar />
                <AvatarFallback>
                    {{ avatarFallback }}
                </AvatarFallback>
            </Avatar>
            <div class="flex flex-col justify-end message-bubble">
                <Letter :html=message.content class="mb-1" :class="{ 'mb-3': message.attachments.length > 0 }" />
                <MessageAttachmentPreview :attachments="message.attachments" />
            </div>
        </div>
        <div class="pl-[47px]">
            <Tooltip>
                <TooltipTrigger>
                    <span class="text-muted-foreground text-xs mt-1">
                        {{ format(message.updated_at, "h:mm a") }}
                    </span>
                </TooltipTrigger>
                <TooltipContent>
                    <p>
                        {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
                    </p>
                </TooltipContent>
            </Tooltip>
        </div>
    </div>
</template>

<script setup>
import { computed } from "vue"
import { format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'

import {
    Tooltip,
    TooltipContent,
    TooltipTrigger
} from '@/components/ui/tooltip'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import MessageAttachmentPreview from "./MessageAttachmentPreview.vue"


defineProps({
    message: Object,
})
const convStore = useConversationStore()

const getAvatar = computed(() => {
    return convStore.conversation.data.avatar_url ? convStore.conversation.avatar_url : ''
})

const getFullName = computed(() => {
    return convStore.conversation.data.first_name + ' ' + convStore.conversation.data.last_name
})

const avatarFallback = computed(() => {
    return convStore.conversation.data.first_name.toUpperCase().substring(0, 2)
})
</script>