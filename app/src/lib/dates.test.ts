import { describe, expect, it } from 'vitest'
import { addDays, formatDay, formatMonth, monthRange, todayISO } from './dates'

describe('dates', () => {
  it('computes today in the user time zone', () => {
    const now = new Date('2026-10-26T20:30:00Z') // 01:30 on the 27th in Tashkent (UTC+5)
    expect(todayISO('Asia/Tashkent', now)).toBe('2026-10-27')
    expect(todayISO('UTC', now)).toBe('2026-10-26')
  })

  it('adds days across month and year boundaries', () => {
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
    expect(addDays('2026-12-31', 1)).toBe('2027-01-01')
  })

  it('returns month bounds', () => {
    expect(monthRange('2026-02')).toEqual({ from: '2026-02-01', to: '2026-02-28' })
    expect(monthRange('2028-02')).toEqual({ from: '2028-02-01', to: '2028-02-29' })
    expect(monthRange('2026-10')).toEqual({ from: '2026-10-01', to: '2026-10-31' })
  })

  it('formats Uzbek dates without relying on Intl locale data', () => {
    expect(formatDay('2026-10-05', 'uz-Latn')).toBe('5-oktabr')
    expect(formatDay('2025-01-31', 'uz-Latn', true)).toBe('2025-yil 31-yanvar')
    expect(formatDay('2026-10-05', 'uz-Cyrl')).toBe('5 октябр')
    expect(formatMonth('2026-10', 'uz-Latn')).toBe('oktabr 2026')
    expect(formatMonth('2026-12', 'uz-Cyrl')).toBe('декабр 2026')
  })

  it('formats other locales with Intl', () => {
    expect(formatDay('2026-10-05', 'en')).toBe('October 5')
    expect(formatDay('2026-10-05', 'ru')).toBe('5 октября')
  })
})
