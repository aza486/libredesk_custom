<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <UserForm :submitForm="onSubmit" :initialValues="{}" :isNewForm="true" />
</template>

<script setup>
import UserForm from '@/components/admin/team/users/UserForm.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import api from '@/api'

const { toast } = useToast()
const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/users', label: 'Users'},
  { path: '#', label: 'Add user' }
]

const onSubmit = (values) => {
  createNewUser(values)
}

const createNewUser = async (values) => {
  try {
    await api.createUser(values)
  } catch (error) {
    toast({
      title: 'Could not create user.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
