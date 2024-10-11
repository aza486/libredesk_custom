<template>
  <LineChart :data="chartData" index="date" :categories="['New conversations']" :y-formatter="(tick) => {
    return tick
  }
    " />
</template>

<script setup>
import { computed } from 'vue'
import { LineChart } from '@/components/ui/chart-line'

const props = defineProps({
  data: {
    type: Array,
    default: () => []
  }
})

const chartData = computed(() =>
  props.data.map(item => ({
    date: item.date,
    "New conversations": item.new_conversations
  }))
)
</script>
