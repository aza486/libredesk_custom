<template>
    <div class="flex justify-between mb-5">
        <PageHeader title="Notifications" description="Manage notifications for agents" />
    </div>
    <div>
        <List :notifications="notifications" />
    </div>
</template>

<script setup>
import List from './NotificationList.vue'
import PageHeader from '../common/PageHeader.vue';

const notifications = [
    {
        name: "New conversation created"
    },
    {
        name: "New conversation created"
    }
]
</script>