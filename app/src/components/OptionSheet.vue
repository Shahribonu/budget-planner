<script setup lang="ts" generic="T extends string | number">
import { computed, ref, watch } from 'vue'
import { haptic } from '@/lib/telegram'
import AppIcon from './AppIcon.vue'
import BottomSheet from './BottomSheet.vue'

export interface SheetOption<V> {
  value: V
  label: string
  /** Emoji or short text shown before the label. */
  icon?: string
  /** Secondary line under the label. */
  hint?: string
  /** Options with the same group are listed under one heading, in order. */
  group?: string
}

const props = defineProps<{
  title: string
  options: SheetOption<T>[]
  /** When set, an extra first row selects `null` (e.g. "All"). */
  allLabel?: string
  searchable?: boolean
}>()
const open = defineModel<boolean>('open', { required: true })
const model = defineModel<T | null>({ required: true })

const query = ref('')
watch(open, (o) => !o && (query.value = ''))

const sections = computed(() => {
  const q = query.value.trim().toLowerCase()
  const out: { group?: string; items: SheetOption<T>[] }[] = []
  for (const opt of props.options) {
    if (q && !`${opt.label} ${opt.hint ?? ''}`.toLowerCase().includes(q)) continue
    const last = out[out.length - 1]
    if (last && last.group === opt.group) last.items.push(opt)
    else out.push({ group: opt.group, items: [opt] })
  }
  return out
})

// Focus the selected row (or the first one), not the search field, so mobile keyboards stay closed.
const autofocusValue = computed<T | null | undefined>(() => {
  if (model.value === null ? props.allLabel : props.options.some((o) => o.value === model.value)) return model.value
  return props.allLabel ? null : props.options[0]?.value
})

function select(value: T | null) {
  haptic.impact('light')
  model.value = value
  open.value = false
}
</script>

<template>
  <BottomSheet v-model:open="open" :title="title">
    <label v-if="searchable" class="mx-4 mb-2 flex shrink-0 items-center gap-2 rounded-xl bg-page px-3 py-2.5">
      <AppIcon name="search" :size="18" class="text-hint" />
      <input
        v-model="query"
        type="search"
        enterkeyhint="search"
        :placeholder="$t('common.search')"
        :aria-label="$t('common.search')"
        class="min-w-0 flex-1 bg-transparent outline-none placeholder:text-hint"
      />
    </label>

    <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain pb-2">
      <button
        v-if="allLabel && !query"
        type="button"
        class="option"
        :aria-pressed="model === null"
        :data-autofocus="autofocusValue === null || undefined"
        @click="select(null)"
      >
        <span class="min-w-0 flex-1 truncate font-medium">{{ allLabel }}</span>
        <AppIcon v-if="model === null" name="check" :size="20" class="shrink-0 text-accent" />
      </button>

      <template v-for="section in sections" :key="section.group ?? ''">
        <h3 v-if="section.group" class="px-4 pt-3 pb-1 text-xs font-semibold tracking-wide text-hint uppercase">
          {{ section.group }}
        </h3>
        <button
          v-for="opt in section.items"
          :key="opt.value"
          type="button"
          class="option"
          :aria-pressed="model === opt.value"
          :data-autofocus="autofocusValue === opt.value || undefined"
          @click="select(opt.value)"
        >
          <span v-if="opt.icon" class="w-7 shrink-0 text-center text-xl" aria-hidden="true">{{ opt.icon }}</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate font-medium">{{ opt.label }}</span>
            <span v-if="opt.hint" class="block truncate text-sm text-hint">{{ opt.hint }}</span>
          </span>
          <AppIcon v-if="model === opt.value" name="check" :size="20" class="shrink-0 text-accent" />
        </button>
      </template>

      <p v-if="!sections.length" class="px-4 py-6 text-center text-sm text-hint">{{ $t('common.no_results') }}</p>
    </div>
  </BottomSheet>
</template>

<style scoped>
@reference '../style.css';

.option {
  @apply flex min-h-12 w-full items-center gap-3 px-4 py-2 text-left outline-none active:bg-page focus-visible:bg-page;
}
.option[aria-pressed='true'] {
  @apply bg-accent/10;
}
</style>
