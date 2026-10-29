import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ApiError, api } from '@/api/client'
import type { Summary, Transaction, TransactionInput } from '@/api/types'
import { i18n } from '@/i18n'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from './accounts'
import { useBudgetsStore } from './budgets'
import { useSavingsStore } from './savings'
import { useMonthlyPlanStore } from './monthlyPlan'
import { useDebtsStore } from './debts'
import { useInstallmentsStore } from './installments'
import { useToastStore } from './toast'

/** Transaction writes plus the cached Home summary; `revision` bumps on every change so lists can reload. */
export const useLedgerStore = defineStore('ledger', () => {
  const summary = ref<Summary | null>(null)
  const revision = ref(0)

  async function loadSummary(): Promise<Summary> {
    summary.value = await api.getSummary()
    return summary.value
  }

  /** Call after any write that moves money: balances, debts and plans may all change. */
  function changed(): void {
    revision.value++
    void useAccountsStore().load(true)
    const debts = useDebtsStore()
    if (debts.loaded) void debts.load(true).catch(() => {})
    const plans = useInstallmentsStore()
    if (plans.loaded) void plans.load(true).catch(() => {})
    const plan = useMonthlyPlanStore()
    if (plan.view) void plan.load().catch(() => {})
    const savings = useSavingsStore()
    if (savings.summary) void savings.load().catch(() => {})
    const budgets = useBudgetsStore()
    if (budgets.data) void budgets.load(true).catch(() => {})
  }

  /** Toast for a just-recorded transaction whose Undo deletes it (e.g. a repayment). */
  function offerUndo(message: string, txId: number): void {
    const { t } = i18n.global
    useToastStore().show(message, {
      label: t('common.undo'),
      run: async () => {
        await api.deleteTransaction(txId)
        haptic.success()
        changed()
      },
    })
  }

  async function save(id: number | null, input: TransactionInput): Promise<Transaction> {
    const tx = id === null ? await api.createTransaction(input) : await api.updateTransaction(id, input)
    changed()
    return tx
  }

  async function remove(id: number): Promise<void> {
    await api.deleteTransaction(id)
    changed()
    const { t } = i18n.global
    useToastStore().show(t('common.deleted'), {
      label: t('common.undo'),
      run: async () => {
        try {
          await api.restoreTransaction(id)
        } catch (e) {
          // Restoring a repayment can't pay back more than is now owed.
          if (e instanceof ApiError && e.code === 'overpayment') {
            haptic.error()
            useToastStore().show(t('common.restore_overpayment'))
            return
          }
          throw e
        }
        haptic.success()
        changed()
        useToastStore().show(t('common.restored'))
      },
    })
  }

  return { summary, revision, loadSummary, changed, offerUndo, save, remove }
})
