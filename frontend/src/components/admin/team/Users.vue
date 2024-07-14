<template>
  <div class="flex justify-between mb-5">
    <div>
      <h1>Teams</h1>
      <p class="text-muted-foreground text-sm">Create teams, manage agents.</p>
    </div>
    <div class="flex justify-end mb-4">
      <Button @click="navigateToAddInbox" size="sm"> New user </Button>
    </div>
  </div>
  <div v-if="showTable">
    <div class="w-full">
      <DataTable :columns="columns" :data="data" />
    </div>
  </div>
  <div v-else>
    <router-view></router-view>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/components/admin/team/UsersDataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'
import { useRouter } from 'vue-router'
const { toast } = useToast()

const router = useRouter()
const data = ref([])
const showTable = ref(true)

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

const navigateToAddInbox = () => {
  showTable.value = false
  router.push('/admin/team/users/new')
}
</script>
