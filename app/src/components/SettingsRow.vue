<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import AppIcon, { type IconName } from './AppIcon.vue'

/** A grouped-list row: soft icon tile, label, value, chevron. A link with `to`, else a button. */
defineProps<{ icon: IconName; tone?: string; label: string; value?: string; to?: RouteLocationRaw }>()
defineEmits<{ click: [] }>()
</script>

<template>
  <component
    :is="to ? 'RouterLink' : 'button'"
    :to="to"
    :type="to ? undefined : 'button'"
    class="flex min-h-14 w-full items-center gap-3 px-4 py-2.5 text-left transition-colors active:bg-page/60"
    @click="$emit('click')"
  >
    <span
      class="flex size-9 shrink-0 items-center justify-center rounded-xl"
      :style="{ background: `color-mix(in srgb, ${tone ?? 'var(--brand)'} 14%, transparent)`, color: tone ?? 'var(--brand)' }"
    >
      <AppIcon :name="icon" :size="18" />
    </span>
    <span class="min-w-0 flex-1 truncate font-medium">{{ label }}</span>
    <slot>
      <span v-if="value" class="max-w-[45%] truncate text-sm text-hint">{{ value }}</span>
    </slot>
    <AppIcon name="chevron" :size="16" class="shrink-0 text-hint" />
  </component>
</template>
