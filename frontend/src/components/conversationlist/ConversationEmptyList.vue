<template>
    <div class="flex flex-col items-center justify-center h-64">
        <MessageCircleWarning :stroke-width="1.4" :size="90" />
        <h1 class="text-lg font-semibold text-gray-800">
            {{ $t('conversations.emptyState') }}
        </h1>
        <p class="text-gray-600">
            {{ $t('globals.messages.adjustFilters') }}
        </p>
    </div>
</template>

<script setup>
import { MessageCircleWarning } from 'lucide-vue-next'
</script>