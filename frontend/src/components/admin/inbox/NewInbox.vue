<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div>
    <div class="mb-8 flex items-center justify-between">
      <div class="flex items-center space-x-4">
        <div
          :class="[
            'w-8 h-8 flex items-center justify-center rounded-full',
            currentStep >= 1
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          ]"
        >
          1
        </div>
        <span :class="[currentStep >= 1 ? 'font-bold text-primary' : 'text-muted']"
          >Choose a channel</span
        >
      </div>
      <div class="flex-grow h-0.5 bg-muted mx-4"></div>
      <div class="flex items-center space-x-4">
        <div
          :class="[
            'w-8 h-8 flex items-center justify-center rounded-full',
            currentStep >= 2
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          ]"
        >
          {{ currentStep >= 2 ? 2 : '' }}
        </div>
        <span :class="[currentStep >= 2 ? 'font-bold text-primary' : 'text-muted']">Configure</span>
      </div>
    </div>

    <div v-if="currentStep === 1" class="space-y-6">
      <h4 class="text-xl text-foreground">Choose a channel</h4>
      <div>
        <Button
          v-for="channel in channels"
          :key="channel"
          @click="selectChannel(channel)"
          variant="outline"
          size="default"
        >
          <Mail size="16" class="mr-2" />
          {{ channel }}
        </Button>
      </div>
    </div>

    <div v-else-if="currentStep === 2" class="space-y-6">
      <Button @click="goBack" variant="link" size="xs">← Back</Button>
      <h4>Configure your email inbox</h4>
      <div v-if="selectedChannel === 'Email'">
        <EmailInboxForm :initial-values="{}" :submitForm="submitForm" />
      </div>
    </div>
    <div v-else>
      <Button @click="goInboxList" variant="link" size="xs" class="mt-10">← Back</Button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/toast/use-toast'
import { handleHTTPError } from '@/utils/http'
import { Mail } from 'lucide-vue-next'
import EmailInboxForm from '@/components/admin/inbox/EmailInboxForm.vue'
import api from '@/api'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb/index.js'

const breadcrumbLinks = [
  { path: '/admin/inboxes', label: 'Inboxes' },
  { path: '#', label: 'New Inbox' },
]

const router = useRouter()
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

const goInboxList = () => {
  router.push('/admin/inboxes')
}

const submitForm = (values) => {
  const channelName = selectedChannel.value.toLowerCase()
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

async function createInbox(payload) {
  try {
    await api.createInbox(payload)
    router.push('/admin/inboxes')
  } catch (error) {
    toast({
      title: 'Could not create inbox.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
