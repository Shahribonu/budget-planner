<script setup lang="ts">
import { computed } from 'vue'
import { budgetLevel, budgetPercent } from '@/lib/budgets'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ spent: number; limit: number; thick?: boolean }>()

const pct = computed(() => budgetPercent(props.spent, props.limit))
const level = computed(() => budgetLevel(props.spent, props.limit))
const fill = computed(() => ({ ok: 'bg-accent', warn: 'bg-warning', over: 'bg-danger' })[level.value])
const text = computed(() => ({ ok: 'text-hint', warn: 'text-warning', over: 'text-danger' })[level.value])
</script>

<template>
  <span class="flex items-center gap-2">
    <span
      role="progressbar"
      :aria-valuenow="Math.min(pct, 100)"
      aria-valuemin="0"
      aria-valuemax="100"
      :aria-valuetext="`${pct}%`"
      class="block flex-1 overflow-hidden rounded-full bg-page"
      :class="thick ? 'h-2.5' : 'h-1.5'"
    >
      <span class="block h-full rounded-full" :class="fill" :style="{ width: `${Math.min(pct, 100)}%` }" />
    </span>
    <!-- Not colour alone: the number, plus an icon from 80% -->
    <span class="tabular flex shrink-0 items-center gap-0.5 text-xs font-semibold" :class="text">
      <AppIcon v-if="level !== 'ok'" name="alert" :size="13" />
      <span v-if="level !== 'ok'" class="sr-only">{{ level === 'over' ? $t('budgets.over_label') : $t('budgets.warn') }}:</span>
      {{ pct }}%
    </span>
  </span>
</template>
