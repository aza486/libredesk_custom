<template>
  <form @submit="onLocalFsSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="provider">
      <FormItem>
        <FormLabel>Provider</FormLabel>
        <FormControl>
          <Select
            v-bind="componentField"
            v-model="componentField.modelValue"
            @update:modelValue="handleProviderUpdate"
          >
            <SelectTrigger>
              <SelectValue placeholder="Select a provider" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="s3"> S3 </SelectItem>
                <SelectItem value="localfs"> Local filesystem </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="upload_path">
      <FormItem v-auto-animate>
        <FormLabel>Upload path</FormLabel>
        <FormControl>
          <Input type="text" placeholder="/home/ubuntu/uploads" v-bind="componentField" />
        </FormControl>
        <FormDescription> Path to the directory where files will be uploaded. </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { localFsFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
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
  }
})

const emit = defineEmits(['provider-update'])

const localFsForm = useForm({
  validationSchema: toTypedSchema(localFsFormSchema)
})

const onLocalFsSubmit = localFsForm.handleSubmit((values) => {
  props.submitForm(values)
})

const handleProviderUpdate = (value) => {
  emit('provider-update', value)
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    localFsForm.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
