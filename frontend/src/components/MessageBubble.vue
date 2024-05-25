<template>
    <div class="flex flex-col" v-if="message.type === 'incoming' || message.type === 'outgoing'">
        <div class="self-start w-11/12 flex px-5 py-3 mt-2 ">
            <Avatar class="mt-1">
                <AvatarImage :src=getAvatar />
                <AvatarFallback>
                    {{ message.first_name.toUpperCase().substring(0, 2) }}
                </AvatarFallback>
            </Avatar>
            <div class="ml-5">
                <div class="flex gap-3">
                    <p class="font-medium">
                        {{ message.first_name + ' ' + message.last_name }}
                    </p>
                    <p class="text-muted-foreground text-xs mt-1">
                        {{ format(message.updated_at, 'h:mm a') }}
                    </p>
                </div>
                <Letter :html=message.content class="mb-1" />
            </div>
        </div>
    </div>
    <div class="self-start flex px-5 py-3 mt-2 bg-[#FFF7E6] opacity-80" v-if="message.type === 'internal_note'">
        <Avatar class="mt-1">
            <AvatarImage :src=getAvatar />
            <AvatarFallback>
                {{ message.first_name.toUpperCase().substring(0, 2) }}
            </AvatarFallback>
        </Avatar>
        <div class="ml-5">
            <div class="flex gap-3">
                <p class="font-medium">
                    {{ message.first_name + ' ' + message.last_name }}
                </p>
                <div class="flex items-center text-xs text-muted-foreground">
                    {{ format(message.updated_at, 'h:mm a') }}
                    <LockKeyhole class="w-4 h-4 ml-2" />
                </div>
            </div>
            <Letter :html=message.content class="mb-1" />
        </div>
    </div>
    <div class="text-center p-2" v-if="message.type === 'activity'">
        <div class="text-sm text-muted-foreground">
            {{ message.content }}
            <span>{{ format(message.updated_at, 'h:mm a') }}</span>
        </div>
    </div>
</template>

<script setup>
import { computed } from "vue"
import { format } from 'date-fns'

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import { LockKeyhole } from "lucide-vue-next"

const props = defineProps({
    message: Object,
})

const getAvatar = computed(() => {
    return props.message.avatar_url ? props.message.avatar_url : ''
})

</script>
