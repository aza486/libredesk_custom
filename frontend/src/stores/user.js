import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useUserStore = defineStore('user', () => {
  const userAvatar = ref('')
  const userFirstName = ref('')
  const userLastName = ref('')
  const userPermissions = ref([])
  const emitter = useEmitter()

  // Setters
  const setAvatar = (avatar) => {
    userAvatar.value = avatar
  }

  const setFirstName = (firstName) => {
    userFirstName.value = firstName
  }

  const setLastName = (lastName) => {
    userLastName.value = lastName
  }

  // Computed properties
  const getInitials = computed(() => {
    const firstInitial = userFirstName.value.charAt(0).toUpperCase()
    const lastInitial = userLastName.value.charAt(0).toUpperCase()
    return `${firstInitial}${lastInitial}`
  })

  const getFullName = computed(() => {
    return `${userFirstName.value} ${userLastName.value}`
  })

  // Fetches current user.
  const getCurrentUser = async () => {
    try {
      const response = await api.getCurrentUser()
      const userData = response?.data?.data
      if (userData) {
        const { avatar_url, first_name, last_name, permissions } = userData
        setAvatar(avatar_url)
        setFirstName(first_name)
        setLastName(last_name)
        userPermissions.value = permissions || []
      }
    } catch (error) {
      if (error.response) {
        if (error.response.status !== 401) {
          emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            title: 'Could not fetch current user',
            variant: 'destructive',
            description: handleHTTPError(error).message
          })
        }
      }
    }
  }

  const clearAvatar = () => {
    userAvatar.value = ''
  }

  return {
    // State
    userFirstName,
    userLastName,
    userAvatar,
    userPermissions,

    // Computed
    getFullName,
    getInitials,

    // Actions
    setAvatar,
    setFirstName,
    setLastName,
    getCurrentUser,
    clearAvatar
  }
})
