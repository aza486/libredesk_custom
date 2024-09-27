<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div class="flex justify-end mb-5">
    <Button @click="navigateToAddTeam" size="sm"> New team </Button>
  </div>
  <div>
    <div>
      <Spinner v-if="isLoading"></Spinner>
      <DataTable :columns="columns" :data="data" v-else />
    </div>
  </div>
  <div>
    <router-view></router-view>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { handleHTTPError } from '@/utils/http'
import { columns } from '@/components/admin/team/teams/TeamsDataTableColumns.js'
import { useToast } from '@/components/ui/toast/use-toast'
import { Button } from '@/components/ui/button'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import DataTable from '@/components/admin/DataTable.vue'
import api from '@/api'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'

const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/', label: 'Teams' }
]

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
      title: 'Could not fetch teams.',
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

onMounted(async () => {
  getData()
})
</script>
