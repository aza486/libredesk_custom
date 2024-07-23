<script setup lang="ts">
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

const sidebarNavItems = [
  {
    title: 'Inboxes',
    href: '/admin/inboxes',
    description: 'Create and manage your inboxes'
  },
  {
    title: 'Teams',
    href: '/admin/teams',
    description: 'Create teams, manage agents and roles'
  },
  {
    title: 'Automations',
    href: '/admin/automations',
    description: 'Create automations and time triggers'
  },
  {
    title: 'Notifications',
    href: '/admin/notifications',
    description: 'Manage notifications for your agents'
  }
]
</script>

<template>
  <nav class="flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1">
    <router-link v-for="item in sidebarNavItems" :key="item.title" :to="item.href">
      <template v-slot="{ navigate, isActive }">
        <Button as="a" :href="item.href" variant="ghost" :class="cn(
          'w-full text-left justify-start rounded-none h-20',
          isActive && 'bg-muted hover:bg-muted'
        )
          " @click="navigate">
          <div class="flex flex-col space-y-1">
            <span class="admin-subtitle">{{ item.title }}</span>
            <p class="text-muted-foreground text-xs">{{ item.description }}</p>
          </div>
        </Button>
      </template>
    </router-link>
  </nav>
</template>
