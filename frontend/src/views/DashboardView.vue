<script setup>
import { onMounted, ref } from 'vue';
import { useUserStore } from '@/stores/user'
import { LineChart } from '@/components/ui/chart-line'
import { format } from 'date-fns';
import api from '@/api';

import {
    Card,
    CardContent,
    CardDescription,
    CardFooter,
    CardHeader,
    CardTitle,
} from '@/components/ui/card'
import { useToast } from '@/components/ui/toast/use-toast'

const { toast } = useToast()
const counts = ref({})
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

onMounted(() => {
    api.getAssigneeStats().then((resp) => {
        counts.value = resp.data.data;
    }).catch((err) => {
        toast({
            title: 'Uh oh! Something went wrong.',
            description: err.response.data.message,
            variant: 'destructive',
        });
    });
})

const labels = {
    total_assigned: "Total Assigned",
    unresolved_count: "Unresolved",
    awaiting_response_count: "Awaiting Response",
    created_today_count: "Created Today"
};


</script>

<template>
    <div class="tab-container-default">
        <div v-if="userStore.getFullName">
            <h4 class="scroll-m-20 text-2xl font-semibold tracking-tight">
                <p>Good morning, {{ userStore.getFullName }}</p>
                <p class="text-xl text-muted-foreground">🌤️ {{ format(new Date(), "EEEE, MMMM d HH:mm a") }}</p>
            </h4>
        </div>
        <div class="flex mt-5 gap-x-7">
            <Card class="w-1/6" v-for="(value, key) in counts" :key="key">
                <CardHeader>
                    <CardTitle class="text-4xl">
                        {{ value }}
                    </CardTitle>
                    <CardDescription>
                        {{ labels[key] }}
                    </CardDescription>
                </CardHeader>
            </Card>
        </div>
        <!-- <div class="w-1/2 flex flex-col items-center justify-between">
            <LineChart :data="data" index="year" :categories="['Export Growth Rate', 'Import Growth Rate']"
                :y-formatter="(tick, i) => {
                    return typeof tick === 'number'
                        ? `$ ${new Intl.NumberFormat('us').format(tick).toString()}`
                        : ''
                }" />
        </div> -->
    </div>
</template>