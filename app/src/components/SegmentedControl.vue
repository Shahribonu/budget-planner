<script setup lang="ts" generic="T extends string">
import { haptic } from '@/lib/telegram'

defineProps<{ options: { value: T; label: string }[]; label?: string }>()
const model = defineModel<T>({ required: true })

function select(value: T) {
  if (value === model.value) return
  haptic.selection()
  model.value = value
}
</script>

<template>
  <div role="radiogroup" :aria-label="label" class="flex gap-1 rounded-xl bg-page p-1">
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      role="radio"
      :aria-checked="model === opt.value"
      class="min-w-0 flex-1 truncate rounded-lg px-1 py-2 text-[13px] font-medium transition-colors"
      :class="model === opt.value ? 'bg-bg text-text shadow-sm' : 'text-hint'"
      @click="select(opt.value)"
    >
      {{ opt.label }}
    </button>
  </div>
</template>
