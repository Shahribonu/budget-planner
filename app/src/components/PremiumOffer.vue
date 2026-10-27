<script setup lang="ts">
import { onMounted } from 'vue'
import type { PremiumPlan } from '@/api/types'
import { type BuyResult, usePremium } from '@/composables/usePremium'
import { formatStars, monthsFree } from '@/lib/premium'
import AppIcon from './AppIcon.vue'

/** What Premium gives today and the two Stars plans. No timers, nothing pre-selected. */
defineProps<{ compact?: boolean }>()
const emit = defineEmits<{ paid: [result: Exclude<BuyResult, null>] }>()

const { buy, buying, canPay, prices, pricesState, loadPrices } = usePremium()
onMounted(loadPrices)

async function choose(plan: PremiumPlan) {
  const result = await buy(plan)
  if (result) emit('paid', result)
}
</script>

<template>
  <div class="space-y-3">
    <ul v-if="!compact" class="space-y-1.5 text-sm">
      <li class="flex items-start gap-2">
        <AppIcon name="check" :size="16" class="mt-0.5 shrink-0 text-accent" />
        {{ $t('premium.benefit_budgets', { n: prices?.free_budgets ?? 1 }) }}
      </li>
      <li class="flex items-start gap-2"><AppIcon name="check" :size="16" class="mt-0.5 shrink-0 text-accent" />{{ $t('premium.benefit_support') }}</li>
    </ul>

    <p v-if="!canPay" class="rounded-xl bg-page px-3 py-2 text-sm text-hint">{{ $t('premium.open_in_telegram') }}</p>
    <p v-else-if="pricesState === 'unavailable'" class="rounded-xl bg-page px-3 py-2 text-sm text-hint">{{ $t('premium.unavailable') }}</p>
    <div v-else-if="prices" class="grid grid-cols-2 gap-2">
      <button
        type="button"
        class="pressable flex flex-col items-center rounded-xl border-2 border-accent/40 bg-page px-2 py-2.5 disabled:opacity-60"
        :disabled="!!buying"
        @click="choose('monthly')"
      >
        <span class="font-semibold">{{ buying === 'monthly' ? $t('common.loading') : $t('premium.monthly', { stars: formatStars(prices.monthly_stars) }) }}</span>
        <span class="text-xs text-hint">{{ $t('premium.monthly_note') }}</span>
      </button>
      <button
        type="button"
        class="pressable relative flex flex-col items-center rounded-xl border-2 border-accent/40 bg-page px-2 py-2.5 disabled:opacity-60"
        :disabled="!!buying"
        @click="choose('yearly')"
      >
        <span
          v-if="monthsFree(prices.monthly_stars, prices.yearly_stars) > 0"
          class="absolute -top-2.5 rounded-full bg-accent px-2 py-0.5 text-[10px] font-semibold text-on-accent"
        >
          {{ $t('premium.yearly_badge', { n: monthsFree(prices.monthly_stars, prices.yearly_stars) }) }}
        </span>
        <span class="font-semibold">{{ buying === 'yearly' ? $t('common.loading') : $t('premium.yearly', { stars: formatStars(prices.yearly_stars) }) }}</span>
        <span class="text-xs text-hint">{{ $t('premium.yearly_note') }}</span>
      </button>
    </div>
    <p v-else class="py-2 text-center text-sm text-hint">{{ $t('common.loading') }}</p>

    <p v-if="!compact" class="text-xs text-hint">{{ $t('premium.free_note') }}</p>
  </div>
</template>
