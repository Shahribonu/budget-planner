import type { Currency, Transaction } from '@/api/types'
import { addDays } from './dates'

/** Monday of the week containing `iso` (weeks start on Monday in Uzbekistan). */
export function weekStart(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  const dow = new Date(Date.UTC(y!, m! - 1, d!)).getUTCDay() // 0 = Sunday
  return addDays(iso, -((dow + 6) % 7))
}

/** Spending (expenses and installment purchases) in one currency between two dates, inclusive. */
export function sumSpent(txs: Pick<Transaction, 'type' | 'currency' | 'amount_minor' | 'occurred_on'>[], currency: Currency, from: string, to: string): number {
  return txs
    .filter((t) => (t.type === 'expense' || t.type === 'installment') && t.currency === currency && t.occurred_on >= from && t.occurred_on <= to)
    .reduce((s, t) => s + t.amount_minor, 0)
}

export interface DonutSegment {
  /** Share of the ring, 0–100. */
  length: number
  /** Where it starts, 0–100. */
  offset: number
}

/**
 * Ring segments for values, as percentages of the circle (for an SVG circle with
 * pathLength=100). `gap` leaves a small space between segments when there are several.
 */
export function donutSegments(values: number[], gap = 1.2): DonutSegment[] {
  const total = values.reduce((s, v) => s + Math.max(v, 0), 0)
  if (total <= 0) return values.map(() => ({ length: 0, offset: 0 }))
  const g = values.filter((v) => v > 0).length > 1 ? gap : 0
  let offset = 0
  return values.map((v) => {
    const share = (Math.max(v, 0) / total) * 100
    const seg = { length: Math.max(share - g, share > 0 ? 0.5 : 0), offset }
    offset += share
    return seg
  })
}
