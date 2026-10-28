<script setup lang="ts">
import { computed } from 'vue'
import { budgetLevel, budgetPercent } from '@/lib/budgets'
import AppIcon from './AppIcon.vue'

/** Circular budget progress: colour by level, plus the % and an icon from 80% (never colour alone). */
const props = defineProps<{ spent: number; limit: number; size?: number }>()

const px = computed(() => props.size ?? 72)
const R = 30
const C = 2 * Math.PI * R
const pct = computed(() => budgetPercent(props.spent, props.limit))
const level = computed(() => budgetLevel(props.spent, props.limit))
const stroke = computed(() => ({ ok: 'var(--brand)', warn: 'var(--amber)', over: 'var(--red)' })[level.value])
const dash = computed(() => `${(Math.min(pct.value, 100) / 100) * C} ${C}`)
</script>

<template>
  <span class="relative inline-flex items-center justify-center" :style="{ width: `${px}px`, height: `${px}px` }">
    <svg :width="px" :height="px" viewBox="0 0 72 72" class="-rotate-90" aria-hidden="true">
      <circle cx="36" cy="36" :r="R" fill="none" stroke="var(--color-page)" stroke-width="7" />
      <circle cx="36" cy="36" :r="R" fill="none" :stroke="stroke" stroke-width="7" stroke-linecap="round" :stroke-dasharray="dash" />
    </svg>
    <span class="absolute flex flex-col items-center leading-none" role="img" :aria-label="`${pct}%`">
      <AppIcon v-if="level !== 'ok'" name="alert" :size="12" :class="level === 'over' ? 'text-danger' : 'text-warning'" />
      <span class="tabular text-sm font-bold" :class="{ 'text-danger': level === 'over', 'text-warning': level === 'warn' }">{{ pct }}%</span>
    </span>
  </span>
</template>
