<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Template name" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="body">
      <FormItem>
        <FormLabel>Body</FormLabel>
        <FormControl>
          <CodeEditor v-model="componentField.modelValue" @update:modelValue="handleChange"></CodeEditor>
        </FormControl>
        <FormDescription>{{ `Make sure the template has \{\{ template "content" . \}\}` }}</FormDescription>
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

    <Button type="submit" size="sm" :isLoading="isLoading"> {{ submitLabel }} </Button>
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
import CodeEditor from '@/components/common/CodeEditor.vue';
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
  },
  isLoading: {
    type: Boolean,
    required: false,
  }
})

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
