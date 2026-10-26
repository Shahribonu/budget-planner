<script setup lang="ts">
import { ref } from 'vue'
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

defineExpose({ focus: () => input.value?.focus() })
</script>

<template>
  <label class="block">
    <span class="sr-only">{{ label }}</span>
    <span
      class="flex items-baseline justify-center gap-2 border-b-2 px-2 pb-2"
      :class="error ? 'border-danger' : 'border-separator focus-within:border-accent'"
    >
      <span v-if="prefix" class="text-3xl font-semibold text-hint">{{ unit }}</span>
      <input
        ref="input"
        :value="model"
        type="text"
        inputmode="decimal"
        autocomplete="off"
        placeholder="0"
        :autofocus="autofocus"
        class="tabular max-w-full min-w-[1ch] flex-initial bg-transparent text-center [field-sizing:content] text-4xl font-bold outline-none placeholder:text-hint/50"
        :aria-invalid="!!error"
        @input="onInput"
      />
      <span v-if="!prefix" class="shrink-0 text-xl font-semibold text-hint">{{ unit }}</span>
    </span>
    <span v-if="error" class="mt-1 block text-center text-sm text-danger">{{ error }}</span>
  </label>
</template>
