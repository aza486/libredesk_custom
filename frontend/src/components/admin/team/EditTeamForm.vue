<template>
  <TeamForm :initial-values="team" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TeamForm from '@/components/admin/team/TeamForm.vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const team = ref({})

const submitForm = (values) => {
  updateTeam(values)
}

const updateTeam = async (payload) => {
  try {
    await api.updateTeam(team.value.id, payload)
    router.push('/admin/team/teams')
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
