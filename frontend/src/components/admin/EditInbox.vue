<template>
  <EmailInboxAutoform :initial-values="inbox" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import EmailInboxAutoform from '@/components/admin/EmailInboxAutoform.vue'
import { useRouter } from 'vue-router'
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
    await api.updateInbox(inbox.value.id, payload)
    router.push('/admin/inboxes')
  } catch (error) {
    console.log(error)
  }
}

onMounted(async () => {
  try {
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
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
