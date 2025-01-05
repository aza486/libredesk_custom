<template>
  <PageHeader title="Roles" description="Manage roles" />
  <div class="w-8/12">
    <div v-if="router.currentRoute.value.path === '/admin/teams/roles'">
      <div class="flex justify-end mb-5">
        <Button @click="navigateToAddRole"> New role </Button>
      </div>
      <div>
        <Spinner v-if="isLoading"></Spinner>
        <DataTable :columns="columns" :data="roles" v-else />
      </div>
    </div>
    <router-view></router-view>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/components/admin/team/roles/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import PageHeader from '@/components/admin/common/PageHeader.vue'
const { toast } = useToast()

const emit = useEmitter()
const router = useRouter()
const roles = ref([])
const isLoading = ref(false)
const breadcrumbLinks = [

  { path: '#', label: 'Roles' }
]

const getRoles = async () => {
  try {
    isLoading.value = true
    const resp = await api.getRoles()
    roles.value = resp.data.data
  } catch (error) {
    toast({
      title: 'Could not fetch roles.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  getRoles()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'team') getRoles()
  })
})

const navigateToAddRole = () => {
  router.push('/admin/teams/roles/new')
}
</script>
