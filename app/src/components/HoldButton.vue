<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { haptic } from '@/lib/telegram'

/**
 * Press and hold to confirm a destructive action. Works with touch, mouse and keyboard
 * (hold Space or Enter); letting go early cancels.
 */
const props = withDefaults(defineProps<{ label: string; holdMs?: number; disabled?: boolean }>(), { holdMs: 2000 })
const emit = defineEmits<{ confirm: [] }>()

const progress = ref(0)
let start = 0
let frame = 0
let active = false

function tick(now: number) {
  progress.value = Math.min(1, (now - start) / props.holdMs)
  if (progress.value >= 1) {
    active = false
    haptic.warning()
    emit('confirm')
    return
  }
  frame = requestAnimationFrame(tick)
}

function begin() {
  if (props.disabled || active) return
  active = true
  haptic.impact('light')
  start = performance.now()
  frame = requestAnimationFrame(tick)
}

function cancel() {
  if (!active) return
  active = false
  cancelAnimationFrame(frame)
  progress.value = 0
}

function onKey(e: KeyboardEvent, down: boolean) {
  if (e.key !== ' ' && e.key !== 'Enter') return
  e.preventDefault()
  if (down && !e.repeat) begin()
  else if (!down) cancel()
}

onBeforeUnmount(() => cancelAnimationFrame(frame))
</script>

<template>
  <button
    type="button"
    class="relative w-full overflow-hidden rounded-2xl bg-danger/12 py-4 font-semibold text-danger select-none disabled:opacity-50"
    :disabled="disabled"
    :aria-label="label"
    @pointerdown="begin"
    @pointerup="cancel"
    @pointerleave="cancel"
    @pointercancel="cancel"
    @contextmenu.prevent
    @keydown="onKey($event, true)"
    @keyup="onKey($event, false)"
  >
    <span class="absolute inset-y-0 left-0 bg-danger" :style="{ width: `${progress * 100}%` }" aria-hidden="true" />
    <span class="relative" :class="{ 'text-white': progress > 0.5 }">{{ label }}</span>
  </button>
</template>
