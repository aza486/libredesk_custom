<template>
    <div class="flex flex-col items-end text-left">

        <div class="pr-[47px] mb-1">
            <p class="text-muted-foreground text-sm">
                {{ getFullName(message) }}
            </p>
        </div>

        <div class="flex flex-row gap-2 justify-end">
            <div class="
                flex
                flex-col
                message-bubble
                justify-end
                items-end
                relative" :class="{
                    'bg-[#FEF1E1]': message.private,
                    'bg-white': !message.private,
                    'opacity-50 animate-pulse': message.status === 'pending',
                    'bg-red': message.status === 'failed'
                }">

                <div v-html=message.content :class="{ 'mb-3': message.attachments.length > 0 }"></div>
                <MessageAttachmentPreview :attachments="message.attachments" />
                <Spinner v-if="message.status === 'pending'" />
                <div class="flex items-center space-x-2 mt-2">
                    <span class="text-slate-500 capitalize text-xs" v-if="message.status != 'pending'">{{
                        message.status}}</span>
                    <RotateCcw size="10" @click="retryMessage(message)" class="cursor-pointer"
                        v-if="message.status === 'failed'"></RotateCcw>
                </div>
            </div>
            <Avatar class="cursor-pointer">
                <AvatarImage :src=getAvatar />
                <AvatarFallback>
                    {{ avatarFallback(message) }}
                </AvatarFallback>
            </Avatar>

        </div>

        <div class="pr-[47px]">
            <Tooltip>
                <TooltipTrigger>
                    <span class="text-muted-foreground text-xs mt-1">
                        {{ format(message.updated_at, "h:mm a") }}
                    </span>
                </TooltipTrigger>
                <TooltipContent>
                    {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
                </TooltipContent>
            </Tooltip>
        </div>
    </div>
</template>

<script setup>
import { format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'
import api from '@/api';

import {
    Tooltip,
    TooltipContent,
    TooltipTrigger
} from '@/components/ui/tooltip'
import { Spinner } from '@/components/ui/spinner'
import { RotateCcw } from 'lucide-vue-next';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import MessageAttachmentPreview from "./MessageAttachmentPreview.vue"


defineProps({
    message: Object,
})
const convStore = useConversationStore()

const getAvatar = (msg) => {
    if (msg.sender_uuid && convStore.conversation.participants) {
        let participant = convStore.conversation.participants[msg.sender_uuid]
        return participant.avatar_url ? participant.avatar_url : ''
    }
    return ''
}

const getFullName = (msg) => {
    if (msg.sender_uuid && convStore.conversation.participants) {
        let participant = convStore.conversation.participants[msg.sender_uuid]
        return participant.first_name + ' ' + participant.last_name
    }
    return ''
}

const avatarFallback = (msg) => {
    if (msg.sender_uuid && convStore.conversation.participants) {
        let participant = convStore.conversation.participants[msg.sender_uuid]
        return participant.first_name.toUpperCase().substring(0, 2)
    }
    return ''
}

const retryMessage = (msg) => {
    api.retryMessage(msg.uuid)
    msg.status = 'pending'
    convStore.updateMessageStatus(msg)
}

</script>