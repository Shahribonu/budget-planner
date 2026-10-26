<script setup lang="ts">
defineProps<{ label: string; busy?: boolean }>()
defineEmits<{ click: [] }>()
</script>

<template>
  <div class="sticky bottom-0 z-10 bg-page/95 px-4 pt-3 pb-3 backdrop-blur" :style="{ paddingBottom: 'calc(var(--safe-bottom) + 12px)' }">
    <button
      type="button"
      class="pressable w-full rounded-xl bg-accent py-3.5 font-semibold text-on-accent disabled:opacity-60"
      :disabled="busy"
      @click="$emit('click')"
    >
      {{ busy ? $t('common.loading') : label }}
    </button>
  </div>
</template>
