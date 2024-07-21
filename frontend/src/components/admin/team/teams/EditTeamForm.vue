<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <TeamForm :initial-values="team" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TeamForm from '@/components/admin/team/teams/TeamForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const router = useRouter()
const team = ref({})

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
    console.log('form payload ', payload)
    await api.updateTeam(team.value.id, payload)
    router.push('/admin/teams/teams')
  } catch (error) {
    console.log(error)
  }
}

onMounted(async () => {
  try {
    const resp = await api.getTeam(props.id)
    team.value = resp.data.data
  } catch (error) {
    console.log(error)
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
