<template>
  <div
    class="page-content p-4 pr-36"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
  >
    <Spinner v-if="isLoading" />
    <div class="space-y-4">
      <div class="text-sm text-gray-500 text-right">
        Last updated: {{ new Date(lastUpdate).toLocaleTimeString() }}
      </div>
      <div class="mt-7 flex w-full space-x-4">
        <Card title="Open conversations" :counts="cardCounts" :labels="agentCountCardsLabels" />
        <Card
          class="w-8/12"
          title="Agent status"
          :counts="sampleAgentStatusCounts"
          :labels="sampleAgentStatusLabels"
        />
      </div>
      <div class="rounded-lg box w-full p-5 bg-white">
        <LineChart :data="chartData.processedData"></LineChart>
      </div>
      <div class="rounded-lg box w-full p-5 bg-white">
        <BarChart :data="chartData.status_summary"></BarChart>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import Card from '@/features/reports/DashboardCard.vue'
import LineChart from '@/features/reports/DashboardLineChart.vue'
import BarChart from '@/features/reports/DashboardBarChart.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import api from '@/api'

const emitter = useEmitter()
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
  Promise.allSettled([getCardStats(), getDashboardCharts()]).finally(() => {
    isLoading.value = false
  })
}

const getCardStats = async () => {
  return api
    .getOverviewCounts()
    .then((resp) => {
      cardCounts.value = resp.data.data
    })
    .catch((error) => {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    })
}

const getDashboardCharts = async () => {
  return api
    .getOverviewCharts()
    .then((resp) => {
      chartData.value.new_conversations = resp.data.data.new_conversations || []
      chartData.value.resolved_conversations = resp.data.data.resolved_conversations || []
      chartData.value.messages_sent = resp.data.data.messages_sent || []
      chartData.value.processedData = resp.data.data.new_conversations.map((item) => ({
        date: item.date,
        'New conversations': item.count,
        'Resolved conversations':
          resp.data.data.resolved_conversations.find((r) => r.date === item.date)?.count || 0,
        'Messages sent': resp.data.data.messages_sent.find((r) => r.date === item.date)?.count || 0
      }))
      chartData.value.status_summary = resp.data.data.status_summary || []
    })
    .catch((error) => {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    })
}
</script>
