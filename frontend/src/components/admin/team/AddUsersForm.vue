<template>
  <UserAutoform :submitForm="onSubmit" :initialValues="{}" :isNewForm="true" />
</template>

<script setup>
import { handleHTTPError } from '@/utils/http'
import UserAutoform from '@/components/admin/team/UserAutoform.vue'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'

const { toast } = useToast()

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
