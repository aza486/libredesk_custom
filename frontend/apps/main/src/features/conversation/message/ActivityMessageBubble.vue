<template>
  <div class="text-center">
    <div class="text-muted-foreground text-sm">
      {{ message.content }}
      <span class="text-xs ml-1">
        {{ format(message.updated_at, 'dd.MM.yyyy HH:mm') }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { format } from 'date-fns'

defineProps({
  message: Object
})
</script>
