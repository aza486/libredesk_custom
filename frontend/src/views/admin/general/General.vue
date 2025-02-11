<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <GeneralSettingForm
          :submitForm="submitForm"
          :initial-values="initialValues"
          submitLabel="Save"
        />
        <Spinner v-if="isLoading" />
      </div>
    </template>
    <template #help>
      <p>
        Configure core helpdesk settings like helpdesk name, timezone, business hours, and more.
      </p>
      <p>These settings affect your entire helpdesk system.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Spinner } from '@/components/ui/spinner'
import GeneralSettingForm from '@/features/admin/general/GeneralSettingForm.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'

import api from '@/api'

const initialValues = ref({})
const isLoading = ref(false)

onMounted(async () => {
  isLoading.value = true
  const response = await api.getSettings('general')
  const data = response.data.data
  isLoading.value = false
  initialValues.value = Object.keys(data).reduce((acc, key) => {
    // Remove 'app.' prefix
    const newKey = key.replace(/^app\./, '')
    acc[newKey] = data[key]
    return acc
  }, {})
})

const submitForm = async (values) => {
  // Prepend keys with `app.`
  const updatedValues = Object.fromEntries(
    Object.entries(values).map(([key, value]) => [`app.${key}`, value])
  )
  await api.updateSettings('general', updatedValues)
}
</script>
