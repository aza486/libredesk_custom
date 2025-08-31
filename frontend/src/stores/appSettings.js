import { defineStore } from 'pinia'

export const useAppSettingsStore = defineStore('settings', {
    state: () => ({
        settings: {},
        public_config: {}
    }),
    actions: {
        setSettings (newSettings) {
            this.settings = newSettings
        },
        setPublicConfig (newPublicConfig) {
            this.public_config = newPublicConfig
        }
    }
})
