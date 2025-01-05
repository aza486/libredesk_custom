<template>
    <PageHeader title="Business hours" description="Manage business hours" />
    <div class="w-8/12">
        <template v-if="router.currentRoute.value.path === '/admin/business-hours'">
            <div class="flex justify-between mb-5">
                <div></div>
                <div>
                    <Button @click="navigateToAddBusinessHour">New business hour</Button>
                </div>
            </div>
            <div>
                <Spinner v-if="isLoading"></Spinner>
                <DataTable :columns="columns" :data="businessHours" v-else />
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
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import PageHeader from '../common/PageHeader.vue'
import { Spinner } from '@/components/ui/spinner'
import { columns } from '@/components/admin/business_hours/dataTableColumns.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const businessHours = ref([])
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
    if (data?.model === 'business_hours') fetchAll()
}

const fetchAll = async () => {
    try {
        isLoading.value = true
        const resp = await api.getAllBusinessHours()
        businessHours.value = resp.data.data
    } finally {
        isLoading.value = false
    }
}

const navigateToAddBusinessHour = () => {
    router.push('/admin/business-hours/new')
}
</script>