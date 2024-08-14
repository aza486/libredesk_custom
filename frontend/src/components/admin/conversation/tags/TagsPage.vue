<template>
  <div>
      <div class="flex justify-between mb-5">
          <PageHeader title="Tags" description="Manage email templates" />
          <div class="flex justify-end mb-4">
              <Button @click="navigateToAddTemplate" size="sm"> New tag </Button>
          </div>
      </div>
      <div class="w-full">
          <DataTable :columns="columns" :data="tags" />
      </div>
  </div>
</template>


<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/templates/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/common/PageHeader.vue'
import { useRouter } from 'vue-router'
import api from '@/api'

const tags = ref([])
const router = useRouter()

onMounted(async () => {
  const resp = await api.getTags()
  tags.value = resp.data.data
})

const navigateToAddTemplate = () => {
  router.push('/admin/templates/new')
}
</script>