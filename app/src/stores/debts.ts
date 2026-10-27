import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Debt } from '@/api/types'

export const useDebtsStore = defineStore('debts', () => {
  const all = ref<Debt[]>([])
  const loaded = ref(false)
  let pending: Promise<Debt[]> | null = null

  const byId = computed(() => new Map(all.value.map((d) => [d.id, d])))

  function load(force = false): Promise<Debt[]> {
    if (loaded.value && !force) return Promise.resolve(all.value)
    pending ??= api
      .listDebts()
      .then((list) => {
        all.value = list
        loaded.value = true
        return list
      })
      .finally(() => (pending = null))
    return pending
  }

  function upsert(debt: Debt): Debt {
    const i = all.value.findIndex((d) => d.id === debt.id)
    if (i === -1) all.value.unshift(debt)
    else all.value[i] = debt
    return debt
  }

  function drop(id: number): void {
    all.value = all.value.filter((d) => d.id !== id)
  }

  return { all, loaded, byId, load, upsert, drop }
})
