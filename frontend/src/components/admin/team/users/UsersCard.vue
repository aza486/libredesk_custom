<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div class="flex justify-end mb-5">
    <Button @click="navigateToAddUser" size="sm"> New user </Button>
  </div>
  <div class="w-full">
    <DataTable :columns="columns" :data="data" />
  </div>
  <router-view></router-view>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/components/admin/team/users/UsersDataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
const { toast } = useToast()

const router = useRouter()
const data = ref([])
const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '#', label: 'Users' }
]

const getData = async () => {
  try {
    const response = await api.getUsers()
    data.value = response.data.data
  } catch (error) {
    toast({
      title: 'Uh oh! Could not fetch users.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

onMounted(async () => {
  getData()
})

const navigateToAddUser = () => {
  router.push('/admin/teams/users/new')
}
</script>
