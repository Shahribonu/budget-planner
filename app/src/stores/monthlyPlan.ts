import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Currency, NewPlanItem, PlanItemPatch, PlanPatch, PlanView } from '@/api/types'

/** The spending plan on the Plan tab. Every write returns the whole recalculated plan. */
export const useMonthlyPlanStore = defineStore('monthlyPlan', () => {
  const view = ref<PlanView | null>(null)
  // null until the first load: the server picks the month (current, or next from the 20th) and currency.
  const month = ref<string | null>(null)
  const currency = ref<Currency | null>(null)
  let seq = 0

  const planned = computed(() => view.value?.items.filter((i) => i.status === 'planned') ?? [])
  const done = computed(() => view.value?.items.filter((i) => i.status !== 'planned') ?? [])

  function apply(v: PlanView): PlanView {
    view.value = v
    month.value = v.month
    currency.value = v.currency
    return v
  }

  async function load(): Promise<PlanView> {
    const mine = ++seq
    const v = await api.getPlan(month.value ?? undefined, currency.value ?? undefined)
    // A quick month switch can return out of order; keep the latest request only.
    return mine === seq ? apply(v) : v
  }

  function key(): [string, Currency] {
    if (!view.value) throw new Error('plan not loaded')
    return [view.value.month, view.value.currency]
  }

  return {
    view,
    month,
    currency,
    planned,
    done,
    load,
    patch: async (p: PlanPatch) => apply(await api.patchPlan(...key(), p)),
    addItems: async (items: NewPlanItem[]) => apply(await api.addPlanItems(...key(), items)),
    patchItem: async (id: number, p: PlanItemPatch) => apply(await api.patchPlanItem(id, p)),
    deleteItem: async (id: number) => apply(await api.deletePlanItem(id)),
  }
})
