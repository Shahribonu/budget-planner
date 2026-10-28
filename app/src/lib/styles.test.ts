import { describe, expect, it } from 'vitest'
import { accountColor, darken, isCard } from './accountStyle'
import { CATEGORY_STYLES, categoryStyle } from './categoryStyle'

const KEYS = [
  'groceries', 'cafe', 'taxi', 'transport', 'fuel', 'utilities', 'rent', 'mobile_internet', 'health', 'clothing',
  'education', 'entertainment', 'gifts', 'family', 'beauty', 'home', 'travel', 'fees', 'other_expense',
  'salary', 'freelance', 'gift_income', 'cashback', 'other_income',
]

describe('categoryStyle', () => {
  it('has a colour and glyph for every seeded key', () => {
    expect(Object.keys(CATEGORY_STYLES).sort()).toEqual([...KEYS].sort())
    for (const k of KEYS) expect(CATEGORY_STYLES[k]!.color).toMatch(/^#[0-9a-f]{6}$/)
  })

  it('gives custom categories a stable colour by id', () => {
    const a = categoryStyle({ id: 41, key: null })
    expect(categoryStyle({ id: 41, key: null })).toEqual(a)
    expect(categoryStyle({ id: 42, key: null }).color).not.toBe(a.color)
    expect(categoryStyle(undefined).color).toMatch(/^#/)
  })
})

describe('accountColor', () => {
  it('uses the account colour, or a default per type', () => {
    expect(accountColor({ type: 'humo', color: '#FF6B5A' })).toBe('#FF6B5A')
    expect(accountColor({ type: 'humo', color: null })).toBe('#3b82f6')
    expect(accountColor({ type: 'cash', color: 'red' })).toBe('#0f9e6e')
  })

  it('darkens for gradients', () => {
    expect(darken('#ffffff', 0.5)).toBe('#808080')
    expect(darken('#5b3df5', 0)).toBe('#5b3df5')
  })

  it('knows card types', () => {
    expect(isCard('humo')).toBe(true)
    expect(isCard('cash')).toBe(false)
  })
})
