<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <RoleForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import RoleForm from './RoleForm.vue'
import api from '@/api'
import { useRouter } from 'vue-router'

const router = useRouter()
const formLoading = ref(false)

const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/roles', label: 'Roles' },
  { path: '#', label: 'Add role' }
]

const submitForm = async (values) => {
  formLoading.value = true
  await api.createRole(values)
  router.push('/admin/teams/roles')
  formLoading.value = false
}
</script>
