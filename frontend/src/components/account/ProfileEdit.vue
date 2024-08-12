<template>
    <div>
        <div class="flex flex-col space-y-5">
            <div class="space-y-1">
                <span class="sub-title">Public avatar</span>
                <p class="text-muted-foreground text-xs">Change your avatar here.</p>
            </div>
            <div class="flex space-x-5">
                <Avatar class="size-28">
                    <AvatarImage :src="userStore.userAvatar" alt="Cropped Image" />
                    <AvatarFallback>{{ userStore.getInitials }}</AvatarFallback>
                </Avatar>

                <div class="flex flex-col space-y-5 justify-center">
                    <input ref="uploadInput" type="file" hidden accept="image/jpg, image/jpeg, image/png, image/gif"
                        @change="selectFile" />
                    <Button class="w-28" @click="selectAvatar" size="sm">
                        Choose a file...
                    </Button>

                    <Button class="w-28" @click="removeAvatar" variant="destructive" size="sm">
                        Remove avatar
                    </Button>
                </div>
            </div>

            <Button class="w-28" @click="saveUser" size="sm">Save Changes</Button>

            <Dialog :open="showCropper">
                <DialogContent class="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle class="text-xl">Crop avatar</DialogTitle>
                    </DialogHeader>

                    <VuePictureCropper
                        :boxStyle="{ width: '100%', height: '400px', backgroundColor: '#f8f8f8', margin: 'auto' }"
                        :img="newUserAvatar" :options="{ viewMode: 1, dragMode: 'crop', aspectRatio: 1 }" />

                    <DialogFooter class="sm:justify-end">
                        <Button variant="secondary" @click="closeDialog">
                            Close
                        </Button>
                        <Button @click="getResult">Save</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    </div>
</template>

<script setup>
import { useUserStore } from '@/stores/user';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { ref } from 'vue';
import VuePictureCropper, { cropper } from 'vue-picture-cropper';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import api from '@/api'

const userStore = useUserStore();
const uploadInput = ref(null);
const newUserAvatar = ref('');
const showCropper = ref(false);
let croppedBlob = null;
let avatarFile = null;

const selectAvatar = () => {
    uploadInput.value.click();
};

const selectFile = (event) => {
    newUserAvatar.value = '';
    const { files } = event.target;
    if (!files || !files.length) return;

    avatarFile = files[0];
    const reader = new FileReader();
    reader.readAsDataURL(avatarFile);
    reader.onload = () => {
        newUserAvatar.value = String(reader.result);
        showCropper.value = true;
        uploadInput.value.value = '';
    };
};

const closeDialog = () => {
    showCropper.value = false;
};

const getResult = async () => {
    if (!cropper) return;
    croppedBlob = await cropper.getBlob();
    if (!croppedBlob) return;
    userStore.userAvatar = URL.createObjectURL(croppedBlob);
    showCropper.value = false;
};

const saveUser = async () => {
    const formData = new FormData();
    formData.append('files', croppedBlob, 'avatar.png');
    try {
        await api.updateCurrentUser(formData);
        // Handle success
    } catch (error) {
        // Handle error
    }
};

const removeAvatar = async () => {
    croppedBlob = null;
    try {
        await api.deleteUserAvatar();
        // Handle success
    } catch (error) {
        // Handle error
    }
};
</script>
