<template>
  <div class="page-content">
    <PageHeader title="Overview" />
    <Spinner v-if="isLoading"></Spinner>
    <div class="mt-7 flex w-full space-x-4" v-auto-animate>
      <Card class="flex-1" title="Open conversations" :counts="cardCounts" :labels="agentCountCardsLabels" />
      <Card class="flex-1" title="Agent status" :counts="sampleAgentStatusCounts" :labels="sampleAgentStatusLabels" />
    </div>
    <div class="w-11/12" :class="{ 'soft-fade': isLoading }">
      <div class="flex my-7 justify-between items-center space-x-5">
        <div class="dashboard-card p-5">
          <LineChart :data="chartData.new_conversations"></LineChart>
        </div>
        <div class="dashboard-card p-5">
          <BarChart :data="chartData.status_summary"></BarChart>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'

import { vAutoAnimate } from '@formkit/auto-animate/vue'
import Card from '@/components/dashboard/DashboardCard.vue'
import LineChart from '@/components/dashboard/DashboardLineChart.vue'
import BarChart from '@/components/dashboard/DashboardBarChart.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'

const { toast } = useToast()
const isLoading = ref(false)
const cardCounts = ref({})
const chartData = ref({})
const agentCountCardsLabels = {
  total_count: 'Total',
  resolved_count: 'Resolved',
  unresolved_count: 'Unresolved',
  awaiting_response_count: 'Awaiting Response'
}
const sampleAgentStatusLabels = {
  online: 'Online',
  offline: 'Offline',
  away: 'Away',
}

const sampleAgentStatusCounts = {
  online: 5,
  offline: 2,
  away: 1,
}

onMounted(() => {
  getDashboardData()
})

const getDashboardData = () => {
  isLoading.value = true
  Promise.all([getCardStats(), getDashboardCharts()])
    .finally(() => {
      isLoading.value = false
    })
}

const getCardStats = () => {
  return api.getGlobalDashboardCounts()
    .then((resp) => {
      cardCounts.value = resp.data.data
    })
    .catch((err) => {
      toast({
        title: 'Something went wrong',
        description: err.response.data.message,
        variant: 'destructive'
      })
    })
}

const getDashboardCharts = () => {
  return api.getGlobalDashboardCharts()
    .then((resp) => {
      chartData.value.new_conversations = resp.data.data.new_conversations || []
      chartData.value.status_summary = resp.data.data.status_summary || []
    })
    .catch((err) => {
      toast({
        title: 'Something went wrong',
        description: err.response.data.message,
        variant: 'destructive'
      })
    })
}
</script>
