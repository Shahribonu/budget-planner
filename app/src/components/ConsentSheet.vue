<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { haptic } from '@/lib/telegram'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import BottomSheet from './BottomSheet.vue'
import TermsLinks from './TermsLinks.vue'

/** One-time, blocking: onboarded users who haven't accepted the Terms and Privacy Policy yet. */
const user = useUserStore()
const route = useRoute()
const toast = useToastStore()
const busy = ref(false)

// Hidden while reading a legal page (opened from this sheet), shown again on return.
const open = computed({
  get: () => !!user.me?.onboarded && user.me.terms_accepted === false && !route.meta.public && route.name !== 'onboarding',
  set: () => {},
})

async function accept() {
  if (busy.value) return
  busy.value = true
  try {
    await user.update({ accept_terms: true })
    haptic.success()
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <BottomSheet v-model:open="open" :title="$t('legal.consent_title')" persistent>
    <div class="space-y-4 px-4 pt-1 pb-5">
      <p class="text-sm text-hint">{{ $t('legal.consent_body') }}</p>
      <p class="text-[15px]"><TermsLinks keypath="legal.consent_links" /></p>
      <button
        type="button"
        class="pressable w-full rounded-2xl bg-accent py-3.5 font-semibold text-on-accent shadow-lg shadow-accent/25 disabled:opacity-60"
        :disabled="busy"
        data-autofocus
        @click="accept"
      >
        {{ busy ? $t('common.loading') : $t('legal.accept_button') }}
      </button>
    </div>
  </BottomSheet>
</template>
