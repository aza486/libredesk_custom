<template>
    <div class="flex flex-col items-end text-left">

        <div class="pr-[47px] mb-1">
            <p class="text-muted-foreground text-sm">
                {{ getFullName }}
            </p>
        </div>
        <div class="flex flex-row gap-2 justify-end">
            <div class="
                flex
                flex-col
                message-bubble
                justify-end
                items-end
                relative
                !rounded-tr-none
                " :class="{
                    'bg-[#FEF1E1]': message.private,
                    'bg-white': !message.private,
                    'opacity-50 animate-pulse': message.status === 'pending',
                    'bg-red': message.status === 'failed'
                }">

                <div v-html=message.content :class="{ 'mb-3': message.attachments.length > 0 }"></div>
                <MessageAttachmentPreview :attachments="message.attachments" />
                <Spinner v-if="message.status === 'pending'" />
                <div class="flex items-center space-x-2 mt-2">
                    <CheckCheck :size="14"  v-if="message.status == 'sent'"/>
                    <RotateCcw size="10" @click="retryMessage(message)" class="cursor-pointer"
                        v-if="message.status === 'failed'"></RotateCcw>
                </div>
            </div>
            <Avatar class="cursor-pointer">
                <AvatarImage :src=getAvatar />
                <AvatarFallback>
                    {{ avatarFallback }}
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
import { computed } from 'vue'
import { format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'
import api from '@/api'

import {
    Tooltip,
    TooltipContent,
    TooltipTrigger
} from '@/components/ui/tooltip'
import { Spinner } from '@/components/ui/spinner'
import { RotateCcw, CheckCheck } from 'lucide-vue-next';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import MessageAttachmentPreview from "@/components/attachment/MessageAttachmentPreview.vue"


const props = defineProps({
    message: Object,
})
const convStore = useConversationStore();

const participant = computed(() => {
    return convStore.conversation?.participants?.[props.message.sender_uuid] ?? {};
});

const getFullName = computed(() => {
    const firstName = participant.value?.first_name ?? 'Unknown'
    const lastName = participant.value?.last_name ?? 'User'
    return `${firstName} ${lastName}`;
});

const getAvatar = computed(() => {
    return participant.value?.avatar_url
});

const avatarFallback = computed(() => {
    const firstName = participant.value?.first_name ?? 'A'
    return firstName.toUpperCase().substring(0, 2);
});

const retryMessage = (msg) => {
    api.retryMessage(msg.uuid)
    convStore.updateMessageStatus(msg.uuid, 'pending')
}

</script>