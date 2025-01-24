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
              <CardTitle class="text-2xl font-bold text-gray-900">Set New Password</CardTitle>
              <p class="text-gray-600">Please enter your new password twice to confirm.</p>
            </div>

            <form @submit.prevent="setPasswordAction" class="space-y-4">
              <div class="space-y-2">
                <Label for="password" class="text-sm font-medium text-gray-700">New Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter new password"
                  v-model="passwordForm.password"
                  :class="{ 'border-red-500': passwordHasError }"
                  class="w-full bg-white border-gray-300 text-gray-900 placeholder:text-gray-400"
                />
              </div>

              <div class="space-y-2">
                <Label for="confirmPassword" class="text-sm font-medium text-gray-700"
                  >Confirm Password</Label
                >
                <Input
                  id="confirmPassword"
                  type="password"
                  placeholder="Confirm new password"
                  v-model="passwordForm.confirmPassword"
                  :class="{ 'border-red-500': confirmPasswordHasError }"
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
                  Setting password...
                </span>
                <span v-else>Set New Password</span>
              </Button>
            </form>

            <Error
              v-if="errorMessage"
              :errorMessage="errorMessage"
              :border="true"
              class="w-full bg-red-50 text-red-600 border-red-200 p-3 rounded-md text-sm"
            />
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
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
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
  return (
    passwordForm.value.confirmPassword !== '' &&
    passwordForm.value.password !== passwordForm.value.confirmPassword
  )
})
</script>
