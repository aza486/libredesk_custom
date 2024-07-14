<template>
  <UserAutoform :initial-values="user" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import UserAutoform from '@/components/admin/team/UserAutoform.vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const user = ref({})

const submitForm = (values) => {
  updateUser(values)
}

const updateUser = async (payload) => {
  try {
    await api.updateUser(user.value.id, payload)
    router.push('/admin/team/users')
  } catch (error) {
    console.log(error)
  }
}

onMounted(async () => {
  try {
    const resp = await api.getUser(props.id)
    user.value = resp.data.data
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
