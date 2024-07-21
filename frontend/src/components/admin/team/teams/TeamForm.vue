<template>
  <form @submit="onSubmit" class="w-2/3 space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Name" v-bind="componentField" />
        </FormControl>
        <FormDescription> Select an unique name </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit"> {{submitLabel}} </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { teamFormSchema } from './teamFormSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => 'Submit'
  },
})

const form = useForm({
  validationSchema: toTypedSchema(teamFormSchema),
})

const onSubmit = form.handleSubmit((values) => {
  console.log(" form vaal ", values)
  console.log(" init val ", props.initialValues)
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    console.log("Setting ->" , newValues)
    form.setValues(newValues)
  },
  { immediate: true }
)
</script>
