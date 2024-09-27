<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <TemplateForm :initial-values="template" :submitForm="submitForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }" :isLoading="formLoading"/>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TemplateForm from './TemplateForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'

const template = ref({})
const isLoading = ref(false)
const formLoading = ref(false)
const router = useRouter()

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})

const submitForm = async (values) => {
  try {
    formLoading.value = true
    if (props.id) {
      await api.updateTemplate(props.id, values)
    } else {
      await api.createTemplate(values)
      router.push('/admin/templates')
    }
  } finally {
    formLoading.value = false
  }
}

const breadCrumLabel = () => {
  return props.id ? 'Edit' : 'New'
}

const breadcrumbLinks = [
  { path: '/admin/templates', label: 'Templates' },
  { path: '#', label: breadCrumLabel() }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getTemplate(props.id)
      template.value = resp.data.data
    } catch (error) {
      console.log(error)
    } finally {
      isLoading.value = false
    }
  }
})
</script>
