<script setup lang="ts">
const PATHS = {
  home: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z',
  list: 'M8 6h13M8 12h13M8 18h13M3.5 6h.01M3.5 12h.01M3.5 18h.01',
  target: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zm0-5a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
  chart: 'M4 20V11M10 20V5M16 20v-7M21 20H3',
  user: 'M20 21a8 8 0 0 0-16 0M12 13a5 5 0 1 0 0-10 5 5 0 0 0 0 10z',
  plus: 'M12 5v14M5 12h14',
  search: 'M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16zm10 2-4.3-4.3',
  x: 'M18 6 6 18M6 6l12 12',
  out: 'M7 17 17 7M8 7h9v9',
  in: 'M17 7 7 17m9 0H7V8',
  swap: 'M7 4 3 8l4 4M3 8h14m0 12 4-4-4-4m4 4H7',
  flag: 'M5 21V4m0 0h12l-2 4 2 4H5',
  chevron: 'm9 18 6-6-6-6',
  down: 'm6 9 6 6 6-6',
  check: 'M20 6 9 17l-5-5',
  back: 'm15 18-6-6 6-6',
  trash: 'M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6',
  archive: 'M3 4h18v4H3zm2 4v12h14V8m-9 4h4',
} as const

export type IconName = keyof typeof PATHS

defineProps<{ name: IconName; size?: number }>()
</script>

<template>
  <svg
    :width="size ?? 24"
    :height="size ?? 24"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="PATHS[name]" />
  </svg>
</template>
