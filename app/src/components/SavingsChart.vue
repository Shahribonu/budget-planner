<script setup lang="ts">
import { computed } from 'vue'
import type { Currency } from '@/api/types'
import { useFormat } from '@/composables/useFormat'
import { formatMonth } from '@/lib/dates'
import type { SeriesPoint } from '@/lib/savings'

/** Net saved per month as bars: mint up, coral down, the current month highlighted. */
const props = defineProps<{ series: SeriesPoint[]; currency: Currency; current: string; label: string }>()
const { lang } = useFormat()

const W = 300
const H = 120
const LABEL = 16

const max = computed(() => Math.max(1, ...props.series.map((p) => Math.abs(p.net_minor))))
const hasNegative = computed(() => props.series.some((p) => p.net_minor < 0))
// The zero line sits at the bottom unless something was taken out.
const zeroY = computed(() => (hasNegative.value ? (H - LABEL) * 0.62 : H - LABEL))
const slot = computed(() => W / Math.max(1, props.series.length))

const bars = computed(() =>
  props.series.map((p, i) => {
    const up = p.net_minor >= 0
    const room = up ? zeroY.value - 4 : H - LABEL - zeroY.value - 2
    const h = p.net_minor === 0 ? 0 : Math.max(3, (Math.abs(p.net_minor) / max.value) * room)
    const name = formatMonth(p.month, lang.value)
    return {
      key: p.month,
      x: i * slot.value + slot.value * 0.2,
      w: slot.value * 0.6,
      y: up ? zeroY.value - h : zeroY.value,
      h,
      up,
      current: p.month === props.current,
      initial: name.charAt(0).toUpperCase(),
    }
  }),
)
</script>

<template>
  <svg :viewBox="`0 0 ${W} ${H}`" class="h-auto w-full" role="img" :aria-label="label">
    <line x1="0" :y1="zeroY" :x2="W" :y2="zeroY" stroke="var(--color-separator)" stroke-width="1" />
    <g v-for="b in bars" :key="b.key">
      <rect
        v-if="b.h > 0"
        :x="b.x"
        :y="b.y"
        :width="b.w"
        :height="b.h"
        rx="3"
        :fill="b.up ? 'var(--mint)' : 'var(--coral)'"
        :opacity="b.current ? 1 : 0.55"
      />
      <rect v-if="b.current" :x="b.x - 2" :y="2" :width="b.w + 4" :height="H - LABEL - 2" rx="5" fill="var(--brand)" opacity="0.08" />
      <text
        :x="b.x + b.w / 2"
        :y="H - 3"
        text-anchor="middle"
        font-size="10"
        :font-weight="b.current ? 700 : 400"
        :fill="b.current ? 'var(--brand)' : 'var(--color-hint)'"
      >
        {{ b.initial }}
      </text>
    </g>
  </svg>
</template>
