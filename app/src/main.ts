import { createPinia } from 'pinia'
import { createApp, watch } from 'vue'
import App from './App.vue'
import { setRateLimitHandler } from './api/client'
import { useToastStore } from './stores/toast'
import { detectLanguage, i18n, setLanguage } from './i18n'
import { prefs } from './lib/prefs'
import * as telegram from './lib/telegram'
import { router } from './router'
import './style.css'

telegram.init()
watch(() => prefs.theme, telegram.applyColorScheme)
const pinia = createPinia()
const app = createApp(App).use(pinia).use(i18n).use(router)
setRateLimitHandler((e) => useToastStore(pinia).rateLimited(e.retryAfter))
void setLanguage(detectLanguage(telegram.telegramLanguageCode() ?? navigator.language)).finally(() => app.mount('#app'))
