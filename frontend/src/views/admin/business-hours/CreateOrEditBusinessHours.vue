<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <Spinner v-if="isLoading"></Spinner>
    <BusinessHoursForm :initial-values="businessHours" :submitForm="submitForm" :isNewForm="isNewForm"
        :class="{ 'opacity-50 transition-opacity duration-300': isLoading }" :isLoading="formLoading" />
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import BusinessHoursForm from '@/features/admin/business-hours/BusinessHoursForm.vue'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'

const businessHours = ref({})
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
            await api.updateBusinessHours(props.id, values)
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Success',
                description: 'Business hours updated successfully',
            })
        } else {
            await api.createBusinessHours(values)
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Success',
                description: 'Business hours created successfully',
            })
            router.push({ name: 'business-hours-list' })
        }
    } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            title: 'Could not save business hours',
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
    { path: 'business-hours-list', label: 'Business hours' },
    { path: '', label: breadCrumLabel() }
]

onMounted(async () => {
    if (props.id) {
        try {
            isLoading.value = true
            const resp = await api.getBusinessHours(props.id)
            businessHours.value = resp.data.data
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Could not fetch business hours',
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        } finally {
            isLoading.value = false
        }
    }
})
</script>