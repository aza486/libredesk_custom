import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useAgentStore = defineStore('agent', () => {
    const agents = ref([])
    const emitter = useEmitter()
    const forSelect = computed(() => agents.value.map(agent => ({
        label: agent.first_name + ' ' + agent.last_name,
        value: agent.id
    })))
    const fetchAgents = async () => {
        if (agents.value.length) return
        try {
            const response = await api.getUsersCompact()
            agents.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Error',
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }
    return {
        agents,
        forSelect,
        fetchAgents,
    }
})