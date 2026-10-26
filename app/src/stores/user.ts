import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { User, UserPatch } from '@/api/types'
import { currentLanguage, setLanguage } from '@/i18n'

export const useUserStore = defineStore('user', () => {
  const me = ref<User | null>(null)
  const error = ref<unknown>(null)
  let pending: Promise<User> | null = null

  const language = computed(() => me.value?.language ?? currentLanguage())
  const mainCurrency = computed(() => me.value?.main_currency ?? 'UZS')

  function apply(user: User): User {
    me.value = user
    void setLanguage(user.language)
    return user
  }

  function load(force = false): Promise<User> {
    if (me.value && !force) return Promise.resolve(me.value)
    pending ??= api
      .getMe()
      .then((u) => {
        error.value = null
        return apply(u)
      })
      .catch((e: unknown) => {
        error.value = e
        throw e
      })
      .finally(() => (pending = null))
    return pending
  }

  async function update(patch: UserPatch): Promise<User> {
    return apply(await api.patchMe(patch))
  }

  return { me, error, language, mainCurrency, load, update }
})
