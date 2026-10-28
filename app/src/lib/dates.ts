import type { Language } from '@/api/types'

const INTL_LOCALE: Record<Language, string> = {
  'uz-Latn': 'uz-Latn',
  'uz-Cyrl': 'uz-Cyrl',
  ru: 'ru',
  en: 'en',
}

export function intlLocale(lang: Language): string {
  return INTL_LOCALE[lang]
}

/** Today's calendar date (YYYY-MM-DD) in the given IANA time zone. */
export function todayISO(timeZone?: string, now: Date = new Date()): string {
  try {
    return new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit' }).format(now)
  } catch {
    return new Intl.DateTimeFormat('en-CA', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(now)
  }
}

function parseISO(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y!, m! - 1, d!))
}

function toISO(date: Date): string {
  return date.toISOString().slice(0, 10)
}

export function addDays(iso: string, days: number): string {
  const d = parseISO(iso)
  d.setUTCDate(d.getUTCDate() + days)
  return toISO(d)
}

/** First and last calendar day of a "YYYY-MM" month. */
export function monthRange(month: string): { from: string; to: string } {
  const [y, m] = month.split('-').map(Number)
  const last = new Date(Date.UTC(y!, m!, 0))
  return { from: `${month}-01`, to: toISO(last) }
}

// Chrome's bundled ICU (desktop and Android WebView) has no Uzbek data, so Intl
// prints "2026 M10"; format Uzbek dates by hand.
const UZ_MONTHS = {
  'uz-Latn': ['yanvar', 'fevral', 'mart', 'aprel', 'may', 'iyun', 'iyul', 'avgust', 'sentabr', 'oktabr', 'noyabr', 'dekabr'],
  'uz-Cyrl': ['январ', 'феврал', 'март', 'апрел', 'май', 'июн', 'июл', 'август', 'сентябр', 'октябр', 'ноябр', 'декабр'],
} as const

function isUz(lang: Language): lang is 'uz-Latn' | 'uz-Cyrl' {
  return lang === 'uz-Latn' || lang === 'uz-Cyrl'
}

export function formatDay(iso: string, lang: Language, withYear = false): string {
  if (isUz(lang)) {
    const [y, m, d] = iso.split('-').map(Number)
    const month = UZ_MONTHS[lang][m! - 1]
    if (lang === 'uz-Latn') return withYear ? `${y}-yil ${d}-${month}` : `${d}-${month}`
    return withYear ? `${y} йил ${d} ${month}` : `${d} ${month}`
  }
  return new Intl.DateTimeFormat(intlLocale(lang), {
    day: 'numeric',
    month: 'long',
    year: withYear ? 'numeric' : undefined,
    timeZone: 'UTC',
  }).format(parseISO(iso))
}

export function formatMonth(month: string, lang: Language): string {
  if (isUz(lang)) {
    const [y, m] = month.split('-').map(Number)
    return `${UZ_MONTHS[lang][m! - 1]} ${y}`
  }
  return new Intl.DateTimeFormat(intlLocale(lang), { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(
    parseISO(`${month}-01`),
  )
}
