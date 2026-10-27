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
  Debt,
  DebtCreate,
  DebtDetail,
  DebtPatch,
  Installment,
  InstallmentCreate,
  PaymentInput,
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
    throw new ApiError(res.status, err?.code ?? `http_${res.status}`, err?.message ?? res.statusText, err?.field)
  }
  return data as T
}

export const api = {
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
