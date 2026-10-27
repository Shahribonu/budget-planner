// Mirrors components/schemas in api/openapi.yaml.

export type Currency = 'UZS' | 'USD'
export type Language = 'uz-Latn' | 'uz-Cyrl' | 'ru' | 'en'
export type AccountType = 'cash' | 'humo' | 'uzcard' | 'visa' | 'mastercard' | 'savings' | 'other'
export type TransactionType = 'expense' | 'income' | 'transfer' | 'exchange' | 'opening'
export type InputTransactionType = Exclude<TransactionType, 'opening'>
export type Source = 'manual' | 'chat' | 'voice' | 'bank' | 'receipt'
export type CategoryKind = 'expense' | 'income'

export interface ApiErrorBody {
  error: { code: string; message: string; field?: string }
}

export interface Money {
  amount_minor: number
  currency: Currency
}

export interface User {
  id: number
  telegram_id: number
  first_name: string
  username?: string | null
  language: Language
  main_currency: Currency
  timezone: string
  onboarded: boolean
  notify_enabled: boolean
  /** HH:MM in the user's time zone. */
  notify_time: string
}

export interface UserPatch {
  language?: Language
  main_currency?: Currency
  timezone?: string
  notify_enabled?: boolean
  notify_time?: string
}

export interface Account {
  id: number
  name: string
  type: AccountType
  currency: Currency
  balance_minor: number
  archived: boolean
}

export interface AccountCreate {
  name: string
  type: AccountType
  currency: Currency
  opening_balance_minor?: number
}

export interface AccountPatch {
  name?: string
  archived?: boolean
}

export interface Category {
  id: number
  kind: CategoryKind
  key: string | null
  name?: string | null
  icon: string
}

export interface TransactionInput {
  type: InputTransactionType
  occurred_on: string
  amount_minor: number
  account_id: number
  to_account_id?: number
  to_amount_minor?: number
  category_id?: number
  description?: string
  source?: Source
  source_ref?: string
}

export interface Transaction {
  id: number
  type: TransactionType
  occurred_on: string
  amount_minor: number
  currency: Currency
  account_id: number
  to_account_id?: number | null
  to_amount_minor?: number | null
  to_currency?: Currency | null
  category_id?: number | null
  description: string
  source: Source
  created_at: string
}

export interface TransactionFilter {
  from?: string
  to?: string
  account_id?: number
  category_id?: number
  type?: TransactionType
  currency?: Currency
  q?: string
}

export interface TransactionPage {
  transactions: Transaction[]
  next_cursor: string | null
}

export interface CategoryAmount {
  category_id: number
  amount_minor: number
  currency: Currency
}

export interface Summary {
  month: string
  balances: Money[]
  income: Money[]
  expense: Money[]
  by_category: CategoryAmount[]
  recent: Transaction[]
}
