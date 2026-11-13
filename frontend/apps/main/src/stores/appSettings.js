import { defineStore } from 'pinia'
import api from '@/api'

export const useAppSettingsStore = defineStore('settings', {
    state: () => ({
        settings: {},
        public_config: {}
    }),
    actions: {
        async fetchSettings (key = 'general') {
            try {
                const response = await api.getSettings(key)
                this.settings = response?.data?.data || {}
                return this.settings
            } catch {
                // Pass
            }
        },
        async fetchPublicConfig () {
            try {
                const response = await api.getConfig()
                this.public_config = response?.data?.data || {}
                return this.public_config
            } catch {
                // Pass
            }
        },
        setSettings (newSettings) {
            this.settings = newSettings
        },
        setPublicConfig (newPublicConfig) {
            this.public_config = newPublicConfig
        }
    }
})
