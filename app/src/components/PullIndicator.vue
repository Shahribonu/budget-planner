<script setup lang="ts">
defineProps<{ pull: number; refreshing: boolean; threshold: number }>()
</script>

<template>
  <div
    v-if="pull > 0 || refreshing"
    class="pointer-events-none fixed inset-x-0 z-30 flex justify-center"
    :style="{ top: `calc(var(--safe-top) + ${Math.min(pull, threshold) * 0.6}px)` }"
    aria-hidden="true"
  >
    <span
      class="block size-8 rounded-full border-[3px] border-accent/25 border-t-accent bg-surface shadow"
      :class="{ 'animate-spin': refreshing }"
      :style="{ transform: refreshing ? undefined : `rotate(${pull * 4}deg)`, opacity: Math.min(1, pull / threshold + (refreshing ? 1 : 0)) }"
    />
  </div>
</template>
