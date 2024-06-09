import { defineStore } from 'pinia'
import { ref } from "vue"
import api from '@/api';

export const useAgents = defineStore('agents', () => {
    const agents = ref([])

    async function fetchAll () {
        try {
            const resp = await api.getAgents();
            agents.value = resp.data.data
        } catch (error) {
            // Pass
        } finally {
            // Pass
        }
    }

    return { agents, fetchAll }
})