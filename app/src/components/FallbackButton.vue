<script setup lang="ts">
defineProps<{ label: string; busy?: boolean; disabled?: boolean }>()
defineEmits<{ click: [] }>()
</script>

<template>
  <div class="sticky bottom-0 z-10 bg-canvas/95 px-4 pt-3 pb-3 backdrop-blur" :style="{ paddingBottom: 'calc(var(--safe-bottom) + 12px)' }">
    <button
      type="button"
      class="pressable w-full rounded-2xl bg-accent py-4 font-semibold text-on-accent shadow-lg shadow-accent/25 disabled:opacity-60"
      :disabled="busy || disabled"
      @click="$emit('click')"
    >
      {{ busy ? $t('common.loading') : label }}
    </button>
  </div>
</template>
