<script setup lang="ts">
import { computed } from 'vue'
import { donutSegments } from '@/lib/insights'

export interface DonutSlice {
  key: string | number
  value: number
  color: string
  label: string
}

/** Pure-SVG donut; the slot renders in the centre. Slices are tappable through the legend. */
const props = defineProps<{ slices: DonutSlice[]; size?: number; label: string }>()
const segs = computed(() => donutSegments(props.slices.map((s) => s.value)))
const px = computed(() => props.size ?? 168)
</script>

<template>
  <div class="relative shrink-0" :style="{ width: `${px}px`, height: `${px}px` }">
    <svg viewBox="0 0 42 42" class="size-full -rotate-90" role="img" :aria-label="label">
      <circle cx="21" cy="21" r="15.9155" fill="none" stroke="var(--color-page)" stroke-width="5" />
      <circle
        v-for="(s, i) in slices"
        :key="s.key"
        cx="21"
        cy="21"
        r="15.9155"
        fill="none"
        :stroke="s.color"
        stroke-width="5"
        pathLength="100"
        :stroke-dasharray="`${segs[i]!.length} ${100 - segs[i]!.length}`"
        :stroke-dashoffset="-segs[i]!.offset"
        class="donut-seg"
        :style="{ animationDelay: `${i * 70}ms` }"
      />
    </svg>
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.donut-seg {
  animation: donut-in 700ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
}
@keyframes donut-in {
  from {
    stroke-dasharray: 0 100;
  }
}
@media (prefers-reduced-motion: reduce) {
  .donut-seg {
    animation: none;
  }
}
</style>
