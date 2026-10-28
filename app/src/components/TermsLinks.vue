no, <script setup lang="ts">
import { Translation } from 'vue-i18n'

/** "I accept the Terms and Privacy Policy" with both names linked to the legal pages. */
defineProps<{ keypath?: string }>()
</script>

<template>
  <Translation :keypath="keypath ?? 'legal.accept'" tag="span" scope="global">
    <template #terms>
      <RouterLink :to="{ name: 'legal', params: { doc: 'terms' } }" class="font-semibold text-link underline underline-offset-2" @click.stop>
        {{ $t('legal.terms') }}
      </RouterLink>
    </template>
    <template #privacy>
      <RouterLink :to="{ name: 'legal', params: { doc: 'privacy' } }" class="font-semibold text-link underline underline-offset-2" @click.stop>
        {{ $t('legal.privacy') }}
      </RouterLink>
    </template>
  </Translation>
</template>
