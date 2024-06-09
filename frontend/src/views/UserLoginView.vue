<template>
    <div class="relative h-screen" id="login-container">
        <div class="absolute left-1/2 top-20 transform -translate-x-1/2 w-96 h-1/2 ">
            <form @submit.prevent="loginAction">
                <Card>
                    <CardHeader class="space-y-1">
                        <CardTitle class="text-2xl text-center">
                            Login to Artemis
                        </CardTitle>
                    </CardHeader>
                    <CardContent class="grid gap-4">
                        <div class="grid grid-cols-1 gap-6">
                            <Button variant="outline">
                                <svg role="img" viewBox="0 0 24 24" class="mr-2 h-4 w-4">
                                    <path fill="currentColor"
                                        d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z" />
                                </svg>
                                Google
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
                            <Input id="email" type="email" placeholder="m@example.com" v-model.trim="loginForm.email" />
                        </div>
                        <div class="grid gap-2">
                            <Label for="password">Password</Label>
                            <Input id="password" type="password" v-model="loginForm.password" />
                        </div>
                    </CardContent>
                    <CardFooter class="flex flex-col gap-5">
                        <Button class="w-full" @click.prevent="loginAction" :disabled="loading" type="submit">
                            Login
                        </Button>
                        <Error :errorMessage="errorMessage" :border="true"></Error>
                        <div>
                            <a href="#" class="text-xs">Forgot ID or Password?</a>
                        </div>
                    </CardFooter>
                </Card>
            </form>
        </div>
    </div>
</template>

<script setup>
import { ref } from 'vue';
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
</script>
