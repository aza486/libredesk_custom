<template>
  <div class="relative h-screen" id="login-container">
    <div class="absolute left-1/2 top-20 transform -translate-x-1/2 w-96 h-1/2">
      <form @submit.prevent="loginAction">
        <Card>
          <CardHeader class="space-y-1">
            <CardTitle class="text-2xl text-center">Login</CardTitle>
          </CardHeader>
          <CardContent class="grid gap-4">
            <div v-for="oidcProvider in enabledOIDCProviders" :key="oidcProvider.id" class="grid grid-cols-1 gap-6">
              <Button variant="outline" @click.prevent="redirectToOIDC(oidcProvider)">
                <img :src="oidcProvider.logo_url" width="15" class="mr-2" />
                {{ oidcProvider.name }}
              </Button>
            </div>
            <div class="relative">
              <div class="absolute inset-0 flex items-center">
                <span class="w-full border-t" />
              </div>
              <div class="relative flex justify-center text-xs uppercase">
                <span class="bg-background px-2 text-muted-foreground">Or continue with</span>
              </div>
            </div>
            <div class="grid gap-2">
              <Label for="email">Email</Label>
              <Input id="email" type="email" placeholder="Enter your email address" v-model.trim="loginForm.email"
                :class="{ 'border-red-500': emailHasError }" />
            </div>
            <div class="grid gap-2">
              <Label for="password">Password</Label>
              <Input id="password" type="password" placeholder="Password" v-model="loginForm.password"
                :class="{ 'border-red-500': passwordHasError }" />
            </div>
          </CardContent>
          <CardFooter class="flex flex-col gap-5">
            <Button class="w-full" @click.prevent="loginAction" :disabled="isLoading" :isLoading="isLoading"
              type="submit">
              Login
            </Button>
            <Error :errorMessage="errorMessage" :border="true"></Error>
            <div>
              <a href="#" class="text-xs">Forgot Email or Password?</a>
            </div>
          </CardFooter>
        </Card>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import { useUserStore } from '@/stores/user'
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
const userStore = useUserStore()
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
  if (!validateEmail(loginForm.value.email)) {
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
      const userData = resp.data.data
      if (userData) {
        userStore.$patch({
          userAvatar: userData.avatar_url,
          userFirstName: userData.first_name,
          userLastName: userData.last_name
        })
        router.push({ name: 'dashboard' })
      }
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

const emailHasError = computed(() => !validateEmail(loginForm.value.email) && loginForm.value.email !== '')
const passwordHasError = computed(() => !loginForm.value.password && loginForm.value.password !== '')

</script>
