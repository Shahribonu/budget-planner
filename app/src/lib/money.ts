import type { Currency, Language } from '@/api/types'

export const CURRENCIES: readonly Currency[] = ['UZS', 'USD']

const MINOR_DIGITS: Record<Currency, number> = { UZS: 2, USD: 2 }

const NNBSP = ' '
const MINUS = '−'

const UZS_SUFFIX: Record<Language, string> = {
  'uz-Latn': "so'm",
  'uz-Cyrl': 'сўм',
  ru: 'сум',
  en: 'UZS',
}

function decimalSeparator(locale: Language): string {
  return locale === 'en' ? '.' : ','
}

function groupThousands(digits: string): string {
  return digits.replace(/\B(?=(\d{3})+(?!\d))/g, NNBSP)
}

/** Splits |amount_minor| into grouped major digits and the minor part ('' when zero). */
function splitMinor(amountMinor: number, currency: Currency): { major: string; minor: string } {
  const digits = MINOR_DIGITS[currency]
  const factor = 10 ** digits
  const abs = Math.abs(amountMinor)
  const major = Math.floor(abs / factor)
  const minor = abs % factor
  return {
    major: groupThousands(String(major)),
    minor: minor === 0 ? '' : String(minor).padStart(digits, '0'),
  }
}

export interface FormatMoneyOptions {
  /** Prefix positive amounts with "+" (negative amounts always get "−"). */
  signed?: boolean
  /** Omit the currency symbol/name. */
  bare?: boolean
}

export function formatMoney(
  amountMinor: number,
  currency: Currency,
  locale: Language,
  opts: FormatMoneyOptions = {},
): string {
  const { major, minor } = splitMinor(amountMinor, currency)
  const number = minor ? `${major}${decimalSeparator(locale)}${minor}` : major
  const sign = amountMinor < 0 ? MINUS : opts.signed && amountMinor > 0 ? '+' : ''
  if (opts.bare) return sign + number
  if (currency === 'USD') return `${sign}$${number}`
  return `${sign}${number}${NNBSP}${UZS_SUFFIX[locale]}`
}

/** The unit label shown next to an amount input. */
export function currencyLabel(currency: Currency, locale: Language): string {
  return currency === 'USD' ? '$' : UZS_SUFFIX[locale]
}

const SPACES = /[\s   ']/g

/**
 * Parses a user-typed major-unit amount into integer minor units without
 * floating point. Accepts "45000", "45 000", "45,000", "12.50", "12,5".
 * Returns null when the input is not a valid non-negative amount.
 */
export function parseAmount(input: string, currency: Currency): number | null {
  const digits = MINOR_DIGITS[currency]
  const s = input.replace(SPACES, '')
  if (!/^[\d.,]+$/.test(s) || !/\d/.test(s)) return null

  let intPart = s
  let fracPart = ''
  const lastSep = Math.max(s.lastIndexOf('.'), s.lastIndexOf(','))
  if (lastSep !== -1) {
    const sepChar = s[lastSep]!
    const otherSep = sepChar === '.' ? ',' : '.'
    const tail = s.slice(lastSep + 1)
    const sameSepCount = s.split(sepChar).length - 1
    // A lone separator followed by exactly 3 digits ("45,000") is grouping, not a decimal point.
    const isGrouping = !s.includes(otherSep) && (sameSepCount > 1 || tail.length === 3)
    if (isGrouping) {
      if (!validGrouping(s.split(sepChar))) return null
      intPart = s.split(sepChar).join('')
    } else {
      const head = s.slice(0, lastSep)
      if (head.includes(sepChar)) return null
      if (head.includes(otherSep) && !validGrouping(head.split(otherSep))) return null
      intPart = head.split(otherSep).join('')
      fracPart = tail
    }
  }

  if (fracPart.length > digits) return null
  if (intPart === '' && fracPart === '') return null
  const minorDigits = (intPart || '0') + fracPart.padEnd(digits, '0')
  const trimmed = minorDigits.replace(/^0+(?=\d)/, '')
  if (trimmed.length > 15) return null
  return Number(trimmed)
}

function validGrouping(groups: string[]): boolean {
  const [first, ...rest] = groups
  if (!first || first.length > 3) return false
  return rest.every((g) => g.length === 3)
}

/** Formats minor units for an amount input field, e.g. 4500000 → "45 000", 1250 → "12.50". */
export function toInputValue(amountMinor: number, currency: Currency): string {
  const { major, minor } = splitMinor(amountMinor, currency)
  return minor ? `${major}.${minor}` : major
}

/**
 * Regroups what the user is typing so it reads "1 250 000" while keeping a
 * trailing decimal part exactly as typed.
 */
export function formatTyping(input: string): string {
  const s = input.replace(SPACES, '').replace(/[^\d.,]/g, '')
  const sep = s.search(/[.,]/)
  if (sep === -1) return groupThousands(s.replace(/^0+(?=\d)/, ''))
  const int = s.slice(0, sep).replace(/^0+(?=\d)/, '')
  const frac = s.slice(sep + 1).replace(/[.,]/g, '')
  return `${groupThousands(int || '0')}${s[sep]}${frac}`
}
