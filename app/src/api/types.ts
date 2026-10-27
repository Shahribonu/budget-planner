// Mirrors components/schemas in api/openapi.yaml.

export type Currency = 'UZS' | 'USD'
export type Language = 'uz-Latn' | 'uz-Cyrl' | 'ru' | 'en'
export type AccountType = 'cash' | 'humo' | 'uzcard' | 'visa' | 'mastercard' | 'savings' | 'other'
export type TransactionType =
  | 'expense'
  | 'income'
  | 'transfer'
  | 'exchange'
  | 'opening'
  | 'debt'
  | 'debt_payment'
  | 'installment'
  | 'installment_payment'
export type InputTransactionType = 'expense' | 'income' | 'transfer' | 'exchange'
/** Created via /debts or /installments; read-only through /transactions. */
export type ManagedTransactionType = 'debt' | 'debt_payment' | 'installment' | 'installment_payment'
export type DebtDirection = 'lent' | 'borrowed'
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
  /** null for a debt or installment that moved no money. */
  account_id: number | null
  to_account_id?: number | null
  to_amount_minor?: number | null
  to_currency?: Currency | null
  category_id?: number | null
  description: string
  source: Source
  created_at: string
  debt_id?: number | null
  installment_id?: number | null
  /** For debt and debt_payment. */
  debt_direction?: DebtDirection | null
  /** The person, for debt and debt_payment. */
  counterparty?: string | null
}

export interface TransactionFilter {
  from?: string
  to?: string
  account_id?: number
  category_id?: number
  /** Several types are sent comma-separated, e.g. expense,installment. */
  type?: TransactionType | TransactionType[]
  currency?: Currency
  q?: string
  /** 404 if the debt or plan isn't the user's. */
  debt_id?: number
  installment_id?: number
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
  owed_to_me: Money[]
  /** Open debts only. */
  i_owe: Money[]
  /** Left to pay on installment plans. */
  installments: Money[]
}

export interface PaymentInput {
  /** 0 or omitted on installments: the rest of the next payment. */
  amount_minor?: number
  account_id: number
  occurred_on: string
}

export interface Debt {
  id: number
  direction: DebtDirection
  counterparty: string
  currency: Currency
  principal_minor: number
  remaining_minor: number
  due_on: string | null
  note: string
  closed: boolean
  created_at: string
  /** When the money was lent or borrowed. */
  occurred_on: string
  /** null for a debt from before tracking. */
  account_id: number | null
}

export interface DebtCreate {
  direction: DebtDirection
  counterparty: string
  currency: Currency
  amount_minor: number
  account_id?: number | null
  occurred_on: string
  /** Not before occurred_on (422 invalid_due_date). */
  due_on?: string | null
  note?: string
}

export interface DebtPatch {
  counterparty?: string
  note?: string
  /** null clears it. */
  due_on?: string | null
}

export interface DebtDetail {
  debt: Debt
  transactions: Transaction[]
}

export interface ScheduleRow {
  n: number
  due_on: string
  /** In `next`: what is left of this payment after partial payments. */
  amount_minor: number
  paid: boolean
}

export interface Installment {
  id: number
  title: string
  provider: string
  category_id: number
  currency: Currency
  total_minor: number
  paid_minor: number
  remaining_minor: number
  months: number
  pay_from_account_id: number | null
  next: ScheduleRow | null
  schedule: ScheduleRow[]
  closed: boolean
  created_at: string
}

export interface InstallmentCreate {
  title: string
  provider?: string
  category_id: number
  currency: Currency
  total_minor: number
  months: number
  first_due_on: string
  pay_from_account_id?: number | null
  occurred_on: string
  record_purchase?: boolean
}

export interface Budget {
  id: number
  /** null is the overall monthly limit. */
  category_id: number | null
  currency: Currency
  amount_minor: number
  /** Expenses incl. installment purchases; transfers, debts and plan payments excluded. */
  spent_minor: number
  /** Negative when over budget. */
  remaining_minor: number
}

export interface BudgetList {
  month: string
  /** Days left in the month after today (today not counted); 0 for a past month. */
  days_left: number
  budgets: Budget[]
}

export interface BudgetCreate {
  category_id: number | null
  currency: Currency
  amount_minor: number
}
