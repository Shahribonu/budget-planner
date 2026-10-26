import { createI18n } from 'vue-i18n'
import type { Category, Language } from '@/api/types'
import uzLatn from './uz-Latn'

export type MessageSchema = typeof uzLatn

// The default/fallback catalog ships in the main bundle; others load on demand.
const loaders: Record<Exclude<Language, 'uz-Latn'>, () => Promise<{ default: MessageSchema }>> = {
  'uz-Cyrl': () => import('./uz-Cyrl'),
  ru: () => import('./ru'),
  en: () => import('./en'),
}

export const LANGUAGES: readonly Language[] = ['uz-Latn', 'uz-Cyrl', 'ru', 'en']

export const i18n = createI18n<[MessageSchema], Language, false>({
  legacy: false,
  locale: 'uz-Latn',
  fallbackLocale: 'uz-Latn',
  messages: { 'uz-Latn': uzLatn } as Record<Language, MessageSchema>,
})

export function currentLanguage(): Language {
  return i18n.global.locale.value
}

let requested: Language = 'uz-Latn'

export async function setLanguage(lang: Language): Promise<void> {
  requested = lang
  if (lang !== 'uz-Latn' && !i18n.global.availableLocales.includes(lang)) {
    try {
      i18n.global.setLocaleMessage(lang, (await loaders[lang]()).default)
    } catch {
      return // offline: stay on the current language
    }
  }
  if (requested !== lang) return // a newer call won the race
  i18n.global.locale.value = lang
  document.documentElement.lang = lang
}

/** Maps a Telegram language_code (IETF tag) to a supported language. */
export function detectLanguage(code: string | undefined): Language {
  const c = (code ?? '').toLowerCase()
  if (c.startsWith('uz')) return c.includes('cyrl') ? 'uz-Cyrl' : 'uz-Latn'
  if (['ru', 'be', 'uk', 'kk', 'ky', 'tg'].some((p) => c.startsWith(p))) return 'ru'
  if (c.startsWith('en')) return 'en'
  return 'uz-Latn'
}

export function categoryLabel(cat: Category | undefined): string {
  if (!cat) return ''
  if (cat.key) {
    const path = `categories.${cat.key}`
    if (i18n.global.te(path)) return i18n.global.t(path)
    return cat.name ?? cat.key
  }
  return cat.name ?? ''
}
