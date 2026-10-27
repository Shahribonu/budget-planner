<script setup lang="ts">
defineProps<{ label: string }>()
const model = defineModel<boolean>({ required: true })
</script>

<template>
  <label class="flex items-center justify-between gap-3">
    <span class="font-medium">{{ label }}</span>
    <input v-model="model" type="checkbox" role="switch" class="peer sr-only" />
    <span
      aria-hidden="true"
      class="relative h-7 w-12 shrink-0 rounded-full transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-accent"
      :class="model ? 'bg-accent' : 'bg-hint/40'"
    >
      <span
        class="absolute top-0.5 left-0.5 size-6 rounded-full bg-white shadow transition-transform"
        :class="{ 'translate-x-5': model }"
      />
    </span>
  </label>
</template>
