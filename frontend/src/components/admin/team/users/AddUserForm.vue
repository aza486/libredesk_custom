<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <UserForm :submitForm="onSubmit" :initialValues="{}" :isNewForm="true" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import UserForm from '@/components/admin/team/users/UserForm.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import api from '@/api'

const { toast } = useToast()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/users', label: 'Users' },
  { path: '#', label: 'Add user' }
]

const onSubmit = (values) => {
  createNewUser(values)
}

const createNewUser = async (values) => {
  try {
    formLoading.value = true
    await api.createUser(values)
  } catch (error) {
    toast({
      title: 'Could not create user.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
