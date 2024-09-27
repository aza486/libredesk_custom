<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <TeamForm :initial-values="team" :submitForm="submitForm" :isLoading="formLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TeamForm from '@/components/admin/team/teams/TeamForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'

const team = ref({})
const formLoading = ref(false)
const isLoading = ref(false)

const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/teams', label: 'Teams' },
  { path: '#', label: 'Edit team' }
]

const submitForm = (values) => {
  updateTeam(values)
}

const updateTeam = async (payload) => {
  try {
    formLoading.value = true
    await api.updateTeam(team.value.id, payload)
  } catch (error) {
    console.log(error)
  } finally {
    formLoading.value = false
  }
}

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getTeam(props.id)
    team.value = resp.data.data
  } catch (error) {
    console.log(error)
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
