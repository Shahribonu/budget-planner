import type { Account, AccountType } from '@/api/types'

/** The 8 swatches offered in account Appearance; all sit well on the violet brand. */
export const ACCOUNT_SWATCHES = ['#5b3df5', '#2bd9a0', '#ff6b5a', '#f5a524', '#1e293b', '#3b82f6', '#ec4899', '#0f9e6e'] as const

const TYPE_COLORS: Record<AccountType, string> = {
  cash: '#0f9e6e',
  humo: '#3b82f6',
  uzcard: '#1e293b',
  visa: '#5b3df5',
  mastercard: '#ff6b5a',
  savings: '#f5a524',
  other: '#8b8aa0',
}

const HEX = /^#[0-9a-f]{6}$/i

/** The account's own colour, or a default for its type when unset or malformed. */
export function accountColor(a: Pick<Account, 'type' | 'color'>): string {
  return a.color && HEX.test(a.color) ? a.color : TYPE_COLORS[a.type]
}

/** A darker shade of #rrggbb for gradients and pressed states. */
export function darken(hex: string, amount = 0.25): string {
  const n = parseInt(hex.slice(1), 16)
  const ch = (shift: number) => Math.round(((n >> shift) & 255) * (1 - amount))
  return `#${[16, 8, 0].map((s) => ch(s).toString(16).padStart(2, '0')).join('')}`
}

export function isCard(type: AccountType): boolean {
  return type === 'humo' || type === 'uzcard' || type === 'visa' || type === 'mastercard'
}
