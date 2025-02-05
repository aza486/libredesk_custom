<template>
  <form
    @submit="onSmtpSubmit"
    class="space-y-6"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
  >
    <!-- Enabled Field -->
    <FormField name="enabled" v-slot="{ value, handleChange }">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>Enabled</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- SMTP Host Field -->
    <FormField v-slot="{ componentField }" name="host">
      <FormItem>
        <FormLabel>SMTP Host</FormLabel>
        <FormControl>
          <Input type="text" placeholder="smtp.gmail.com" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- SMTP Port Field -->
    <FormField v-slot="{ componentField }" name="port">
      <FormItem>
        <FormLabel>SMTP Port</FormLabel>
        <FormControl>
          <Input type="number" placeholder="587" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Username Field -->
    <FormField v-slot="{ componentField }" name="username">
      <FormItem>
        <FormLabel>Username</FormLabel>
        <FormControl>
          <Input type="text" placeholder="admin@yourcompany.com" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Password Field -->
    <FormField v-slot="{ componentField }" name="password">
      <FormItem>
        <FormLabel>Password</FormLabel>
        <FormControl>
          <Input type="password" placeholder="Enter your password" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Max Connections Field -->
    <FormField v-slot="{ componentField }" name="max_conns">
      <FormItem>
        <FormLabel>Max Connections</FormLabel>
        <FormControl>
          <Input type="number" placeholder="2" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Idle Timeout Field -->
    <FormField v-slot="{ componentField }" name="idle_timeout">
      <FormItem>
        <FormLabel>Idle Timeout</FormLabel>
        <FormControl>
          <Input type="text" placeholder="15s" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Wait Timeout Field -->
    <FormField v-slot="{ componentField }" name="wait_timeout">
      <FormItem>
        <FormLabel>Wait Timeout</FormLabel>
        <FormControl>
          <Input type="text" placeholder="5s" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Authentication Protocol Field -->
    <FormField v-slot="{ componentField }" name="auth_protocol">
      <FormItem>
        <FormLabel>Authentication Protocol</FormLabel>
        <FormControl>
          <Select v-bind="componentField" v-model="componentField.modelValue">
            <SelectTrigger>
              <SelectValue placeholder="Select an authentication protocol" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="plain">Plain</SelectItem>
                <SelectItem value="login">Login</SelectItem>
                <SelectItem value="cram">CRAM-MD5</SelectItem>
                <SelectItem value="none">None</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Email Address Field -->
    <FormField v-slot="{ componentField }" name="email_address">
      <FormItem>
        <FormLabel>From Email Address</FormLabel>
        <FormControl>
          <Input
            type="text"
            placeholder="From email address. e.g. My Support <mysupport@example.com>"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
        <FormDescription
          >From email address. e.g. My Support &lt;mysupport@example.com&gt;</FormDescription
        >
      </FormItem>
    </FormField>

    <!-- Max Message Retries Field -->
    <FormField v-slot="{ componentField }" name="max_msg_retries">
      <FormItem>
        <FormLabel>Max Message Retries</FormLabel>
        <FormControl>
          <Input type="number" placeholder="2" v-bind="componentField" />
        </FormControl>
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
import { smtpConfigSchema } from './formSchema.js'
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
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
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
  isLoading: {
    type: Boolean,
    required: false
  }
})

const smtpForm = useForm({
  validationSchema: toTypedSchema(smtpConfigSchema)
})

const onSmtpSubmit = smtpForm.handleSubmit((values) => {
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    smtpForm.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
