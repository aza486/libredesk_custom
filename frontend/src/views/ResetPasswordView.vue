<template>
    <Spinner v-if="isLoading"></Spinner>
    <div class="relative h-screen" id="reset-password-container" :class="{ 'soft-fade': isLoading }">
        <div class="absolute left-1/2 top-20 transform -translate-x-1/2 w-96 h-1/2">
            <form @submit.prevent="requestResetAction">
                <Card>
                    <CardHeader class="space-y-1">
                        <CardTitle class="text-2xl text-center">Reset Password</CardTitle>
                        <p class="text-sm text-muted-foreground text-center">
                            Enter your email to receive a password reset link.
                        </p>
                    </CardHeader>
                    <CardContent class="grid gap-4">
                        <div class="grid gap-2">
                            <Label for="email">Email</Label>
                            <Input id="email" type="email" placeholder="Enter your email address"
                                v-model.trim="resetForm.email" :class="{ 'border-red-500': emailHasError }" />
                        </div>
                    </CardContent>
                    <CardFooter class="flex flex-col gap-5">
                        <Button class="w-full" @click.prevent="requestResetAction" :disabled="isLoading" type="submit">
                            Send Reset Link
                        </Button>
                        <Error :errorMessage="errorMessage" :border="true"></Error>
                        <div>
                            <router-link to="/" class="text-xs">Back to Login</router-link>
                        </div>
                    </CardFooter>
                </Card>
            </form>
        </div>
    </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { validateEmail } from '@/utils/strings'
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
const { toast } = useToast()
const resetForm = ref({
    email: ''
})

const validateForm = () => {
    if (!validateEmail(resetForm.value.email)) {
        errorMessage.value = 'Invalid email address.'
        useTemporaryClass('reset-password-container', 'animate-shake')
        return false
    }
    return true
}

const requestResetAction = async () => {
    if (!validateForm()) return

    errorMessage.value = ''
    isLoading.value = true

    try {
        await api.resetPassword({
            email: resetForm.value.email
        })
        toast({
            title: 'Reset link sent',
            description: 'Please check your email for the reset link.',
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
        useTemporaryClass('reset-password-container', 'animate-shake')
    } finally {
        isLoading.value = false
    }
}

const emailHasError = computed(() => {
    return !validateEmail(resetForm.value.email) && resetForm.value.email !== ''
})
</script>