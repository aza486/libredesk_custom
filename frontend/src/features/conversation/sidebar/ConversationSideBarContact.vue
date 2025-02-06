<template>
  <div>
    <div class="flex justify-between items-start">
      <Avatar class="size-20">
        <AvatarImage
          :src="conversation?.contact?.avatar_url"
          v-if="conversation?.contact?.avatar_url"
        />
        <AvatarFallback>
          {{ conversation?.contact?.first_name?.toUpperCase().substring(0, 2) }}
        </AvatarFallback>
      </Avatar>
      <PanelLeft
        class="cursor-pointer"
        @click="emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)"
        size="16"
      />
    </div>
    <div>
      <h4 class="mt-3">
        {{ conversation?.contact?.first_name + ' ' + conversation?.contact?.last_name }}
      </h4>
      <p class="text-sm text-muted-foreground flex gap-2 mt-1" v-if="conversation?.contact?.email">
        <Mail class="size-3 mt-1" />
        {{ conversation.contact.email }}
      </p>
      <p class="text-sm text-muted-foreground flex gap-2 mt-1">
        <Phone class="size-3 mt-1" />
        <span>
          {{ conversation?.contact?.phone_number || 'Not available' }}
        </span>
      </p>
    </div>
  </div>
</template>

<script setup>
import { PanelLeft } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const emitter = useEmitter()

defineProps({
  conversation: Object
})
</script>
