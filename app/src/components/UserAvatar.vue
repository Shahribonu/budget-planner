<script setup lang="ts">
import { computed } from 'vue'
import { useAvatar } from '@/composables/useAvatar'
import { useUserStore } from '@/stores/user'

/** Telegram photo in a ring, or initials on mint when there's no photo. */
defineProps<{ size?: number; ring?: string }>()
const user = useUserStore()
const { src, failed } = useAvatar()
const initials = computed(() => {
  const name = (user.me?.first_name ?? '?').trim()
  return name.slice(0, 1).toUpperCase()
})
</script>

<template>
  <span
    class="inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full"
    :style="{ width: `${size ?? 44}px`, height: `${size ?? 44}px`, boxShadow: `0 0 0 3px ${ring ?? '#fff'}` }"
  >
    <img v-if="src" :src="src" alt="" class="size-full object-cover" @error="failed" />
    <span
      v-else
      class="flex size-full items-center justify-center bg-mint font-bold text-[#0b2b20]"
      :style="{ fontSize: `${(size ?? 44) * 0.42}px` }"
      aria-hidden="true"
    >
      {{ initials }}
    </span>
  </span>
</template>
