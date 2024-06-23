import { ref, computed } from "vue"
import { defineStore } from 'pinia'
import api from '@/api';

export const useUserStore = defineStore('user', () => {
    const userAvatar = ref('')
    const userFirstName = ref('')
    const userLastName = ref('')

    const setAvatar = (v) => {
        userAvatar.value = v
    }

    const setFirstName = (v) => {
        userFirstName.value = v
    }

    const setLastName = (v) => {
        userLastName.value = v
    }

    const getFullName = computed(() => {
        return userFirstName.value + " " + userLastName.value
    })

    const getCurrentUser = () => {
        return api.getCurrentUser().then((resp) => {
            if (resp.data.data) {
                userAvatar.value = resp.data.data.avatar_url
                userFirstName.value = resp.data.data.first_name
                userLastName.value = resp.data.data.last_name
            }
        })
    }

    return { userFirstName, userLastName, userAvatar, getFullName, setAvatar, setFirstName, setLastName, getCurrentUser }
})
