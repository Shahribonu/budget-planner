<script setup lang="ts">
/** Placeholder rows while a list loads (instead of "Loading…" text). */
withDefaults(defineProps<{ rows?: number; hero?: boolean }>(), { rows: 4 })
</script>

<template>
  <div class="space-y-4 px-4" aria-busy="true" :aria-label="$t('common.loading')">
    <div v-if="hero" class="skeleton h-36 rounded-[24px]" />
    <div class="card space-y-4 p-4">
      <div v-for="i in rows" :key="i" class="flex items-center gap-3">
        <div class="skeleton size-11 shrink-0 rounded-full" />
        <div class="flex-1 space-y-2"><div class="skeleton h-4 w-2/3" /><div class="skeleton h-3 w-1/3" /></div>
      </div>
    </div>
  </div>
</template>
