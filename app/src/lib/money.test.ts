import { describe, expect, it } from 'vitest'
import { formatMoney, formatTyping, parseAmount, toInputValue } from './money'

const N = ' '

describe('formatMoney', () => {
  it('groups thousands with a narrow no-break space and hides zero minor part', () => {
    expect(formatMoney(1_200_000_000, 'UZS', 'uz-Latn')).toBe(`12${N}000${N}000${N}so'm`)
  })

  it('uses the locale currency name for UZS', () => {
    expect(formatMoney(4_500_000, 'UZS', 'uz-Cyrl')).toBe(`45${N}000${N}сўм`)
    expect(formatMoney(4_500_000, 'UZS', 'ru')).toBe(`45${N}000${N}сум`)
    expect(formatMoney(4_500_000, 'UZS', 'en')).toBe(`45${N}000${N}UZS`)
  })

  it('prefixes USD with $', () => {
    expect(formatMoney(2_000, 'USD', 'en')).toBe('$20')
    expect(formatMoney(123_456_789, 'USD', 'en')).toBe(`$1${N}234${N}567.89`)
  })

  it('shows the minor part with the locale decimal separator', () => {
    expect(formatMoney(1_250, 'USD', 'en')).toBe('$12.50')
    expect(formatMoney(1_205, 'USD', 'ru')).toBe('$12,05')
    expect(formatMoney(150, 'UZS', 'uz-Latn')).toBe(`1,50${N}so'm`)
  })

  it('handles zero, small and negative amounts', () => {
    expect(formatMoney(0, 'UZS', 'en')).toBe(`0${N}UZS`)
    expect(formatMoney(5, 'USD', 'en')).toBe('$0.05')
    expect(formatMoney(-150_000, 'UZS', 'ru')).toBe(`−1${N}500${N}сум`)
    expect(formatMoney(-2_000, 'USD', 'en')).toBe('−$20')
  })

  it('adds + for positive amounts when signed', () => {
    expect(formatMoney(2_000, 'USD', 'en', { signed: true })).toBe('+$20')
    expect(formatMoney(0, 'USD', 'en', { signed: true })).toBe('$0')
  })

  it('omits the currency when bare', () => {
    expect(formatMoney(100_000_00, 'UZS', 'en', { bare: true })).toBe(`100${N}000`)
  })
})

describe('parseAmount', () => {
  it.each([
    ['45000', 4_500_000],
    ['45 000', 4_500_000],
    [`45${N}000`, 4_500_000],
    ['45 000', 4_500_000],
    ['45,000', 4_500_000],
    ['1,234,567', 123_456_700],
    ['1.234.567', 123_456_700],
    ['12.50', 1_250],
    ['12,50', 1_250],
    ['12.5', 1_250],
    ['12,5', 1_250],
    ['0.01', 1],
    ['.5', 50],
    ['12.', 1_200],
    ['1,234.56', 123_456],
    ['1.234,56', 123_456],
    ['0', 0],
    ['007', 700],
  ])('%s → %i', (input, expected) => {
    expect(parseAmount(input, 'USD')).toBe(expected)
  })

  it('avoids float rounding errors', () => {
    // 0.29 * 100 === 28.999999999999996 in floating point
    expect(parseAmount('0.29', 'USD')).toBe(29)
    expect(parseAmount('1.15', 'USD')).toBe(115)
    expect(parseAmount('4.35', 'USD')).toBe(435)
    expect(parseAmount('9007199254.99', 'USD')).toBe(900_719_925_499)
  })

  it.each(['', ' ', 'abc', '12a', '-5', '1.234.5', '12.345.67', '1,2,3', '1.2.3', '1234,567,890', ',', '1e5'])(
    'rejects %j',
    (input) => {
      expect(parseAmount(input, 'USD')).toBeNull()
    },
  )

  it('rejects amounts beyond safe integer precision', () => {
    expect(parseAmount('99999999999999', 'UZS')).toBeNull()
    expect(parseAmount('9999999999999', 'UZS')).toBe(999_999_999_999_900)
  })
})

describe('toInputValue', () => {
  it('round-trips through parseAmount', () => {
    for (const minor of [0, 1, 99, 100, 4_500_000, 123_456_789]) {
      expect(parseAmount(toInputValue(minor, 'USD'), 'USD')).toBe(minor)
    }
  })

  it('hides a zero minor part', () => {
    expect(toInputValue(4_500_000, 'UZS')).toBe(`45${N}000`)
    expect(toInputValue(1_250, 'USD')).toBe('12.50')
  })
})

describe('formatTyping', () => {
  it('regroups the integer part and keeps the typed decimal part', () => {
    expect(formatTyping('1250000')).toBe(`1${N}250${N}000`)
    expect(formatTyping(`1${N}25000`)).toBe(`125${N}000`)
    expect(formatTyping('1234.5')).toBe(`1${N}234.5`)
    expect(formatTyping('12,')).toBe('12,')
    expect(formatTyping('0012')).toBe('12')
    expect(formatTyping('12a3')).toBe('123')
  })
})
