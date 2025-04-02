<template>
  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">{{ $t('form.field.subject') }}</p>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <p v-else>
      {{ conversation.subject || '-' }}
    </p>
  </div>

  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">{{ $t('form.field.referenceNumber') }}</p>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <p v-else>
      {{ conversation.reference_number }}
    </p>
  </div>
  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">{{ $t('form.field.initiatedAt') }}</p>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <p v-if="conversation.created_at">
      {{ format(conversation.created_at, 'PPpp') }}
    </p>
    <p v-else>-</p>
  </div>

  <div class="flex flex-col gap-1 mb-5">
    <div class="flex justify-start items-center space-x-2">
      <p class="font-medium">{{ $t('form.field.firstReplyAt') }}</p>
      <SlaBadge
        v-if="conversation.first_response_deadline_at"
        :dueAt="conversation.first_response_deadline_at"
        :actualAt="conversation.first_reply_at"
        :key="conversation.uuid"
      />
    </div>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <div v-else>
      <p v-if="conversation.first_reply_at">
        {{ format(conversation.first_reply_at, 'PPpp') }}
      </p>
      <p v-else>-</p>
    </div>
  </div>

  <div class="flex flex-col gap-1 mb-5">
    <div class="flex justify-start items-center space-x-2">
      <p class="font-medium">{{ $t('form.field.resolvedAt') }}</p>
      <SlaBadge
        v-if="conversation.resolution_deadline_at"
        :dueAt="conversation.resolution_deadline_at"
        :actualAt="conversation.resolved_at"
        :key="conversation.uuid"
      />
    </div>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <div v-else>
      <p v-if="conversation.resolved_at">
        {{ format(conversation.resolved_at, 'PPpp') }}
      </p>
      <p v-else>-</p>
    </div>
  </div>

  <div class="flex flex-col gap-1 mb-5" v-if="conversation.closed_at">
    <p class="font-medium">{{ $t('form.field.closedAt') }}</p>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <p v-else>
      {{ format(conversation.closed_at, 'PPpp') }}
    </p>
  </div>

  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">{{ $t('form.field.slaPolicy') }}</p>
    <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
    <div v-else>
      <p v-if="conversation.sla_policy_name">
        {{ conversation.sla_policy_name }}
      </p>
      <p v-else>-</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { format } from 'date-fns'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'

const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
</script>
