<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <!-- Basic Fields -->
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('form.field.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription> {{ $t('admin.inbox.name.description') }} </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="from">
      <FormItem>
        <FormLabel>{{ $t('form.field.from_email_address') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="t('admin.inbox.from_email_address.placeholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>
          {{ $t('admin.inbox.from_email_address.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Toggle Fields -->
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('form.field.enabled') }}</FormLabel>
          <FormDescription>{{ $t('admin.inbox.enabled.description') }}</FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('admin.inbox.csat_surveys') }}</FormLabel>
          <FormDescription>
            {{ $t('admin.inbox.csat_surveys.description_1') }}<br />
            {{ $t('admin.inbox.csat_surveys.description_2') }}
          </FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- IMAP Section -->
    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.imap_configuration') }}</h3>

      <FormField v-slot="{ componentField }" name="imap.host">
        <FormItem>
          <FormLabel>Host</FormLabel>
          <FormControl>
            <Input type="text" placeholder="imap.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.port">
        <FormItem>
          <FormLabel>{{ $t('form.field.port') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="993" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.mailbox">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.mailbox') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              placeholder="INBOX"
              v-bind="componentField"
              :defaultValue="'INBOX'"
            />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.mailbox.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.username">
        <FormItem>
          <FormLabel>{{ $t('form.field.username') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="inbox@example.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.password">
        <FormItem>
          <FormLabel>{{ $t('form.field.password') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="••••••••" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.tls_type">
        <FormItem>
          <FormLabel>TLS</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('form.field.selectTLS')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">OFF</SelectItem>
                <SelectItem value="tls">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.imap_tls.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.read_interval">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.imap_scan_interval') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="5m" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.imap_scan_interval.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.scan_inbox_since">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.imap_scan_inbox_since') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="48h" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.imap_scan_inbox_since.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="imap.tls_skip_verify">
        <FormItem class="flex flex-row items-center justify-between box p-4">
          <div class="space-y-0.5">
            <FormLabel class="text-base">{{ $t('admin.inbox.skipTLSVerification') }}</FormLabel>
            <FormDescription>
              {{ $t('admin.inbox.skipTLSVerification.description') }}
            </FormDescription>
          </div>
          <FormControl>
            <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <!-- SMTP Section -->
    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.smtp_configuration') }}</h3>

      <FormField v-slot="{ componentField }" name="smtp.host">
        <FormItem>
          <FormLabel>{{ $t('form.field.host') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="smtp.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.port">
        <FormItem>
          <FormLabel>{{ $t('form.field.port') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="587" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.username">
        <FormItem>
          <FormLabel>{{ $t('form.field.username') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="user@example.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.password">
        <FormItem>
          <FormLabel>{{ $t('form.field.password') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="••••••••" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_conns">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.max_connections') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="10" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.max_connections.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_msg_retries">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.max_retries') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="3" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.max_retries.description') }} </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.idle_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.idle_timeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="25s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.idle_timeout.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.wait_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.wait_timeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="60s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.wait_timeout.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.auth_protocol">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.auth_protocol') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue placeholder="Select protocol" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="login">Login</SelectItem>
                <SelectItem value="cram">CRAM</SelectItem>
                <SelectItem value="plain">Plain</SelectItem>
                <SelectItem value="none">None</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription> {{ $t('admin.inbox.auth_protocol.description') }} </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.tls_type">
        <FormItem>
          <FormLabel>{{ t('admin.inbox.tls') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue placeholder="Select TLS" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">OFF</SelectItem>
                <SelectItem value="tls">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription> {{ $t('admin.inbox.tls.description') }} </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.hello_hostname">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.helo_hostname') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.helo_hostname.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="smtp.tls_skip_verify">
        <FormItem class="flex flex-row items-center justify-between box p-4">
          <div class="space-y-0.5">
            <FormLabel class="text-base">{{ $t('admin.inbox.skipTLSVerification') }}</FormLabel>
            <FormDescription>
              {{ $t('admin.inbox.skipTLSVerification.description') }}
            </FormDescription>
          </div>
          <FormControl>
            <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
})

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.buttons.save')
})

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    form.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
