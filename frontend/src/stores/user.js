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

    const getInitials = computed(() => {
        const firstInitial = userFirstName.value.charAt(0).toUpperCase();
        const lastInitial = userLastName.value.charAt(0).toUpperCase();
        return firstInitial + lastInitial;
    });

    const getFullName = computed(() => {
        return userFirstName.value + " " + userLastName.value
    })

    async function getCurrentUser () {
        try {
            const resp = await api.getCurrentUser();
            if (resp.data.data) {
                userAvatar.value = resp.data.data.avatar_url;
                userFirstName.value = resp.data.data.first_name;
                userLastName.value = resp.data.data.last_name;
            }
        } catch (error) {
            console.error("Error fetching current user:", error);
        }
    }

    return { userFirstName, userLastName, userAvatar, getFullName, getInitials, setAvatar, setFirstName, setLastName, getCurrentUser }
})
