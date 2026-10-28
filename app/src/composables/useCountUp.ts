import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

/** Animates a number toward `target` (ease-out, ~700 ms). Jumps straight there with reduced motion. */
export function useCountUp(target: Ref<number>, durationMs = 700) {
  const shown = ref(0)
  let frame = 0
  const reduce = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

  watch(
    target,
    (to) => {
      cancelAnimationFrame(frame)
      const from = shown.value
      if (reduce || from === to) {
        shown.value = to
        return
      }
      const start = performance.now()
      const step = (now: number) => {
        const p = Math.min(1, (now - start) / durationMs)
        const eased = 1 - (1 - p) ** 3
        // Whole minor units only: the display never shows a fraction of a tiyin.
        shown.value = Math.round(from + (to - from) * eased)
        if (p < 1) frame = requestAnimationFrame(step)
      }
      frame = requestAnimationFrame(step)
    },
    { immediate: true },
  )
  onBeforeUnmount(() => cancelAnimationFrame(frame))
  return shown
}
