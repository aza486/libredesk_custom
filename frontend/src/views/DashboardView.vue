<script setup>
import { onMounted, ref } from 'vue';
import { useUserStore } from '@/stores/user'
import { format } from 'date-fns'
import api from '@/api';
import { useToast } from '@/components/ui/toast/use-toast'

import CountCards from '@/components/dashboard/agent/CountCards.vue'
import ConversationsOverTime from '@/components/dashboard/agent/ConversationsOverTime.vue';

const { toast } = useToast()
const counts = ref({})
const newConversationsStats = ref([])
const userStore = useUserStore()
const agentCountCardsLabels = {
    total_assigned: "Total Assigned",
    unresolved_count: "Unresolved",
    awaiting_response_count: "Awaiting Response",
    created_today_count: "Created Today"
};

onMounted(() => {
    getCardStats()
    getnewConversationsStatsStats()
})

const getCardStats = () => {
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

const getnewConversationsStatsStats = () => {
    api.getNewConversationsStats().then((resp) => {
        newConversationsStats.value = resp.data.data
    }).catch((err) => {
        toast({
            title: 'Uh oh! Something went wrong.',
            description: err.response.data.message,
            variant: 'destructive',
        })
    })
}

function getGreeting () {
    const now = new Date();
    const hours = now.getHours();

    if (hours >= 5 && hours < 12) {
        return "Good morning";
    } else if (hours >= 12 && hours < 17) {
        return "Good afternoon";
    } else if (hours >= 17 && hours < 21) {
        return "Good evening";
    } else {
        return "Good night";
    }
}

</script>

<template>
    <div class="page-content">
        <div v-if="userStore.getFullName">
            <h4 class="scroll-m-20 text-2xl font-semibold tracking-tight">
                <p>{{ getGreeting() }}, {{ userStore.getFullName }}</p>
                <p class="text-xl text-muted-foreground">🌤️ {{ format(new Date(), "EEEE, MMMM d HH:mm a") }}</p>
            </h4>
        </div>
        <CountCards :counts="counts" :labels="agentCountCardsLabels" />
        <!-- <AssignedByStatusDonut /> -->
        <div class="flex my-10">
            <ConversationsOverTime class="flex-1" :data=newConversationsStats />
        </div>
    </div>
</template>