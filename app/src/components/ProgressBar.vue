<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: number; total: number; danger?: boolean; label?: string }>()
const pct = computed(() => (props.total > 0 ? Math.min(100, Math.max(0, (props.value / props.total) * 100)) : 0))
</script>

<template>
  <span
    role="progressbar"
    :aria-valuenow="Math.round(pct)"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-label="label"
    class="block h-1.5 overflow-hidden rounded-full bg-page"
  >
    <span class="block h-full rounded-full" :class="danger ? 'bg-danger' : 'bg-accent'" :style="{ width: `${pct}%` }" />
  </span>
</template>
