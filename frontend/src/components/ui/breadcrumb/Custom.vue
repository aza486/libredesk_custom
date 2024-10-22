<template>
  <Breadcrumb>
    <BreadcrumbList>
      <BreadcrumbItem v-for="(item, index) in links" :key="index">
        <router-link :to="item.path">
          {{ item.label }}
        </router-link>
        <BreadcrumbSeparator v-if="index < links.length - 1">
          <ChevronRight />
        </BreadcrumbSeparator>
      </BreadcrumbItem>
    </BreadcrumbList>
  </Breadcrumb>
</template>

<script setup>
import { ChevronRight } from 'lucide-vue-next'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbSeparator
} from '@/components/ui/breadcrumb'

defineProps({
  links: {
    type: Array,
    required: true,
    default: () => []
  }
})
</script>
