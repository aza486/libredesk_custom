<template>
  <div>
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-macro' }">
        <Button> New macro </Button>
      </router-link>
    </div>
    <div>
      <Spinner v-if="isLoading"></Spinner>
      <DataTable v-else :columns="columns" :data="macros" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import { columns } from '../../../features/admin/macros/dataTableColumns.js'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { Button } from '@/components/ui/button'
import api from '@/api'

const formLoading = ref(false)
const macros = ref([])
const emit = useEmitter()

onMounted(() => {
  getMacros()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'macros') getMacros()
}

const getMacros = async () => {
  try {
    formLoading.value = true
    const resp = await api.getAllMacros()
    macros.value = resp.data.data
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
