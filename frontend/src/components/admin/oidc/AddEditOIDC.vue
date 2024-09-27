<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <OIDCForm :initial-values="oidc" :submitForm="submitForm" :isNewForm="isNewForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }" :isLoading="formLoading" />
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import OIDCForm from './OIDCForm.vue'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const oidc = ref({
  provider: 'Google'
})
const isLoading = ref(false)
const formLoading = ref(false)
const router = useRouter()

const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const submitForm = async (values) => {
  try {
    formLoading.value = true
    if (props.id) {
      await api.updateOIDC(props.id, values)
    } else {
      await api.createOIDC(values)
      router.push('/admin/oidc')
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
      console.log(error)
    } finally {
      isLoading.value = false
    }
  }
})
</script>
