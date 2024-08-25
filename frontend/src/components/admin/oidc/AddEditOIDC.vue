<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <OIDCForm :initial-values="oidc" :submitForm="submitForm" :isNewForm="isNewForm" />
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import OIDCForm from './OIDCForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const oidc = ref({
  provider: 'Google'
})
const router = useRouter()

const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const submitForm = async (values) => {
  if (props.id) {
    await api.updateOIDC(props.id, values)
  } else {
    await api.createOIDC(values)
    router.push('/admin/oidc')
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
      const resp = await api.getOIDC(props.id)
      oidc.value = resp.data.data
    } catch (error) {
      console.log(error)
    }
  }
})
</script>
