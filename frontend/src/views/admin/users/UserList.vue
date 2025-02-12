<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-user' }">
        <Button>New User</Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="columns" :data="data" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/features/admin/users/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/datatable/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const isLoading = ref(false)
const data = ref([])
const emitter = useEmitter()

onMounted(async () => {
  getData()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'user') getData()
  })
})

const getData = async () => {
  try {
    isLoading.value = true
    const response = await api.getUsers()
    data.value = response.data.data
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
</script>
