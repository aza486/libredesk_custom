<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <TeamForm :initial-values="{}" :submitForm="submitForm" />
</template>

<script setup>
import { handleHTTPError } from '@/utils/http'
import TeamForm from '@/components/admin/team/teams/TeamForm.vue'
import { useToast } from '@/components/ui/toast/use-toast'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import api from '@/api'

const { toast } = useToast()
const router = useRouter()
const breadcrumbLinks = [
  { path: '/admin/teams', label: 'Teams' },
  { path: '/admin/teams/teams', label: 'Teams'},
  { path: '/admin/teams/teams/new', label: 'New team'},
]

const submitForm = (values) => {
    console.log("form val ", values)
    createTeam(values)
}

const createTeam = async (values) => {
    try {
        await api.createTeam(values)
    } catch (error) {
        toast({
            title: 'Could not create team.',
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    router.push('/admin/teams/teams')
}
</script>