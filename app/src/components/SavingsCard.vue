<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { Money } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import { useFormat } from '@/composables/useFormat'
import { netFor, plannedProgress, savingsTotals } from '@/lib/savings'
import { useAccountsStore } from '@/stores/accounts'
import { useSavingsStore } from '@/stores/savings'

/** Home's savings card. `planned`: this month's planned savings, when a plan exists. */
const props = defineProps<{ planned?: Money | null }>()

const router = useRouter()
const { money } = useFormat()
const accounts = useAccountsStore()
const savings = useSavingsStore()

const totals = computed(() => savingsTotals(accounts.active))
const month = computed(() => {
  const s = savings.summary
  // The month figure is in the plan's currency when there is one, else the first savings currency.
  const currency = props.planned?.currency ?? totals.value[0]?.currency
  if (!s || !currency) return null
  return { currency, net: netFor(s.months, s.current_month, currency) }
})
const progress = computed(() =>
  month.value && props.planned && props.planned.amount_minor > 0 ? plannedProgress(month.value.net, props.planned.amount_minor) : null,
)
</script>

<template>
  <!-- Empty: no savings account yet -->
  <section v-if="!savings.savingsAccounts.length" class="card flex items-center gap-3 px-4 py-3">
    <span class="flex size-11 shrink-0 items-center justify-center rounded-full bg-mint/15 text-mint" aria-hidden="true">
      <AppIcon name="piggy" :size="22" />
    </span>
    <span class="min-w-0 flex-1">
      <span class="block font-semibold">{{ $t('savings.start_title') }}</span>
      <span class="block text-xs text-hint">{{ $t('savings.start_body') }}</span>
    </span>
    <button type="button" class="pressable shrink-0 rounded-full bg-mint px-4 py-2 text-sm font-semibold text-[#0b2b20]" @click="savings.start('in', router)">
      {{ $t('savings.start_cta') }}
    </button>
  </section>

  <section v-else class="card relative">
    <RouterLink :to="{ name: 'savings' }" class="pressable block rounded-[inherit] px-4 pt-4 pb-[4.25rem]" :aria-label="$t('savings.open')">
      <span class="eyebrow flex items-center gap-1.5 text-mint"><AppIcon name="piggy" :size="14" />{{ $t('savings.title') }}</span>
      <span class="mt-1 flex items-start justify-between gap-2">
        <span>
          <span v-for="tot in totals" :key="tot.currency" class="display block text-[22px] leading-tight">{{ money(tot.amount_minor, tot.currency) }}</span>
        </span>
        <AppIcon name="chevron" :size="16" class="mt-1.5 shrink-0 text-hint" />
      </span>
      <span v-if="month" class="mt-2 flex flex-wrap gap-x-1 text-sm">
        <span class="text-hint">{{ $t('savings.this_month') }}</span>
        <span class="tabular font-semibold" :class="month.net < 0 ? 'text-expense' : 'text-income'">{{ money(month.net, month.currency, { signed: true }) }}</span>
        <span v-if="progress !== null && planned" class="text-hint">{{ $t('savings.of_planned', { amount: money(planned.amount_minor, planned.currency) }) }}</span>
      </span>
      <span
        v-if="progress !== null"
        class="mt-2 block h-1.5 overflow-hidden rounded-full bg-mint/15"
        role="progressbar"
        :aria-valuenow="Math.round(progress * 100)"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-label="$t('savings.progress')"
      >
        <span class="block h-full rounded-full bg-mint transition-[width] duration-500" :style="{ width: `${progress * 100}%` }" />
      </span>
    </RouterLink>
    <button
      type="button"
      class="pressable absolute inset-x-4 bottom-4 flex items-center justify-center gap-1.5 rounded-2xl bg-mint py-2.5 font-semibold text-[#0b2b20]"
      @click="savings.start('in', router)"
    >
      <AppIcon name="plus" :size="18" />{{ $t('savings.put_aside') }}
    </button>
  </section>
</template>
