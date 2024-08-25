<template>
  <nav class="flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1">
    <router-link v-for="item in navItems" :key="item.title" :to="item.href">
      <template v-slot="{ navigate, isActive }">
        <Button
          as="a"
          :href="item.href"
          variant="ghost"
          :class="
            cn(
              'w-full text-left justify-start h-16 pl-3',
              isActive || isChildActive(item.href) ? 'bg-muted hover:bg-muted' : ''
            )
          "
          @click="navigate"
        >
          <div class="flex flex-col items-start space-y-1">
            <span class="text-sm">{{ item.title }}</span>
            <p class="text-xs-muted break-words whitespace-normal">{{ item.description }}</p>
          </div>
        </Button>
      </template>
    </router-link>
  </nav>
</template>

<script setup>
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { defineProps } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

defineProps({
  navItems: {
    type: Array,
    required: true,
    default: () => []
  }
})

const isChildActive = (href) => {
  return route.path.startsWith(href)
}
</script>
