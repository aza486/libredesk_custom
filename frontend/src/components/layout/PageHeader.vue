<template>
  <div v-if="!isHidden">
    <div class="flex items-center space-x-4 h-12 px-2">
      <SidebarTrigger class="cursor-pointer w-4 h-4" />
      <span class="text-xl font-semibold text-gray-800">
        {{ title }}
      </span>
    </div>
    <Separator />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { useRoute } from 'vue-router'

const route = useRoute()
const title = computed(() => route.meta.title || '')
const isHidden = computed(() => route.meta.hidePageHeader === true)
</script>
