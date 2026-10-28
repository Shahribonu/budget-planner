import { initData } from '@/lib/telegram'
import type {
  Account,
  AccountCreate,
  AccountPatch,
  ApiErrorBody,
  Budget,
  BudgetCreate,
  BudgetList,
  Category,
  Currency,
  Debt,
  DebtCreate,
  DebtDetail,
  DebtPatch,
  Installment,
  InstallmentCreate,
  NewPlanItem,
  PaymentInput,
  PlanItemPatch,
  PlanPatch,
  PlanView,
  Recurring,
  RecurringCreate,
  RecurringPatch,
  Summary,
  Transaction,
  TransactionFilter,
  TransactionInput,
  TransactionPage,
  User,
  UserPatch,
} from './types'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly field?: string,
    /** Seconds from Retry-After on 429. */
    readonly retryAfter?: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

function authHeader(): string | undefined {
  const data = initData()
  if (data) return `tma ${data}`
  const devId = import.meta.env.VITE_DEV_TELEGRAM_ID
  if (devId) return `dev ${devId}`
  return undefined
}

type Query = Record<string, string | number | boolean | string[] | undefined | null>

function buildUrl(path: string, query?: Query): string {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(query ?? {})) {
    if (v !== undefined && v !== null && v !== '') params.set(k, Array.isArray(v) ? v.join(',') : String(v))
  }
  const qs = params.toString()
  return `/api${path}${qs ? `?${qs}` : ''}`
}

async function request<T>(method: string, path: string, opts: { query?: Query; body?: unknown } = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  const auth = authHeader()
  if (auth) headers.Authorization = auth
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'

  let res: Response
  try {
    res = await fetch(buildUrl(path, opts.query), {
      method,
      headers,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    })
  } catch (e) {
    throw new ApiError(0, 'network', e instanceof Error ? e.message : 'Network error')
  }

  if (res.status === 204) return undefined as T
  const text = await res.text()
  let data: unknown
  try {
    data = text ? JSON.parse(text) : undefined
  } catch {
    data = undefined
  }
  if (!res.ok) {
    const err = (data as ApiErrorBody | undefined)?.error
    const retryAfter = Number(res.headers.get('Retry-After')) || undefined
    const error = new ApiError(res.status, err?.code ?? `http_${res.status}`, err?.message ?? res.statusText, err?.field, retryAfter)
    // Any endpoint may rate-limit: one friendly toast here, no automatic retry.
    if (res.status === 429) onRateLimited?.(error)
    throw error
  }
  return data as T
}

/** The user's Telegram photo as a Blob, or null when there is none (404) or it can't be loaded. */
async function getAvatar(): Promise<Blob | null> {
  const auth = authHeader()
  try {
    const res = await fetch(buildUrl('/me/avatar'), { headers: auth ? { Authorization: auth } : {} })
    return res.ok ? await res.blob() : null
  } catch {
    return null
  }
}

let onRateLimited: ((e: ApiError) => void) | undefined

/** Set by main.ts so the client can show the rate-limit toast without importing stores. */
export function setRateLimitHandler(fn: (e: ApiError) => void): void {
  onRateLimited = fn
}

export const api = {
  /** 202: the bot sends a CSV and a JSON to the user's chat. 409 bot_blocked, 429 rate_limited. */
  exportData: () => request<{ sent: boolean }>('POST', '/me/export'),
  /** Permanently deletes the user and all their data (204). */
  deleteMe: () => request<void>('DELETE', '/me', { body: { confirm: 'delete' } }),

  getAvatar,
  getMe: () => request<User>('GET', '/me'),
  patchMe: (patch: UserPatch) => request<User>('PATCH', '/me', { body: patch }),

  listAccounts: async (includeArchived = false) =>
    (await request<{ accounts: Account[] }>('GET', '/accounts', { query: { include_archived: includeArchived } }))
      .accounts,
  createAccount: (input: AccountCreate) => request<Account>('POST', '/accounts', { body: input }),
  patchAccount: (id: number, patch: AccountPatch) => request<Account>('PATCH', `/accounts/${id}`, { body: patch }),

  listCategories: async () => (await request<{ categories: Category[] }>('GET', '/categories')).categories,

  listTransactions: (filter: TransactionFilter & { cursor?: string; limit?: number }) =>
    request<TransactionPage>('GET', '/transactions', { query: { ...filter } }),
  getTransaction: (id: number) => request<Transaction>('GET', `/transactions/${id}`),
  createTransaction: (input: TransactionInput) => request<Transaction>('POST', '/transactions', { body: input }),
  updateTransaction: (id: number, input: TransactionInput) =>
    request<Transaction>('PUT', `/transactions/${id}`, { body: input }),
  deleteTransaction: (id: number) => request<void>('DELETE', `/transactions/${id}`),
  restoreTransaction: (id: number) => request<Transaction>('POST', `/transactions/${id}/restore`),

  getSummary: (month?: string) => request<Summary>('GET', '/summary', { query: { month } }),

  /** create=false only returns an existing plan (404 if none), for previews like Home. */
  getPlan: (month?: string, currency?: Currency, create = true) =>
    request<PlanView>('GET', '/plan', { query: { month, currency, create: create ? undefined : false } }),
  patchPlan: (month: string, currency: Currency, patch: PlanPatch) =>
    request<PlanView>('PATCH', '/plan', { query: { month, currency }, body: patch }),
  addPlanItems: (month: string, currency: Currency, items: NewPlanItem[]) =>
    request<PlanView>('POST', '/plan/items', { query: { month, currency }, body: { items } }),
  parsePlan: async (text: string, currency: Currency) =>
    (await request<{ items: NewPlanItem[] }>('POST', '/plan/parse', { query: { currency }, body: { text } })).items,
  patchPlanItem: (id: number, patch: PlanItemPatch) => request<PlanView>('PATCH', `/plan/items/${id}`, { body: patch }),
  deletePlanItem: (id: number) => request<PlanView>('DELETE', `/plan/items/${id}`),

  listRecurring: async () => (await request<{ recurring: Recurring[] }>('GET', '/recurring')).recurring,
  createRecurring: (input: RecurringCreate) => request<Recurring>('POST', '/recurring', { body: input }),
  patchRecurring: (id: number, patch: RecurringPatch) => request<Recurring>('PATCH', `/recurring/${id}`, { body: patch }),
  deleteRecurring: (id: number) => request<void>('DELETE', `/recurring/${id}`),


  listBudgets: (month?: string) => request<BudgetList>('GET', '/budgets', { query: { month } }),
  createBudget: (input: BudgetCreate) => request<Budget>('POST', '/budgets', { body: input }),
  patchBudget: (id: number, amountMinor: number) =>
    request<Budget>('PATCH', `/budgets/${id}`, { body: { amount_minor: amountMinor } }),
  deleteBudget: (id: number) => request<void>('DELETE', `/budgets/${id}`),

  listDebts: async () => (await request<{ debts: Debt[] }>('GET', '/debts')).debts,
  getDebt: (id: number) => request<DebtDetail>('GET', `/debts/${id}`),
  createDebt: (input: DebtCreate) => request<Debt>('POST', '/debts', { body: input }),
  patchDebt: (id: number, patch: DebtPatch) => request<Debt>('PATCH', `/debts/${id}`, { body: patch }),
  deleteDebt: (id: number) => request<void>('DELETE', `/debts/${id}`),
  payDebt: (id: number, input: PaymentInput) => request<Transaction>('POST', `/debts/${id}/payments`, { body: input }),

  listInstallments: async () =>
    (await request<{ installments: Installment[] }>('GET', '/installments')).installments,
  getInstallment: (id: number) => request<Installment>('GET', `/installments/${id}`),
  createInstallment: (input: InstallmentCreate) => request<Installment>('POST', '/installments', { body: input }),
  deleteInstallment: (id: number) => request<void>('DELETE', `/installments/${id}`),
  payInstallment: (id: number, input: PaymentInput) =>
    request<Transaction>('POST', `/installments/${id}/payments`, { body: input }),
}
