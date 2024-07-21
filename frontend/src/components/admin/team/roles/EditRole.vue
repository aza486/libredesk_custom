<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <RoleForm :initial-values="role" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import RoleForm from './RoleForm.vue'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api'

const role = ref({})
const { toast } = useToast()
const props = defineProps({
    id: {
        type: String,
        required: true
    }
})

onMounted(async () => {
    const resp = await api.getRole(props.id)
    role.value = resp.data.data
})

const breadcrumbLinks = [
    { path: '/admin/teams', label: 'Teams' },
    { path: '/admin/teams/roles', label: 'Roles' },
    { path: '#', label: 'Edit role' }
]

const submitForm = async (values) => {
    await api.updateRole(props.id, values)
    toast({
        description: 'Role saved successfully',
    });
}
</script>