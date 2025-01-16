<template>
  <div 
    class="relative p-4 transition-all duration-200 ease-in-out cursor-pointer hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
    :class="{ 'bg-blue-50': conversation.uuid === currentConversation?.uuid }"
    @click="navigateToConversation(conversation.uuid)"
  >
    <div class="flex items-start space-x-4">
      <Avatar class="w-12 h-12 rounded-full ring-2 ring-white">
        <AvatarImage :src="conversation.avatar_url" v-if="conversation.avatar_url" />
        <AvatarFallback>
          {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
        </AvatarFallback>
      </Avatar>

      <div class="flex-1 min-w-0">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-medium text-gray-900 truncate">
            {{ contactFullName }}
          </h3>
          <span class="text-xs text-gray-500" v-if="conversation.last_message_at">
            {{ formatTime(conversation.last_message_at) }}
          </span>
        </div>

        <p class="mt-1 text-xs text-gray-500 flex items-center space-x-1">
          <Mail class="w-3 h-3" />
          <span>{{ conversation.inbox_name }}</span>
        </p>

        <p class="mt-2 text-sm text-gray-600 line-clamp-2">
          <CheckCheck class="inline w-4 h-4 mr-1 text-green-500" />
          {{ trimmedLastMessage }}
        </p>

        <div class="flex items-center mt-2 space-x-2">
          <SlaDisplay :dueAt="conversation.first_reply_due_at" :actualAt="conversation.first_reply_at" :label="'FRD'" :showSLAHit="false" />
          <SlaDisplay :dueAt="conversation.resolution_due_at" :actualAt="conversation.resolved_at" :label="'RD'" :showSLAHit="false" />
        </div>
      </div>
    </div>

    <div 
      v-if="conversation.unread_message_count > 0"
      class="absolute top-4 right-4 flex items-center justify-center w-6 h-6 bg-blue-500 text-white text-xs font-bold rounded-full"
    >
      {{ conversation.unread_message_count }}
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { formatTime } from '@/utils/datetime'
import { Mail, CheckCheck } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import SlaDisplay from '@/components/sla/SlaDisplay.vue'

const router = useRouter()
const route = useRoute()

const props = defineProps({
  conversation: Object,
  currentConversation: Object,
  contactFullName: String
})

const navigateToConversation = (uuid) => {
  const baseRoute = route.name.includes('team')
    ? 'team-inbox-conversation'
    : route.name.includes('view')
      ? 'view-inbox-conversation'
      : 'inbox-conversation'

  router.push({
    name: baseRoute,
    params: {
      uuid,
      ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID }),
    },
  })
}

const trimmedLastMessage = computed(() => {
  const message = props.conversation.last_message || ''
  return message.length > 100 ? message.slice(0, 100) + "..." : message
})
</script>

