<template>
    <TeamForm :initial-values="{}" :submitForm="submitForm" />
</template>

<script setup>
import { handleHTTPError } from '@/utils/http'
import TeamForm from '@/components/admin/team/TeamForm.vue'
import { useToast } from '@/components/ui/toast/use-toast'
import { useRouter } from 'vue-router'
import api from '@/api'

const { toast } = useToast()
const router = useRouter()

const submitForm = (values) => {
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
    router.push('/admin/team/teams')
}
</script>