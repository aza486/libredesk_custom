import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useCannedResponses = defineStore('canned_responses', () => {
  const responses = ref([])

  async function fetchAll() {
    try {
      const resp = await api.getCannedResponses()
      responses.value = resp.data.data
    } catch (error) {
      // Pass
    } finally {
      // Pass
    }
  }

  return { responses, fetchAll }
})
