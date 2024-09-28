<template>
  <BarChart
    :data="data"
    index="status"
    :categories="['Low', 'Medium', 'High']"
    :show-grid-line="true"
    :margin="{ top: 0, bottom: 0, left: 0, right: 0 }"
  />
</template>

<script setup>
import { BarChart } from '@/components/ui/chart-bar'

defineProps({
  data: {
    type: Array,
    required: true,
    default: () => []
  }
})
</script>
