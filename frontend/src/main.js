import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import router from './router'
import mitt from 'mitt'
import api from './api'
import './assets/styles/main.scss'
import './utils/strings.js'
import Root from './Root.vue'

async function initApp () {
  const settings = (await api.getSettings('general')).data.data
  const emitter = mitt()
  const lang = settings['app.lang'] || 'en'
  const langMessages = await api.getLanguage(lang)

  // Initialize i18n
  const i18nConfig = {
    legacy: false,
    locale: lang,
    fallbackLocale: 'en',
    messages: {
      [lang]: langMessages.data
    }
  }

  const i18n = createI18n(i18nConfig)
  const app = createApp(Root)
  const pinia = createPinia()

  // Add emitter to global properties.
  app.config.globalProperties.emitter = emitter

  app.use(router)
  app.use(pinia)
  app.use(i18n)
  app.mount('#app')
}

initApp().catch((error) => {
  console.error('Error initializing app: ', error)
})