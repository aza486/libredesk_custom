<template>
  <div v-if="dueAt" class="flex items-center justify-center">
    <span
      v-if="actualAt && isAfterDueTime"
      class="flex items-center bg-red-100 p-1 rounded-lg text-xs text-red-700 border border-red-300"
    >
      <AlertCircle class="w-4 h-4 mr-1" />
      <span class="flex items-center">{{ label }} Overdue</span>
    </span>

    <span v-else-if="actualAt && !isAfterDueTime" class="flex items-center text-xs text-green-700">
      <template v-if="showSLAHit">
        <CheckCircle class="w-4 h-4 mr-1" />
        <span class="flex items-center">{{ label }} SLA Hit</span>
      </template>
    </span>

    <span
      v-else-if="sla?.status === 'remaining'"
      class="flex items-center bg-yellow-100 p-1 rounded-lg text-xs text-yellow-700 border border-yellow-300"
    >
      <Clock class="w-4 h-4 mr-1" />
      <span class="flex items-center">{{ label }} {{ sla.value }}</span>
    </span>

    <span
      v-else-if="sla?.status === 'overdue'"
      class="flex items-center bg-red-100 p-1 rounded-lg text-xs text-red-700 border border-red-300"
    >
      <AlertCircle class="w-4 h-4 mr-1" />
      <span class="flex items-center">{{ label }} Overdue by {{ sla.value }}</span>
    </span>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useSla } from '@/composables/useSla'
import { AlertCircle, CheckCircle, Clock } from 'lucide-vue-next'

const props = defineProps({
  dueAt: String,
  actualAt: String,
  label: String,
  showSLAHit: {
    type: Boolean,
    default: true
  }
})

const { sla, isAfterDueTime } = useSla(ref(props.dueAt), ref(props.actualAt))
</script>
