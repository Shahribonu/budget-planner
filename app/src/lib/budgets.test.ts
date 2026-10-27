import { describe, expect, it } from 'vitest'
import { budgetLevel, budgetPercent, dailyPace } from './budgets'

describe('budgetLevel', () => {
  it('uses the 80% and 100% thresholds', () => {
    expect(budgetLevel(0, 1000)).toBe('ok')
    expect(budgetLevel(799, 1000)).toBe('ok')
    expect(budgetLevel(800, 1000)).toBe('warn')
    expect(budgetLevel(999, 1000)).toBe('warn')
    expect(budgetLevel(1000, 1000)).toBe('over')
    expect(budgetLevel(1500, 1000)).toBe('over')
  })

  it('stays exact for large amounts', () => {
    expect(budgetLevel(400_000_000_000, 500_000_000_000)).toBe('warn')
    expect(budgetLevel(399_999_999_999, 500_000_000_000)).toBe('ok')
  })
})

describe('budgetPercent', () => {
  it('rounds down', () => {
    expect(budgetPercent(996, 1000)).toBe(99)
    expect(budgetPercent(1000, 1000)).toBe(100)
    expect(budgetPercent(2500, 1000)).toBe(250)
    expect(budgetPercent(-5, 1000)).toBe(0)
    expect(budgetPercent(5, 0)).toBe(0)
  })
})

describe('dailyPace', () => {
  it('splits what is left over today and the days left, in whole units', () => {
    // 1 000 000 so'm left, 9 days after today → 10 days → 100 000 so'm a day
    expect(dailyPace(100_000_000, 9)).toBe(10_000_000)
    // 100 so'm over 3 days → 33 so'm (whole units)
    expect(dailyPace(10_000, 2)).toBe(3_300)
  })

  it('uses at least one day and never goes negative', () => {
    expect(dailyPace(50_000, 0)).toBe(50_000)
    expect(dailyPace(50_000, -3)).toBe(50_000)
    expect(dailyPace(-100, 5)).toBe(0)
  })
})
