<template>
    <div class="flex flex-col space-y-6" v-if="userStore.getFullName">
        <div>
            <span class="font-medium text-xl space-y-1">
                <p class="font-semibold text-2xl">Hi, {{ userStore.getFullName }}</p>
                <p class="text-muted-foreground text-lg">🌤️ {{ format(new Date(), 'EEEE, MMMM d, HH:mm a') }}</p>
            </span>
        </div>
    </div>
</template>

<script setup>
import { format } from 'date-fns'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()
</script>