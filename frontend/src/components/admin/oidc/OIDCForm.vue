<template>
  <form @submit="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="provider">
      <FormItem>
        <FormLabel>Provider</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select a provider" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="Google"> Google </SelectItem>
                <SelectItem value="Custom"> Custom </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Google" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="provider_url">
      <FormItem v-auto-animate>
        <FormLabel>Provider URL</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Provider URL" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="client_id">
      <FormItem v-auto-animate>
        <FormLabel>Client ID</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Client ID" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="client_secret">
      <FormItem v-auto-animate>
        <FormLabel>Client Secret</FormLabel>
        <FormControl>
          <Input type="password" placeholder="Client Secret" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="redirect_uri" v-if="!isNewForm">
      <FormItem v-auto-animate>
        <FormLabel>Redirect URI</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Redirect URI" v-bind="componentField" readonly />
        </FormControl>
        <FormDescription>Set this URI for callback.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="enabled" v-slot="{ value, handleChange }" v-if="!isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>Enabled</Label>
          </div>
        </FormControl>
        <FormDescription></FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { oidcLoginFormSchema } from './formSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
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
    default: () => 'Save'
  },
  isNewForm: {
    type: Boolean
  },
  isLoading: {
    type: Boolean,
    required: false
  },
})

const form = useForm({
  validationSchema: toTypedSchema(oidcLoginFormSchema)
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
  { deep: true, immediate: true }
)
</script>
