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
import { Tag, TrendingUp, MessageCircleReply } from 'lucide-vue-next'
import AdminMenuCard from '@/components/admin/common/MenuCard.vue'
import PageHeader from '../common/PageHeader.vue'

const router = useRouter()

const navigateToTags = () => {
  router.push('/admin/conversations/tags')
}

const navigateToStatus = () => {
  router.push('/admin/conversations/statuses')
}

const navigateToCannedResponse = () => {
  router.push('/admin/conversations/canned-responses')
}

const cards = [
  {
    title: 'Tags',
    subTitle: 'Manage conversation tags.',
    onClick: navigateToTags,
    icon: Tag
  },
  {
    title: 'Canned response',
    subTitle: 'Manage canned responses.',
    onClick: navigateToCannedResponse,
    icon: MessageCircleReply
  },
  {
    title: 'Status',
    subTitle: 'Manage conversation statuses.',
    onClick: navigateToStatus,
    icon: TrendingUp
  }
]
</script>
