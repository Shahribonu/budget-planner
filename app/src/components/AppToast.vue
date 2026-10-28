<script setup lang="ts">
import { useToastStore } from '@/stores/toast'

const toast = useToastStore()
</script>

<template>
  <Transition
    enter-from-class="translate-y-4 opacity-0"
    leave-to-class="translate-y-4 opacity-0"
    enter-active-class="transition duration-200"
    leave-active-class="transition duration-150"
  >
    <div
      v-if="toast.current"
      :key="toast.current.id"
      role="status"
      class="fixed inset-x-3 z-30 mx-auto flex max-w-md items-center gap-3 rounded-xl bg-neutral-900/95 px-4 py-3 text-sm text-white shadow-xl"
      :style="{ bottom: 'calc(var(--safe-bottom) + var(--tabbar-h) + var(--fab-stack) + 12px)' }"
    >
      <span class="flex-1">{{ toast.current.message }}</span>
      <button
        v-if="toast.current.action"
        type="button"
        class="pressable -my-1 rounded-lg px-2 py-1 font-semibold text-sky-300"
        @click="toast.runAction()"
      >
        {{ toast.current.action.label }}
      </button>
    </div>
  </Transition>
</template>
