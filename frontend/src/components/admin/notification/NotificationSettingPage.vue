<template>
    <div>
        <PageHeader title="Notification" description="Manage notification settings" />
    </div>
    <div>
        <NotificationsForm :initial-values="initialValues" :submit-form="submitForm"></NotificationsForm>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

import api from '@/api'
import PageHeader from '../common/PageHeader.vue'
import NotificationsForm from './NotificationSettingForm.vue';

const initialValues = ref({})

onMounted(() => {
    getNotificationSettings()
})

const getNotificationSettings = async () => {
    const resp = await api.getEmailNotificationSettings();
    initialValues.value = Object.fromEntries(
        Object.entries(resp.data.data).map(([key, value]) => [key.replace('notification.email.', ''), value])
    );
};

const submitForm = (values) => {
    const updatedValues = Object.fromEntries(
        Object.entries(values).map(([key, value]) => [`notification.email.${key}`, value])
    );
    api.updateEmailNotificationSettings(updatedValues);
};

</script>