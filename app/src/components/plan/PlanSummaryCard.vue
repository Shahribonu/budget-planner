<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RouteLocationRaw } from 'vue-router'
import type { PlanLine, PlanView } from '@/api/types'
import { useFormat } from '@/composables/useFormat'
import { formatDay } from '@/lib/dates'
import AppIcon from '../AppIcon.vue'
import PlanInputSheet, { type PlanInputField } from './PlanInputSheet.vue'

/** income − reserved − savings − everyday = free, every figure with its source. */
const props = defineProps<{ view: PlanView }>()

const { money, lang } = useFormat()
const showLines = ref(false)
const expanded = ref(false)
const editing = ref<PlanInputField>('income')
const sheetOpen = ref(false)

const cur = computed(() => props.view.currency)
const inputs = computed(() => props.view.inputs)
const result = computed(() => props.view.result)

const { t } = useI18n()

/** Income split into reserved / savings / everyday / free, as shares of income (what's left of 100% if over). */
const split = computed(() => {
  const i = inputs.value
  const income = Math.max(i.income_minor, 1)
  const parts = [
    { key: 'reserved', label: t('mplan.reserved'), value: i.reserved_minor, color: 'var(--coral)' },
    { key: 'savings', label: t('mplan.savings'), value: i.savings_minor, color: 'var(--amber)' },
    { key: 'everyday', label: t('mplan.everyday'), value: i.buffer_minor, color: 'var(--brand)' },
    { key: 'free', label: t('mplan.free'), value: Math.max(result.value.free_minor, 0), color: 'var(--mint)' },
  ]
  let used = 0
  return parts.map((p) => {
    const pct = Math.max(0, Math.min(100 - used, (p.value / income) * 100))
    used += pct
    return { ...p, pct }
  })
})
const splitLabel = computed(() => split.value.map((s) => `${s.label} ${Math.round(s.pct)}%`).join(', '))

const LINE_ICON: Record<PlanLine['kind'], string> = { recurring: '🔁', installment: '🛍️', debt: '🤝' }

/** Each reserved line opens what it comes from. */
function lineRoute(l: PlanLine): RouteLocationRaw {
  if (l.kind === 'installment') return { name: 'installment', params: { id: l.ref_id } }
  if (l.kind === 'debt') return { name: 'debt', params: { id: l.ref_id } }
  return { name: 'recurring' }
}

function edit(field: PlanInputField) {
  editing.value = field
  sheetOpen.value = true
}
</script>

<template>
  <section class="card divide-y divide-separator overflow-hidden">
    <!-- Summary first: one big number and where the income goes; the rows open on tap -->
    <button type="button" class="block w-full px-4 pt-4 pb-3 text-left" :aria-expanded="expanded" @click="expanded = !expanded">
      <span class="eyebrow">{{ $t('mplan.free') }}</span>
      <span class="flex items-baseline justify-between gap-2">
        <span class="display text-[32px] leading-tight" :class="{ 'text-danger': result.free_minor < 0 }">{{ money(result.free_minor, cur) }}</span>
        <AppIcon name="down" :size="18" class="shrink-0 text-hint transition-transform" :class="{ 'rotate-180': expanded }" />
      </span>
      <span class="mt-3 flex h-2.5 overflow-hidden rounded-full bg-page" role="img" :aria-label="splitLabel">
        <span v-for="seg in split" :key="seg.key" class="h-full" :style="{ width: `${seg.pct}%`, background: seg.color }" />
      </span>
      <span class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-hint">
        <span v-for="seg in split" :key="seg.key" class="flex items-center gap-1">
          <span class="size-2 rounded-full" :style="{ background: seg.color }" />{{ seg.label }}
        </span>
      </span>
    </button>

    <template v-if="expanded">
    <!-- Income -->
    <button type="button" class="row" @click="edit('income')">
      <span class="min-w-0 flex-1">
        <span class="block font-medium">{{ $t('mplan.income') }}</span>
        <span class="block text-xs" :class="inputs.income_source === 'none' ? 'text-danger' : 'text-hint'">
          {{ $t(`mplan.src_${inputs.income_source}`) }}<template v-if="inputs.income_day">
            · {{ $t('mplan.salary_day_n', { n: inputs.income_day }) }}</template
          >
        </span>
      </span>
      <span class="tabular font-semibold text-income">{{ money(inputs.income_minor, cur, { signed: true }) }}</span>
      <AppIcon name="edit" :size="14" class="shrink-0 text-hint" />
    </button>

    <!-- Reserved: recurring bills, installments, debts -->
    <div>
      <button type="button" class="row" :aria-expanded="showLines" :disabled="!inputs.lines.length" @click="showLines = !showLines">
        <span class="min-w-0 flex-1">
          <span class="block font-medium">{{ $t('mplan.reserved') }}</span>
          <span class="block truncate text-xs text-hint">
            {{ inputs.lines.map((l) => l.title).join(', ') || '—' }}
          </span>
        </span>
        <span class="tabular font-semibold">{{ money(-inputs.reserved_minor, cur) }}</span>
        <AppIcon v-if="inputs.lines.length" name="down" :size="14" class="shrink-0 text-hint transition-transform" :class="{ 'rotate-180': showLines }" />
        <span v-else class="w-3.5" />
      </button>
      <ul v-if="showLines" class="pb-2">
        <li v-for="(l, i) in inputs.lines" :key="i">
          <RouterLink :to="lineRoute(l)" class="flex items-center gap-2 px-4 py-2 text-sm active:bg-page/60">
            <span aria-hidden="true">{{ LINE_ICON[l.kind] }}</span>
            <span class="min-w-0 flex-1 truncate">{{ l.title }}</span>
            <span v-if="l.due_on" class="shrink-0 text-xs text-hint">{{ formatDay(l.due_on, lang) }}</span>
            <span class="tabular shrink-0 text-hint">{{ money(l.amount_minor, cur) }}</span>
            <AppIcon name="chevron" :size="14" class="shrink-0 text-hint" />
          </RouterLink>
        </li>
      </ul>
    </div>

    <!-- Savings -->
    <button type="button" class="row" @click="edit('savings')">
      <span class="min-w-0 flex-1 font-medium">{{ $t('mplan.savings') }}</span>
      <span class="tabular font-semibold">{{ money(-inputs.savings_minor, cur) }}</span>
      <AppIcon name="edit" :size="14" class="shrink-0 text-hint" />
    </button>

    <!-- Everyday spending -->
    <button type="button" class="row" @click="edit('buffer')">
      <span class="min-w-0 flex-1">
        <span class="block font-medium">{{ $t('mplan.everyday') }}</span>
        <span class="block truncate text-xs text-hint">
          {{ inputs.buffer_source === 'override' ? $t('mplan.src_override') : $t(`mplan.buf_${inputs.buffer_source}`) }}
        </span>
      </span>
      <span class="tabular font-semibold">{{ money(-inputs.buffer_minor, cur) }}</span>
      <AppIcon name="edit" :size="14" class="shrink-0 text-hint" />
    </button>

    <!-- = Free -->
    <div class="px-4 py-3">
      <div class="flex items-baseline justify-between gap-3">
        <span class="font-semibold">= {{ $t('mplan.free') }}</span>
        <span class="tabular shrink-0 text-xl font-bold" :class="{ 'text-danger': result.free_minor < 0 }">
          {{ money(result.free_minor, cur) }}
        </span>
      </div>
      <div class="mt-1 flex flex-wrap justify-between gap-x-3 text-sm text-hint">
        <span>{{ $t('mplan.planned') }}: <span class="tabular">{{ money(result.allocated_minor, cur) }}</span></span>
        <span :class="{ 'font-semibold text-danger': result.left_minor < 0 }">
          {{ result.left_minor < 0 ? $t('mplan.short') : $t('mplan.left') }}:
          <span class="tabular">{{ money(Math.abs(result.left_minor), cur) }}</span>
        </span>
      </div>
      <p class="mt-1 text-xs text-hint">{{ $t('mplan.cash_now', { amount: money(inputs.cash_now_minor, cur) }) }}</p>
    </div>

    </template>

    <!-- No income known: the plan can't say much without it -->
    <div v-if="inputs.income_source === 'none'" class="space-y-2 bg-accent/10 px-4 py-3">
      <p class="text-sm">{{ $t('mplan.nudge_income') }}</p>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="pressable rounded-full bg-accent px-3 py-1.5 text-sm font-medium text-on-accent" @click="edit('income')">
          {{ $t('mplan.set_income') }}
        </button>
        <RouterLink
          :to="{ name: 'recurring', query: { new: 'income' } }"
          class="pressable rounded-full bg-surface px-3 py-1.5 text-sm font-medium text-link"
        >
          {{ $t('mplan.add_salary_btn') }}
        </RouterLink>
      </div>
    </div>

    <RouterLink :to="{ name: 'recurring' }" class="flex items-center justify-between px-4 py-3 text-sm text-link">
      {{ $t('mplan.regular') }}<AppIcon name="chevron" :size="16" />
    </RouterLink>
  </section>

  <PlanInputSheet v-model:open="sheetOpen" :field="editing" />
</template>

<style scoped>
@reference '../../style.css';

.row {
  @apply flex w-full items-center gap-2 px-4 py-2.5 text-left active:bg-page/60 disabled:active:bg-transparent;
}
</style>
