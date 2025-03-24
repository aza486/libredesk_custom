<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input
            type="text"
            placeholder="Template name"
            v-bind="componentField"
            :disabled="!isOutgoingTemplate"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="subject" v-if="!isOutgoingTemplate">
      <FormItem>
        <FormLabel>Subject</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Subject for email" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="body">
      <FormItem>
        <FormLabel>Body</FormLabel>
        <FormControl>
          <CodeEditor
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
          />
        </FormControl>
        <FormDescription v-if="isOutgoingTemplate">
          {{ `Make sure the template has \{\{ template "content" . \}\} only once.` }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="is_default" v-slot="{ value, handleChange }" v-if="isOutgoingTemplate">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>Is default</Label>
          </div>
        </FormControl>
        <FormDescription>You can have only one default outgoing email template.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, computed } from 'vue'
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
import CodeEditor from '@/components/editor/CodeEditor.vue'
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
    required: false
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

const isOutgoingTemplate = computed(() => {
  return props.initialValues?.type === 'email_outgoing'
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
