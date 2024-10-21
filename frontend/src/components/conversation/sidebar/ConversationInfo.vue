<template>
    <div class="flex flex-col gap-1 mb-5">
        <p class="font-medium">Reference number</p>
        <p>
            #{{ conversation.reference_number }}
        </p>
    </div>
    <div class="flex flex-col gap-1 mb-5">
        <p class="font-medium">Initiated at</p>
        <p>
            {{ format(conversation.created_at, 'PPpp') }}
        </p>
    </div>

    <div class="flex flex-col gap-1 mb-5">
        <p class="font-medium">First reply at</p>
        <p v-if="conversation.first_reply_at">
            {{ format(conversation.first_reply_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
    </div>

    <div class="flex flex-col gap-1 mb-5">
        <p class="font-medium">Resolved at</p>
        <p v-if="conversation.resolved_at">
            {{ format(conversation.resolved_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
    </div>

    <div class="flex flex-col gap-1 mb-5">
        <p class="font-medium">Closed at</p>
        <p v-if="conversation.closed_at">
            {{ format(conversation.closed_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
    </div>
</template>

<script setup>
import { format } from 'date-fns'
defineProps({
    conversation: Object
})
</script>