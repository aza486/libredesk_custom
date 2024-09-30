<script setup>
import { computed } from 'vue'
import PageHeader from '@/components/common/PageHeader.vue'
import SidebarNav from '@/components/common/SidebarNav.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

const allNavItems = [
  {
    title: 'General',
    href: '/admin/general',
    description: 'Configure general app settings',
    permission: null,
  },
  {
    title: 'Conversations',
    href: '/admin/conversations',
    description: 'Manage tags, canned responses and statuses.',
    permission: null
  },
  {
    title: 'Inboxes',
    href: '/admin/inboxes',
    description: 'Manage your inboxes',
    permission: null,
  },
  {
    title: 'Teams',
    href: '/admin/teams',
    description: 'Manage teams, manage agents and roles',
    permission: null,
  },
  {
    title: 'Automations',
    href: '/admin/automations',
    description: 'Manage automations and time triggers',
    permission: null,
  },
  {
    title: 'Notification',
    href: '/admin/notification',
    description: 'Manage email notification settings',
    permission: null,
  },
  {
    title: 'Templates',
    href: '/admin/templates',
    description: 'Manage email templates',
    permission: null,
  },
  {
    title: 'OpenID Connect SSO',
    href: '/admin/oidc',
    description: 'Manage OpenID SSO configurations',
    permission: null,
  }
]

const sidebarNavItems = computed(() =>
  allNavItems.filter((item) => userStore.hasPermission(item.permission))
)
</script>

<template>
  <div class="space-y-4 md:block overflow-y-auto">
    <PageHeader title="Admin settings" subTitle="Manage your helpdesk settings." />
    <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-10 lg:space-y-5">
      <aside class="lg:w-1/6 md:w-1/7 h-[calc(100vh-10rem)] border-r pr-3">
        <SidebarNav :navItems="sidebarNavItems" />
      </aside>
      <div class="flex-1 lg:max-w-5xl admin-main-content min-h-[700px]">
        <div class="space-y-6">
          <slot></slot>
        </div>
      </div>
    </div>
  </div>
</template>
