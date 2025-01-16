<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <MacroForm
    :submitForm="onSubmit"
    :isLoading="formLoading"
  />
</template>

<script setup>
import { ref } from 'vue'
import MacroForm from '@/components/admin/conversation/macros/MacroForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { handleHTTPError } from '@/utils/http'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const router = useRouter()
const emit = useEmitter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: '/admin/conversations/macros', label: 'Macros' },
  { path: '#', label: 'New macro' }
]

const onSubmit = (values) => {
  createMacro(values)
}

const createMacro = async (values) => {
  try {
    formLoading.value = true
    await api.createMacro(values)
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      variant: 'success',
      description: 'Macro created successfully'
    })
    router.push('/admin/conversations/macros')
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
