<script setup>
import { DonutChart } from '@/components/ui/chart-donut'
const valueFormatter = (tick) => typeof tick === 'number' ? `$ ${new Intl.NumberFormat('us').format(tick).toString()}` : ''
const dataDonut = [
     { name: 'Jan', total: Math.floor(Math.random() * 2000) + 500, predicted: Math.floor(Math.random() * 2000) + 500 },
     { name: 'Feb', total: Math.floor(Math.random() * 2000) + 500, predicted: Math.floor(Math.random() * 2000) + 500 },
     { name: 'Mar', total: Math.floor(Math.random() * 2000) + 500, predicted: Math.floor(Math.random() * 2000) + 500 },
     { name: 'Apr', total: Math.floor(Math.random() * 2000) + 500, predicted: Math.floor(Math.random() * 2000) + 500 },
     { name: 'May', total: Math.floor(Math.random() * 2000) + 500, predicted: Math.floor(Math.random() * 2000) + 500 },
     { name: 'Jun', total: Math.floor(Math.random() * 2000) + 500, predicted: Math.floor(Math.random() * 2000) + 500 },
]

</script>

<template>
     <DonutChart index="name" :category="'total'" :data="dataDonut" :value-formatter="valueFormatter" type="pie" />
</template>
