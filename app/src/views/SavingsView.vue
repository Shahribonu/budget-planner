<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Currency, Transaction } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import SavingsChart from '@/components/SavingsChart.vue'
import SkeletonList from '@/components/SkeletonList.vue'
import TxRow from '@/components/TxRow.vue'
import { useFormat } from '@/composables/useFormat'
import { formatMonth } from '@/lib/dates'
import { savingsCurrencies, savingsSeries, savingsTotals } from '@/lib/savings'
import { txRoute } from '@/lib/transactions'
import { useAccountsStore } from '@/stores/accounts'
import { useLedgerStore } from '@/stores/ledger'
import { useSavingsStore } from '@/stores/savings'

const { t } = useI18n()
const router = useRouter()
const { money, lang } = useFormat()
const accounts = useAccountsStore()
const savings = useSavingsStore()
const ledger = useLedgerStore()

const loading = ref(!savings.summary)
const failed = ref(false)
const recent = ref<Transaction[]>([])

async function load() {
  failed.value = false
  try {
    await Promise.all([savings.load(), accounts.load()])
    void loadRecent()
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch(() => ledger.revision, load)

/** Recent transfers in and out of savings: one request per savings account (usually one or two). */
async function loadRecent() {
  const pages = await Promise.all(
    savings.savingsAccounts.slice(0, 3).map((a) => api.listTransactions({ account_id: a.id, limit: 10 }).catch(() => null)),
  )
  const seen = new Set<number>()
  recent.value = pages
    .flatMap((p) => p?.transactions ?? [])
    .filter((tx) => !seen.has(tx.id) && seen.add(tx.id))
    .sort((a, b) => (a.occurred_on < b.occurred_on ? 1 : a.occurred_on > b.occurred_on ? -1 : b.id - a.id))
    .slice(0, 8)
}

/** Month rows drop "so'm": the section heading and chart already say the currency. */
const rowMoney = (amount: number, currency: Currency) => money(amount, currency, { signed: true, bare: currency === 'UZS' })

const totals = computed(() => savingsTotals(accounts.active))
const current = computed(() => savings.summary?.current_month ?? '')
const charts = computed(() => {
  const s = savings.summary
  if (!s) return []
  return savingsCurrencies(accounts.active, s.months).map((currency) => {
    const series = savingsSeries(s.months, currency, s.current_month)
    const total = series.reduce((sum, p) => sum + p.net_minor, 0)
    return {
      currency,
      series,
      label: t('savings.chart_label', { total: money(total, currency, { signed: true }) }),
      // Text equivalent of the chart: months with movement, newest first.
      rows: [...series].reverse().filter((p) => p.put_in_minor || p.taken_out_minor),
    }
  })
})
</script>

<template>
  <PageHeader :title="$t('savings.title')" back />

  <SkeletonList v-if="loading" hero />
  <EmptyState v-else-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

  <div v-else class="space-y-6 px-4 pb-8">
    <!-- Total -->
    <section class="savings-hero relative overflow-hidden rounded-[26px] p-5 text-white">
      <span class="absolute -top-12 -right-10 size-40 rounded-full bg-white/15" aria-hidden="true" />
      <p class="relative flex items-center gap-1.5 text-xs font-semibold tracking-[0.12em] uppercase opacity-80">
        <AppIcon name="piggy" :size="16" />{{ $t('savings.total') }}
      </p>
      <p v-for="tot in totals" :key="tot.currency" class="display relative text-[clamp(26px,9vw,34px)] leading-tight">
        {{ money(tot.amount_minor, tot.currency) }}
      </p>
      <p v-if="!totals.length" class="display relative text-[clamp(26px,9vw,34px)] leading-tight">{{ money(0, 'UZS') }}</p>
      <!-- Wraps to two rows when both labels don't fit (320px, long Uzbek labels). -->
      <div class="relative mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          class="pressable flex grow items-center justify-center gap-1.5 rounded-2xl bg-white px-4 py-3 font-semibold whitespace-nowrap text-[#0b2b20]"
          @click="savings.start('in', router)"
        >
          <AppIcon name="plus" :size="18" />{{ $t('savings.put_aside') }}
        </button>
        <button
          type="button"
          class="pressable flex grow items-center justify-center gap-1.5 rounded-2xl bg-white/20 px-4 py-3 font-semibold whitespace-nowrap text-white disabled:opacity-50"
          :disabled="!savings.savingsAccounts.length"
          @click="savings.start('out', router)"
        >
          <AppIcon name="out" :size="18" />{{ $t('savings.take_out') }}
        </button>
      </div>
    </section>

    <!-- Accounts -->
    <section v-if="savings.savingsAccounts.length">
      <h2 class="eyebrow mb-2 px-1">{{ $t('savings.accounts') }}</h2>
      <div class="card divide-y divide-separator overflow-hidden">
        <RouterLink
          v-for="a in savings.savingsAccounts"
          :key="a.id"
          :to="{ name: 'account-edit', params: { id: a.id } }"
          class="flex items-center gap-3 px-4 py-3 active:bg-page/60"
        >
          <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-mint/15 text-mint" aria-hidden="true"><AppIcon name="piggy" :size="18" /></span>
          <span class="min-w-0 flex-1 truncate font-semibold">{{ a.name }}</span>
          <span class="tabular font-bold whitespace-nowrap">{{ money(a.balance_minor, a.currency) }}</span>
        </RouterLink>
      </div>
    </section>
    <EmptyState v-else icon="🐷" :body="$t('savings.empty_body')" />

    <!-- 12 months, per currency -->
    <section v-for="c in charts" :key="c.currency">
      <h2 class="eyebrow mb-2 px-1">{{ $t('savings.history') }}<template v-if="charts.length > 1"> · {{ c.currency }}</template></h2>
      <div class="card p-4">
        <SavingsChart :series="c.series" :currency="c.currency" :current="current" :label="c.label" />
        <p class="mt-1 flex justify-center gap-4 text-xs text-hint" aria-hidden="true">
          <span class="flex items-center gap-1"><span class="size-2.5 rounded-sm bg-mint" />{{ $t('savings.put_in') }}</span>
          <span class="flex items-center gap-1"><span class="size-2.5 rounded-sm bg-coral" />{{ $t('savings.taken_out') }}</span>
        </p>
        <ul v-if="c.rows.length" class="mt-3 divide-y divide-separator border-t border-separator">
          <li v-for="p in c.rows" :key="p.month" class="flex items-center gap-2 py-2.5 text-sm" :class="{ 'font-semibold': p.month === current }">
            <span class="min-w-0 flex-1 truncate first-letter:uppercase">{{ formatMonth(p.month, lang) }}</span>
            <span v-if="p.put_in_minor" class="tabular text-income">{{ rowMoney(p.put_in_minor, c.currency) }}</span>
            <template v-if="p.put_in_minor && p.taken_out_minor"><span class="text-hint" aria-hidden="true">/</span></template>
            <span v-if="p.taken_out_minor" class="tabular text-expense">{{ rowMoney(-p.taken_out_minor, c.currency) }}</span>
          </li>
        </ul>
        <p v-else class="mt-3 text-center text-sm text-hint">{{ $t('savings.no_history') }}</p>
      </div>
    </section>

    <!-- Recent movements -->
    <section v-if="recent.length">
      <h2 class="eyebrow mb-2 px-1">{{ $t('savings.recent') }}</h2>
      <div class="card overflow-hidden py-1">
        <TxRow v-for="tx in recent" :key="tx.id" :tx="tx" dated @open="router.push(txRoute(tx))" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.savings-hero {
  background: linear-gradient(140deg, #13b884 0%, #0b8a63 100%);
  box-shadow: 0 14px 30px -12px rgb(15 158 110 / 55%);
}
</style>
