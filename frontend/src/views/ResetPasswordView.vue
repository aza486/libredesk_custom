<template>
  <div class="min-h-screen flex flex-col bg-gray-50">
    <header class="p-6">
      <h1 class="text-xl font-bold text-gray-900">LibreDesk</h1>
    </header>

    <main class="flex-1 flex items-center justify-center p-4">
      <div class="w-full max-w-[400px]">
        <Card class="bg-white border border-gray-200 shadow-lg">
          <CardContent class="p-8 space-y-6">
            <div class="space-y-2 text-center">
              <CardTitle class="text-2xl font-bold text-gray-900">Reset Password</CardTitle>
              <p class="text-gray-600">Enter your email to receive a password reset link.</p>
            </div>

            <form @submit.prevent="requestResetAction" class="space-y-4">
              <div class="space-y-2">
                <Label for="email" class="text-sm font-medium text-gray-700">Email</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="Enter your email address"
                  v-model.trim="resetForm.email"
                  :class="{ 'border-red-500': emailHasError }"
                  class="w-full bg-white border-gray-300 text-gray-900 placeholder:text-gray-400"
                />
              </div>

              <Button
                class="w-full bg-primary hover:bg-slate-500 text-white"
                :disabled="isLoading"
                type="submit"
              >
                <span v-if="isLoading" class="flex items-center justify-center">
                  <svg
                    class="animate-spin -ml-1 mr-3 h-5 w-5 text-white"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  Sending...
                </span>
                <span v-else>Send Reset Link</span>
              </Button>
            </form>

            <Error
              v-if="errorMessage"
              :errorMessage="errorMessage"
              :border="true"
              class="w-full bg-red-50 text-red-600 border-red-200 p-3 rounded-md text-sm"
            />

            <div class="text-center">
              <router-link to="/" class="text-sm text-blue-600 hover:text-blue-500"
                >Back to Login</router-link
              >
            </div>
          </CardContent>
        </Card>
      </div>
    </main>

    <footer class="p-6 text-center">
      <div class="text-sm text-gray-500 space-x-4">
        <a href="#" class="hover:text-gray-700">Privacy Policy</a>
        <span>•</span>
        <a href="#" class="hover:text-gray-700">Terms of Service</a>
        <span>•</span>
        <a href="#" class="hover:text-gray-700">Legal Notice</a>
      </div>
    </footer>
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
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

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
