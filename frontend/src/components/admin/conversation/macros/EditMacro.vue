<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <MacroForm :initialValues="macro" :submitForm="submitForm" :isLoading="formLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import MacroForm from '@/components/admin/conversation/macros/MacroForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'

const macro = ref({})
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()

const breadcrumbLinks = [
  { path: '/admin/conversations/macros', label: 'Macros' },
  { path: '#', label: 'Edit macro' }
]

const submitForm = (values) => {
  updateMacro(values)
}

const updateMacro = async (payload) => {
  try {
    formLoading.value = true
    await api.updateMacro(macro.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Saved',
      description: 'Macro updated successfully'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not update macro',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getMacro(props.id)
    macro.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
