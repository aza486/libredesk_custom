<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div class="flex justify-end mb-5">
    <Button @click="navigateToAddTeam" size="sm"> New team </Button>
  </div>
  <div>
    <div class="w-full">
      <DataTable :columns="columns" :data="data" />
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

const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/', label: 'Teams' }
]

const router = useRouter()
const data = ref([])
const { toast } = useToast()

const getData = async () => {
  try {
    const response = await api.getTeams()
    data.value = response.data.data
  } catch (error) {
    toast({
      title: 'Could not fetch teams.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const navigateToAddTeam = () => {
  router.push('/admin/teams/teams/new')
}

onMounted(async () => {
  getData()
})
</script>
