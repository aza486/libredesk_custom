<template>
    <div class="flex justify-between mb-5">
    <PageHeader title="OIDC" description="Manage OpenID Connect configurations" />
    <div>
      <Button size="sm" @click="navigateToAddOIDC">New OIDC</Button>
    </div>
  </div>
  <div class="w-full">
    <DataTable :columns="columns" :data="oidc" />
  </div>
</template>


<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/oidc/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import PageHeader from '../common/PageHeader.vue'
import api from '@/api'

const oidc = ref([])
const router = useRouter()

onMounted(async () => {
  const resp = await api.getAllOIDC()
  oidc.value = resp.data.data
})

const navigateToAddOIDC = () => {
  router.push("/admin/oidc/new")
}
</script>