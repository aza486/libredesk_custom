<template>
  <div v-if="showTable">
    <div class="flex justify-between mb-5">
      <div>
        <span class="admin-title">Inboxes</span>
        <p class="text-muted-foreground text-sm">Create and manage inboxes.</p>
      </div>
      <div class="flex justify-end mb-4">
        <Button @click="navigateToAddInbox" size="sm"> New inbox </Button>
      </div>
    </div>
    <div class="w-full">
      <DataTable :columns="columns" :data="data" />
    </div>
  </div>
  <div v-else>
    <router-view></router-view>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/components/admin/InboxDataTableColumns.js'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import { useRouter } from 'vue-router'
import api from '@/api'

const { toast } = useToast()
const router = useRouter()

const data = ref([])
const showTable = ref(true)

const getData = async () => {
  try {
    const response = await api.getInboxes()
    data.value = response.data.data
  } catch (error) {
    toast({
      title: 'Could not fetch inboxes',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

onMounted(async () => {
  await getData()
})

const navigateToAddInbox = () => {
  showTable.value = false
  router.push('/admin/inboxes/new').catch((err) => {
    if (err.name !== 'NavigationDuplicated') {
      toast({
        title: 'Navigation error',
        variant: 'destructive',
        description: 'Failed to navigate to the new inbox page.'
      })
      showTable.value = true
    }
  })
}
</script>
