<template>
    <div class="flex justify-between mb-5">
        <div>
            <span class="admin-title">Notifications</span>
            <p class="text-muted-foreground text-sm">Manage notifications.</p>
        </div>
    </div>
    <div>
        <List :notifications="notifications" />
    </div>
</template>

<script setup>
import List from './NotificationList.vue'
const notifications = [
    {
        name: "New conversation created"
    },
    {
        name: "New conversation created"
    }
]
</script>