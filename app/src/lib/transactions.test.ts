import { describe, expect, it } from 'vitest'
import { amountSign, txRoute } from './transactions'

describe('txRoute', () => {
  it('opens debts and plans on their own screens', () => {
    expect(txRoute({ id: 1, type: 'debt', debt_id: 7 })).toEqual({ name: 'debt', params: { id: 7 } })
    expect(txRoute({ id: 2, type: 'debt_payment', debt_id: 7 })).toEqual({ name: 'debt', params: { id: 7 } })
    expect(txRoute({ id: 3, type: 'installment_payment', installment_id: 4 })).toEqual({
      name: 'installment',
      params: { id: 4 },
    })
  })

  it('opens other transactions in the form', () => {
    expect(txRoute({ id: 5, type: 'expense', debt_id: null })).toEqual({ name: 'tx-edit', params: { id: 5 } })
  })
})

describe('amountSign', () => {
  it('signs income and spending only', () => {
    expect(amountSign({ type: 'income', amount_minor: 1 })).toBe('in')
    expect(amountSign({ type: 'installment', amount_minor: 1 })).toBe('out')
    expect(amountSign({ type: 'debt', amount_minor: 1 })).toBe('none')
    expect(amountSign({ type: 'debt_payment', amount_minor: 1 })).toBe('none')
    expect(amountSign({ type: 'installment_payment', amount_minor: 1 })).toBe('none')
    expect(amountSign({ type: 'transfer', amount_minor: 1 })).toBe('none')
    expect(amountSign({ type: 'opening', amount_minor: -5 })).toBe('out')
  })
})
