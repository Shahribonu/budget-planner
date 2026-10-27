import { describe, expect, it } from 'vitest'
import { addMonthsClamped, splitPayments, tooSmallForMonths } from './installments'

describe('splitPayments', () => {
  it('splits into whole units with the remainder on the last payment', () => {
    // 1 000 000.50 so'm over 3 months: 333 333 + 333 333 + 333 334.50
    expect(splitPayments(100_000_050, 3)).toEqual([33_333_300, 33_333_300, 33_333_450])
  })

  it('handles an even split and a single month', () => {
    expect(splitPayments(1_200_000_00, 12)).toEqual(Array(12).fill(10_000_000))
    expect(splitPayments(12_345, 1)).toEqual([12_345])
  })

  it('always sums to the total', () => {
    for (const [total, months] of [[999_999, 7], [1_000_001, 60], [150, 1]] as const) {
      expect(splitPayments(total, months).reduce((a, b) => a + b, 0)).toBe(total)
    }
  })

  it('returns nothing for invalid input', () => {
    expect(splitPayments(0, 3)).toEqual([])
    expect(splitPayments(1000, 0)).toEqual([])
  })

  it('flags totals too small for the term', () => {
    expect(tooSmallForMonths(250, 3)).toBe(true) // 2.50 over 3 months rounds to 0 per month
    expect(tooSmallForMonths(300, 3)).toBe(false)
    expect(tooSmallForMonths(50, 1)).toBe(true)
    expect(tooSmallForMonths(100, 1)).toBe(false)
  })
})

describe('addMonthsClamped', () => {
  it('keeps the day of month', () => {
    expect(addMonthsClamped('2026-10-15', 0)).toBe('2026-10-15')
    expect(addMonthsClamped('2026-10-15', 3)).toBe('2027-01-15')
  })

  it('clamps to the end of shorter months', () => {
    expect(addMonthsClamped('2026-01-31', 1)).toBe('2026-02-28')
    expect(addMonthsClamped('2028-01-31', 1)).toBe('2028-02-29')
    expect(addMonthsClamped('2026-01-31', 2)).toBe('2026-03-31')
  })
})
