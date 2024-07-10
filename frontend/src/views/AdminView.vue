<script setup>
import Admin from '@/components/admin/Admin.vue'

defineProps({
  page: {
    type: String,
    default: 'inboxes'
  }
})
</script>

<template>
  <Admin class="h-screen overflow-auto">
    <router-view></router-view>
  </Admin>
</template>
