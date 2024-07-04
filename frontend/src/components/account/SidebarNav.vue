<script setup lang="ts">

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

defineProps({
  currentPage: {
    type: String,
    required: true
  }
})


const sidebarNavItems = [
  {
    title: 'Profile',
    href: '/account/profile',
  },
]
</script>

<template>
  <nav class="flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1">
    <router-link v-for="item in sidebarNavItems" :key="item.title" :to="item.href" custom>
      <template v-slot="{ navigate, isActive }">
        <Button as="a" :href="item.href" variant="ghost" :class="cn(
          'w-full text-left justify-start',
          isActive && 'bg-muted hover:bg-muted'
        )" @click="navigate">
          {{ item.title }}
        </Button>
      </template>
    </router-link>
  </nav>
</template>
