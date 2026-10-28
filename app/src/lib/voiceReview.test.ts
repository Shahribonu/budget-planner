import { describe, expect, it } from 'vitest'
import { allResolved, needsPick, reviewOptions } from './voiceReview'

const uzs = (amount_minor: number, needs_review: boolean) => ({ amount_minor, currency: 'UZS' as const, needs_review })
const usd = (amount_minor: number, needs_review: boolean) => ({ amount_minor, currency: 'USD' as const, needs_review })

describe('reviewOptions', () => {
  it('offers as-heard and × 1000 for flagged so’m amounts', () => {
    expect(reviewOptions(uzs(1800, true))).toEqual([1800, 1_800_000])
  })

  it('offers nothing for unflagged or USD amounts', () => {
    expect(reviewOptions(uzs(1800, false))).toEqual([])
    expect(reviewOptions(usd(1800, true))).toEqual([])
  })
})

describe('resolve', () => {
  it('blocks only flagged so’m rows without a pick', () => {
    expect(needsPick(uzs(1800, true), false)).toBe(true)
    expect(needsPick(uzs(1800, true), true)).toBe(false)
    expect(needsPick(usd(1800, true), false)).toBe(false)
    expect(needsPick(uzs(2_500_000, false), false)).toBe(false)
  })

  it('allows Save all once every flagged row is resolved', () => {
    const rows = [
      { item: uzs(2_500_000, false), resolved: false },
      { item: uzs(1800, true), resolved: false },
      { item: usd(500, true), resolved: false },
    ]
    expect(allResolved(rows)).toBe(false)
    rows[1]!.resolved = true
    expect(allResolved(rows)).toBe(true)
    expect(allResolved([])).toBe(true)
  })
})
