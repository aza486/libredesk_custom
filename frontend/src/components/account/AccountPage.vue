<script setup>
import PageHeader from '@/components/common/PageHeader.vue'
import SidebarNav from '@/components/common/SidebarNav.vue'

const sidebarNavItems = [
  {
    title: 'Profile',
    href: '/account/profile',
    description: 'Update your profile'
  }
]
</script>

<template>
  <div class="space-y-4 md:block page-content">
    <PageHeader title="Account settings" subTitle="Manage your account settings." />
    <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
      <div class="flex-1 lg:max-w-3xl admin-main-content min-h-[700px]">
        <div class="space-y-6">
          <slot></slot>
        </div>
      </div>
    </div>
  </div>
</template>
