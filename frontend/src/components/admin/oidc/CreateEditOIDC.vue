<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <OIDCForm
    :initial-values="oidc"
    :submitForm="submitForm"
    :isNewForm="isNewForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    :isLoading="formLoading"
  />
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import OIDCForm from './OIDCForm.vue'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useRouter } from 'vue-router'

const router = useRouter()
const oidc = ref({
  provider: 'Google'
})
const emitter = useEmitter()
const isLoading = ref(false)
const formLoading = ref(false)
const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const submitForm = async (values) => {
  try {
    formLoading.value = true
    let toastDescription = ''
    if (props.id) {
      if (values.client_secret.includes('•')) {
        values.client_secret = ''
      }
      await api.updateOIDC(props.id, values)
      toastDescription = 'Provider updated successfully'
    } else {
      await api.createOIDC(values)
      toastDescription = 'Provider created successfully'
      router.push('/admin/oidc')
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      description: toastDescription
    })
  } catch (error) {
    if (props.id) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error reloading OIDC providers',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } else {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  } finally {
    formLoading.value = false
  }
}

const breadCrumLabel = () => {
  return props.id ? 'Edit' : 'New'
}

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: '/admin/oidc', label: 'OIDC' },
  { path: '#', label: breadCrumLabel() }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getOIDC(props.id)
      oidc.value = resp.data.data
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  }
})
</script>
