<template>
  <div class="space-y-2">
    <div class="flex justify-between items-start">
      <Avatar class="size-20">
        <AvatarImage :src="conversation?.contact?.avatar_url || ''" />
        <AvatarFallback>
          {{ conversation?.contact?.first_name?.toUpperCase().substring(0, 2) }}
        </AvatarFallback>
      </Avatar>
      <PanelLeft
        class="cursor-pointer"
        @click="emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)"
        size="20"
      />
    </div>

    <div class="h-6 flex items-center gap-2">
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-24 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.first_name + ' ' + conversation?.contact?.last_name }}
      </span>
      <ExternalLink
        v-if="!conversationStore.conversation.loading"
        size="20"
        class="text-muted-foreground cursor-pointer flex-shrink-0"
        @click="$router.push({ name: 'contact-detail', params: { id: conversation?.contact_id } })"
      />
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Mail size="18" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else class="break-all">
        {{ conversation?.contact?.email }}
      </span>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Phone size="18" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ phoneNumber }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { PanelLeft } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone, ExternalLink } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'
import { useI18n } from 'vue-i18n'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()

const phoneNumber = computed(() => {
  const callingCode = conversation.value?.contact?.phone_number_calling_code || ''
  const number = conversation.value?.contact?.phone_number || t('conversation.sidebar.notAvailable')
  return callingCode ? `${callingCode} ${number}` : number
})
</script>
