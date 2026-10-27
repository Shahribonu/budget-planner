<script setup lang="ts">
import type { PlanPriority } from '@/api/types'

defineProps<{ priority: PlanPriority }>()
defineEmits<{ cycle: [] }>()

const STYLE: Record<PlanPriority, string> = {
  must: 'bg-accent text-on-accent',
  should: 'bg-accent/15 text-accent',
  nice: 'bg-page text-hint',
}
</script>

<template>
  <button
    type="button"
    class="pressable shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold"
    :class="STYLE[priority]"
    :aria-label="$t('mplan.prio_change', { p: $t(`mplan.${priority}`) })"
    @click.stop="$emit('cycle')"
  >
    {{ $t(`mplan.${priority}`) }}
  </button>
</template>
