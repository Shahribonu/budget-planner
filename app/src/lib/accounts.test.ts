import { describe, expect, it } from 'vitest'
import type { Account } from '@/api/types'
import { chooseAccount } from './accounts'

const acc = (id: number, currency: 'UZS' | 'USD'): Account => ({
  id, name: `a${id}`, type: 'cash', currency, balance_minor: 0, archived: false, color: null, include_in_total: true, card_mask: null,
})
const active = [acc(1, 'USD'), acc(2, 'UZS'), acc(3, 'UZS')]

describe('chooseAccount', () => {
  it('prefers the given account, then the last used one', () => {
    expect(chooseAccount(active, { preferredId: 3, lastId: 2, mainCurrency: 'UZS' })?.id).toBe(3)
    expect(chooseAccount(active, { lastId: 2, mainCurrency: 'UZS' })?.id).toBe(2)
  })

  it('falls back to the main currency, then anything', () => {
    expect(chooseAccount(active, { mainCurrency: 'UZS' })?.id).toBe(2)
    expect(chooseAccount([acc(1, 'USD')], { mainCurrency: 'UZS' })?.id).toBe(1)
  })

  it('respects a required currency', () => {
    expect(chooseAccount(active, { currency: 'USD', lastId: 2, mainCurrency: 'UZS' })?.id).toBe(1)
    expect(chooseAccount(active, { currency: 'UZS', preferredId: 1, mainCurrency: 'UZS' })?.id).toBe(2)
    expect(chooseAccount([acc(2, 'UZS')], { currency: 'USD', mainCurrency: 'UZS' })).toBeUndefined()
  })
})
