import type { Account, Currency, SavingsMonth } from '@/api/types'
import { addMonthsYM } from './plan'

export function isSavings(a: Pick<Account, 'type'>): boolean {
  return a.type === 'savings'
}

/** Total balance of savings accounts, per currency, in a stable order (UZS first). */
export function savingsTotals(accounts: Pick<Account, 'type' | 'currency' | 'balance_minor'>[]): { currency: Currency; amount_minor: number }[] {
  const totals = new Map<Currency, number>()
  for (const a of accounts) if (isSavings(a)) totals.set(a.currency, (totals.get(a.currency) ?? 0) + a.balance_minor)
  return (['UZS', 'USD'] as Currency[]).filter((c) => totals.has(c)).map((c) => ({ currency: c, amount_minor: totals.get(c)! }))
}

/** Put in minus taken out for one month and currency (0 when there was no movement). */
export function netFor(months: SavingsMonth[], month: string, currency: Currency): number {
  const m = months.find((x) => x.month === month && x.currency === currency)
  return m ? m.put_in_minor - m.taken_out_minor : 0
}

export interface SeriesPoint {
  month: string
  put_in_minor: number
  taken_out_minor: number
  net_minor: number
}

/** The last `n` months up to `current`, oldest first, with empty months filled in as zero. */
export function savingsSeries(months: SavingsMonth[], currency: Currency, current: string, n = 12): SeriesPoint[] {
  return Array.from({ length: n }, (_, i) => {
    const month = addMonthsYM(current, i - (n - 1))
    const m = months.find((x) => x.month === month && x.currency === currency)
    const put = m?.put_in_minor ?? 0
    const out = m?.taken_out_minor ?? 0
    return { month, put_in_minor: put, taken_out_minor: out, net_minor: put - out }
  })
}

/** Currencies that have savings accounts or movement, UZS first. */
export function savingsCurrencies(accounts: Pick<Account, 'type' | 'currency'>[], months: SavingsMonth[]): Currency[] {
  const set = new Set<Currency>([...accounts.filter(isSavings).map((a) => a.currency), ...months.map((m) => m.currency)])
  return (['UZS', 'USD'] as Currency[]).filter((c) => set.has(c))
}

/** Share of the planned monthly saving reached, 0–1 (capped). 0 when nothing is planned. */
export function plannedProgress(netMinor: number, plannedMinor: number): number {
  if (plannedMinor <= 0) return 0
  return Math.max(0, Math.min(1, netMinor / plannedMinor))
}
