<template>
    <div class="relative h-screen" id="login-container">
        <div class="absolute left-1/2 top-20 transform -translate-x-1/2 w-96 h-1/2 ">
            <form @submit.prevent="loginAction">
                <Card>
                    <CardHeader class="space-y-1">
                        <CardTitle class="text-2xl text-center">
                            Login
                        </CardTitle>
                    </CardHeader>
                    <CardContent class="grid gap-4">
                        <div v-for="oidcProvider in enabledOIDCProviders" :key="oidcProvider.id"
                            class="grid grid-cols-1 gap-6">
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
                                <span class="bg-background px-2 text-muted-foreground">
                                    Or continue with
                                </span>
                            </div>
                        </div>
                        <div class="grid gap-2">
                            <Label for="email">Email</Label>
                            <Input id="email" type="email" placeholder="Enter your email address"
                                v-model.trim="loginForm.email" />
                        </div>
                        <div class="grid gap-2">
                            <Label for="password">Password</Label>
                            <Input id="password" type="password" placeholder="Password" v-model="loginForm.password" />
                        </div>
                    </CardContent>
                    <CardFooter class="flex flex-col gap-5">
                        <Button class="w-full" @click.prevent="loginAction" :disabled="loading" type="submit">
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
import { ref, onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { handleHTTPError } from '@/utils/http'
import { useUserStore } from '@/stores/user'
import api from '@/api';

import { useTemporaryClass } from '@/composables/useTemporaryClass';
import { Button } from '@/components/ui/button'
import { Error } from '@/components/ui/error'
import {
    Card,
    CardContent,
    CardFooter,
    CardHeader,
    CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const errorMessage = ref("");
const loading = ref(false);
const router = useRouter();
const loginForm = ref({
    email: "",
    password: ""
})

const userStore = useUserStore()
const oidcProviders = ref([])

const redirectToOIDC = (provider) => {
    window.location.href = `/api/oidc/${provider.id}/login`
};

const loginAction = () => {
    errorMessage.value = ""
    loading.value = true
    let loginParams = {
        "email": loginForm.value.email,
        "password": loginForm.value.password
    }
    api.login(loginParams).then((resp) => {
        if (resp.data.data) {
            userStore.$patch((state) => {
                state.userAvatar = resp.data.data.avatar_url
                state.userFirstName = resp.data.data.first_name
                state.userLastName = resp.data.data.last_name
            })
            router.push({ name: 'dashboard' });
        }
    }).catch((error) => {
        errorMessage.value = handleHTTPError(error).message
        useTemporaryClass("login-container", "animate-shake")
    }).finally(() => {
        loading.value = false
    })
}

onMounted(async () => {
    const resp = await api.getAllOIDC()
    oidcProviders.value = resp.data.data
})

const enabledOIDCProviders = computed(() => {
    return oidcProviders.value.filter(provider => !provider.disabled);
});

</script>
