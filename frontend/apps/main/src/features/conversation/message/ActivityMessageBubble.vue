<template>
  <div class="text-center">
    <div class="text-muted-foreground text-sm">
      {{ activityLabel }}
      <span class="text-xs ml-1">
        {{ format(message.updated_at, 'dd.MM.yyyy HH:mm') }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { format } from 'date-fns'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { conversationActivityLabel } from '@/utils/conversation-activity'

const props = defineProps({
  message: Object
})

const { t } = useI18n()
const activityLabel = computed(() => conversationActivityLabel(props.message, t))
</script>
