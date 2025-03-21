<template>
  <LineChart :data="data" index="date" :categories="['New conversations', 'Resolved conversations']"
    :x-formatter="xFormatter" :y-formatter="yFormatter" />
</template>

<script setup>
import { LineChart } from '@/components/ui/chart-line'
const props = defineProps({
  data: {
    type: Array,
    default: () => []
  }
})

const xFormatter = (tick) => {
  return props.data[tick]?.date ?? ''
}

const yFormatter = (tick) => {
  return Number.isInteger(tick) ? tick : ''
}

</script>
