<script setup lang="ts">
import { computed, ref } from 'vue'
import { formatTyping } from '@/lib/money'

defineProps<{ unit: string; prefix?: boolean; label: string; error?: string; autofocus?: boolean }>()
const model = defineModel<string>({ required: true })
const input = ref<HTMLInputElement>()

function onInput(e: Event) {
  const el = e.target as HTMLInputElement
  const formatted = formatTyping(el.value)
  model.value = formatted
  // Regrouping can change the length; keep the caret at the end where digits are typed.
  if (el.value !== formatted) el.value = formatted
}

// Big, but stepped down so long amounts still fit a narrow card.
const sizeClass = computed(() => {
  const n = model.value.length
  return n > 11 ? 'text-[30px]' : n > 8 ? 'text-[36px]' : 'text-[44px]'
})

defineExpose({ focus: () => input.value?.focus() })
</script>

<template>
  <label class="block">
    <span class="sr-only">{{ label }}</span>
    <span
      class="flex items-center justify-center gap-2 rounded-[22px] px-3 py-3 transition-colors"
      :class="error ? 'bg-danger/8 ring-2 ring-danger' : 'focus-within:ring-2 focus-within:ring-accent/40'"
    >
      <input
        ref="input"
        :value="model"
        type="text"
        inputmode="decimal"
        autocomplete="off"
        placeholder="0"
        :autofocus="autofocus"
        class="display max-w-full min-w-[1ch] flex-initial bg-transparent text-center [field-sizing:content] leading-tight outline-none placeholder:text-hint/40 focus-visible:outline-none"
        :class="sizeClass"
        :aria-invalid="!!error"
        @input="onInput"
      />
      <!-- Currency as a chip, not part of the number -->
      <span class="shrink-0 rounded-full bg-accent-soft px-3 py-1 text-sm font-bold text-accent">{{ unit }}</span>
    </span>
    <span v-if="error" class="mt-1 block text-center text-sm text-danger">{{ error }}</span>
  </label>
</template>
