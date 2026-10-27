<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter, type LocationQueryRaw } from 'vue-router'
import type { Currency, Money, Transaction } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import BudgetBar from '@/components/BudgetBar.vue'
import EmptyState from '@/components/EmptyState.vue'
import TxRow from '@/components/TxRow.vue'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { formatMonth, monthRange } from '@/lib/dates'
import { txRoute } from '@/lib/transactions'
import { useAccountsStore } from '@/stores/accounts'
import { useBudgetsStore } from '@/stores/budgets'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'
import { useLedgerStore } from '@/stores/ledger'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const user = useUserStore()
const ledger = useLedgerStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const budgets = useBudgetsStore()
const plans = useInstallmentsStore()
const { money, lang } = useFormat()

const failed = ref(false)

async function refresh() {
  try {
    await Promise.all([ledger.loadSummary(), categories.load(), accounts.load()])
    // Only used to word debt rows in Recent; Home works without them.
    void plans.load().catch(() => {})
    void budgets.load(true).catch(() => {})
    failed.value = false
  } catch {
    failed.value = true
  }
}
onMounted(refresh)
watch(() => ledger.revision, refresh)

const summary = computed(() => ledger.summary)
const range = computed(() => (summary.value ? monthRange(summary.value.month) : null))
const isEmpty = computed(() => !!summary.value && summary.value.recent.length === 0)

const debtTiles = computed(() => {
  const s = summary.value
  if (!s) return []
  return [
    { key: 'owed_to_me', list: s.owed_to_me ?? [], to: { name: 'debts' }, icon: 'in' as const },
    { key: 'i_owe', list: s.i_owe ?? [], to: { name: 'debts' }, icon: 'out' as const },
    { key: 'installments', list: s.installments ?? [], to: { name: 'installments' }, icon: 'calendar' as const },
  ].filter((tile) => tile.list.some((m) => m.amount_minor !== 0))
})

const balances = computed<Money[]>(() => {
  const list = summary.value?.balances ?? []
  return list.length ? list : [{ amount_minor: 0, currency: user.mainCurrency }]
})

/** Union of currencies seen this month so income and expense line up per currency. */
const monthCurrencies = computed<Currency[]>(() => {
  const s = summary.value
  if (!s) return []
  const set = new Set([...s.income, ...s.expense].map((m) => m.currency))
  return set.size ? [...set] : [user.mainCurrency]
})

function amountFor(list: Money[] | undefined, currency: Currency): number {
  return list?.find((m) => m.currency === currency)?.amount_minor ?? 0
}

const breakdown = computed(() => {
  const s = summary.value
  if (!s) return []
  const groups = new Map<Currency, typeof s.by_category>()
  for (const row of s.by_category) {
    const g = groups.get(row.currency) ?? []
    g.push(row)
    groups.set(row.currency, g)
  }
  return [...groups].map(([currency, rows]) => {
    const total = rows.reduce((sum, r) => sum + r.amount_minor, 0)
    const max = Math.max(...rows.map((r) => r.amount_minor), 1)
    return {
      currency,
      rows: rows.map((r) => ({
        ...r,
        category: categories.byId.get(r.category_id),
        width: Math.max(2, Math.round((r.amount_minor / max) * 100)),
        share: total ? Math.round((r.amount_minor / total) * 100) : 0,
      })),
    }
  })
})

function drill(query: LocationQueryRaw, withMonth = true) {
  const q = { ...query }
  if (withMonth && range.value) Object.assign(q, range.value)
  void router.push({ name: 'activity', query: q })
}

function openTx(tx: Transaction) {
  void router.push(txRoute(tx))
}
</script>

<template>
  <div class="px-4 pt-4">
    <EmptyState v-if="failed && !summary" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="refresh" />

    <template v-else-if="summary">
      <!-- Balance per currency -->
      <section class="mb-3 rounded-2xl bg-surface p-4" :aria-label="$t('home.balance')">
        <h2 class="mb-1 text-sm text-hint">{{ $t('home.balance') }}</h2>
        <button
          v-for="b in balances"
          :key="b.currency"
          type="button"
          class="pressable tabular block w-full truncate text-left text-3xl leading-tight font-bold"
          @click="drill({ currency: b.currency }, false)"
        >
          {{ money(b.amount_minor, b.currency) }}
        </button>
      </section>

      <!-- Debts: only when something is open -->
      <section v-if="debtTiles.length" class="mb-3 divide-y divide-separator rounded-2xl bg-surface">
        <RouterLink
          v-for="tile in debtTiles"
          :key="tile.key"
          :to="tile.to"
          class="flex items-start justify-between gap-3 px-4 py-3 active:bg-page/60"
        >
          <span class="flex shrink-0 items-center gap-1 pt-0.5 text-sm text-hint">
            <AppIcon :name="tile.icon" :size="14" />{{ $t(`home.${tile.key}`) }}
          </span>
          <span class="flex min-w-0 items-center gap-1">
            <span class="min-w-0 text-right">
              <span
                v-for="m in tile.list.filter((x) => x.amount_minor !== 0)"
                :key="m.currency"
                class="tabular block truncate font-semibold"
              >
                {{ money(m.amount_minor, m.currency) }}
              </span>
            </span>
            <AppIcon name="chevron" :size="16" class="shrink-0 text-hint" />
          </span>
        </RouterLink>
      </section>

      <!-- This month: income / expense per currency -->
      <section class="mb-3 rounded-2xl bg-surface p-4">
        <h2 class="mb-3 text-sm text-hint first-letter:uppercase">
          {{ $t('home.this_month') }} · {{ formatMonth(summary.month, lang) }}
        </h2>
        <div class="space-y-3">
          <div class="flex items-start justify-between gap-3">
            <div class="flex items-center gap-1 pt-0.5 text-sm text-hint">
              <AppIcon name="in" :size="16" class="text-income" />{{ $t('home.income') }}
            </div>
            <div class="min-w-0 text-right">
              <button
                v-for="c in monthCurrencies"
                :key="c"
                type="button"
                class="pressable tabular block w-full truncate text-right text-lg font-semibold text-income"
                @click="drill({ type: 'income', currency: c })"
              >
                {{ money(amountFor(summary.income, c), c, { signed: true }) }}
              </button>
            </div>
          </div>
          <div class="flex items-start justify-between gap-3">
            <div class="flex items-center gap-1 pt-0.5 text-sm text-hint">
              <AppIcon name="out" :size="16" />{{ $t('home.expense') }}
            </div>
            <div class="min-w-0 text-right">
              <button
                v-for="c in monthCurrencies"
                :key="c"
                type="button"
                class="pressable tabular block w-full truncate text-right text-lg font-semibold"
                @click="drill({ type: 'expense', currency: c })"
              >
                {{ money(-amountFor(summary.expense, c), c) }}
              </button>
            </div>
          </div>
          <!-- Overall budget, compact; full card is on Plan -->
          <RouterLink
            v-for="b in budgets.overall"
            :key="b.id"
            :to="{ name: 'plan' }"
            class="block"
            :aria-label="`${$t('budgets.overall')} ${b.currency}`"
          >
            <span class="mb-1 flex justify-between gap-2 text-xs text-hint">
              <span class="truncate">{{ $t('budgets.overall') }}</span>
              <span class="tabular shrink-0">
                {{ $t('budgets.spent_of', { spent: money(b.spent_minor, b.currency, { bare: true }), limit: money(b.amount_minor, b.currency) }) }}
              </span>
            </span>
            <BudgetBar :spent="b.spent_minor" :limit="b.amount_minor" />
          </RouterLink>
        </div>
      </section>

      <EmptyState
        v-if="isEmpty"
        icon="🧾"
        :title="$t('home.empty_title')"
        :body="$t('home.empty_body')"
        :action="$t('home.empty_cta')"
        @action="router.push({ name: 'tx-new', query: { type: 'expense' } })"
      />

      <template v-else>
        <!-- Category breakdown -->
        <section class="mb-3 rounded-2xl bg-surface py-3">
          <h2 class="mb-1 px-4 font-semibold">{{ $t('home.by_category') }}</h2>
          <p v-if="!breakdown.length" class="px-4 py-2 text-sm text-hint">{{ $t('home.no_expenses') }}</p>
          <div v-for="group in breakdown" :key="group.currency">
            <button
              v-for="row in group.rows"
              :key="`${row.category_id}-${row.currency}`"
              type="button"
              class="block w-full px-4 py-2 text-left active:bg-page/60"
              @click="drill({ category_id: row.category_id, currency: row.currency, type: 'expense' })"
            >
              <span class="flex items-center gap-2">
                <span aria-hidden="true">{{ row.category?.icon }}</span>
                <span class="min-w-0 flex-1 truncate text-sm">{{ categoryLabel(row.category) }}</span>
                <span class="tabular text-xs text-hint">{{ row.share }}%</span>
                <span class="tabular text-sm font-semibold whitespace-nowrap">{{ money(row.amount_minor, row.currency) }}</span>
              </span>
              <span class="mt-1.5 block h-2 overflow-hidden rounded-full bg-page">
                <span class="block h-full rounded-full bg-accent" :style="{ width: `${row.width}%` }" />
              </span>
            </button>
          </div>
        </section>

        <!-- Recent activity -->
        <section class="rounded-2xl bg-surface py-2">
          <div class="flex items-center justify-between px-4 py-1">
            <h2 class="font-semibold">{{ $t('home.recent') }}</h2>
            <RouterLink :to="{ name: 'activity' }" class="flex items-center text-sm text-link">
              {{ $t('common.see_all') }}<AppIcon name="chevron" :size="16" />
            </RouterLink>
          </div>
          <TxRow v-for="tx in summary.recent" :key="tx.id" :tx="tx" @open="openTx" />
        </section>
      </template>
    </template>

    <div v-else class="py-20 text-center text-hint">{{ $t('common.loading') }}</div>
  </div>
</template>
