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

  function error(): void {
    show(i18n.global.t('common.error'))
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

  return { current, show, error, runAction }
})
