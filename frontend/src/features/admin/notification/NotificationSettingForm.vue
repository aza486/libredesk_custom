<template>
  <form @submit="onSmtpSubmit" class="space-y-6">
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
        <FormDescription> Maximum concurrent connections to the server. </FormDescription>
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
        <FormDescription>
          Time to wait for new activity on a connection before closing it and removing it from the
          pool (s for second, m for minute)
        </FormDescription>
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
        <FormDescription>
          Time to wait for new activity on a connection before closing it and removing it from the
          pool (s for second, m for minute, h for hour).
        </FormDescription>
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
        <FormDescription> Number of times to retry when a message fails. </FormDescription>
      </FormItem>
    </FormField>

    <!-- HELO Hostname Field -->
    <FormField v-slot="{ componentField }" name="hello_hostname">
      <FormItem>
        <FormLabel>HELO Hostname</FormLabel>
        <FormControl>
          <Input type="text" placeholder="smtp.example.com" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          The hostname to use in the HELO/EHLO command. If not set, defaults to localhost.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- TLS Type Field -->
    <FormField v-slot="{ componentField }" name="tls_type">
      <FormItem>
        <FormLabel>TLS</FormLabel>
        <FormControl>
          <Select v-bind="componentField" v-model="componentField.modelValue">
            <SelectTrigger>
              <SelectValue placeholder="Select a TLS type" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="none">Off</SelectItem>
                <SelectItem value="tls">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Skip TLS Verification Field -->
    <FormField v-slot="{ componentField, handleChange }" name="tls_skip_verify">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">Skip TLS Verification</FormLabel>
          <FormDescription> Skip hostname check on the TLS certificate. </FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, ref } from 'vue'
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
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

const isLoading = ref(false)
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

const smtpForm = useForm({
  validationSchema: toTypedSchema(smtpConfigSchema)
})

const onSmtpSubmit = smtpForm.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    await props.submitForm(values)
  } finally {
    isLoading.value = false
  }
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
