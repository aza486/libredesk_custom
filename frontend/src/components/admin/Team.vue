<template>
  <div v-if="!isSubpageRoute">
    <div>
      <h2
        class="scroll-m-20 pb-2 text-3xl font-semibold tracking-tight transition-colors first:mt-0"
      >
        Teams
      </h2>
      <p>Create agents, teams and set working hours.</p>
    </div>
    <div class="flex space-x-10">
      <router-link v-for="item in options" :key="item.title" :to="item.href">
        <template v-slot="{ navigate, isActive }">
          <div
            :class="
              cn(
                'w-full text-left justify-start px-14 py-3 cursor-pointer',
                isActive && 'bg-muted hover:bg-muted'
              )
            "
            @click="navigate"
          >
            {{ item.title }}
          </div>
        </template>
      </router-link>
    </div>
  </div>
  <router-view></router-view>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { cn } from '@/lib/utils'
const route = useRoute()
const options = [
  {
    title: 'Users',
    href: '/admin/team/users',
    description: 'Manage users'
  },
  {
    title: 'Roles',
    href: '/admin/team/roles',
    description: 'Manage roles'
  }
]
const isSubpageRoute = computed(() =>
  ['/admin/team/users', '/admin/team/roles'].includes(route.path)
)
</script>
