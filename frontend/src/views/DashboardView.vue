<script setup>
import { onMounted, ref } from 'vue';
import { useUserStore } from '@/stores/user'
import { format } from 'date-fns'
import api from '@/api';
import { useToast } from '@/components/ui/toast/use-toast'

import CountCards from '@/components/dashboard/agent/CountCards.vue'

const { toast } = useToast()
const counts = ref({})
const userStore = useUserStore()
const agentCountCardsLabels = {
    total_assigned: "Total Assigned",
    unresolved_count: "Unresolved",
    awaiting_response_count: "Awaiting Response",
    created_today_count: "Created Today"
};

onMounted(() => {
    getStats()
})

const getStats = () => {
    api.getAssigneeStats().then((resp) => {
        counts.value = resp.data.data
    }).catch((err) => {
        toast({
            title: 'Uh oh! Something went wrong.',
            description: err.response.data.message,
            variant: 'destructive',
        })
    })
}
</script>

<template>
    <div class="tab-container-default">
        <div v-if="userStore.getFullName">
            <h4 class="scroll-m-20 text-2xl font-semibold tracking-tight">
                <p>Good morning, {{ userStore.getFullName }}</p>
                <p class="text-xl text-muted-foreground">🌤️ {{ format(new Date(), "EEEE, MMMM d HH:mm a") }}</p>
            </h4>
        </div>
        <CountCards :counts="counts" :labels="agentCountCardsLabels" />
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