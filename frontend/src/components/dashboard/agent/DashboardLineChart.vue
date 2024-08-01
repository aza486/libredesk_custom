<template>
    <LineChart :data="renamedData" index="date" :categories="['New conversations']" :y-formatter="(tick) => {
        return tick
    }" />
</template>

<script setup>
import { computed } from 'vue';
import { LineChart } from '@/components/ui/chart-line'

const props = defineProps({
    data: {
        type: Array,
        required: true,
        default: () => []
    }
});

const renamedData = computed(() => 
    props.data.map(item => ({
        ...item,
        'New conversations': item.new_conversations
    }))
);
</script>
