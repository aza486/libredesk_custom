<template>
    <div>
        <div class="mb-5">
            <PageHeader title="Conversation" description="Manage conversation settings" />
        </div>
        <div class="flex space-x-5">
            <AdminMenuCard v-for="card in cards" :key="card.title" :onClick="card.onClick" :title="card.title"
                :subTitle="card.subTitle" :icon="card.icon">
            </AdminMenuCard>
        </div>
    </div>
    <router-view></router-view>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { Tag, TrendingUp, Activity } from 'lucide-vue-next'
import AdminMenuCard from '@/components/admin/common/MenuCard.vue'
import PageHeader from '../common/PageHeader.vue';

const router = useRouter()

const navigateToTags = () => {
    router.push('/admin/conversations/tags')
}

const navigateToStatus = () => {
    router.push('/admin/conversations/status')
}

const navigateToPriority = () => {
    router.push('/admin/conversations/priority')
}

const cards = [
    {
        title: 'Tags',
        subTitle: 'Manage conversation tags.',
        onClick: navigateToTags,
        icon: Tag
    },
    {
        title: 'Priority',
        subTitle: 'Manage conversation priorities.',
        onClick: navigateToPriority,
        icon: Activity
    },
    {
        title: 'Status',
        subTitle: 'Manage conversation statuses.',
        onClick: navigateToStatus,
        icon: TrendingUp
    },
]
</script>