<script setup lang="ts" generic="T extends string | number">
import { computed, ref } from 'vue'
import AppIcon from './AppIcon.vue'
import OptionSheet, { type SheetOption } from './OptionSheet.vue'

/** A form field that shows the chosen option and opens an OptionSheet to change it. */
const props = defineProps<{
  label: string
  options: SheetOption<T>[]
  placeholder?: string
  /** Offer a first "none" row that sets null, labelled with this text. */
  noneLabel?: string
  searchable?: boolean
  error?: string
}>()
const model = defineModel<T | null>({ required: true })
const open = ref(false)

const selected = computed(() => props.options.find((o) => o.value === model.value))
const display = computed(() => (model.value === null ? (props.noneLabel ?? '') : (selected.value?.label ?? '')))
</script>

<template>
  <div>
    <span class="mb-2 block text-sm font-medium text-hint">{{ label }}</span>
    <button
      type="button"
      class="pressable flex w-full items-center gap-3 rounded-xl border-2 bg-surface px-3 py-2.5 text-left"
      :class="error ? 'border-danger' : 'border-transparent'"
      aria-haspopup="dialog"
      :aria-invalid="!!error"
      @click="open = true"
    >
      <span v-if="selected?.icon" class="text-xl" aria-hidden="true">{{ selected.icon }}</span>
      <span class="min-w-0 flex-1">
        <span class="block truncate" :class="display ? 'font-medium' : 'text-hint'">{{ display || placeholder }}</span>
        <span v-if="selected?.hint" class="block truncate text-xs text-hint">{{ selected.hint }}</span>
      </span>
      <AppIcon name="down" :size="18" class="shrink-0 text-hint" />
    </button>
    <span v-if="error" class="mt-1 block text-sm text-danger">{{ error }}</span>
    <OptionSheet
      v-model:open="open"
      v-model="model"
      :title="label"
      :options="options"
      :all-label="noneLabel"
      :searchable="searchable"
    />
  </div>
</template>
