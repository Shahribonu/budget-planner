import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { User } from '@/api/types'
import { useToastStore } from './toast'
import { useUserStore } from './user'
import { useVoiceStore } from './voice'

// The Telegram wrapper reads window at import; these tests run in node.
vi.mock('@/lib/telegram', () => ({
  haptic: { selection: () => {}, impact: () => {}, success: () => {}, error: () => {}, warning: () => {} },
  initData: () => '',
}))

describe('voice store', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('says "coming soon" and never touches the mic when voice is off', () => {
    useUserStore().me = { voice_enabled: false } as User
    const voice = useVoiceStore()
    voice.begin()
    expect(voice.open).toBe(false)
    expect(voice.recorder).toBeNull()
    expect(useToastStore().current?.message).toBeTruthy()
  })

  it('hands one prefill to the form, once', () => {
    const voice = useVoiceStore()
    voice.prefill = {
      text: 'taksi 25',
      item: { type: 'expense', amount_minor: 2500, currency: 'UZS', description: 'taksi', category_id: 3, account_id: null, occurred_on: '2026-10-28', needs_review: true },
    }
    expect(voice.takePrefill()?.text).toBe('taksi 25')
    expect(voice.takePrefill()).toBeNull()
  })
})
