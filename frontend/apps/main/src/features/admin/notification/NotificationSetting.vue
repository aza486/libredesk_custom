<template>
  <AdminSplitLayout>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <Spinner v-if="isLoading" />
        <NotificationsForm :initial-values="initialValues" :inbox-options="inboxOptions" :legacy-configuration="usesLegacyConfiguration" :submit-form="submitForm" />
      </div>
    </template>

    <template #help>
      <p>{{ $t('admin.notification.help.description') }}</p>
      <p>{{ $t('admin.notification.help.detail') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@main/api'
import AdminSplitLayout from '@main/layouts/admin/AdminSplitLayout.vue'
import { useI18n } from 'vue-i18n'
import NotificationsForm from './NotificationSettingForm.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useAppSettingsStore } from '@main/stores/appSettings'

const initialValues = ref({})
const inboxOptions = ref([])
const usesLegacyConfiguration = ref(false)
const { t } = useI18n()
const isLoading = ref(false)
const emitter = useEmitter()
const appSettingsStore = useAppSettingsStore()

onMounted(() => {
  getNotificationSettings()
})

const getNotificationSettings = async () => {
  try {
    isLoading.value = true
    const resp = await api.getEmailNotificationSettings()
    const data = resp.data.data
    usesLegacyConfiguration.value = data['notification.email.inbox_uuid'] == null
    inboxOptions.value = data['notification.email.inbox_options'] || []
    initialValues.value = Object.fromEntries(
      Object.entries(data).filter(([key]) => key !== "notification.email.team_contact_email" && !key.endsWith("inbox_options") && !key.includes("effective_")).map(([key, value]) => [
        key.replace('notification.email.', ''),
        value
      ])
    )
    initialValues.value.inbox_uuid = data['notification.email.inbox_uuid'] ?? data['notification.email.effective_inbox_uuid'] ?? ''
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const submitForm = async (values) => {
  try {
    const updatedValues = Object.fromEntries(
      Object.entries(values).map(([key, value]) => {
        if (key === 'password' && value.includes('•')) {
          return [`notification.email.${key}`, '']
        }
        return [`notification.email.${key}`, value]
      })
    )
    await api.updateEmailNotificationSettings(updatedValues)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('admin.notification.savedLive')
    })
    await getNotificationSettings()
    appSettingsStore.fetchSettings()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
