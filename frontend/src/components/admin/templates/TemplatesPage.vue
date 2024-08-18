<template>
    <div>
        <div class="flex justify-between mb-5">
            <PageHeader title="Templates" description="Manage email templates" />
            <div class="flex justify-end mb-4">
                <Button @click="navigateToAddTemplate" size="sm"> New template </Button>
            </div>
        </div>
        <div class="w-full">
            <DataTable :columns="columns" :data="templates" />
        </div>
    </div>
</template>


<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/templates/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
import { useRouter } from 'vue-router'
import api from '@/api'

const templates = ref([])
const router = useRouter()

onMounted(async () => {
    const resp = await api.getTemplates()
    templates.value = resp.data.data
})

const navigateToAddTemplate = () => {
    router.push('/admin/templates/new')
}
</script>