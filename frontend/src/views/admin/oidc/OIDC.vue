<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view/>
    </template>

    <template #help>
      <p>Configure single sign-on with one or more OpenID Connect providers.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
