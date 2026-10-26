import type { AccountType } from '@/api/types'

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
