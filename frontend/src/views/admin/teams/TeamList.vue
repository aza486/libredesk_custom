<template>
  <div>
    <div class="flex justify-end mb-5">
      <Button @click="navigateToAddTeam"> New team </Button>
    </div>
    <div>
      <div>
        <Spinner v-if="isLoading"></Spinner>
        <DataTable :columns="columns" :data="data" v-else />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { handleHTTPError } from '@/utils/http'
import { columns } from '@/features/admin/teams/TeamsDataTableColumns.js'
import { useToast } from '@/components/ui/toast/use-toast'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/datatable/DataTable.vue'
import api from '@/api'

import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const emit = useEmitter()
const router = useRouter()
const data = ref([])
const isLoading = ref(false)
const { toast } = useToast()

const getData = async () => {
  try {
    isLoading.value = true
    const response = await api.getTeams()
    data.value = response.data.data
  } catch (error) {
    toast({
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const navigateToAddTeam = () => {
  router.push('/admin/teams/teams/new')
}

const listenForRefresh = () => {
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (event) => {
    if (event.model === 'team') {
      getData()
    }
  })
}

const removeListeners = () => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST)
}

onMounted(async () => {
  getData()
  listenForRefresh()
})

onUnmounted(() => {
  removeListeners()
})
</script>
