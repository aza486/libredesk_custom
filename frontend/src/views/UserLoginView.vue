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
              <CardTitle class="text-2xl font-bold text-gray-900">Sign in</CardTitle>
              <p class="text-gray-600">Sign in to your account</p>
            </div>

            <div v-if="enabledOIDCProviders.length" class="space-y-3">
              <Button
                v-for="oidcProvider in enabledOIDCProviders"
                :key="oidcProvider.id"
                variant="outline"
                type="button"
                @click="redirectToOIDC(oidcProvider)"
                class="w-full bg-white hover:bg-gray-50 text-gray-700 border-gray-300"
              >
                <img :src="oidcProvider.logo_url" width="20" class="mr-2" alt="" />
                {{ oidcProvider.name }}
              </Button>

              <div class="relative">
                <div class="absolute inset-0 flex items-center">
                  <span class="w-full border-t border-gray-200"></span>
                </div>
                <div class="relative flex justify-center text-xs uppercase">
                  <span class="px-2 text-gray-500 bg-white">Or continue with</span>
                </div>
              </div>
            </div>

            <form @submit.prevent="loginAction" class="space-y-4">
              <div class="space-y-2">
                <Label for="email" class="text-sm font-medium text-gray-700">Email</Label>
                <Input
                  id="email"
                  type="text"
                  placeholder="Enter your email"
                  v-model.trim="loginForm.email"
                  :class="{ 'border-red-500': emailHasError }"
                  class="w-full bg-white border-gray-300 text-gray-900 placeholder:text-gray-400"
                />
              </div>

              <div class="space-y-2">
                <Label for="password" class="text-sm font-medium text-gray-700">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter your password"
                  v-model="loginForm.password"
                  :class="{ 'border-red-500': passwordHasError }"
                  class="w-full bg-white border-gray-300 text-gray-900 placeholder:text-gray-400"
                />
              </div>

              <div class="flex items-center justify-between">
                <div class="flex items-center space-x-2">
                  <!-- <input
                    type="checkbox"
                    id="remember"
                    class="w-4 h-4 rounded bg-white border-gray-300 text-blue-600"
                  />
                  <Label for="remember" class="text-sm text-gray-600">Remember me</Label> -->
                </div>
                <router-link to="/reset-password" class="text-sm text-blue-600 hover:text-blue-500">
                  Forgot password?
                </router-link>
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
                  Logging in...
                </span>
                <span v-else>Sign in</span>
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
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
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
const loginForm = ref({
  email: '',
  password: ''
})
const oidcProviders = ref([])

onMounted(async () => {
  fetchOIDCProviders()
})

const fetchOIDCProviders = async () => {
  try {
    const resp = await api.getAllOIDC()
    oidcProviders.value = resp.data.data
  } catch (error) {
    toast({
      title: 'Failed to load SSO providers.',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const redirectToOIDC = (provider) => {
  window.location.href = `/api/oidc/${provider.id}/login`
}

const validateForm = () => {
  if (!validateEmail(loginForm.value.email) && loginForm.value.email !== 'System') {
    errorMessage.value = 'Invalid email address.'
    useTemporaryClass('login-container', 'animate-shake')
    return false
  }
  if (!loginForm.value.password) {
    errorMessage.value = 'Password cannot be empty.'
    useTemporaryClass('login-container', 'animate-shake')
    return false
  }
  return true
}

const loginAction = () => {
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true

  api
    .login({
      email: loginForm.value.email,
      password: loginForm.value.password
    })
    .then(() => {
      router.push({ name: 'inboxes' })
    })
    .catch((error) => {
      errorMessage.value = handleHTTPError(error).message
      useTemporaryClass('login-container', 'animate-shake')
    })
    .finally(() => {
      isLoading.value = false
    })
}

const enabledOIDCProviders = computed(() => {
  return oidcProviders.value.filter((provider) => !provider.disabled)
})

const emailHasError = computed(() => {
  const email = loginForm.value.email
  return email !== 'System' && !validateEmail(email) && email !== ''
})

const passwordHasError = computed(
  () => !loginForm.value.password && loginForm.value.password !== ''
)
</script>
