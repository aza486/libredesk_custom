<template>
    <div class="page-content w-11/12">
        <div class="flex flex-col space-y-6">
            <div>
                <span class="font-medium text-2xl space-y-1" v-if="userStore.getFullName">
                    <p>Hi, {{ userStore.getFullName }}</p>
                    <p class="text-sm-muted">🌤️ {{ format(new Date(), "EEEE, MMMM d, HH:mm a") }}</p>
                </span>
            </div>
            <div>
                <Select :model-value="filter" @update:model-value="onDashboardFilterChange">
                    <SelectTrigger class="w-[130px]">
                        <SelectValue placeholder="Select a filter" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectGroup>
                            <SelectItem value="me">
                                Mine
                            </SelectItem>
                            <SelectItem value="all_teams">
                                All teams
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
            <div class="dashboard-card p-5">
                <Select :model-value="lineChartFilter" @update:model-value="onLineChartFilterChange">
                    <SelectTrigger class="w-[7rem] text-xs">
                        <SelectValue placeholder="Select a filter" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectGroup>
                            <SelectItem value="last_week">
                                Last week
                            </SelectItem>
                            <SelectItem value="last_month">
                                Last month
                            </SelectItem>
                            <SelectItem value="last_year">
                                Last year
                            </SelectItem>
                        </SelectGroup>
                    </SelectContent>
                </Select>
                <LineChart :data="chartData.new_conversations"> </LineChart>
            </div>
            <div class="dashboard-card p-5">
                <Select :model-value="barChartFilter" @update:model-value="onBarChartFilterChange">
                    <SelectTrigger class="w-[7rem] text-xs">
                        <SelectValue placeholder="Select a filter" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectGroup>
                            <SelectItem value="last_week">
                                Last week
                            </SelectItem>
                            <SelectItem value="last_month">
                                Last month
                            </SelectItem>
                            <SelectItem value="last_year">
                                Last year
                            </SelectItem>
                        </SelectGroup>
                    </SelectContent>
                </Select>
                <BarChart :data="chartData.status_summary">
                </BarChart>
            </div>
        </div>
    </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue';
import { useUserStore } from '@/stores/user'
import { format, subWeeks, subMonths, subYears, formatISO } from 'date-fns'
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
const userStore = useUserStore()
const cardCounts = ref({})
const chartData = ref({})
const filter = ref("me")
const barChartFilter = ref("last_week")
const lineChartFilter = ref("last_week")
const agentCountCardsLabels = {
    total_count: "Total",
    resolved_count: "Resolved",
    unresolved_count: "Unresolved",
    awaiting_response_count: "Awaiting Response",
};

const chartFilters = {
    last_week: getLastWeekRange(),
    last_month: getLastMonthRange(),
    last_year: getLastYearRange(),
};
``
onMounted(() => {
    getCardStats()
    getDashboardCharts()
})

watch(filter, () => {
    getDashboardCharts()
    getCardStats()
})


const onDashboardFilterChange = (v) => {
    filter.value = v
}

const onLineChartFilterChange = (v) => {
    console.log("chart filter  -> ", chartFilters)
    lineChartFilter.value = v
}

const onBarChartFilterChange = (v) => {
    barChartFilter.value = v
}

function getLastWeekRange () {
    const today = new Date();
    const lastWeekStart = subWeeks(today, 1);
    return {
        start: formatISO(lastWeekStart, { representation: 'date' }),
        end: formatISO(today, { representation: 'date' }),
    };
}

function getLastMonthRange () {
    const today = new Date();
    const lastMonthStart = subMonths(today, 1);
    return {
        start: formatISO(lastMonthStart, { representation: 'date' }),
        end: formatISO(today, { representation: 'date' }),
    };
}

function getLastYearRange () {
    const today = new Date();
    const lastYearStart = subYears(today, 1);
    return {
        start: formatISO(lastYearStart, { representation: 'date' }),
        end: formatISO(today, { representation: 'date' }),
    };
}

const getCardStats = () => {
    let apiCall;
    switch (filter.value) {
        case "all_teams":
            apiCall = api.getGlobalDashboardCounts;
            break;
        case "me":
            apiCall = api.getUserDashboardCounts;
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
        case "all_teams":
            apiCall = api.getGlobalDashboardCharts;
            break;
        case "me":
            apiCall = api.getUserDashboardCharts;
            break;
        default:
            throw new Error("Invalid filter value");
    }

    apiCall().then((resp) => {
        chartData.value = resp.data.data;
    }).catch((err) => {
        toast({
            title: 'Something went wrong',
            description: err.response.data.message,
            variant: 'destructive',
        });
    });
};

</script>
