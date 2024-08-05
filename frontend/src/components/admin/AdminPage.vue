<script setup>
import PageHeader from '@/components/common/PageHeader.vue';
import SidebarNav from '@/components/common/SidebarNav.vue'

const sidebarNavItems = [
  {
    title: 'General',
    href: '/admin/general',
    description: 'Configure general app settings'
  },
  {
    title: 'Inboxes',
    href: '/admin/inboxes',
    description: 'Manage your inboxes'
  },
  {
    title: 'Teams',
    href: '/admin/teams',
    description: 'Manage teams, manage agents and roles'
  },
  {
    title: 'Automations',
    href: '/admin/automations',
    description: 'Manage automations and time triggers'
  },
  {
    title: 'Notifications',
    href: '/admin/notifications',
    description: 'Manage notifications for your agents'
  },
  {
    title: 'Email templates',
    href: '/admin/email-templates',
    description: 'Manage email templates'
  },
  {
    title: 'File uploads',
    href: '/admin/uploads',
    description: 'Manage file upload settings'
  },
  {
    title: 'Social Login',
    href: '/admin/oidc',
    description: 'Configure social login'
  },
]
</script>

<template>
  <div class="space-y-4 md:block">
    <PageHeader title="Admin settings" subTitle="Manage your helpdesk settings." />
    <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-10 lg:space-y-5">
      <aside class="lg:w-1/6 md:w-1/7 h-[calc(100vh-10rem)] border-r pr-3">
        <SidebarNav :navItems="sidebarNavItems" />
      </aside>
      <div class="flex-1 lg:max-w-4xl admin-main-content min-h-[700px]">
        <div class="space-y-6">
          <slot></slot>
        </div>
      </div>
    </div>
  </div>
</template>
