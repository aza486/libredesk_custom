<script setup>
import { LineChart } from '@/components/ui/chart-line'

defineProps({
    data: {
        type: Array,
        required: true,
        default: () => []
    }
});

</script>

<template>
    <LineChart :data="data" index="date" :categories="['new_conversations']" :y-formatter="(tick, i) => {
        return typeof tick === 'number'
            ? `${new Intl.NumberFormat('us').format(tick).toString()}`
            : ''
    }" />
</template>