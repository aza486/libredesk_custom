<template>
  <div class="flex justify-between mb-5">
    <PageHeader title="OIDC" description="Manage OpenID Connect configurations" />
    <div>
      <Button size="sm" @click="navigateToAddOIDC">New OIDC</Button>
    </div>
  </div>
  <div>
    <Spinner v-if="isLoading"></Spinner>
    <DataTable :columns="columns" :data="oidc" v-else />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/oidc/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import PageHeader from '../common/PageHeader.vue'
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'

const oidc = ref([])
const isLoading = ref(false)
const router = useRouter()
const emit = useEmitter()

onMounted(() => {
  fetchAll()
  emit.on('refresh-list', (data) => {
    if (data?.name === 'inbox') fetchAll
  })
})

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAllOIDC()
    oidc.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}

const navigateToAddOIDC = () => {
  router.push('/admin/oidc/new')
}
</script>
