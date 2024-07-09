<script setup>
import { ref } from 'vue';
import { AutoForm } from '@/components/ui/auto-form'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/toast/use-toast'
import { handleHTTPError } from '@/utils/http'
import { Mail } from 'lucide-vue-next';
import * as z from 'zod'
import api from '@/api'

const { toast } = useToast()

// Step management
const currentStep = ref(1)
const selectedChannel = ref(null)

const channels = ['Email']

const selectChannel = (channel) => {
  selectedChannel.value = channel
  currentStep.value = 2
}

const goBack = () => {
  currentStep.value = 1
  selectedChannel.value = null
}

const submitInboxForm = (values) => {
  // Combine the imap, smtp and from_email_address for the `config` field.
  const channelName = 'email'
  const payload = {
    name: values.name,
    from: values.from,
    channel: channelName,
    config: {
      imap: [values.imap],
      smtp: values.smtp
    }
  }
  createInbox(payload)
}

async function createInbox (payload) {
  try {
    await api.createInbox(payload);
  } catch (error) {
    toast({
      title: 'Could not create inbox.',
      variant: 'destructive',
      description: handleHTTPError(error).message,
    });
  }
}

const emailChannelFormSchema = z.object({
  name: z.string().describe('Name').default(''),
  from: z.string().describe('From address').default(''),
  imap: z.object({
    host: z.string().describe('Host').default('imap.gmail.com'),
    port: z
      .number({
        invalid_type_error: 'Port must be a number.',
      })
      .min(1, {
        message: 'Port must be at least 1.',
      })
      .max(65535, {
        message: 'Port must be at most 65535.',
      })
      .default(993).describe('Port'),
    mailbox: z.string().describe('Mailbox name').default('INBOX'),
    username: z.string().describe('Username'),
    password: z.string().describe('Password'),
    read_interval: z.string().describe('Email scan interval').default('30s'),
  }).describe('IMAP client'),
  smtp: z.array(z.object({
    host: z.string().describe('Host').default('smtp.yourmailserver.com'),
    port: z
      .number({
        invalid_type_error: 'Port must be a number.',
      })
      .min(1, {
        message: 'Port must be at least 1.',
      })
      .max(65535, {
        message: 'Port must be at most 65535.',
      })
      .default(25).describe('Port'),
    username: z.string().describe('Username'),
    password: z.string().describe('Password'),
    max_conns: z
      .number({
        invalid_type_error: 'Must be a number.',
      })
      .min(1, {
        message: 'Must be at least 1.',
      })
      .default(10).describe('Maximum concurrent connections to the server.'),
    max_msg_retries: z
      .number({
        invalid_type_error: 'Must be a number.',
      })
      .min(0, {
        message: 'Must be at least 0.',
      })
      .max(100, {
        message: 'Max retries allowed are 100.',
      })
      .default(2).describe('Number of times to retry when a message fails.'),
    idle_timeout: z.string().default('5s').describe('Time to wait for new activity on a connection before closing it and removing it from the pool (s for second, m for minute).'),
    wait_timeout: z.string().default('5s').describe('Time to wait for new activity on a connection before closing it and removing it from the pool (s for second, m for minute).'),
    auth_protocol: z.enum(['login', 'cram', 'plain', 'none']).default('none').optional(),
  }).describe('SMTP')).describe('SMTP servers'),
});

</script>

<template>
  <div class="p-8 box border">
    <div class="mb-8 flex items-center justify-between">
      <div class="flex items-center space-x-4">
        <div
          :class="['w-8 h-8 flex items-center justify-center rounded-full', currentStep >= 1 ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground']">
          1</div>
        <span :class="[currentStep >= 1 ? 'font-bold text-primary' : 'text-muted']">Choose a channel</span>
      </div>
      <div class="flex-grow h-0.5 bg-muted mx-4"></div>
      <div class="flex items-center space-x-4">
        <div
          :class="['w-8 h-8 flex items-center justify-center rounded-full', currentStep >= 2 ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground']">
          2</div>
        <span :class="[currentStep >= 2 ? 'font-bold text-primary' : 'text-muted']">Configure</span>
      </div>
    </div>

    <div v-if="currentStep === 1" class="space-y-6">
      <h4 class="text-2xl font-semibold text-foreground">Choose a channel</h4>
      <div>
        <Button v-for="channel in channels" :key="channel" @click="selectChannel(channel)" variant="outline" size="sm">
          <Mail size="16" class="mr-2" />
          {{ channel }}
        </Button>
      </div>
    </div>

    <div v-else-if="currentStep === 2" class="space-y-6">
      <Button @click="goBack" variant="outline" size="xs">← Back</Button>
      <h4>Configure your email inbox</h4>
      <div v-if="selectedChannel === 'Email'">
        <AutoForm class="w-11/12 space-y-6" :schema="emailChannelFormSchema" :field-config="{
          name: {
            description: 'Name for your inbox.'
          },
          from: {
            label: 'Email address',
            description: 'From email address. e.g. `Support <support@example.com>`'
          },
          imap: {
            label: 'IMAP',
            password: {
              inputProps: {
                type: 'password',
                placeholder: '••••••••',
              },
            },
            read_interval: {
              label: 'Emails scan interval',
            },
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
                before closing it and removing it from the pool.`,
            },
            wait_timeout: {
              label: 'Wait timeout',
              description: `PoolWaitTimeout is the maximum time to wait to obtain a connection from
              a pool before timing out. This may happen when all open connections are
              busy sending e-mails and they're not returning to the pool fast enough.
              This is also the timeout used when creating new SMTP connections.
              `,
            },
            auth_protocol: {
              label: 'Auth protocol',
            },
            password: {
              inputProps: {
                type: 'password',
                placeholder: '••••••••',
              },
            }
          }
        }" @submit="submitInboxForm">
          <Button type="submit">
            Submit
          </Button>
        </AutoForm>
      </div>
    </div>
  </div>
</template>