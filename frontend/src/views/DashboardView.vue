<template>
  <div class="page-content">
    <Spinner v-if="isLoading"></Spinner>
    <div class="space-y-4">
      <div class="text-sm text-gray-500 text-right">
        Last updated: {{ new Date(lastUpdate).toLocaleTimeString() }}
      </div>
      <div class="mt-7 flex w-full space-x-4" v-auto-animate>
        <Card title="Open conversations" :counts="cardCounts" :labels="agentCountCardsLabels" />
        <Card class="w-8/12" title="Agent status" :counts="sampleAgentStatusCounts" :labels="sampleAgentStatusLabels" />
      </div>
      <div class="dashboard-card p-5">
        <LineChart :data="chartData.processedData"></LineChart>
      </div>
      <div class="dashboard-card p-5">
        <BarChart :data="chartData.status_summary"></BarChart>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import Card from '@/components/dashboard/DashboardCard.vue'
import LineChart from '@/components/dashboard/DashboardLineChart.vue'
import BarChart from '@/components/dashboard/DashboardBarChart.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'

const { toast } = useToast()
const isLoading = ref(false)
const cardCounts = ref({})
const chartData = ref({})
const lastUpdate = ref(new Date())
let updateInterval

const agentCountCardsLabels = {
  open: 'Total',
  awaiting_response: 'Awaiting Response',
  unassigned: 'Unassigned',
  pending: 'Pending'
}

// TODO: Build agent status feature.
const sampleAgentStatusLabels = {
  online: 'Online',
  offline: 'Offline',
  away: 'Away'
}
const sampleAgentStatusCounts = {
  online: 5,
  offline: 2,
  away: 1
}

onMounted(() => {
  getDashboardData()
  startRealtimeUpdates()
})

onUnmounted(() => {
  stopRealtimeUpdates()
})

const startRealtimeUpdates = () => {
  updateInterval = setInterval(() => {
    getDashboardData()
    lastUpdate.value = new Date()
  }, 60000)
}

const stopRealtimeUpdates = () => {
  clearInterval(updateInterval)
}

const getDashboardData = () => {
  isLoading.value = true
  Promise.all([getCardStats(), getDashboardCharts()])
    .finally(() => {
      isLoading.value = false
    })
}

const getCardStats = async () => {
  return api.getOverviewCounts()
    .then((resp) => {
      cardCounts.value = resp.data.data
    })
    .catch((err) => {
      toast({
        title: 'Error',
        description: err.response.data.message,
        variant: 'destructive'
      })
    })
}

const getDashboardCharts = async () => {
  return api.getOverviewCharts()
    .then((resp) => {
      chartData.value.new_conversations = resp.data.data.new_conversations || []
      chartData.value.resolved_conversations = resp.data.data.resolved_conversations || []
      chartData.value.messages_sent = resp.data.data.messages_sent || []
      chartData.value.processedData = resp.data.data.new_conversations.map(item => ({
        date: item.date,
        'New conversations': item.count,
        'Resolved conversations': resp.data.data.resolved_conversations.find(r => r.date === item.date)?.count || 0,
        'Messages sent': resp.data.data.messages_sent.find(r => r.date === item.date)?.count || 0
      }))
      chartData.value.status_summary = resp.data.data.status_summary || []
    })
    .catch((err) => {
      toast({
        title: 'Error',
        description: err.response.data.message,
        variant: 'destructive'
      })
    })
}
</script>
