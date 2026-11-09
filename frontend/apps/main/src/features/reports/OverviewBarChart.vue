<template>
  <BarChart :data="displayData" index="status" :categories="priorities" :show-grid-line="true" :show-x-axis="true"
    :show-y-axis="true" type="grouped" :x-formatter="xFormatter" :y-formatter="yFormatter"   :rounded-corners="4"/>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { conversationPriorityLabel } from '@/utils/conversation-priority'
import { BarChart } from '@shared-ui/components/ui/chart-bar'

const props = defineProps({
  data: {
    type: Array,
    required: true,
    default: () => []
  }
})

const xFormatter = (tick) => {
  return props.data[tick]?.status ?? ''
}

const yFormatter = (tick) => {  
  return Number.isInteger(tick) ? tick : ''
}

const { t } = useI18n()
const series = ['Low', 'Medium', 'High']
const seriesLabels = computed(() => Object.fromEntries(series.map((name) => {
  const label = conversationPriorityLabel(name, t)
  // Preserve a custom column whose name coincides with a German label.
  return [name, label !== name && props.data.some((row) => Object.hasOwn(row, label)) ? name : label]
})))
const priorities = computed(() => series.map((name) => seriesLabels.value[name]))
// The chart library uses series keys as legend/tooltip text. Localize a
// presentation copy only; API rows and their original keys remain untouched.
const displayData = computed(() => props.data.map((row) => Object.fromEntries(
  Object.entries(row).map(([key, value]) => [Object.hasOwn(seriesLabels.value, key) ? seriesLabels.value[key] : key, value])
)))
</script>
