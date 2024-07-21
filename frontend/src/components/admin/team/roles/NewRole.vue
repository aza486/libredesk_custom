<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <RoleForm :initial-values="{}" :submitForm="submitForm"/>
</template>

<script setup>
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import RoleForm from './RoleForm.vue';
import api from '@/api';
import { useRouter } from 'vue-router'

const router = useRouter()

const breadcrumbLinks = [
    { path: '/admin/teams', label: 'Teams' },
    { path: '/admin/teams/roles', label: 'Roles' },
    { path: '#', label: 'Add role' }
]

const submitForm = async (values) => {
    await api.createRole(values)
    router.push('/admin/teams/roles')
}
</script>