import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Router } from 'vue-router'
import { api } from '@/api/client'
import type { Account, Currency, SavingsSummary } from '@/api/types'
import { chooseAccount, lastAccountId } from '@/lib/accounts'
import { isSavings } from '@/lib/savings'
import { useAccountsStore } from './accounts'
import { useUserStore } from './user'

export const LAST_SAVINGS_KEY = 'bp.lastSavings'

function lastSavingsId(): number | undefined {
  try {
    return Number(localStorage.getItem(LAST_SAVINGS_KEY)) || undefined
  } catch {
    return undefined
  }
}

/** Remember the savings account used last, so "Put aside" goes there next time. */
export function rememberSavings(id: number): void {
  try {
    localStorage.setItem(LAST_SAVINGS_KEY, String(id))
  } catch {
    /* storage unavailable */
  }
}

type Direction = 'in' | 'out'

/** Savings: monthly movement from GET /savings, plus "Put aside" / "Take out" into the transfer form. */
export const useSavingsStore = defineStore('savings', () => {
  const summary = ref<SavingsSummary | null>(null)
  /** The "create a savings account" sheet, and what to do once it exists. */
  const createOpen = ref(false)
  const pending = ref<Direction>('in')

  const accounts = useAccountsStore()
  const savingsAccounts = computed(() => accounts.active.filter(isSavings))
  const spendingAccounts = computed(() => accounts.active.filter((a) => !isSavings(a)))

  // Shared while in flight: ledger.changed() and the open page both reload after a save.
  let inflight: Promise<SavingsSummary> | null = null
  function load(): Promise<SavingsSummary> {
    inflight ??= api
      .getSavings(12)
      .then((s) => (summary.value = s))
      .finally(() => (inflight = null))
    return inflight
  }

  /** The savings account to use: the last used one in `currency` if possible, else the first. */
  function pickSavings(currency?: Currency): Account | undefined {
    const list = savingsAccounts.value
    const fits = (a: Account) => !currency || a.currency === currency
    const last = lastSavingsId()
    return list.find((a) => a.id === last && fits(a)) ?? list.find(fits) ?? list.find((a) => a.id === last) ?? list[0]
  }

  /**
   * Opens the transfer form: "in" moves money from a spending account into savings, "out" back.
   * With no savings account yet, asks to create one first (see CreateSavingsSheet).
   */
  function start(direction: Direction, router: Router): void {
    pending.value = direction
    if (!savingsAccounts.value.length) {
      createOpen.value = true
      return
    }
    const user = useUserStore()
    const spending = chooseAccount(spendingAccounts.value, { lastId: lastAccountId(), mainCurrency: user.mainCurrency })
    const savings = pickSavings(spending?.currency)
    const other =
      spending && savings && spending.currency === savings.currency
        ? spending
        : savings
          ? chooseAccount(spendingAccounts.value, { currency: savings.currency, lastId: lastAccountId(), mainCurrency: user.mainCurrency })
          : spending
    const [from, to] = direction === 'in' ? [other, savings] : [savings, other]
    void router.push({
      name: 'tx-new',
      query: { type: 'transfer', ...(from ? { from: String(from.id) } : {}), ...(to ? { to: String(to.id) } : {}) },
    })
  }

  return { summary, createOpen, pending, savingsAccounts, load, pickSavings, start }
})
