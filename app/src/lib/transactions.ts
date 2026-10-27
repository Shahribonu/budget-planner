import type { RouteLocationRaw } from 'vue-router'
import type { ManagedTransactionType, Transaction, TransactionType } from '@/api/types'

const MANAGED: readonly TransactionType[] = ['debt', 'debt_payment', 'installment', 'installment_payment']

export function isManaged(type: TransactionType): type is ManagedTransactionType {
  return MANAGED.includes(type)
}

/** Where tapping a transaction leads: debts and plans are edited on their own screens. */
export function txRoute(tx: Pick<Transaction, 'id' | 'type' | 'debt_id' | 'installment_id'>): RouteLocationRaw {
  if (isManaged(tx.type)) {
    if (tx.debt_id) return { name: 'debt', params: { id: tx.debt_id } }
    if (tx.installment_id) return { name: 'installment', params: { id: tx.installment_id } }
  }
  return { name: 'tx-edit', params: { id: tx.id } }
}

/**
 * How an amount is signed in lists. Debts, repayments, plan payments and transfers
 * are neither income nor expense, so they get no sign.
 */
export function amountSign(tx: Pick<Transaction, 'type' | 'amount_minor'>): 'in' | 'out' | 'none' {
  switch (tx.type) {
    case 'income':
      return 'in'
    case 'expense':
    case 'installment':
    case 'exchange':
      return 'out'
    case 'opening':
      return tx.amount_minor < 0 ? 'out' : 'in'
    default:
      return 'none'
  }
}
