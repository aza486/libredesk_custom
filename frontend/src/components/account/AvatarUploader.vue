<script setup>
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ref } from 'vue'

defineProps({
    avatarURL: {
        type: String,
        default: '',
    }
})

const emit = defineEmits(['change', 'onAvatarDelete'])
const fileInput = ref(null)

const handleImageUpload = (event) => {
    const file = event.target.files[0];
    emit('change', {
        file: file,
        url: file ? URL.createObjectURL(file) : null,
    });
}

const triggerFileInput = () => {
    fileInput.value.click();
}

const onAvatarDelete = () => {
    fileInput.value.value = null;
    emit('onAvatarDelete');
}
</script>

<template>
    <div class="space-y-2">
        <div class="text-3xl font-semibold">
            Public avatar
        </div>
        <p class="text-lg text-muted-foreground">
            Change your avatar here
        </p>
        <div class="flex flex-row gap-5">
            <Avatar class="size-20">
                <AvatarImage :src="avatarURL" alt="Current avatar" />
                <AvatarFallback></AvatarFallback>
            </Avatar>
            <div class="flex flex-col gap-2">
                <FormField v-slot="{ componentField }" name="file">
                    <FormItem v-auto-animate>
                        <FormLabel class="font-semi-bold">Upload new avatar</FormLabel>
                        <FormControl>
                            <Input ref="fileInput" type="file"
                                accept="image/png, image/jpeg, image/jpg, image/gif, image/webp"
                                @change="handleImageUpload" v-bind="componentField" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <Button variant="destructive" class="w-32" @click="onAvatarDelete">
                    Remove avatar
                </Button>
            </div>
        </div>
    </div>
</template>
