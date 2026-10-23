<script setup>
import { Badge } from '@shared-ui/components/ui/badge'

defineProps({
  count: { type: Number, default: 0 }
})
</script>

<template>
  <Badge
    v-if="count > 0"
    variant="outline"
    class="ml-auto shrink-0 tabular-nums bg-background px-1.5 py-0 font-medium"
  >
    {{ count > 9 ? '9+' : count }}
  </Badge>
</template>
