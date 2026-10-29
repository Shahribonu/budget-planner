import { describe, expect, it } from 'vitest'
import { netFor, plannedProgress, savingsCurrencies, savingsSeries, savingsTotals } from './savings'

const months = [
  { month: '2026-10', currency: 'UZS' as const, put_in_minor: 200_000_000, taken_out_minor: 30_000_000 },
  { month: '2026-08', currency: 'UZS' as const, put_in_minor: 0, taken_out_minor: 50_000_000 },
  { month: '2026-10', currency: 'USD' as const, put_in_minor: 5000, taken_out_minor: 0 },
]

describe('savings', () => {
  it('totals savings accounts per currency only', () => {
    expect(
      savingsTotals([
        { type: 'savings', currency: 'USD', balance_minor: 100 },
        { type: 'savings', currency: 'UZS', balance_minor: 500 },
        { type: 'savings', currency: 'UZS', balance_minor: 250 },
        { type: 'cash', currency: 'UZS', balance_minor: 999 },
      ]),
    ).toEqual([
      { currency: 'UZS', amount_minor: 750 },
      { currency: 'USD', amount_minor: 100 },
    ])
  })

  it('nets a month, zero when absent', () => {
    expect(netFor(months, '2026-10', 'UZS')).toBe(170_000_000)
    expect(netFor(months, '2026-08', 'UZS')).toBe(-50_000_000)
    expect(netFor(months, '2026-09', 'UZS')).toBe(0)
  })

  it('builds a 12-month series ending at the current month, oldest first', () => {
    const s = savingsSeries(months, 'UZS', '2026-10')
    expect(s).toHaveLength(12)
    expect(s[0]!.month).toBe('2025-11')
    expect(s[11]).toEqual({ month: '2026-10', put_in_minor: 200_000_000, taken_out_minor: 30_000_000, net_minor: 170_000_000 })
    expect(s[9]!.net_minor).toBe(-50_000_000)
    expect(s[10]!.net_minor).toBe(0)
  })

  it('lists currencies from accounts and movement', () => {
    expect(savingsCurrencies([{ type: 'savings', currency: 'UZS' }], months)).toEqual(['UZS', 'USD'])
    expect(savingsCurrencies([{ type: 'cash', currency: 'USD' }], [])).toEqual([])
  })

  it('caps progress toward the plan', () => {
    expect(plannedProgress(50, 200)).toBe(0.25)
    expect(plannedProgress(300, 200)).toBe(1)
    expect(plannedProgress(-10, 200)).toBe(0)
    expect(plannedProgress(10, 0)).toBe(0)
  })
})
