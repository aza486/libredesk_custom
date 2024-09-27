<template>
  <div>
    <div class="flex justify-between mb-5">
      <PageHeader title="Templates" description="Manage email templates" />
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
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/templates/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'

const templates = ref([])
const isLoading = ref(false)
const router = useRouter()

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getTemplates()
    templates.value = resp.data.data
  } finally {
    isLoading.value = false
  }
})

const navigateToAddTemplate = () => {
  router.push('/admin/templates/new')
}
</script>
