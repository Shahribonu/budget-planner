import { defineStore } from 'pinia'
import { ref } from 'vue'
import { i18n } from '@/i18n'

export interface ToastAction {
  label: string
  run: () => Promise<void> | void
}

export interface Toast {
  id: number
  message: string
  action?: ToastAction
}

export const useToastStore = defineStore('toast', () => {
  const current = ref<Toast | null>(null)
  let seq = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  function show(message: string, action?: ToastAction, ms = action ? 5000 : 2500): void {
    clearTimeout(timer)
    const id = ++seq
    current.value = { id, message, action }
    timer = setTimeout(() => {
      if (current.value?.id === id) current.value = null
    }, ms)
  }

  // After a rate-limit toast, callers' own generic error toasts would replace it; hold them briefly.
  let holdUntil = 0

  function error(): void {
    if (Date.now() < holdUntil) return
    show(i18n.global.t('common.error'))
  }

  function rateLimited(retryAfter?: number): void {
    show(retryAfter && retryAfter > 0 ? i18n.global.t('common.rate_limited_s', { s: retryAfter }) : i18n.global.t('common.rate_limited'))
    holdUntil = Date.now() + 1500
  }

  async function runAction(): Promise<void> {
    const action = current.value?.action
    current.value = null
    clearTimeout(timer)
    if (!action) return
    try {
      await action.run()
    } catch {
      error()
    }
  }

  return { current, show, error, rateLimited, runAction }
})
