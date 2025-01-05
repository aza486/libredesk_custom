<template>
    <PageHeader title="Notifications" description="Manage your email notification settings" />
    <div class="w-8/12">
        <div>
            <Spinner v-if="formLoading"></Spinner>
            <NotificationsForm :initial-values="initialValues" :submit-form="submitForm" :isLoading="formLoading" />
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import PageHeader from '../common/PageHeader.vue'
import NotificationsForm from './NotificationSettingForm.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'

const initialValues = ref({})
const formLoading = ref(false)
const emitter = useEmitter()

onMounted(() => {
    getNotificationSettings()
})

const getNotificationSettings = async () => {
    try {
        formLoading.value = true
        const resp = await api.getEmailNotificationSettings()
        initialValues.value = Object.fromEntries(
            Object.entries(resp.data.data).map(([key, value]) => [key.replace('notification.email.', ''), value])
        )
    } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            title: 'Could not fetch',
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    } finally {
        formLoading.value = false
    }
}

const submitForm = async (values) => {
    try {
        formLoading.value = true
        const updatedValues = Object.fromEntries(
            Object.entries(values).map(([key, value]) => {
                if (key === 'password' && value.includes('•')) {
                    return [`notification.email.${key}`, '']
                }
                return [`notification.email.${key}`, value]
            })
        );
        await api.updateEmailNotificationSettings(updatedValues)
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            description: "Saved successfully"
        })
    } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            title: 'Could not save',
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    } finally {
        formLoading.value = false
    }
}

</script>