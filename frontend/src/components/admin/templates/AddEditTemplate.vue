<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <TemplateForm :initial-values="template" :submitForm="submitForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }" :isLoading="formLoading" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TemplateForm from './TemplateForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'

const template = ref({})
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()
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
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: "Saved"
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not save template',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
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
