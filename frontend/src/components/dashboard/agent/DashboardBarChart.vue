<template>
    <BarChart :data="data" index="status" :categories="['Low', 'Medium', 'High']" :show-grid-line="true" :y-formatter="(tick) => {
        return tick
    }" :margin="{ top: 0, bottom: 0, left: 20, right: 20 }" />
</template>

<script setup>
import { BarChart } from '@/components/ui/chart-bar'

defineProps({
    data: {
        type: Array,
        required: true,
        default: () => []
    }
});
</script>