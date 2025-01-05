<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <TeamForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import TeamForm from '@/components/admin/team/teams/TeamForm.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import api from '@/api'

const formLoading = ref(false)
const router = useRouter()
const emitter = useEmitter()
const breadcrumbLinks = [
  { path: '/admin/teams/teams', label: 'Teams' },
  { path: '/admin/teams/teams/new', label: 'New team' }
]

const submitForm = (values) => {
  createTeam(values)
}

const createTeam = async (values) => {
  try {
    formLoading.value = true
    await api.createTeam(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Saved',
      description: "Team created successfully"
    })
    router.push('/admin/teams/teams')
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
