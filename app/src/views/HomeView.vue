<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter, type LocationQueryRaw } from 'vue-router'
import type { Currency, Money } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import TxRow from '@/components/TxRow.vue'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { formatMonth, monthRange } from '@/lib/dates'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useLedgerStore } from '@/stores/ledger'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const user = useUserStore()
const ledger = useLedgerStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const { money, lang } = useFormat()

const failed = ref(false)

async function refresh() {
  try {
    await Promise.all([ledger.loadSummary(), categories.load(), accounts.load()])
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

function openTx(id: number) {
  void router.push({ name: 'tx-edit', params: { id } })
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
