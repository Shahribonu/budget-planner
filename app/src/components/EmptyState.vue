<script setup lang="ts">
import EmptyArt from './EmptyArt.vue'

/** `art` draws an illustration; `icon` is a smaller emoji for minor states (errors, sub-pages). */
defineProps<{ icon?: string; art?: 'wallet' | 'search' | 'plan'; title?: string; body: string; action?: string }>()
defineEmits<{ action: [] }>()
</script>

<template>
  <div class="flex flex-col items-center px-6 py-10 text-center">
    <EmptyArt v-if="art" :kind="art" class="mb-4" />
    <div v-else-if="icon" class="mb-3 text-5xl" aria-hidden="true">{{ icon }}</div>
    <h2 v-if="title" class="display mb-1 text-xl">{{ title }}</h2>
    <p class="max-w-xs text-sm text-hint">{{ body }}</p>
    <button
      v-if="action"
      type="button"
      class="pressable mt-5 rounded-2xl bg-accent px-6 py-3.5 font-semibold text-on-accent shadow-lg shadow-accent/25"
      @click="$emit('action')"
    >
      {{ action }}
    </button>
  </div>
</template>
