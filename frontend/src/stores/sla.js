import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useSlaStore = defineStore('sla', () => {
    const slas = ref([])
    const emitter = useEmitter()
    const options = computed(() => slas.value.map(sla => ({
        label: sla.name,
        value: String(sla.id)
    })))
    const fetchSlas = async () => {
        if (slas.value.length) return
        try {
            const response = await api.getAllSLAs()
            slas.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                title: 'Error',
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }
    return {
        slas,
        options,
        fetchSlas
    }
})
