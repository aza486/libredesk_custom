<template>
  <div>
    <div class="flex justify-between mb-5">
      <div></div>
      <div>
        <Button @click="navigateToNewSLA">New SLA</Button>
      </div>
    </div>
    <div>
      <Spinner v-if="isLoading"></Spinner>
      <DataTable :columns="columns" :data="slas" v-else />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from './dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'

import { Spinner } from '@/components/ui/spinner'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const slas = ref([])
const isLoading = ref(false)
const router = useRouter()
const emit = useEmitter()

onMounted(() => {
  fetchAll()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'sla') fetchAll()
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAllSLAs()
    slas.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}

const navigateToNewSLA = () => {
  router.push('/admin/sla/new')
}
</script>
