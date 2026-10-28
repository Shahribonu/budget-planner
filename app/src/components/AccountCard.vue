<script setup lang="ts">
import { computed } from 'vue'
import type { Account } from '@/api/types'
import { useFormat } from '@/composables/useFormat'
import { accountColor, darken, isCard } from '@/lib/accountStyle'
import { ACCOUNT_ICONS } from '@/lib/accounts'

/** An account as a bank card. `big`: the Home carousel size (~60% of the screen). */
const props = defineProps<{ account: Account; big?: boolean }>()
const { money } = useFormat()

const color = computed(() => accountColor(props.account))
const bg = computed(() => `linear-gradient(135deg, ${color.value} 0%, ${darken(color.value, 0.35)} 100%)`)
</script>

<template>
  <RouterLink
    :to="{ name: 'account-edit', params: { id: account.id } }"
    class="pressable relative flex shrink-0 snap-start flex-col justify-between overflow-hidden p-4 text-white"
    :class="big ? 'h-40 w-[62%] min-w-[13.5rem] rounded-[26px]' : 'h-32 w-52 rounded-3xl'"
    :style="{ background: bg, boxShadow: `0 14px 28px -10px ${color}aa, 0 2px 6px rgb(0 0 0 / 12%)` }"
  >
    <!-- card pattern: a soft wave and a chip -->
    <svg class="pointer-events-none absolute inset-0 size-full" viewBox="0 0 200 130" preserveAspectRatio="none" aria-hidden="true">
      <path d="M0 95 C 50 70, 90 125, 140 95 S 200 70, 200 70 V130 H0Z" fill="#fff" opacity=".08" />
      <path d="M0 110 C 60 90, 100 135, 150 110 S 200 95, 200 95 V130 H0Z" fill="#fff" opacity=".06" />
      <circle cx="175" cy="18" r="42" fill="#fff" opacity=".08" />
    </svg>
    <span class="relative flex items-center justify-between gap-2">
      <span class="truncate text-sm font-semibold">{{ account.name }}</span>
      <span class="flex size-8 items-center justify-center rounded-full bg-white/20 text-base" aria-hidden="true">{{ ACCOUNT_ICONS[account.type] }}</span>
    </span>
    <span v-if="isCard(account.type)" class="chip relative" aria-hidden="true" />
    <span class="relative">
      <span v-if="account.card_mask" class="block text-xs tracking-[0.2em] text-white/80">•••• {{ account.card_mask }}</span>
      <span class="tabular block truncate font-extrabold tracking-normal" :class="big ? 'text-[22px]' : 'text-xl'">{{ money(account.balance_minor, account.currency) }}</span>
    </span>
  </RouterLink>
</template>

<style scoped>
.chip {
  width: 30px;
  height: 22px;
  border-radius: 6px;
  background: linear-gradient(135deg, #ffe6a3 0%, #e8b64a 100%);
  box-shadow: inset 0 0 0 1px rgb(0 0 0 / 12%);
  opacity: 0.9;
}
</style>
