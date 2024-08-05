<template>
  <div>
    <div class="mb-5">
      <PageHeader title="Teams" description="Manage teams, users and roles" />
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
import { Users, UserRoundCog } from 'lucide-vue-next'
import AdminMenuCard from '@/components/admin/common/MenuCard.vue'
import PageHeader from '../common/PageHeader.vue';

const router = useRouter()

const navigateToUsers = () => {
  router.push('/admin/teams/users')
}

const navigateToTeams = () => {
  router.push('/admin/teams/teams')
}

const navigateToRoles = () => {
  router.push('/admin/teams/roles')
}

const cards = [
  {
    title: 'Users',
    subTitle: 'Create and manage users.',
    onClick: navigateToUsers,
    icon: UserRoundCog
  },
  {
    title: 'Teams',
    subTitle: 'Create and manage teams.',
    onClick: navigateToTeams,
    icon: Users
  },
  {
    title: 'Roles',
    subTitle: 'Create and manage roles.',
    onClick: navigateToRoles,
    icon: Users
  },
]
</script>
