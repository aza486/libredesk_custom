<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <Spinner v-if="isLoading"></Spinner>
    <SLAForm :initial-values="slaData" :submitForm="submitForm" :isNewForm="isNewForm"
        :class="{ 'opacity-50 transition-opacity duration-300': isLoading }" :isLoading="formLoading" />
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import SLAForm from './SLAForm.vue'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'

const slaData = ref({})
const emitter = useEmitter()
const isLoading = ref(false)
const formLoading = ref(false)
const router = useRouter()
const props = defineProps({
    id: {
        type: String,
        required: false
    }
})

const submitForm = async (values) => {
    try {
        formLoading.value = true
        if (props.id) {
            await api.updateSLA(props.id, values)
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Success',
                description: 'SLA updated successfully',
            })
        } else {
            await api.createSLA(values)
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Success',
                description: 'SLA created successfully',
            })
            router.push('/admin/sla')
        }
    } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            title: 'Could not save SLA',
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    } finally {
        formLoading.value = false
    }
}

const breadCrumLabel = () => {
    return props.id ? 'Edit' : 'New'
}

const isNewForm = computed(() => {
    return props.id ? false : true
})

const breadcrumbLinks = [
    { path: '/admin/sla', label: 'SLA' },
    { path: '#', label: breadCrumLabel() }
]

onMounted(async () => {
    if (props.id) {
        try {
            isLoading.value = true
            const resp = await api.getSLA(props.id)
            slaData.value = resp.data.data
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Could not fetch SLA',
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        } finally {
            isLoading.value = false
        }
    }
})
</script>