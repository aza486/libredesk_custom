import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import api from '@/api'

export const useUserStore = defineStore('user', () => {
  const userAvatar = ref('')
  const userFirstName = ref('')
  const userLastName = ref('')
  const userPermissions = ref([])

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

  // Fetch current user data
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
      console.error('Error fetching current user:', error)
    }
  }

  // Check if user has a specific permission
  const hasPermission = (permission) => {
    if (!permission) return true
    return userPermissions.value.includes(permission)
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
    hasPermission,
    clearAvatar
  }
})
