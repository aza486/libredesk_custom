<template>
  <AdminPageWithHelp>
    <template #content>
      <GeneralSettingForm
        :submitForm="submitForm"
        :initial-values="initialValues"
        submitLabel="Save"
      />
    </template>

    <template #help>
      <p>Configure core helpdesk settings like helpdesk name, timezone, business hours, and more.</p>
      <p>
        These settings affect your entire helpdesk system.
      </p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import GeneralSettingForm from '@/features/admin/general/GeneralSettingForm.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'

import api from '@/api'
const initialValues = ref({})

onMounted(async () => {
  const response = await api.getSettings('general')
  const data = response.data.data
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
