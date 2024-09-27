<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="formLoading"></Spinner>
  <EmailInboxForm :initialValues="inbox" :submitForm="submitForm" :isLoading="isLoading" v-else/>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import EmailInboxForm from '@/components/admin/inbox/EmailInboxForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb/index.js'
import { Spinner } from '@/components/ui/spinner'

const breadcrumbLinks = [
  { path: '/admin/inboxes', label: 'Inboxes' },
  { path: '#', label: 'Edit Inbox' }
]

const formLoading = ref(false)
const isLoading = ref(false)
const router = useRouter()
const inbox = ref({})

const submitForm = (values) => {
  const channelName = inbox.value.channel
  const payload = {
    name: values.name,
    from: values.from,
    channel: channelName,
    config: {
      imap: [values.imap],
      smtp: values.smtp
    }
  }
  updateInbox(payload)
}

const updateInbox = async (payload) => {
  try {
    isLoading.value = true
    await api.updateInbox(inbox.value.id, payload)
    router.push('/admin/inboxes')
  } catch (error) {
    console.log(error)
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  try {
    formLoading.value = true
    const resp = await api.getInbox(props.id)
    let inboxData = resp.data.data

    // Modify the inbox data as per the zod schema.
    if (inboxData?.config?.imap) {
      inboxData.imap = inboxData?.config?.imap[0]
    }
    if (inboxData?.config?.smtp) {
      inboxData.smtp = inboxData?.config?.smtp
    }

    inbox.value = inboxData
  } catch (error) {
    console.log(error)
  } finally {
    formLoading.value = false
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
