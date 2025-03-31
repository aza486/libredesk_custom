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

    <div class="mt-3 h-6">
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-24 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.first_name + ' ' + conversation?.contact?.last_name }}
      </span>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 h-4 mt-2">
      <Mail class="size-3 mt-1" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.email }}
      </span>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 mt-2 h-4">
      <Phone class="size-3 mt-1" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.phone_number || t('conversation.sidebar.notAvailable') }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { PanelLeft } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'
import {useI18n} from 'vue-i18n'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()
</script>
