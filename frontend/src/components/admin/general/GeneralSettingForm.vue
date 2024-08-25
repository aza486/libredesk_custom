<template>
  <form @submit="onSubmit" class="w-2/3 space-y-6">
    <FormField v-slot="{ field }" name="site_name">
      <FormItem v-auto-animate>
        <FormLabel>Site Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Site Name" v-bind="field" />
        </FormControl>
        <FormDescription>Enter the site name</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="lang">
      <FormItem>
        <FormLabel>Language</FormLabel>
        <FormControl>
          <Select v-bind="field" :modelValue="field.value">
            <SelectTrigger>
              <SelectValue placeholder="Select a language" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="en"> English </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>Select language for the app.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="root_url">
      <FormItem v-auto-animate>
        <FormLabel>Root URL</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Root URL" v-bind="field" />
        </FormControl>
        <FormDescription>Root URL of the app.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="favicon_url" :value="props.initialValues.favicon_url">
      <FormItem v-auto-animate>
        <FormLabel>Favicon URL</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Favicon URL" v-bind="field" />
        </FormControl>
        <FormDescription>Favicon URL for the app.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ field }"
      name="max_file_upload_size"
      :value="props.initialValues.max_file_upload_size"
    >
      <FormItem v-auto-animate>
        <FormLabel>Max allowed file upload size</FormLabel>
        <FormControl>
          <Input type="number" placeholder="10" v-bind="field" />
        </FormControl>
        <FormDescription>In megabytes.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="allowed_file_upload_extensions" v-slot="{ componentField }">
      <FormItem>
        <FormLabel>Allowed file extensions</FormLabel>
        <FormControl>
          <TagsInput v-model="componentField.modelValue">
            <TagsInputItem v-for="item in componentField.modelValue" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="jpg" />
          </TagsInput>
        </FormControl>
        <FormDescription>Use `*` to allow any file.</FormDescription>
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
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@/components/ui/tags-input'
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
