import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import router from './router'
import mitt from 'mitt';
import './assets/styles/main.scss'
import './utils/strings.js';
import api from './api'

async function initApp () {
    const emitter = mitt();

    // TODO: fetch def lang from cfg.
    const defaultLang = 'en';
    const langMessages = await api.getLanguage(defaultLang);

    // Initialize i18n
    const i18nConfig = {
        legacy: false,
        locale: defaultLang,
        fallbackLocale: defaultLang,
        messages: {
            [defaultLang]: langMessages.data
        }
    }
    const i18n = createI18n(i18nConfig);
    const app = createApp(App);
    const pinia = createPinia();

    app.config.globalProperties.emitter = emitter;

    app.use(router);
    app.use(pinia);
    app.use(i18n);
    app.mount('#app');
}

initApp().catch((error) => {
    console.error("Error initializing app: ", error);
});
