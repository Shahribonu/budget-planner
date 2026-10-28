<script setup lang="ts">
import { inTelegram } from '@/lib/telegram'
import { useBack } from '@/composables/useBack'
import AppIcon from './AppIcon.vue'

defineProps<{ title: string; back?: boolean }>()
const goBack = useBack()
</script>

<template>
  <header class="flex min-h-12 items-center gap-2 px-4 pt-3 pb-2">
    <button
      v-if="back && !inTelegram"
      type="button"
      class="pressable -ml-2 rounded-full p-2 text-link"
      :aria-label="$t('common.cancel')"
      @click="goBack"
    >
      <AppIcon name="back" />
    </button>
    <h1 class="display min-w-0 flex-1 truncate text-[26px]">{{ title }}</h1>
    <slot />
  </header>
</template>
