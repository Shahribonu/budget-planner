import type { Currency, PlanDecision, PlanItem, PlanPriority } from '@/api/types'
import { parseAmount } from './money'

export const PRIORITIES: readonly PlanPriority[] = ['must', 'should', 'nice']

export function nextPriority(p: PlanPriority): PlanPriority {
  return PRIORITIES[(PRIORITIES.indexOf(p) + 1) % PRIORITIES.length]!
}

/** "2026-10" + n months → "YYYY-MM". */
export function addMonthsYM(month: string, n: number): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(Date.UTC(y!, m! - 1 + n, 1))
  return d.toISOString().slice(0, 7)
}

/** One i18n message (key under `plan.reason.`) with its parameters. */
export interface ReasonPart {
  key: string
  params?: Record<string, string | number>
}

export interface ReasonLine {
  parts: ReasonPart[]
  /** danger only for a Must that doesn't fit; everything else stays neutral or positive. */
  tone: 'ok' | 'neutral' | 'danger'
}

export interface ReasonContext {
  /** The plan's month, to tell "early next month" from a later month. */
  planMonth: string
  money: (minor: number) => string
  month: (ym: string) => string
}

/**
 * The one-line explanation under a plan item, built only from the server's decision
 * and numbers. Messages are i18n templates; this picks which and fills them in.
 */
export function buildReason(d: PlanDecision, ctx: ReasonContext): ReasonLine {
  const parts: ReasonPart[] = []
  let tone: ReasonLine['tone'] = 'neutral'

  switch (d.reason) {
    case 'must':
      parts.push({ key: 'must' })
      tone = 'ok'
      break
    case 'fixed_date':
      parts.push({ key: 'fixed_date' })
      tone = 'ok'
      break
    case 'fits':
      parts.push(
        d.left_after_minor === null
          ? { key: 'fits_plain' }
          : { key: 'fits', params: { left: ctx.money(Math.max(d.left_after_minor, 0)) } },
      )
      tone = 'ok'
      break
    case 'must_shortfall':
      parts.push({ key: 'must_shortfall' })
      tone = 'danger'
      break
    case 'before_salary':
      parts.push({ key: 'before_salary' })
      break
    case 'not_enough':
      if (!d.suggest_month) parts.push({ key: 'not_enough' })
      break
    case 'too_large':
      parts.push({ key: 'too_large' })
      break
    case 'postponed':
      parts.push({ key: 'postponed' })
      break
  }

  // When it becomes affordable, then the ways to get there sooner.
  if (d.suggest_month && tone !== 'ok') {
    parts.push(
      d.suggest_month === addMonthsYM(ctx.planMonth, 1)
        ? { key: 'next_month' }
        : { key: 'in_month', params: { month: ctx.month(d.suggest_month) } },
    )
  }
  if (d.from_savings_minor > 0) parts.push({ key: 'from_savings', params: { amount: ctx.money(d.from_savings_minor) } })
  if (d.monthly_saving_minor > 0) {
    parts.push({ key: 'save_monthly', params: { amount: ctx.money(d.monthly_saving_minor), n: d.months } })
  }
  return { parts, tone }
}

/** "700 000", "700-900 ming"-style ranges typed by the user: "700 000 – 900 000" → {min, max}. */
export function parseRange(text: string, currency: Currency): { min: number; max: number } | null {
  const pieces = text.split(/\s*[-–—]\s*/).filter((s) => s.trim() !== '')
  if (pieces.length === 0 || pieces.length > 2) return null
  const min = parseAmount(pieces[0]!, currency)
  const max = pieces.length === 2 ? parseAmount(pieces[1]!, currency) : min
  if (!min || !max || max < min) return null
  return { min, max }
}

/** The amount the plan counts for an item: the max (conservative). */
export function itemAmount(item: Pick<PlanItem, 'amount_min_minor' | 'amount_max_minor'>): number {
  return Math.max(item.amount_min_minor, item.amount_max_minor)
}
