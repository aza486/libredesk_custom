<template>
  <div v-if="dueAt" class="flex items-center justify-center">
    <TransitionGroup name="fade" class="animate-fade-in-down">
      <span
        v-if="actualAt && isAfterDueTime"
        key="overdue"
        class="inline-flex items-center bg-red-50 px-1 py-1 rounded-full text-xs font-medium text-red-700 border border-red-200 shadow-sm transition-all duration-300 ease-in-out hover:bg-red-100 animate-fade-in-down min-w-[90px]"
      >
        <AlertCircle class="w-3 h-3  flex-shrink-0" />
        <span class="flex-1 text-center">{{ label }} Overdue</span>
      </span>

      <span
        v-else-if="actualAt && !isAfterDueTime && showSLAHit"
        key="sla-hit"
        class="inline-flex items-center bg-green-50 px-1 py-1 rounded-full text-xs font-medium text-green-700 border border-green-200 shadow-sm transition-all duration-300 ease-in-out hover:bg-green-100 animate-fade-in-down min-w-[90px]"
      >
        <CheckCircle class="w-3 h-3  flex-shrink-0" />
        <span class="flex-1 text-center">{{ label }} SLA Hit</span>
      </span>

      <span
        v-else-if="sla?.status === 'remaining'"
        key="remaining"
        class="inline-flex items-center bg-yellow-50 px-1 py-1 rounded-full text-xs font-medium text-yellow-700 border border-yellow-200 shadow-sm transition-all duration-300 ease-in-out hover:bg-yellow-100 animate-fade-in-down min-w-[90px]"
      >
        <Clock class="w-3 h-3  flex-shrink-0" />
        <span class="flex-1 text-center">{{ label }} {{ sla.value }}</span>
      </span>

      <span
        v-else-if="sla?.status === 'overdue'"
        key="sla-overdue"
        class="inline-flex items-center bg-red-50 px-1 py-1 rounded-full text-xs font-medium text-red-700 border border-red-200 shadow-sm transition-all duration-300 ease-in-out hover:bg-red-100 animate-fade-in-down min-w-[90px]"
      >
        <AlertCircle class="w-3 h-3  flex-shrink-0" />
        <span class="flex-1 text-center">{{ label }} overdue</span>
      </span>
    </TransitionGroup>
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
