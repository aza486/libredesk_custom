<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <UserForm :initial-values="user" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import UserForm from '@/components/admin/team/users/UserForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const router = useRouter()
const user = ref({})

const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/users', label: 'Users' },
  { path: '#', label: 'Edit user' }
]

const submitForm = (values) => {
  updateUser(values)
}

const updateUser = async (payload) => {
  try {
    await api.updateUser(user.value.id, payload)
    router.push('/admin/teams/users')
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
