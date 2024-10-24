<template>
  <div>
    <div class="flex justify-between mb-5">
      <PageHeader title="Email Templates" description="Manage outgoing email templates" />
      <div class="flex justify-end mb-4">
        <Button @click="navigateToAddTemplate" size="sm"> New template </Button>
      </div>
    </div>
    <div>
      <Spinner v-if="isLoading"></Spinner>
      <DataTable :columns="columns" :data="templates" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/templates/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const templates = ref([])
const isLoading = ref(false)
const router = useRouter()
const emit = useEmitter()

onMounted(async () => {
  fetchAll()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getTemplates()
    templates.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}

const refreshList = (data) => {
  if (data?.model === 'templates') fetchAll()
}

const navigateToAddTemplate = () => {
  router.push('/admin/templates/new')
}
</script>
