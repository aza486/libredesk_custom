<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <UserForm :submitForm="onSubmit" :initialValues="{}" :isNewForm="true" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import UserForm from '@/features/admin/users/UserForm.vue'
import { handleHTTPError } from '@/utils/http'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const emitter = useEmitter()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'user-list', label: 'Users' },
  { path: '', label: 'Add user' }
]

const onSubmit = (values) => {
  createNewUser(values)
}

const createNewUser = async (values) => {
  try {
    formLoading.value = true
    await api.createUser(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      description: 'User created successfully'
    })
    router.push({ name: 'user-list' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
