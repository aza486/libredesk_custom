import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import router from './router'
import mitt from 'mitt'
import './assets/styles/main.scss'
import './utils/strings.js'
import api from './api'

async function initApp () {
  const emitter = mitt()

  let lang = 'en'
  const settings = (await api.getSettings('general')).data.data

  // Set the language
  lang = settings['app.lang']
  const langMessages = await api.getLanguage(lang)

  // Set the favicon based on settings
  const faviconUrl = settings['app.favicon_url'] || '/default-favicon.ico'
  const link = document.querySelector("link[rel~='icon']")
  if (link) {
    link.href = faviconUrl
  } else {
    const newLink = document.createElement('link')
    newLink.rel = 'icon'
    newLink.href = faviconUrl
    document.head.appendChild(newLink)
  }

  // Set the page title based on settings
  const pageTitle = settings['app.site_name'] || 'Support'
  document.title = pageTitle

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
  const app = createApp(App)
  const pinia = createPinia()

  app.config.globalProperties.emitter = emitter
  app.use(router)
  app.use(pinia)
  app.use(i18n)
  app.mount('#app')
}

initApp().catch((error) => {
  console.error('Error initializing app: ', error)
})
