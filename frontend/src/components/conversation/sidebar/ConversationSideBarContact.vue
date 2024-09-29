<template>
    <div>
        <Avatar class="size-20">
            <AvatarImage :src="conversation?.avatar_url" v-if="conversation?.avatar_url" />
            <AvatarFallback>
                {{ conversation?.first_name.toUpperCase().substring(0, 2) }}
            </AvatarFallback>
        </Avatar>
        <h4 class="mt-3">
            {{ conversation?.first_name + ' ' + conversation?.last_name }}
        </h4>
        <p class="text-sm text-muted-foreground flex gap-2 mt-1" v-if="conversation?.email">
            <Mail class="size-3 mt-1"></Mail>
            {{ conversation.email }}
        </p>
        <p class="text-sm text-muted-foreground flex gap-2 mt-1" v-if="conversation?.phone_number">
            <Phone class="size-3 mt-1"></Phone>
            {{ conversation.phone_number }}
        </p>
    </div>
</template>

<script setup>
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone } from 'lucide-vue-next'
defineProps({
    conversation: Object
});
</script>
