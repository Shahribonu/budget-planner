import type { Account, AccountType, Currency } from '@/api/types'

export const ACCOUNT_TYPES: readonly AccountType[] = ['cash', 'humo', 'uzcard', 'visa', 'mastercard', 'savings', 'other']

export const ACCOUNT_ICONS: Record<AccountType, string> = {
  cash: '💵',
  humo: '💳',
  uzcard: '💳',
  visa: '💳',
  mastercard: '💳',
  savings: '🏦',
  other: '👛',
}

/** localStorage key for the account last used to save a transaction. */
export const LAST_ACCOUNT_KEY = 'bp.lastAccount'

export function lastAccountId(): number | undefined {
  try {
    return Number(localStorage.getItem(LAST_ACCOUNT_KEY)) || undefined
  } catch {
    return undefined
  }
}

/**
 * The account a new transaction goes to: the preferred one if usable, else the last used,
 * else the first in the given currency (or the main currency when none is required).
 */
export function chooseAccount(
  active: Account[],
  opts: { currency?: Currency; preferredId?: number | null; lastId?: number; mainCurrency: Currency },
): Account | undefined {
  const fits = (a: Account) => !opts.currency || a.currency === opts.currency
  const byId = (id?: number | null) => (id ? active.find((a) => a.id === id && fits(a)) : undefined)
  return (
    byId(opts.preferredId) ??
    byId(opts.lastId) ??
    active.find((a) => fits(a) && (opts.currency || a.currency === opts.mainCurrency)) ??
    active.find(fits)
  )
}
