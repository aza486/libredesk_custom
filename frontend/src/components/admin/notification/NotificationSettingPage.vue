<template>
    <div>
        <PageHeader title="Notification" description="Manage notification settings" />
    </div>
    <div>
        <Spinner v-if="formLoading"></Spinner>
        <NotificationsForm :initial-values="initialValues" :submit-form="submitForm"
            :class="{ 'opacity-50 transition-opacity duration-300': formLoading }" :isLoading="formLoading"/>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import PageHeader from '../common/PageHeader.vue'
import NotificationsForm from './NotificationSettingForm.vue'
import { Spinner } from '@/components/ui/spinner'

const initialValues = ref({})
const formLoading = ref(false)

onMounted(() => {
    getNotificationSettings()
})

const getNotificationSettings = async () => {
    try {
        formLoading.value = true
        const resp = await api.getEmailNotificationSettings();
        initialValues.value = Object.fromEntries(
            Object.entries(resp.data.data).map(([key, value]) => [key.replace('notification.email.', ''), value])
        );
    } finally {
        formLoading.value = false
    }
};

const submitForm = (values) => {
    try {
        formLoading.value = true
        const updatedValues = Object.fromEntries(
            Object.entries(values).map(([key, value]) => [`notification.email.${key}`, value])
        );
        api.updateEmailNotificationSettings(updatedValues)
    } finally {
        formLoading.value = false
    }
};

</script>