
import { ref, computed } from "vue"
import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', () => {
    const userAvatar = ref('')
    const userFirstName = ref('')
    const userLastName = ref('')

    function setAvatar (v) {
        userAvatar.value = v
    }

    function setFirstName (v) {
        userFirstName.value = v
    }

    function setLastName (v) {
        userLastName.value = v
    }

    const getFullName = computed(() => {
        return userFirstName.value + " " + userLastName.value
    })


    return { userFirstName, userLastName, userAvatar, getFullName, setAvatar, setFirstName, setLastName }
})