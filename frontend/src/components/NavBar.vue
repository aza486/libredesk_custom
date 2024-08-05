<script setup>
import { RouterLink, useRoute } from 'vue-router'
import { cn } from '@/lib/utils'

import { Icon } from '@iconify/vue'
import { buttonVariants } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from '@/components/ui/tooltip'

defineProps({
  isCollapsed: Boolean,
  links: Array
})

const route = useRoute()

const getFirstSegment = (path) => {
  return path.split('/')[1]
}

const getButtonVariant = (to) => {
  const currentSegment = getFirstSegment(route.path.toLowerCase())
  const targetSegment = getFirstSegment(to.toLowerCase())
  return currentSegment === targetSegment ? '' : 'ghost'
}
</script>

<template>
  <div :data-collapsed="isCollapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
    <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
      <template v-for="(link, index) of links">
        <!-- Collapsed -->
        <router-link :to="link.to" v-if="isCollapsed" :key="`1-${index}`">
          <TooltipProvider :delay-duration="10">
            <Tooltip>
              <TooltipTrigger as-child>
                <span :class="cn(
                  buttonVariants({ variant: getButtonVariant(link.to), size: 'icon' }),
                  'h-9 w-9',
                  link.variant === getButtonVariant(link.to) &&
                  'dark:bg-muted dark:text-muted-foreground dark:hover:bg-muted dark:hover:text-white'
                )
                  ">
                  <Icon :icon="link.icon" class="size-5" />
                  <span class="sr-only">{{ link.title }}</span>
                </span>
              </TooltipTrigger>
              <TooltipContent side="right" class="flex items-center gap-4">
                {{ link.title }}
                <span v-if="link.label" class="ml-auto text-muted-foreground">
                  {{ link.label }}
                </span>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </router-link>

        <!-- Expanded -->
        <router-link v-else :to="link.to" :key="`2-${index}`" :class="cn(
          buttonVariants({ variant: getButtonVariant(link.to), size: 'sm' }),
          link.variant === getButtonVariant(link.to) &&
          'dark:bg-muted dark:text-white dark:hover:bg-muted dark:hover:text-white',
          'justify-start'
        )
          ">
          <Icon :icon="link.icon" class="mr-2 size-5" />
          {{ link.title }}
          <span v-if="link.label" :class="cn(
            'ml-',
            link.variant === getButtonVariant(link.to) && 'text-background dark:text-white'
          )
            ">
            {{ link.label }}
          </span>
        </router-link>
      </template>
    </nav>
  </div>
</template>
