<script setup lang="ts">
defineProps<{ icon?: string; title?: string; body: string; action?: string }>()
defineEmits<{ action: [] }>()
</script>

<template>
  <div class="flex flex-col items-center px-6 py-10 text-center">
    <div v-if="icon" class="mb-3 text-5xl" aria-hidden="true">{{ icon }}</div>
    <h2 v-if="title" class="mb-1 text-lg font-semibold">{{ title }}</h2>
    <p class="max-w-xs text-sm text-hint">{{ body }}</p>
    <button
      v-if="action"
      type="button"
      class="pressable mt-5 rounded-xl bg-accent px-6 py-3 font-semibold text-on-accent"
      @click="$emit('action')"
    >
      {{ action }}
    </button>
  </div>
</template>
