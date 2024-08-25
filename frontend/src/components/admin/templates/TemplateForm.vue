<template>
  <form @submit="onSubmit" class="w-2/3 space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Template name" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="body">
      <FormItem v-auto-animate>
        <FormLabel>HTML body</FormLabel>
        <FormControl>
          <Textarea placeholder="HTML here.." v-bind="componentField" class="h-52" />
        </FormControl>
        <FormDescription>{{ templateBodyDescription() }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="is_default" v-slot="{ value, handleChange }">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>Is default</Label>
          </div>
        </FormControl>
        <FormDescription>There can be only one default template.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" size="sm"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
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
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'

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
    default: () => 'Save'
  }
})

const templateBodyDescription = () => 'Make sure the template has {{ .Content }}'

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
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
  { deep: true }
)
</script>
