<script setup>
import * as z from 'zod'
import { Button } from '@/components/ui/button'
import { AutoForm } from '@/components/ui/auto-form'
import { handleHTTPError } from '@/utils/http'
import api from '@/api';
import { useToast } from '@/components/ui/toast/use-toast'

const { toast } = useToast()

const schema = z.object({
  first_name: z
    .string({
      required_error: 'First name is required.',
    })
    .min(2, {
      message: 'First name must be at least 2 characters.',
    }),

  last_name: z.string().optional(),

  email: z
    .string({
      required_error: 'Email is required.',
    })
    .email({
      message: 'Invalid email address.',
    }),

  team: z.enum(['red', 'green', 'blue']).optional(),

  send_welcome_email: z.boolean().optional(),
})


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
      description: handleHTTPError(error).message,
    });
  }
}
</script>

<template>
  <AutoForm class="w-2/3 space-y-6" :schema="schema" :field-config="{
    first_name: { label: 'First name' },
    last_name: { label: 'Last name' },
    email: { label: 'Email' },
    team: { label: 'Team' },
    send_welcome_email: { label: 'Send welcome email' },
  }" @submit="onSubmit">
    <Button type="submit">
      Submit
    </Button>
  </AutoForm>
</template>
