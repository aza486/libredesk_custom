<template>
  <div class="min-h-screen flex flex-col bg-background">
    <main class="flex-1 flex items-center justify-center p-4">
      <div class="w-full max-w-[400px]">
        <Card
          class="bg-card border border-border shadow-xl rounded-xl"
          :class="{ 'animate-shake': shakeCard }"
          ref="cardRef"
        >
          <CardContent class="p-8 space-y-6">
            <div class="space-y-2 text-center">
              <CardTitle class="text-3xl font-bold text-foreground">Libredesk</CardTitle>
              <p class="text-muted-foreground">Sign in to your account</p>
            </div>

            <div v-if="enabledOIDCProviders.length" class="space-y-4">
              <Button
                v-for="oidcProvider in enabledOIDCProviders"
                :key="oidcProvider.id"
                variant="outline"
                type="button"
                @click="redirectToOIDC(oidcProvider)"
                class="w-full bg-card hover:bg-secondary text-foreground border-border rounded-lg py-2 transition-all duration-200 ease-in-out transform hover:scale-105"
              >
                <img
                  :src="oidcProvider.logo_url"
                  width="20"
                  class="mr-2"
                  alt=""
                  v-if="oidcProvider.logo_url"
                />
                {{ oidcProvider.name }}
              </Button>

              <div class="relative">
                <div class="absolute inset-0 flex items-center">
                  <span class="w-full border-t border-border"></span>
                </div>
                <div class="relative flex justify-center text-xs uppercase">
                  <span class="px-2 text-muted-foreground bg-card">Or continue with</span>
                </div>
              </div>
            </div>

            <form @submit.prevent="loginAction" class="space-y-4">
              <div class="space-y-2">
                <Label for="email" class="text-sm font-medium text-foreground">Email</Label>
                <Input
                  id="email"
                  type="text"
                  placeholder="Enter your email"
                  v-model.trim="loginForm.email"
                  :class="{ 'border-destructive': emailHasError }"
                  class="w-full bg-card border-border text-foreground placeholder:text-muted-foreground rounded-lg py-2 px-3 focus:ring-2 focus:ring-ring focus:border-ring transition-all duration-200 ease-in-out"
                />
              </div>

              <div class="space-y-2">
                <Label for="password" class="text-sm font-medium text-foreground">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter your password"
                  v-model="loginForm.password"
                  :class="{ 'border-destructive': passwordHasError }"
                  class="w-full bg-card border-border text-foreground placeholder:text-muted-foreground rounded-lg py-2 px-3 focus:ring-2 focus:ring-ring focus:border-ring transition-all duration-200 ease-in-out"
                />
              </div>

              <div class="flex items-center justify-between">
                <router-link
                  to="/reset-password"
                  class="text-sm text-primary hover:text-primary/80 transition-all duration-200 ease-in-out"
                >
                  Forgot password?
                </router-link>
              </div>

              <Button
                class="w-full bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg py-2 transition-all duration-200 ease-in-out transform hover:scale-105"
                :disabled="isLoading"
                type="submit"
              >
                <span v-if="isLoading" class="flex items-center justify-center">
                  <svg
                    class="animate-spin -ml-1 mr-3 h-5 w-5 text-primary-foreground"
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
              class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded-lg text-sm"
            />
          </CardContent>
        </Card>
      </div>
    </main>

    <footer class="p-6 text-center">
      <div class="text-sm text-muted-foreground space-x-4"></div>
    </footer>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { validateEmail } from '@/utils/strings'
import { useTemporaryClass } from '@/composables/useTemporaryClass'
import { Button } from '@/components/ui/button'
import { Error } from '@/components/ui/error'
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useEmitter } from '@/composables/useEmitter'
import { useUserStore } from '@/stores/user'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const emitter = useEmitter()
const errorMessage = ref('')
const isLoading = ref(false)
const router = useRouter()
const userStore = useUserStore()
const loginForm = ref({
  email: '',
  password: ''
})
const oidcProviders = ref([])
const isDemoBuild = import.meta.env.VITE_DEMO_BUILD === 'true'

const demoCredentials = {
  email: 'demo@libredesk.io',
  password: 'demo@libredesk.io'
}

onMounted(async () => {
  // Prefill the login form with demo credentials if it's a demo build
  if (isDemoBuild) {
    loginForm.value.email = demoCredentials.email
    loginForm.value.password = demoCredentials.password
  }
  fetchOIDCProviders()
})

const fetchOIDCProviders = async () => {
  try {
    const resp = await api.getAllEnabledOIDC()
    oidcProviders.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const redirectToOIDC = (provider) => {
  window.location.href = `/api/v1/oidc/${provider.id}/login`
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
    .then((resp) => {
      if (resp?.data?.data) {
        userStore.setCurrentUser(resp.data.data)
      }
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
