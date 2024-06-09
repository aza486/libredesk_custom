<script setup>
import { useUserStore } from '@/stores/user'
import { LineChart } from '@/components/ui/chart-line'
import { format } from 'date-fns';


const data = [
    {
        'year': 1970,
        'Export Growth Rate': 2.04,
        'Import Growth Rate': 1.53,
    },
    {
        'year': 1971,
        'Export Growth Rate': 1.96,
        'Import Growth Rate': 1.58,
    },
]
const userStore = useUserStore()

</script>

<template>
    <div class="tab-container-default">
        <div v-if="userStore.getFullName">
            <h4 class="scroll-m-20 text-2xl font-semibold tracking-tight">
                <p>Good morning, {{ userStore.getFullName }}</p>
                <p class="text-xl text-muted-foreground">🌤️ {{ format(new Date(), "EEEE, MMMM d HH:mm a") }}</p>
            </h4>
        </div>
        <div class="w-1/2 flex flex-col items-center justify-between">
            <LineChart :data="data" index="year" :categories="['Export Growth Rate', 'Import Growth Rate']"
                :y-formatter="(tick, i) => {
                    return typeof tick === 'number'
                        ? `$ ${new Intl.NumberFormat('us').format(tick).toString()}`
                        : ''
                }" />
        </div>
    </div>
</template>