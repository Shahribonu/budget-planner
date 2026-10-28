<script setup lang="ts">
/** Friendly inline illustrations for empty states. */
defineProps<{ kind?: 'wallet' | 'search' | 'plan' }>()
</script>

<template>
  <svg viewBox="0 0 160 120" class="mx-auto h-28 w-auto" aria-hidden="true">
    <ellipse cx="80" cy="108" rx="52" ry="6" fill="var(--color-page)" />
    <circle cx="128" cy="26" r="10" fill="var(--mint)" opacity=".35" />
    <circle cx="30" cy="40" r="6" fill="var(--brand)" opacity=".25" />
    <template v-if="kind === 'search'">
      <circle cx="72" cy="56" r="30" fill="var(--color-surface)" stroke="var(--brand)" stroke-width="7" />
      <path d="M94 78l22 22" stroke="var(--brand)" stroke-width="9" stroke-linecap="round" />
      <path d="M60 52h24M60 64h16" stroke="var(--color-hint)" stroke-width="5" stroke-linecap="round" />
    </template>
    <template v-else-if="kind === 'plan'">
      <rect x="40" y="24" width="80" height="78" rx="14" fill="var(--color-surface)" stroke="var(--brand)" stroke-width="5" />
      <rect x="40" y="24" width="80" height="20" rx="10" fill="var(--brand)" />
      <path d="M56 62l8 8 16-16" stroke="var(--mint)" stroke-width="7" fill="none" stroke-linecap="round" stroke-linejoin="round" />
      <path d="M88 64h16M88 84h16M56 84h16" stroke="var(--color-hint)" stroke-width="5" stroke-linecap="round" />
    </template>
    <template v-else>
      <rect x="30" y="34" width="100" height="66" rx="16" fill="var(--brand)" />
      <rect x="30" y="34" width="100" height="22" rx="11" fill="#7b5cff" />
      <rect x="96" y="58" width="40" height="24" rx="12" fill="var(--mint)" />
      <circle cx="108" cy="70" r="5" fill="#fff" />
      <path d="M58 22l6 10m16-14v12m18-8-6 10" stroke="var(--mint)" stroke-width="5" stroke-linecap="round" />
    </template>
  </svg>
</template>
