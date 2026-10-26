import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Category, CategoryKind } from '@/api/types'

export const useCategoriesStore = defineStore('categories', () => {
  const all = ref<Category[]>([])
  let pending: Promise<Category[]> | null = null

  const byId = computed(() => new Map(all.value.map((c) => [c.id, c])))

  function ofKind(kind: CategoryKind): Category[] {
    return all.value.filter((c) => c.kind === kind)
  }

  function load(): Promise<Category[]> {
    if (all.value.length) return Promise.resolve(all.value)
    pending ??= api
      .listCategories()
      .then((list) => (all.value = list))
      .finally(() => (pending = null))
    return pending
  }

  return { all, byId, ofKind, load }
})
