<template>
    <div class="flex flex-col space-y-6" v-if="userStore.getFullName">
        <div>
            <span class="font-medium text-3xl space-y-1">
                <p>Hi, {{ userStore.getFullName }}</p>
                <p class="text-sm-muted">🌤️ {{ format(new Date(), 'EEEE, MMMM d, HH:mm a') }}</p>
            </span>
        </div>
    </div>
</template>

<script setup>
import { format } from 'date-fns'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()
</script>