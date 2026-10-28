import { defineStore } from 'pinia'
import { markRaw, ref, shallowRef } from 'vue'
import type { VoiceItem } from '@/api/types'
import { i18n } from '@/i18n'
import { MicUnsupportedError, WavRecorder } from '@/lib/recorder'
import { haptic } from '@/lib/telegram'
import { useToastStore } from './toast'
import { useUserStore } from './user'

/** Voice capture: the open sheet, its recorder, and an item handed to the transaction form. */
export const useVoiceStore = defineStore('voice', () => {
  const open = ref(false)
  const recorder = shallowRef<WavRecorder | null>(null)
  /** One recognised item for tx-new to prefill, with what was heard. */
  const prefill = ref<{ item: VoiceItem; text: string } | null>(null)

  /**
   * Call from the tap handler itself: the recorder's AudioContext must be created during
   * the user gesture or iOS keeps it suspended.
   */
  function newRecorder(): WavRecorder | null {
    try {
      return markRaw(new WavRecorder())
    } catch (e) {
      const t = i18n.global.t
      useToastStore().show(e instanceof MicUnsupportedError ? t('voice.unsupported') : t('common.error'))
      return null
    }
  }

  function begin(): void {
    const user = useUserStore()
    if (!user.me?.voice_enabled) {
      // No mic permission prompt when voice isn't available.
      haptic.selection()
      useToastStore().show(i18n.global.t('voice.coming_soon'))
      return
    }
    const r = newRecorder()
    if (!r) return
    recorder.value = r
    open.value = true
  }

  function takePrefill() {
    const p = prefill.value
    prefill.value = null
    return p
  }

  return { open, recorder, prefill, newRecorder, begin, takePrefill }
})
