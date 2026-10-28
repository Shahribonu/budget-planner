import { createPinia } from 'pinia'
import { createApp, watch } from 'vue'
import App from './App.vue'
import { detectLanguage, i18n, setLanguage } from './i18n'
import { prefs } from './lib/prefs'
import * as telegram from './lib/telegram'
import { router } from './router'
import './style.css'

telegram.init()
watch(() => prefs.theme, telegram.applyColorScheme)
const app = createApp(App).use(createPinia()).use(i18n).use(router)
void setLanguage(detectLanguage(telegram.telegramLanguageCode() ?? navigator.language)).finally(() => app.mount('#app'))
