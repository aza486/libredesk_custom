import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useUserStore = defineStore('user', () => {
  const user = ref({
    id: null,
    first_name: '',
    last_name: '',
    avatar_url: '',
    email: '',
    teams: [],
    permissions: []
  })

  const emitter = useEmitter()

  const userID = computed(() => user.value.id)
  const firstName = computed(() => user.value.first_name)
  const lastName = computed(() => user.value.last_name)
  const avatar = computed(() => user.value.avatar_url)
  const permissions = computed(() => user.value.permissions || [])
  const email = computed(() => user.value.email)
  const teams = computed(() => user.value.teams || [])

  const getFullName = computed(() => {
    if (!user.value.first_name && !user.value.last_name) return ''
    return `${user.value.first_name || ''} ${user.value.last_name || ''}`.trim()
  })

  const getInitials = computed(() => {
    const firstInitial = user.value.first_name?.charAt(0)?.toUpperCase() || ''
    const lastInitial = user.value.last_name?.charAt(0)?.toUpperCase() || ''
    return `${firstInitial}${lastInitial}`
  })

  const getCurrentUser = async () => {
    try {
      const response = await api.getCurrentUser()
      const userData = response?.data?.data
      if (userData) {
        user.value = userData
      } else {
        throw new Error('No user data found')
      }
    } catch (error) {
      if (error.response?.status !== 401) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          title: 'Could not fetch current user',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
  }

  const setAvatar = (avatarURL) => {
    if (typeof avatarURL !== 'string') {
      console.warn('Avatar URL must be a string')
      return
    }
    user.value.avatar_url = avatarURL
  }

  const clearAvatar = () => {
    user.value.avatar_url = ''
  }

  return {
    user,
    userID,
    firstName,
    lastName,
    avatar,
    email,
    teams,
    permissions,
    getFullName,
    getInitials,
    getCurrentUser,
    clearAvatar,
    setAvatar
  }
})