import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Budget, BudgetList } from '@/api/types'

/** This month's budgets with spending. */
export const useBudgetsStore = defineStore('budgets', () => {
  const data = ref<BudgetList | null>(null)
  let pending: Promise<BudgetList> | null = null

  const all = computed(() => data.value?.budgets ?? [])
  const overall = computed(() => all.value.filter((b) => b.category_id === null))
  const byCategory = computed(() => all.value.filter((b) => b.category_id !== null))

  function load(force = false): Promise<BudgetList> {
    if (data.value && !force) return Promise.resolve(data.value)
    pending ??= api
      .listBudgets()
      .then((list) => (data.value = list))
      .finally(() => (pending = null))
    return pending
  }

  /** Put a created or edited budget (the API returns it with this month's spending). */
  function upsert(budget: Budget): void {
    if (!data.value) return
    const list = data.value.budgets
    const i = list.findIndex((b) => b.id === budget.id)
    if (i === -1) list.push(budget)
    else list[i] = budget
  }

  function drop(id: number): void {
    if (data.value) data.value.budgets = data.value.budgets.filter((b) => b.id !== id)
  }

  return { data, all, overall, byCategory, load, upsert, drop }
})
