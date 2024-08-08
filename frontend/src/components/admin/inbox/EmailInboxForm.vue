<template>
  <AutoForm class="w-11/12 space-y-6" :schema="emailChannelFormSchema" :form="form" :field-config="{
    name: {
      description: 'Name for your inbox.'
    },
    from: {
      label: 'Email address',
      description: 'From email address. e.g. My Support <mysupport@example.com>'
    },
    imap: {
      label: 'IMAP',
      password: {
        inputProps: {
          type: 'password',
          placeholder: '••••••••'
        }
      },
      read_interval: {
        label: 'Emails scan interval'
      }
    },
    smtp: {
      label: 'SMTP',
      max_conns: {
        label: 'Max connections',
        description: 'Maximum number of concurrent connections to the server.'
      },
      max_msg_retries: {
        label: 'Retries',
        description: 'Number of times to retry when a message fails.'
      },
      idle_timeout: {
        label: 'Idle timeout',
        description: `IdleTimeout is the maximum time to wait for new activity on a connection
        before closing it and removing it from the pool.`
      },
      wait_timeout: {
        label: 'Wait timeout',
        description: `PoolWaitTimeout is the maximum time to wait to obtain a connection from
      a pool before timing out. This may happen when all open connections are
      busy sending e-mails and they're not returning to the pool fast enough.
      This is also the timeout used when creating new SMTP connections.
      `
      },
      auth_protocol: {
        label: 'Auth protocol'
      },
      password: {
        inputProps: {
          type: 'password',
          placeholder: '••••••••'
        }
      }
    }
  }" @submit="submitForm">
    <Button type="submit"> {{ props.submitLabel }} </Button>
  </AutoForm>
</template>

<script setup>
import { watch } from 'vue'
import { AutoForm } from '@/components/ui/auto-form'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { emailChannelFormSchema } from './emailChannelFormSchema.js'
import { Button } from '@/components/ui/button'

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
  validationSchema: toTypedSchema(emailChannelFormSchema),
  initialValues: props.initialValues
})

// Watch for changes in initialValues and update the form
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
  },
  { deep: true }
)
</script>
