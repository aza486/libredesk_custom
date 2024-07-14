<template>
  <form @submit="onSubmit" class="w-2/3 space-y-6">
    <FormField v-slot="{ componentField }" name="first_name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Name" v-bind="componentField" />
        </FormControl>
        <FormDescription> Select an unique name for the team. </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit"> Submit </Button>
  </form>
</template>

<script setup>
import { watch, onMounted, ref } from 'vue'
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
import api from '@/api'

const teams = ref([])

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
  isNewForm: {
    type: Boolean,
    required: false,
    default: () => false
  }
})

onMounted(async () => {
  try {
    const resp = await api.getTeams()
    teams.value = resp.data.data
  } catch (err) {
    console.log(err)
  }
})

const form = useForm({
  validationSchema: toTypedSchema(teamFormSchema),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true }
)
</script>
