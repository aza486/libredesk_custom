<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <RoleForm :initial-values="role" :submitForm="submitForm" :isLoading="formLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import RoleForm from './RoleForm.vue'
import { useToast } from '@/components/ui/toast/use-toast'
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'

const role = ref({})
const isLoading = ref(false)
const formLoading = ref(false)
const { toast } = useToast()
const props = defineProps({
  id: {
    type: String,
    required: true
  }
})

onMounted(async () => {
  isLoading.value = true
  const resp = await api.getRole(props.id)
  role.value = resp.data.data
  isLoading.value = false
})

const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/roles', label: 'Roles' },
  { path: '#', label: 'Edit role' }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.updateRole(props.id, values)
    toast({
      description: 'Role saved successfully'
    })
  } finally {
    formLoading.value = false
  }
}
</script>
