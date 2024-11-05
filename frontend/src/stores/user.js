import { computed, reactive } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useUserStore = defineStore('user', () => {
  let user = reactive({
    id: null,
    first_name: '',
    last_name: '',
    avatar_url: '',
    permissions: []
  })
  const emitter = useEmitter()

  const userID = computed(() => user.id)
  const firstName = computed(() => user.first_name)
  const lastName = computed(() => user.last_name)
  const avatar = computed(() => user.avatar_url)
  const permissions = computed(() => user.permissions || [])

  const getFullName = computed(() => {
    if (!user.first_name && !user.last_name) return ''
    return `${user.first_name || ''} ${user.last_name || ''}`.trim()
  })

  const getInitials = computed(() => {
    const firstInitial = user.first_name?.charAt(0)?.toUpperCase() || ''
    const lastInitial = user.last_name?.charAt(0)?.toUpperCase() || ''
    return `${firstInitial}${lastInitial}`
  })

  const getCurrentUser = async () => {
    try {
      const response = await api.getCurrentUser()
      const userData = response?.data?.data
      if (userData) {
        Object.assign(user, userData)
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
    user.avatar_url = avatarURL
  }

  const clearAvatar = () => {
    user.avatar_url = ''
  }

  return {
    userID,
    firstName,
    lastName,
    avatar,
    permissions,
    getFullName,
    getInitials,
    getCurrentUser,
    clearAvatar,
    setAvatar,
  }
})