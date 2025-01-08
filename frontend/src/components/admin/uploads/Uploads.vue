<template>
  <div>
    
  </div>
  <div>
    <component
      :is="formProvider === 's3' ? S3Form : LocalFsForm"
      :submitForm="submitForm"
      :initialValues="initialValues"
      submitLabel="Save"
      @provider-update="handleProviderUpdate"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import S3Form from './S3Form.vue'
import LocalFsForm from './LocalFsForm.vue'
import api from '@/api'


const initialValues = ref({})

onMounted(async () => {
  const resp = await api.getSettings('upload')
  const config = resp.data.data
  const modifiedConfig = {}
  for (const key in config) {
    if (key.startsWith('upload.localfs.')) {
      const newKey = key.replace('upload.localfs.', '')
      modifiedConfig[newKey] = config[key]
    } else if (key.startsWith('upload.s3')) {
      const newKey = key.replace('upload.s3.', '')
      modifiedConfig[newKey] = config[key]
    } else if (key.startsWith('upload.')) {
      const newKey = key.replace('upload.', '')
      modifiedConfig[newKey] = config[key]
    }
  }
  initialValues.value = modifiedConfig
})

const submitForm = async (values) => {
  const prefixedValues = {}
  for (const key in values) {
    if (values.provider === 'localfs') {
      prefixedValues[`upload.localfs.${key}`] = values[key]
    } else if (values.provider === 's3') {
      prefixedValues[`upload.s3.${key}`] = values[key]
    }
  }
  prefixedValues['upload.provider'] = values.provider
  await api.updateSettings('upload', prefixedValues)
}

const formProvider = computed(() => {
  return initialValues.value.provider
})

const handleProviderUpdate = (value) => {
  initialValues.value.provider = value
}
</script>
