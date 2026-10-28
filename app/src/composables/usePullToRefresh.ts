import { onBeforeUnmount, onMounted, ref } from 'vue'
import { haptic } from '@/lib/telegram'

const THRESHOLD = 70

/** Pull down at the top of the page to refresh. Touch only; mouse users have the normal reload paths. */
export function usePullToRefresh(refresh: () => Promise<unknown>) {
  const pull = ref(0)
  const refreshing = ref(false)
  let startY: number | null = null

  function onStart(e: TouchEvent) {
    startY = window.scrollY <= 0 && !refreshing.value ? (e.touches[0]?.clientY ?? null) : null
  }
  function onMove(e: TouchEvent) {
    if (startY === null) return
    const dy = (e.touches[0]?.clientY ?? startY) - startY
    pull.value = dy > 0 ? Math.min(dy * 0.5, THRESHOLD * 1.4) : 0
  }
  async function onEnd() {
    if (startY === null) return
    startY = null
    if (pull.value >= THRESHOLD) {
      refreshing.value = true
      haptic.impact('light')
      try {
        await refresh()
      } finally {
        refreshing.value = false
      }
    }
    pull.value = 0
  }

  onMounted(() => {
    window.addEventListener('touchstart', onStart, { passive: true })
    window.addEventListener('touchmove', onMove, { passive: true })
    window.addEventListener('touchend', onEnd)
  })
  onBeforeUnmount(() => {
    window.removeEventListener('touchstart', onStart)
    window.removeEventListener('touchmove', onMove)
    window.removeEventListener('touchend', onEnd)
  })
  return { pull, refreshing, threshold: THRESHOLD }
}
