<template>
    <PageHeader title="SLA" description="Manage service level agreements" />
    <div class="w-8/12">
        <template v-if="router.currentRoute.value.path === '/admin/sla'">
            <div class="flex justify-between mb-5">
                <div></div>
                <div>
                    <Button @click="navigateToAddSLA">New SLA</Button>
                </div>
            </div>
            <div>
                <Spinner v-if="isLoading"></Spinner>
                <DataTable :columns="columns" :data="slas" v-else />
            </div>
        </template>
        <template v-else>
            <router-view/>
        </template>
    </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from './dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import PageHeader from '../common/PageHeader.vue'
import { Spinner } from '@/components/ui/spinner'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const slas = ref([])
const isLoading = ref(false)
const router = useRouter()
const emit = useEmitter()

onMounted(() => {
    fetchAll()
    emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
    emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
    if (data?.model === 'sla') fetchAll()
}

const fetchAll = async () => {
    try {
        isLoading.value = true
        const resp = await api.getAllSLAs()
        slas.value = resp.data.data
    } finally {
        isLoading.value = false
    }
}

const navigateToAddSLA = () => {
    router.push('/admin/sla/new')
}
</script>