<script setup lang="ts">
import EmptyArt from '@/components/EmptyArt.vue'
import { closeApp } from '@/lib/telegram'
</script>

<template>
  <div class="flex min-h-[80dvh] flex-col items-center justify-center px-8 text-center">
    <EmptyArt kind="wallet" class="mb-6" />
    <h1 class="display text-2xl">{{ $t('privacy.deleted_title') }}</h1>
    <p class="mt-2 text-hint">{{ $t('privacy.deleted_body') }}</p>
    <button
      type="button"
      class="pressable mt-8 rounded-2xl bg-accent px-8 py-3.5 font-semibold text-on-accent shadow-lg shadow-accent/25"
      @click="closeApp"
    >
      {{ $t('common.close') }}
    </button>
  </div>
</template>
