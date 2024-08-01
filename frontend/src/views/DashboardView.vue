<template>
    <div class="page-content w-11/12">
        <div class="flex flex-col space-y-5">
            <div>
                <span class="font-semibold text-xl" v-if="userStore.getFullName">
                    <p>Hi, {{ userStore.getFullName }}</p>
                    <p class="text-sm text-muted-foreground">🌤️ {{ format(new Date(), "EEEE, MMMM d, HH:mm a") }}</p>
                </span>
            </div>
            <div>
                <Select :model-value="filter" @update:model-value="onFilterChange">
                    <SelectTrigger class="w-[130px]">
                        <SelectValue placeholder="Select a filter" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectGroup>
                            <SelectItem value="me">
                                Mine
                            </SelectItem>
                            <SelectItem value="global">
                                Global
                            </SelectItem>
                            <SelectItem value="3">
                                Funds team
                            </SelectItem>
                        </SelectGroup>
                    </SelectContent>
                </Select>
            </div>
        </div>
        <div class="mt-7">
            <Card :counts="cardCounts" :labels="agentCountCardsLabels" />
        </div>
        <div class="flex my-7 justify-between items-center space-x-5">
            <LineChart :data="chartData.new_conversations" class="dashboard-card p-5"></LineChart>
            <BarChart :data="chartData.status_summary" class="dashboard-card p-5"></BarChart>
        </div>
    </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue';
import { useUserStore } from '@/stores/user'
import { format } from 'date-fns'
import api from '@/api';
import { useToast } from '@/components/ui/toast/use-toast'

import Card from '@/components/dashboard/agent/DashboardCard.vue'
import LineChart from '@/components/dashboard/agent/DashboardLineChart.vue';
import BarChart from '@/components/dashboard/agent/DashboardBarChart.vue';
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'

const { toast } = useToast()
const cardCounts = ref({})
const chartData = ref({})
const filter = ref("me")
const userStore = useUserStore()
const agentCountCardsLabels = {
    total_count: "Total",
    resolved_count: "Resolved",
    unresolved_count: "Unresolved",
    awaiting_response_count: "Awaiting Response",
};

onMounted(() => {
    getCardStats()
    getDashboardCharts()
})

watch(filter, () => {
    getDashboardCharts()
    getCardStats()
})

const onFilterChange = (v) => {
    filter.value = v
}

const getCardStats = () => {
    let apiCall;
    switch (filter.value) {
        case "global":
            apiCall = api.getGlobalDashboardCounts;
            break;
        case "me":
            apiCall = api.getUserDashboardCounts;
            break;
        case "team":
            apiCall = api.getTeamDashboardCounts;
            break;
        default:
            throw new Error("Invalid filter value");
    }

    apiCall().then((resp) => {
        cardCounts.value = resp.data.data;
    }).catch((err) => {
        toast({
            title: 'Something went wrong',
            description: err.response.data.message,
            variant: 'destructive',
        });
    });
};

const getDashboardCharts = () => {
    let apiCall;
    switch (filter.value) {
        case "global":
            apiCall = api.getGlobalDashboardCharts;
            break;
        case "me":
            apiCall = api.getUserDashboardCharts;
            break;
        case "team":
            apiCall = api.getTeamDashboardCharts;
            break;
        default:
            throw new Error("Invalid filter value");
    }

    apiCall().then((resp) => {
        chartData.value = resp.data.data;
        console.log("chart data ->", chartData.value);
    }).catch((err) => {
        toast({
            title: 'Something went wrong',
            description: err.response.data.message,
            variant: 'destructive',
        });
    });
};

</script>
