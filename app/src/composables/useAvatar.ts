import { ref } from 'vue'
import { api } from '@/api/client'
import { telegramPhotoUrl } from '@/lib/telegram'

// One lookup per session, shared by Home and Profile.
const src = ref<string | null>(null)
let started = false

async function resolve(): Promise<void> {
  const direct = telegramPhotoUrl()
  if (direct) {
    src.value = direct
    return
  }
  const blob = await api.getAvatar()
  if (blob && blob.size > 0) src.value = URL.createObjectURL(blob)
}

/**
 * The user's Telegram photo: photo_url from initData, else the server proxy
 * (/me/avatar), else null so callers show initials. `failed()` drops a URL
 * that turned out not to load (e.g. an expired photo_url) and tries the proxy.
 */
export function useAvatar() {
  if (!started) {
    started = true
    void resolve()
  }
  async function failed() {
    const wasDirect = src.value !== null && src.value === telegramPhotoUrl()
    src.value = null
    if (wasDirect) {
      const blob = await api.getAvatar()
      if (blob && blob.size > 0) src.value = URL.createObjectURL(blob)
    }
  }
  return { src, failed }
}
