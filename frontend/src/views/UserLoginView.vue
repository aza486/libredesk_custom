<template>
  <div class="relative h-screen" id="login-container">
    <div class="absolute left-1/2 top-20 transform -translate-x-1/2 w-96 h-1/2">
      <Card>
        <CardHeader class="space-y-1">
          <CardTitle class="text-2xl text-center">Login</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-4">
          <div v-for="oidcProvider in enabledOIDCProviders" :key="oidcProvider.id" class="grid grid-cols-1 gap-6">
            <Button variant="outline" type="button" @click="redirectToOIDC(oidcProvider)">
              <img :src="oidcProvider.logo_url" width="15" class="mr-2" />
              {{ oidcProvider.name }}
            </Button>
          </div>
          <div class="relative" v-if="enabledOIDCProviders.length">
            <div class="absolute inset-0 flex items-center">
              <span class="w-full border-t"></span>
            </div>
            <div class="relative flex justify-center text-xs uppercase">
              <span class="bg-background px-2 text-muted-foreground">Or continue with</span>
            </div>
          </div>
          <form @submit.prevent="loginAction" class="space-y-4">
            <div class="grid gap-2">
              <Label for="email">Email</Label>
              <Input id="email" type="text" placeholder="Enter your email address" v-model.trim="loginForm.email"
                :class="{ 'border-red-500': emailHasError }" />
            </div>
            <div class="grid gap-2">
              <Label for="password">Password</Label>
              <Input id="password" type="password" placeholder="Password" v-model="loginForm.password"
                :class="{ 'border-red-500': passwordHasError }" />
            </div>
            <div>
              <Button class="w-full" :disabled="isLoading" :isLoading="isLoading" type="submit">
                Login
              </Button>
            </div>
          </form>
        </CardContent>
        <CardFooter class="flex flex-col gap-5">
          <Error :errorMessage="errorMessage" :border="true"></Error>
          <div>
            <router-link to="/reset-password" class="text-xs text-primary">
              Forgot password?
            </router-link>
          </div>
        </CardFooter>
      </Card>
    </div>
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
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
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
        router.push({ name: 'dashboard' })
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
  const email = loginForm.value.email;
  return email !== 'System' && !validateEmail(email) && email !== '';
})
const passwordHasError = computed(() => !loginForm.value.password && loginForm.value.password !== '')
</script>