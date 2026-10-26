import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Account, AccountCreate, AccountPatch } from '@/api/types'

export const useAccountsStore = defineStore('accounts', () => {
  // Always includes archived accounts so history rows can still show their names.
  const all = ref<Account[]>([])
  const loaded = ref(false)
  let pending: Promise<Account[]> | null = null

  const active = computed(() => all.value.filter((a) => !a.archived))
  const byId = computed(() => new Map(all.value.map((a) => [a.id, a])))

  function load(force = false): Promise<Account[]> {
    if (loaded.value && !force) return Promise.resolve(all.value)
    pending ??= api
      .listAccounts(true)
      .then((list) => {
        all.value = list
        loaded.value = true
        return list
      })
      .finally(() => (pending = null))
    return pending
  }

  function upsert(account: Account): Account {
    const i = all.value.findIndex((a) => a.id === account.id)
    if (i === -1) all.value.push(account)
    else all.value[i] = account
    return account
  }

  async function create(input: AccountCreate): Promise<Account> {
    return upsert(await api.createAccount(input))
  }

  async function update(id: number, patch: AccountPatch): Promise<Account> {
    return upsert(await api.patchAccount(id, patch))
  }

  return { all, active, byId, loaded, load, create, update }
})
