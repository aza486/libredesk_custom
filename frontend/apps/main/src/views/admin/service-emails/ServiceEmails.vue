<script setup>
import { useI18n } from 'vue-i18n'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import ServiceEmailManager from '@main/components/sidebar/ServiceEmailManager.vue'
const { t } = useI18n()
</script>

<template>
  <AdminSplitLayout>
    <template #content>
      <div class="space-y-5">
        <h2 class="text-lg font-semibold">{{ t('serviceEmails.title') }}</h2>
        <ServiceEmailManager />
      </div>
    </template>
    <template #help>{{ t('serviceEmails.help') }}</template>
  </AdminSplitLayout>
</template>
