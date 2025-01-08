<template>
  <AutoForm
    class="space-y-6"
    :schema="formSchema"
    :form="form"
    :field-config="{
      name: {
        description: 'Name for your inbox.'
      },
      from: {
        label: 'From email address',
        description: 'From email address. e.g. My Support <mysupport@example.com>'
      },
      csat_enabled: {
        label: 'CSAT',
        description: 'Send a CSAT survey after a conversation is marked as resolved.'
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
    }"
    @submit="submitForm"
  >
    <Button type="submit" :is-loading="isLoading"> {{ props.submitLabel }} </Button>
  </AutoForm>
</template>

<script setup>
import { watch } from 'vue'
import { AutoForm } from '@/components/ui/auto-form'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
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
  },
  isLoading: {
    type: Boolean,
    required: false,
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: props.initialValues
})

// Watch for changes in initialValues and update the form
watch(
  () => props.initialValues,
  (newValues) => {
    if (newValues) form.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
