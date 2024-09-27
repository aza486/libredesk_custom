<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <UserForm :initialValues="user" :submitForm="submitForm" :isLoading="formLoading" v-else/>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import UserForm from '@/components/admin/team/users/UserForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'

const user = ref({})
const isLoading = ref(false)
const formLoading = ref(false)

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
    formLoading.value = true
    await api.updateUser(user.value.id, payload)
  } catch (error) {
    console.log(error)
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
