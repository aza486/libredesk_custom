<template>
    <div>
        <div class="flex flex-col box border p-5 mb-5 rounded-lg" v-for="notification in notifications" :key="notification">
            <div class="flex items-center space-x-2">
                <Switch id="airplane-mode" />
                <Label for="airplane-mode">{{ notification.name }}</Label>
            </div>
        </div>
    </div>
</template>

<script setup>
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

defineProps({
    notifications: {
        type: Array,
        required: true,
    },
})

</script>