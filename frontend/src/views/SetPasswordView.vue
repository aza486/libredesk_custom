<template>
    <Spinner v-if="isLoading"></Spinner>
    <div class="relative h-screen" id="set-password-container" :class="{ 'soft-fade': isLoading }">
        <div class="absolute left-1/2 top-20 transform -translate-x-1/2 w-96 h-1/2">
            <form @submit.prevent="setPasswordAction">
                <Card>
                    <CardHeader class="space-y-1">
                        <CardTitle class="text-2xl text-center">Set New Password</CardTitle>
                        <p class="text-sm text-muted-foreground text-center">
                            Please enter your new password twice to confirm.
                        </p>
                    </CardHeader>
                    <CardContent class="grid gap-4">
                        <div class="grid gap-2">
                            <Label for="password">New Password</Label>
                            <Input id="password" type="password" placeholder="Enter new password"
                                v-model="passwordForm.password" :class="{ 'border-red-500': passwordHasError }" />
                        </div>
                        <div class="grid gap-2">
                            <Label for="confirmPassword">Confirm Password</Label>
                            <Input id="confirmPassword" type="password" placeholder="Confirm new password"
                                v-model="passwordForm.confirmPassword"
                                :class="{ 'border-red-500': confirmPasswordHasError }" />
                        </div>
                    </CardContent>
                    <CardFooter class="flex flex-col gap-5">
                        <Button class="w-full" @click.prevent="setPasswordAction" :disabled="isLoading" type="submit">
                            Set New Password
                        </Button>
                        <Error :errorMessage="errorMessage" :border="true"></Error>
                    </CardFooter>
                </Card>
            </form>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { useToast } from '@/components/ui/toast/use-toast'
import { useTemporaryClass } from '@/composables/useTemporaryClass'
import { Button } from '@/components/ui/button'
import { Error } from '@/components/ui/error'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import Spinner from '@/components/ui/spinner/Spinner.vue'

const errorMessage = ref('')
const isLoading = ref(false)
const router = useRouter()
const route = useRoute()
const { toast } = useToast()
const passwordForm = ref({
    password: '',
    confirmPassword: '',
    token: ''
})

onMounted(() => {
    passwordForm.value.token = route.query.token
    if (!passwordForm.value.token) {
        router.push({ name: 'login' })
        toast({
            title: 'Error',
            description: 'Invalid reset link. Please request a new password reset link.',
            variant: 'destructive'
        })
    }
})

const validateForm = () => {
    if (!passwordForm.value.password || passwordForm.value.password.length < 8) {
        errorMessage.value = 'Password must be at least 8 characters long.'
        useTemporaryClass('set-password-container', 'animate-shake')
        return false
    }

    if (passwordForm.value.password !== passwordForm.value.confirmPassword) {
        errorMessage.value = 'Passwords do not match.'
        useTemporaryClass('set-password-container', 'animate-shake')
        return false
    }

    return true
}

const setPasswordAction = async () => {
    if (!validateForm()) return

    errorMessage.value = ''
    isLoading.value = true

    try {
        await api.setPassword({
            token: passwordForm.value.token,
            password: passwordForm.value.password
        })

        toast({
            title: 'Password set successfully',
            description: 'You can now login with your new password.',
            variant: 'success'
        })

        router.push({ name: 'login' })
    } catch (err) {
        toast({
            title: 'Error',
            description: err.response.data.message,
            variant: 'destructive'
        })
        errorMessage.value = handleHTTPError(err).message
        useTemporaryClass('set-password-container', 'animate-shake')
    } finally {
        isLoading.value = false
    }
}

const passwordHasError = computed(() => {
    return passwordForm.value.password !== '' && passwordForm.value.password.length < 8
})

const confirmPasswordHasError = computed(() => {
    return passwordForm.value.confirmPassword !== '' &&
        passwordForm.value.password !== passwordForm.value.confirmPassword
})
</script>