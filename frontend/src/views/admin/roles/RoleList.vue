<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <div class="flex justify-end mb-5">
      <Button @click="navigateToNewRole"> New role </Button>
    </div>
    <div>
      <DataTable :columns="columns" :data="roles" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/features/admin/roles/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/datatable/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const emitter = useEmitter()
const router = useRouter()
const roles = ref([])
const isLoading = ref(false)

const getRoles = async () => {
  try {
    isLoading.value = true
    const resp = await api.getRoles()
    roles.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  getRoles()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'team') getRoles()
  })
})

const navigateToNewRole = () => {
  router.push('/admin/teams/roles/new')
}
</script>
