<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: number; total: number; danger?: boolean; label?: string; onDark?: boolean }>()
const pct = computed(() => (props.total > 0 ? Math.min(100, Math.max(0, (props.value / props.total) * 100)) : 0))
</script>

<template>
  <span
    role="progressbar"
    :aria-valuenow="Math.round(pct)"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-label="label"
    :class="onDark ? 'bg-white/25' : 'bg-page'"
    class="block h-1.5 overflow-hidden rounded-full"
  >
    <span class="block h-full rounded-full" :class="onDark ? (danger ? 'bg-[#ffb3aa]' : 'bg-white') : danger ? 'bg-danger' : 'bg-accent'" :style="{ width: `${pct}%` }" />
  </span>
</template>
