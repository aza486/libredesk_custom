<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <RoleForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import RoleForm from '@/features/admin/roles/RoleForm.vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useRouter } from 'vue-router'

const emitter = useEmitter()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: '/admin/teams/roles', label: 'Roles' },
  { path: '#', label: 'Add role' }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.createRole(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      description: 'Role created successfully'
    })
    router.push('/admin/teams/roles')
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not create role',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
