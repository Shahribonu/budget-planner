import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useToastStore } from './toast'

describe('toast rate limiting', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  it('keeps the rate-limit message over a following generic error', () => {
    const toast = useToastStore()
    toast.rateLimited(30)
    const msg = toast.current?.message
    toast.error() // the caller's own catch block
    expect(toast.current?.message).toBe(msg)
    expect(msg).toContain('30')
  })

  it('lets generic errors through again after a moment', () => {
    const toast = useToastStore()
    toast.rateLimited()
    vi.advanceTimersByTime(2000)
    toast.error()
    expect(toast.current?.message).not.toContain('30')
    expect(toast.current?.message).toBeTruthy()
  })
})
