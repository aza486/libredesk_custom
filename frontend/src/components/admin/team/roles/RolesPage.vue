<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <div class="flex justify-end mb-5">
        <Button @click="navigateToAddRole" size="sm"> New role </Button>
    </div>
    <div class="w-full">
        <DataTable :columns="columns" :data="roles" />
    </div>
    <router-view></router-view>
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
const { toast } = useToast()

const router = useRouter()
const roles = ref([])
const breadcrumbLinks = [
    { path: '/admin/teams', label: 'Teams' },
    { path: '#', label: 'Roles' }
]

const getRoles = async () => {
    try {
        const resp = await api.getRoles()
        roles.value = resp.data.data
    } catch (error) {
        toast({
            title: 'Could not fetch roles.',
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
}

onMounted(async () => {
    getRoles()
})

const navigateToAddRole = () => {
    router.push('/admin/teams/roles/new')
}
</script>