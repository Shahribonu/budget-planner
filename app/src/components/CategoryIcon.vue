<script setup lang="ts">
import { computed } from 'vue'
import type { Category } from '@/api/types'
import { categoryStyle } from '@/lib/categoryStyle'
import { prefs } from '@/lib/prefs'

/** Round category badge: white glyph on the category colour, or the emoji (Profile → App). */
const props = defineProps<{ category?: Category; size?: number }>()

const style = computed(() => categoryStyle(props.category))
const px = computed(() => props.size ?? 44)
</script>

<template>
  <span
    class="flex shrink-0 items-center justify-center rounded-full"
    :style="{
      width: `${px}px`,
      height: `${px}px`,
      background: prefs.categoryStyle === 'icons' ? style.color : 'var(--color-page)',
      fontSize: `${px * 0.5}px`,
    }"
    aria-hidden="true"
  >
    <svg
      v-if="prefs.categoryStyle === 'icons'"
      :width="px * 0.5"
      :height="px * 0.5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="#fff"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path :d="style.path" />
    </svg>
    <template v-else>{{ category?.icon ?? '🏷️' }}</template>
  </span>
</template>
