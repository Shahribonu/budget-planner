import { describe, expect, it } from 'vitest'
import type { PlanDecision } from '@/api/types'
import { addMonthsYM, buildReason, nextPriority, parseRange } from './plan'

const ctx = {
  planMonth: '2026-11',
  money: (m: number) => `${m / 100}`,
  month: (ym: string) => `M${ym}`,
}

function d(over: Partial<PlanDecision>): PlanDecision {
  return {
    group: 'fits',
    reason: 'fits',
    suggest_month: null,
    monthly_saving_minor: 0,
    months: 0,
    from_savings_minor: 0,
    left_after_minor: null,
    ...over,
  }
}

describe('buildReason', () => {
  it('says what is left after this item when it fits', () => {
    expect(buildReason(d({ reason: 'fits', left_after_minor: 150_000_000 }), ctx)).toEqual({
      parts: [{ key: 'fits', params: { left: '1500000' } }],
      tone: 'ok',
    })
  })

  it('never shows a negative "leaves", and omits it when unknown', () => {
    expect(buildReason(d({ reason: 'fits', left_after_minor: -500 }), ctx).parts[0]).toEqual({ key: 'fits', params: { left: '0' } })
    expect(buildReason(d({ reason: 'fits' }), ctx).parts).toEqual([{ key: 'fits_plain' }])
  })

  it('handles must and fixed dates', () => {
    expect(buildReason(d({ reason: 'must' }), ctx)).toEqual({ parts: [{ key: 'must' }], tone: 'ok' })
    expect(buildReason(d({ reason: 'fixed_date' }), ctx)).toEqual({ parts: [{ key: 'fixed_date' }], tone: 'ok' })
  })

  it('says "early next month" for next month and names a later month', () => {
    expect(buildReason(d({ group: 'later', reason: 'not_enough', suggest_month: '2026-12' }), ctx)).toEqual({
      parts: [{ key: 'next_month' }],
      tone: 'neutral',
    })
    expect(buildReason(d({ group: 'later', reason: 'not_enough', suggest_month: '2027-02' }), ctx).parts).toEqual([
      { key: 'in_month', params: { month: 'M2027-02' } },
    ])
  })

  it('falls back to "not enough" without a suggested month', () => {
    expect(buildReason(d({ group: 'later', reason: 'not_enough' }), ctx).parts).toEqual([{ key: 'not_enough' }])
  })

  it('offers taking from savings once', () => {
    expect(
      buildReason(d({ group: 'later', reason: 'not_enough', suggest_month: '2026-12', from_savings_minor: 50_000_000 }), ctx).parts,
    ).toEqual([{ key: 'next_month' }, { key: 'from_savings', params: { amount: '500000' } }])
  })

  it('suggests a monthly saving for large items', () => {
    expect(
      buildReason(d({ group: 'reconsider', reason: 'too_large', monthly_saving_minor: 100_000_000, months: 4 }), ctx).parts,
    ).toEqual([{ key: 'too_large' }, { key: 'save_monthly', params: { amount: '1000000', n: 4 } }])
  })

  it('explains payday timing and carried-over items', () => {
    expect(buildReason(d({ group: 'later', reason: 'before_salary' }), ctx).parts).toEqual([{ key: 'before_salary' }])
    expect(buildReason(d({ group: 'reconsider', reason: 'postponed' }), ctx).parts).toEqual([{ key: 'postponed' }])
  })

  it('marks a Must that does not fit as danger', () => {
    expect(buildReason(d({ reason: 'must_shortfall' }), ctx)).toEqual({ parts: [{ key: 'must_shortfall' }], tone: 'danger' })
  })
})

describe('parseRange', () => {
  it('reads a single amount or a range', () => {
    expect(parseRange('700 000', 'UZS')).toEqual({ min: 70_000_000, max: 70_000_000 })
    expect(parseRange('700 000-900 000', 'UZS')).toEqual({ min: 70_000_000, max: 90_000_000 })
    expect(parseRange('700 000 – 900 000', 'UZS')).toEqual({ min: 70_000_000, max: 90_000_000 })
  })

  it('rejects empty, reversed or broken ranges', () => {
    expect(parseRange('', 'UZS')).toBeNull()
    expect(parseRange('900-700', 'UZS')).toBeNull()
    expect(parseRange('1-2-3', 'UZS')).toBeNull()
    expect(parseRange('abc', 'UZS')).toBeNull()
  })
})

describe('helpers', () => {
  it('cycles priority and months', () => {
    expect(nextPriority('must')).toBe('should')
    expect(nextPriority('nice')).toBe('must')
    expect(addMonthsYM('2026-12', 1)).toBe('2027-01')
    expect(addMonthsYM('2026-01', -1)).toBe('2025-12')
  })
})
