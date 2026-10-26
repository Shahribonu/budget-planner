import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { detectLanguage, i18n, setLanguage } from './i18n'
import * as telegram from './lib/telegram'
import { router } from './router'
import './style.css'

telegram.init()
const app = createApp(App).use(createPinia()).use(i18n).use(router)
void setLanguage(detectLanguage(telegram.telegramLanguageCode() ?? navigator.language)).finally(() => app.mount('#app'))
