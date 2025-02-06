<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>
    <template #help>
      <p>Combine multiple conversation actions into single-click macros.</p>
      <p>Create reusable action sets for common agent responses.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
