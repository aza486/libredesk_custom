<template>
  <div class="flex items-center cursor-pointer flex-row hover:bg-gray-100 hover:rounded-lg hover:box"
    :class="{ 'bg-white rounded-lg box': conversation.uuid === currentConversation?.uuid }"
    @click="navigateToConversation(conversation.uuid)">

    <div class="pl-3">
      <Avatar class="size-[45px]">
        <AvatarImage :src="conversation.avatar_url" v-if="conversation.avatar_url" />
        <AvatarFallback>
          {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
        </AvatarFallback>
      </Avatar>
    </div>

    <div class="ml-3 w-full pb-2">
      <div class="flex justify-between pt-2 pr-3">
        <div>
          <p class="text-xs text-gray-600 flex gap-x-1">
            <Mail size="13" />
            {{ conversation.inbox_name }}
          </p>
          <p class="text-base font-normal">
            {{ contactFullName }}
          </p>
        </div>
        <div>
          <span class="text-sm text-muted-foreground" v-if="conversation.last_message_at">
            {{ formatTime(conversation.last_message_at) }}
          </span>
        </div>
      </div>
      <div class="pt-2 pr-3">
        <div class="flex justify-between">
          <p class="text-gray-800 max-w-xs text-sm dark:text-white text-ellipsis flex gap-1">
            <CheckCheck :size="14" /> {{ trimmedLastMessage }}
          </p>
          <div class="flex items-center justify-center bg-green-500 rounded-full w-[20px] h-[20px]"
            v-if="conversation.unread_message_count > 0">
            <span class="text-white text-xs font-extrabold">
              {{ conversation.unread_message_count }}
            </span>
          </div>
        </div>
      </div>
      <div class="flex space-x-2 mt-2">
        <SlaDisplay :dueAt="conversation.first_reply_due_at" :actualAt="conversation.first_reply_at" :label="'FRD'"
          :showSLAHit="false" />
        <SlaDisplay :dueAt="conversation.resolution_due_at" :actualAt="conversation.resolved_at" :label="'RD'"
          :showSLAHit="false" />
      </div>
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
  return message.length > 45 ? message.slice(0, 45) + "..." : message
})
</script>
