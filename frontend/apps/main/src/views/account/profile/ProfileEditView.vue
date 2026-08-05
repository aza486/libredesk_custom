<template>
  <div class="h-full">
    <div class="flex flex-col space-y-5">
      <div class="space-y-1">
        <span class="sub-title">{{ $t('account.publicAvatar') }}</span>
        <p class="text-muted-foreground text-xs">{{ $t('account.changeAvatar') }}</p>
      </div>
      <AvatarUpload
        :src="userStore.avatar"
        :initials="userStore.getInitials"
        :label="$t('globals.messages.upload')"
        :disabled="isSaving"
        @upload="onCropped"
        @remove="removeAvatar"
      />

      <div class="space-y-4 border-t pt-5">
        <div class="space-y-1">
          <span class="sub-title">Passwort ändern</span>
          <p class="text-muted-foreground text-xs">
            Neues Passwort für deinen Account festlegen
          </p>
        </div>

        <Input
          v-model="newPassword"
          type="password"
          placeholder="Neues Passwort"
        />

        <Input
          v-model="confirmPassword"
          type="password"
          placeholder="Passwort wiederholen"
        />
      </div>

      <Button
        class="self-start"
        @click="saveUser"
        :isLoading="isSaving"
        :disabled="!pendingFile && !newPassword"
      >
        {{ $t('globals.messages.saveChanges') }}
      </Button>
    </div>
  </div>
</template>

<script setup>
import { useUserStore } from '../../../stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import { ref } from 'vue'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'
import { Input } from '@shared-ui/components/ui/input'

const emitter = useEmitter()
const { t } = useI18n()
const isSaving = ref(false)
const userStore = useUserStore()
const pendingFile = ref(null)
const newPassword = ref('')
const confirmPassword = ref('')

const onCropped = (file) => {
  if (isSaving.value) return
  pendingFile.value = file
  userStore.setAvatar(URL.createObjectURL(file))
}

const saveUser = async () => {
  if (newPassword.value !== confirmPassword.value) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Passwörter stimmen nicht überein'
    })
    return
  }

  if (!pendingFile.value && !newPassword.value) return

  const formData = new FormData()

  if (pendingFile.value) {
    formData.append('files', pendingFile.value, 'avatar.png')
  }

  if (newPassword.value) {
    formData.append('new_password', newPassword.value)
  }
  try {
    isSaving.value = true
    await api.updateCurrentUser(formData)
    pendingFile.value = null
    newPassword.value = ''
    confirmPassword.value = ''
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSaving.value = false
  }
}

const removeAvatar = async () => {
  if (isSaving.value) return
  try {
    await api.deleteUserAvatar()
    pendingFile.value = null
    userStore.clearAvatar()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('account.avatarRemoved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
