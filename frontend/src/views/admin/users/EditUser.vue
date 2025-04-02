<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"/>
  <UserForm :initialValues="user" :submitForm="submitForm" :isLoading="formLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import UserForm from '@/features/admin/users/UserForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import { useI18n } from 'vue-i18n'

const user = ref({})
const { t } = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()

const breadcrumbLinks = [
  { path: 'user-list', label: t('globals.terms.user', 2) },
  {
    path: '',
    label: t('globals.messages.edit', {
      name: t('globals.terms.user', 1)
    })
  }
]

const submitForm = (values) => {
  updateUser(values)
}

const updateUser = async (payload) => {
  try {
    formLoading.value = true
    await api.updateUser(user.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.user', 1)
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
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
    const resp = await api.getUser(props.id)
    user.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
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
