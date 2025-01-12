<template>
    <div>
        <Avatar class="size-20">
            <AvatarImage :src="conversation?.avatar_url" v-if="conversation?.avatar_url" />
            <AvatarFallback>
                {{ conversation?.contact.first_name.toUpperCase().substring(0, 2) }}
            </AvatarFallback>
        </Avatar>
        <h4 class="mt-3">
            {{ conversation?.contact.first_name + ' ' + conversation?.contact.last_name }}
        </h4>
        <p class="text-sm text-muted-foreground flex gap-2 mt-1" v-if="conversation?.contact.email">
            <Mail class="size-3 mt-1"></Mail>
            {{ conversation.contact.email }}
        </p>
        <p class="text-sm text-muted-foreground flex gap-2 mt-1" v-if="conversation?.contact.phone_number">
            <Phone class="size-3 mt-1"></Phone>
            {{ conversation.contact.phone_number }}
        </p>
    </div>
</template>

<script setup>
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone } from 'lucide-vue-next'
defineProps({
    conversation: Object
})
</script>
