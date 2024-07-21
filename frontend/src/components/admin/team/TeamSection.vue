<template>
  <div>
    <div class="mb-10">
      <span class="admin-title">Teams</span>
      <p class="text-muted-foreground text-sm">Create agents, teams and set working hours.</p>
    </div>
    <div class="flex space-x-5">
      <AdminMenuCard v-for="card in cards" :key="card.title" :onClick="card.onClick" :title="card.title"
        :subtitle="card.subtitle" :icon="card.icon">
      </AdminMenuCard>
    </div>
  </div>
  <router-view></router-view>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { Users, UserRoundCog } from 'lucide-vue-next'
import AdminMenuCard from '@/components/admin/common/MenuCard.vue'

const router = useRouter()

const navigateToUsers = () => {
  router.push('/admin/teams/users')
}

const navigateToTeams = () => {
  router.push('/admin/teams/users')
}

const navigateToRoles = () => {
  router.push('/admin/teams/roles')
}

const cards = [
  {
    title: 'Users',
    subtitle: 'Create and manage users.',
    onClick: navigateToUsers,
    icon: UserRoundCog
  },
  {
    title: 'Teams',
    subtitle: 'Create and manage teams.',
    onClick: navigateToTeams,
    icon: Users
  },
  {
    title: 'Roles',
    subtitle: 'Create and manage roles.',
    onClick: navigateToRoles,
    icon: Users
  },
]
</script>
