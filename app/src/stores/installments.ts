import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Installment } from '@/api/types'

export const useInstallmentsStore = defineStore('installments', () => {
  const all = ref<Installment[]>([])
  const loaded = ref(false)
  let pending: Promise<Installment[]> | null = null

  const byId = computed(() => new Map(all.value.map((d) => [d.id, d])))

  function load(force = false): Promise<Installment[]> {
    if (loaded.value && !force) return Promise.resolve(all.value)
    pending ??= api
      .listInstallments()
      .then((list) => {
        all.value = list
        loaded.value = true
        return list
      })
      .finally(() => (pending = null))
    return pending
  }

  function upsert(plan: Installment): Installment {
    const i = all.value.findIndex((d) => d.id === plan.id)
    if (i === -1) all.value.unshift(plan)
    else all.value[i] = plan
    return plan
  }

  function drop(id: number): void {
    all.value = all.value.filter((d) => d.id !== id)
  }

  return { all, loaded, byId, load, upsert, drop }
})
