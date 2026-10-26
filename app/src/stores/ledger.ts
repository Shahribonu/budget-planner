import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { Summary, Transaction, TransactionInput } from '@/api/types'
import { i18n } from '@/i18n'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from './accounts'
import { useToastStore } from './toast'

/** Transaction writes plus the cached Home summary; `revision` bumps on every change so lists can reload. */
export const useLedgerStore = defineStore('ledger', () => {
  const summary = ref<Summary | null>(null)
  const revision = ref(0)

  async function loadSummary(): Promise<Summary> {
    summary.value = await api.getSummary()
    return summary.value
  }

  function changed(): void {
    revision.value++
    void useAccountsStore().load(true)
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
        await api.restoreTransaction(id)
        haptic.success()
        changed()
        useToastStore().show(t('common.restored'))
      },
    })
  }

  return { summary, revision, loadSummary, save, remove }
})
