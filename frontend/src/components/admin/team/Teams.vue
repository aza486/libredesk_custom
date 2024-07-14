<template>
  <div class="flex justify-between mb-5">
    <div>
      <h1>Teams</h1>
      <p class="text-muted-foreground text-sm">Create teams, manage agents.</p>
    </div>
    <div class="flex justify-end mb-4">
      <Button size="sm" @click="navigateToAddTeam">New team </Button>
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
import { ref, onMounted } from 'vue'
import { handleHTTPError } from '@/utils/http'
import { columns } from '@/components/admin/team/TeamsDataTableColumns.js'
import { useToast } from '@/components/ui/toast/use-toast'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import api from '@/api'
import { useRouter } from 'vue-router'


const router = useRouter()
const data = ref([])
const showTable = ref(true)
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
  router.push('/admin/team/teams/new')
}

onMounted(async () => {
  getData()
})
</script>
