import { describe, expect, it } from 'vitest'
import { donutSegments, sumSpent, weekStart } from './insights'

describe('weekStart', () => {
  it('goes back to Monday', () => {
    expect(weekStart('2026-10-27')).toBe('2026-10-26') // Tuesday
    expect(weekStart('2026-10-26')).toBe('2026-10-26') // Monday
    expect(weekStart('2026-11-01')).toBe('2026-10-26') // Sunday
  })
})

describe('sumSpent', () => {
  const txs = [
    { type: 'expense' as const, currency: 'UZS' as const, amount_minor: 100, occurred_on: '2026-10-27' },
    { type: 'installment' as const, currency: 'UZS' as const, amount_minor: 50, occurred_on: '2026-10-26' },
    { type: 'income' as const, currency: 'UZS' as const, amount_minor: 999, occurred_on: '2026-10-27' },
    { type: 'expense' as const, currency: 'USD' as const, amount_minor: 7, occurred_on: '2026-10-27' },
    { type: 'expense' as const, currency: 'UZS' as const, amount_minor: 30, occurred_on: '2026-10-20' },
  ]
  it('sums spending in the range and currency only', () => {
    expect(sumSpent(txs, 'UZS', '2026-10-27', '2026-10-27')).toBe(100)
    expect(sumSpent(txs, 'UZS', '2026-10-26', '2026-10-27')).toBe(150)
    expect(sumSpent(txs, 'USD', '2026-10-01', '2026-10-31')).toBe(7)
  })
})

describe('donutSegments', () => {
  it('splits the ring by share with gaps', () => {
    const s = donutSegments([75, 25], 0)
    expect(s).toEqual([
      { length: 75, offset: 0 },
      { length: 25, offset: 75 },
    ])
  })

  it('leaves gaps between several segments but not around one', () => {
    expect(donutSegments([10], 2)).toEqual([{ length: 100, offset: 0 }])
    const s = donutSegments([50, 50], 2)
    expect(s[0]!.length).toBe(48)
    expect(s[1]!.offset).toBe(50)
  })

  it('handles empty input', () => {
    expect(donutSegments([0, 0])).toEqual([
      { length: 0, offset: 0 },
      { length: 0, offset: 0 },
    ])
  })
})
