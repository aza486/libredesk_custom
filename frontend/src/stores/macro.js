import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useUserStore } from './user'
import api from '@/api'

export const useMacroStore = defineStore('macroStore', () => {
    const macroList = ref([])
    const emitter = useEmitter()
    const userStore = useUserStore()
    const macroOptions = computed(() => {
        const userTeams = userStore.teams.map(team => String(team.id))
        return macroList.value.filter(macro =>
            macro.visibility === 'all' || userTeams.includes(macro.team_id) || String(macro.user_id) === String(userStore.userID)
        ).map(macro => ({
            ...macro,
            label: macro.name,
            value: String(macro.id),
        }))
    })
    const loadMacros = async () => {
        if (macroList.value.length) return
        try {
            const response = await api.getAllMacros()
            macroList.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }
    return {
        macroList,
        macroOptions,
        loadMacros,
    }
})