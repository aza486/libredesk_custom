<template>
  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">SLA policy</p>
    <p v-if="conversation.sla_policy_name">
      {{ conversation.sla_policy_name }}
    </p>
    <p v-else>-</p>
  </div>

  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">Reference number</p>
    <p>
      {{ conversation.reference_number }}
    </p>
  </div>
  <div class="flex flex-col gap-1 mb-5">
    <p class="font-medium">Initiated at</p>
    <p>
      {{ format(conversation.created_at, 'PPpp') }}
    </p>
  </div>

  <div class="flex flex-col gap-1 mb-5" v-if="conversation.first_reply_at">
    <div class="flex justify-start items-center space-x-2">
      <p class="font-medium">First reply at</p>
      <SlaDisplay
        :dueAt="conversation.first_response_due_at"
        :actualAt="conversation.first_reply_at"
      />
    </div>
    <p>
      {{ format(conversation.first_reply_at, 'PPpp') }}
    </p>
  </div>

  <div class="flex flex-col gap-1 mb-5" v-if="conversation.resolved_at">
    <div class="flex justify-start items-center space-x-2">
      <p class="font-medium">Resolved at</p>
      <SlaDisplay :dueAt="conversation.resolution_due_at" :actualAt="conversation.resolved_at" />
    </div>
    <p>
      {{ format(conversation.resolved_at, 'PPpp') }}
    </p>
  </div>

  <div class="flex flex-col gap-1 mb-5" v-if="conversation.closed_at">
    <p class="font-medium">Closed at</p>
    <p>
      {{ format(conversation.closed_at, 'PPpp') }}
    </p>
  </div>
</template>

<script setup>
import { format } from 'date-fns'
import SlaDisplay from '@/features/sla/SlaDisplay.vue'
defineProps({
  conversation: Object
})
</script>
