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
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'

const emitter = useEmitter()
const role = ref({})
const isLoading = ref(false)
const formLoading = ref(false)
const props = defineProps({
  id: {
    type: String,
    required: true
  }
})

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getRole(props.id)
    role.value = resp.data.data

  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not fetch role',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const breadcrumbLinks = [
  
  { path: '/admin/teams/roles', label: 'Roles' },
  { path: '#', label: 'Edit role' }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.updateRole(props.id, values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Saved',
      description: 'Role updated successfully'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not update role',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
