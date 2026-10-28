<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, type LocationQueryRaw } from 'vue-router'
import { api } from '@/api/client'
import type { Currency, Money, Transaction } from '@/api/types'
import AccountCard from '@/components/AccountCard.vue'
import AppIcon from '@/components/AppIcon.vue'
import BudgetBar from '@/components/BudgetBar.vue'
import DonutChart, { type DonutSlice } from '@/components/DonutChart.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import PullIndicator from '@/components/PullIndicator.vue'
import EmptyState from '@/components/EmptyState.vue'
import TxRow from '@/components/TxRow.vue'
import { useCountUp } from '@/composables/useCountUp'
import { useFormat } from '@/composables/useFormat'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { categoryLabel } from '@/i18n'
import { formatMonth, monthRange } from '@/lib/dates'
import { categoryStyle } from '@/lib/categoryStyle'
import { sumSpent, weekStart } from '@/lib/insights'
import { prefs } from '@/lib/prefs'
import { haptic, setHeaderTint } from '@/lib/telegram'
import { txRoute } from '@/lib/transactions'
import { useAccountsStore } from '@/stores/accounts'
import { useBudgetsStore } from '@/stores/budgets'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'
import { useLedgerStore } from '@/stores/ledger'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const { t: t_ } = useI18n()
const user = useUserStore()
const ledger = useLedgerStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const budgets = useBudgetsStore()
const plans = useInstallmentsStore()
const { money, lang, today, dayLabel } = useFormat()

const failed = ref(false)

async function refresh() {
  try {
    await Promise.all([ledger.loadSummary(), categories.load(), accounts.load()])
    // Only used to word debt rows in Recent; Home works without them.
    void plans.load().catch(() => {})
    void budgets.load(true).catch(() => {})
    void loadPlanRow(ledger.summary?.month)
    void loadWeek()
    failed.value = false
  } catch {
    failed.value = true
  }
}
onMounted(refresh)
watch(() => ledger.revision, refresh)

const summary = computed(() => ledger.summary)

// "Plan: N fit · M later" for this month, only when the plan has items.
const planRow = ref<{ fit: number; later: number } | null>(null)
async function loadPlanRow(month?: string) {
  try {
    // create=false: a preview must not create (and carry over into) this month's plan; 404 hides the row.
    const v = await api.getPlan(month, undefined, false)
    const planned = v.items.filter((i) => i.status === 'planned')
    const fit = planned.filter((i) => i.decision?.group === 'fits').length
    planRow.value = planned.length ? { fit, later: planned.length - fit } : null
  } catch {
    planRow.value = null
  }
}
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

const { pull, refreshing, threshold } = usePullToRefresh(refresh)

// Hero: the main-currency balance, big; other currencies small underneath.
const heroBalance = computed<Money>(
  () => balances.value.find((b) => b.currency === user.mainCurrency) ?? balances.value[0]!,
)
const otherBalances = computed(() => balances.value.filter((b) => b !== heroBalance.value))
const shownBalance = useCountUp(computed(() => heroBalance.value.amount_minor))
// Big but always on one line at 360px: step the size down for long amounts.
const heroSize = computed(() => {
  const len = money(heroBalance.value.amount_minor, heroBalance.value.currency).length
  return len > 16 ? 'text-[28px]' : len > 13 ? 'text-[34px]' : 'text-[40px]'
})

// The hero runs into Telegram's header: tint it violet while Home is open.
const HERO_TOP = '#5b3df5'
onMounted(() => setHeaderTint(HERO_TOP))
onBeforeUnmount(() => setHeaderTint(null))


function toggleHidden() {
  haptic.selection()
  prefs.hideAmounts = !prefs.hideAmounts
}

const heroIncome = computed(() => amountFor(summary.value?.income, heroBalance.value.currency))
const heroExpense = computed(() => amountFor(summary.value?.expense, heroBalance.value.currency))
const overallBudget = computed(() => budgets.overall.find((b) => b.currency === heroBalance.value.currency))
const budgetSheet = ref(false)
// Only loaded when someone taps "Set a monthly limit": keeps it out of the first-load bundle.
const BudgetSheet = defineAsyncComponent(() => import('@/components/BudgetSheet.vue'))

const activeAccounts = computed(() => accounts.active)

// Insight strip: today / this week from the last 7 days of spending, top category from the summary.
const weekTxs = ref<Transaction[]>([])
async function loadWeek() {
  try {
    const t = today()
    const page = await api.listTransactions({ type: ['expense', 'installment'], from: weekStart(t), to: t, limit: 200 })
    weekTxs.value = page.transactions
  } catch {
    weekTxs.value = []
  }
}
const mainBreakdown = computed(() => breakdown.value.find((g) => g.currency === heroBalance.value.currency))
const insights = computed(() => {
  const cur = heroBalance.value.currency
  const t = today()
  const ws = weekStart(t)
  const top = mainBreakdown.value?.rows[0]
  return [
    { key: 'today', label: t_('home.chip_today'), value: money(-sumSpent(weekTxs.value, cur, t, t), cur), color: '#ff6b5a', query: { type: 'expense,installment', currency: cur, from: t, to: t } },
    { key: 'week', label: t_('home.chip_week'), value: money(-sumSpent(weekTxs.value, cur, ws, t), cur), color: '#f5a524', query: { type: 'expense,installment', currency: cur, from: ws, to: t } },
    ...(top
      ? [{ key: 'top', label: t_('home.chip_top'), value: `${categoryLabel(top.category)} ${top.share}%`, color: categoryStyle(top.category).color, query: { category_id: top.category_id, currency: cur, ...(range.value ?? {}) } }]
      : []),
  ]
})

// Donut: top 5 categories plus "other", in the hero currency.
const donut = computed<DonutSlice[]>(() => {
  const rows = mainBreakdown.value?.rows ?? []
  const top = rows.slice(0, 5).map((r) => ({ key: r.category_id, value: r.amount_minor, color: categoryStyle(r.category).color, label: categoryLabel(r.category) }))
  const rest = rows.slice(5).reduce((s, r) => s + r.amount_minor, 0)
  return rest > 0 ? [...top, { key: 'other', value: rest, color: '#8b8aa0', label: t_('home.other') }] : top
})
const donutTotal = computed(() => (mainBreakdown.value?.rows ?? []).reduce((s, r) => s + r.amount_minor, 0))

// Recent, grouped as Today / Yesterday / date.
const recentGroups = computed(() => {
  const out: { day: string; items: Transaction[] }[] = []
  for (const tx of (summary.value?.recent ?? []).slice(0, 5)) {
    const last = out[out.length - 1]
    if (last?.day === tx.occurred_on) last.items.push(tx)
    else out.push({ day: tx.occurred_on, items: [tx] })
  }
  return out
})

</script>

<template>
  <PullIndicator :pull="pull" :refreshing="refreshing" :threshold="threshold" />

  <!-- Skeleton -->
  <div v-if="!summary && !failed" aria-busy="true">
    <div class="hero-bg h-[25rem]" />
    <div class="-mt-12 flex gap-3 overflow-hidden px-5">
      <div class="skeleton h-40 w-[62%] shrink-0 rounded-[26px]" />
      <div class="skeleton h-40 w-[62%] shrink-0 rounded-[26px]" />
    </div>
  </div>

  <EmptyState v-else-if="failed && !summary" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" class="pt-20" @action="refresh" />

  <template v-else-if="summary">
    <!-- Full-bleed hero: runs under Telegram's (tinted) header -->
    <section class="hero-bg relative overflow-hidden px-5 pt-3 pb-20 text-white" :aria-label="$t('home.balance')">
      <span class="blob blob-1" aria-hidden="true" />
      <span class="blob blob-2" aria-hidden="true" />
      <span class="blob blob-3" aria-hidden="true" />

      <RouterLink :to="{ name: 'profile' }" class="pressable relative flex w-fit max-w-full items-center gap-3 rounded-full pr-2">
        <UserAvatar :size="42" />
        <span class="truncate font-semibold">{{ $t('home.hello', { name: user.me?.first_name ?? '' }) }}</span>
      </RouterLink>

      <div class="rise relative mt-6 text-center" style="--i: 0">
        <p class="text-[11px] font-semibold tracking-[0.22em] text-white/70 uppercase">{{ $t('home.balance') }}</p>
        <button type="button" class="pressable mx-auto block max-w-full" @click="drill({ currency: heroBalance.currency }, false)">
          <span class="display block leading-tight whitespace-nowrap" :class="heroSize">{{ money(shownBalance, heroBalance.currency) }}</span>
        </button>
        <p v-for="b in otherBalances" :key="b.currency" class="display text-lg text-white/80">{{ money(b.amount_minor, b.currency) }}</p>
        <button
          type="button"
          class="pressable mx-auto mt-3 flex size-10 items-center justify-center rounded-full bg-white text-[#5b3df5] shadow-lg"
          :aria-label="prefs.hideAmounts ? $t('home.show_amounts') : $t('home.hide_amounts')"
          :aria-pressed="prefs.hideAmounts"
          @click="toggleHidden"
        >
          <AppIcon :name="prefs.hideAmounts ? 'eye-off' : 'eye'" :size="20" />
        </button>
      </div>

      <!-- Monthly limit -->
      <div class="rise relative mt-4 flex justify-center" style="--i: 1">
        <RouterLink
          v-if="overallBudget"
          :to="{ name: 'plan' }"
          class="pressable w-full max-w-[18rem] rounded-full bg-white px-4 py-2 text-[#17142b] shadow-lg"
        >
          <span class="mb-1 flex justify-between gap-2 text-xs font-semibold">
            <span>{{ $t('home.limit') }}</span>
            <span class="tabular text-[#8a87a0]">{{ money(overallBudget.spent_minor, overallBudget.currency, { bare: true }) }} / {{ money(overallBudget.amount_minor, overallBudget.currency) }}</span>
          </span>
          <BudgetBar :spent="overallBudget.spent_minor" :limit="overallBudget.amount_minor" />
        </RouterLink>
        <button
          v-else
          type="button"
          class="pressable flex items-center gap-1.5 rounded-full bg-white px-4 py-2 text-sm font-semibold text-[#5b3df5] shadow-lg"
          @click="budgetSheet = true"
        >
          <AppIcon name="target" :size="16" />{{ $t('home.set_limit') }}<AppIcon name="down" :size="16" />
        </button>
      </div>

      <div class="rise relative mt-4 grid grid-cols-2 gap-3" style="--i: 2">
        <!-- White on violet in both themes, so the text colour is fixed dark -->
        <button type="button" class="pressable rounded-[20px] bg-white p-3 text-left text-[#17142b] shadow-lg" @click="drill({ type: 'income', currency: heroBalance.currency })">
          <span class="flex items-center gap-2 text-xs text-[#8a87a0]">
            <span class="flex size-7 items-center justify-center rounded-full bg-[#2bd9a0]/20 text-[#0f9e6e]"><AppIcon name="in" :size="15" /></span>
            {{ $t('home.income') }}
          </span>
          <span class="display mt-1.5 block truncate text-[17px]">{{ money(heroIncome, heroBalance.currency, { signed: true, bare: true }) }}</span>
        </button>
        <button type="button" class="pressable rounded-[20px] bg-white p-3 text-left text-[#17142b] shadow-lg" @click="drill({ type: 'expense,installment', currency: heroBalance.currency })">
          <span class="flex items-center gap-2 text-xs text-[#8a87a0]">
            <span class="flex size-7 items-center justify-center rounded-full bg-[#ff6b5a]/20 text-[#e2483a]"><AppIcon name="out" :size="15" /></span>
            {{ $t('home.expense') }}
          </span>
          <span class="display mt-1.5 block truncate text-[17px]">{{ money(-heroExpense, heroBalance.currency, { bare: true }) }}</span>
        </button>
      </div>
    </section>

    <!-- Big account cards, overlapping the hero -->
    <div
      class="relative -mt-12 flex snap-x snap-mandatory gap-3 overflow-x-auto scroll-px-5 px-5 pb-3 [scrollbar-width:none]"
      role="list"
      :aria-label="$t('home.accounts')"
    >
      <AccountCard
        v-for="(a, i) in activeAccounts"
        :key="a.id"
        :account="a"
        big
        role="listitem"
        class="rise"
        :style="{ '--i': i + 3 }"
      />
      <RouterLink
        :to="{ name: 'account-new' }"
        class="pressable rise flex h-40 w-28 shrink-0 snap-start flex-col items-center justify-center gap-1 rounded-[26px] border-2 border-dashed border-accent/40 bg-surface/90 text-accent backdrop-blur"
        role="listitem"
        :style="{ '--i': activeAccounts.length + 3 }"
      >
        <AppIcon name="plus" :size="28" />
        <span class="text-xs font-semibold">{{ $t('home.add_account') }}</span>
      </RouterLink>
    </div>

    <div class="space-y-6 px-4 pt-2">
      <!-- Insight strip -->
      <div v-if="!isEmpty" class="-mx-4 flex gap-2 overflow-x-auto px-4 [scrollbar-width:none]">
        <button
          v-for="(c, i) in insights"
          :key="c.key"
          type="button"
          class="pressable rise flex shrink-0 items-center gap-2 rounded-full bg-surface py-2 pr-4 pl-2"
          :style="{ '--i': i + 5, boxShadow: 'var(--shadow-card)' }"
          @click="drill(c.query, false)"
        >
          <span class="size-6 rounded-full" :style="{ background: c.color }" aria-hidden="true" />
          <span class="text-xs text-hint">{{ c.label }}</span>
          <span class="text-sm font-bold whitespace-nowrap">{{ c.value }}</span>
        </button>
      </div>

      <div v-if="debtTiles.length" class="flex gap-2 overflow-x-auto [scrollbar-width:none]">
        <RouterLink v-for="tile in debtTiles" :key="tile.key" :to="tile.to" class="card pressable min-w-[10.5rem] flex-1 px-4 py-3">
          <span class="eyebrow flex items-center gap-1"><AppIcon :name="tile.icon" :size="12" />{{ $t(`home.${tile.key}`) }}</span>
          <span v-for="m in tile.list.filter((x) => x.amount_minor !== 0)" :key="m.currency" class="display mt-1 block text-base">
            {{ money(m.amount_minor, m.currency) }}
          </span>
        </RouterLink>
      </div>

      <RouterLink v-if="planRow" :to="{ name: 'plan' }" class="card pressable flex items-center gap-3 px-4 py-3">
        <span class="flex size-10 items-center justify-center rounded-full bg-accent-soft text-accent"><AppIcon name="calendar" :size="20" /></span>
        <span class="min-w-0 flex-1 truncate font-semibold">{{ $t('mplan.home_row', planRow) }}</span>
        <AppIcon name="chevron" :size="16" class="shrink-0 text-hint" />
      </RouterLink>

      <div v-if="isEmpty" class="card">
        <EmptyState
          art="wallet"
          :title="$t('home.empty_title')"
          :body="$t('home.empty_body')"
          :action="$t('home.empty_cta')"
          @action="router.push({ name: 'tx-new', query: { type: 'expense' } })"
        />
      </div>

      <template v-else>
        <!-- Spending by category: donut -->
        <section v-if="donut.length">
          <h2 class="eyebrow mb-2 px-1">{{ $t('home.by_category') }} · {{ formatMonth(summary.month, lang) }}</h2>
          <div class="card flex items-center gap-4 p-4">
            <DonutChart :slices="donut" :size="132" :label="$t('home.by_category')">
              <span class="text-[10px] text-hint uppercase">{{ $t('home.expense') }}</span>
              <span class="display text-sm leading-tight">{{ money(donutTotal, heroBalance.currency, { bare: true }) }}</span>
            </DonutChart>
            <ul class="min-w-0 flex-1 space-y-1.5">
              <li v-for="s in donut" :key="s.key">
                <button
                  type="button"
                  class="flex w-full items-center gap-2 text-left text-sm"
                  :disabled="s.key === 'other'"
                  @click="drill({ category_id: s.key, currency: heroBalance.currency, type: 'expense,installment' })"
                >
                  <span class="size-2.5 shrink-0 rounded-full" :style="{ background: s.color }" />
                  <span class="min-w-0 flex-1 truncate">{{ s.label }}</span>
                  <span class="tabular text-xs font-semibold text-hint">{{ donutTotal ? Math.round((s.value / donutTotal) * 100) : 0 }}%</span>
                </button>
              </li>
            </ul>
          </div>
        </section>

        <!-- Recent -->
        <section>
          <div class="mb-2 flex items-center justify-between px-1">
            <h2 class="eyebrow">{{ $t('home.recent_long') }}</h2>
            <RouterLink :to="{ name: 'activity' }" class="pressable flex items-center gap-0.5 rounded-full bg-accent-soft px-3 py-1 text-xs font-semibold text-accent">
              {{ $t('home.all') }}<AppIcon name="chevron" :size="14" />
            </RouterLink>
          </div>
          <div class="space-y-3">
            <div v-for="g in recentGroups" :key="g.day">
              <p class="mb-1 px-1 text-xs font-semibold text-hint first-letter:uppercase">{{ dayLabel(g.day) }}</p>
              <div class="card overflow-hidden py-1">
                <TxRow v-for="tx in g.items" :key="tx.id" :tx="tx" @open="openTx" />
              </div>
            </div>
          </div>
        </section>
      </template>
    </div>

    <BudgetSheet v-if="budgetSheet" v-model:open="budgetSheet" :budget="null" />
  </template>
</template>

<style scoped>
.hero-bg {
  /* Bleeds under the safe area (App pads it) so the hero meets Telegram's tinted header. */
  margin-top: calc(-1 * var(--safe-top));
  padding-top: calc(var(--safe-top) + 0.75rem);
  background: linear-gradient(165deg, #5b3df5 0%, #6a4bfa 55%, #7b5cff 100%);
  border-bottom-left-radius: 32px;
  border-bottom-right-radius: 32px;
}
.blob {
  position: absolute;
  border-radius: 9999px;
  filter: blur(28px);
  pointer-events: none;
}
.blob-1 {
  width: 220px;
  height: 220px;
  top: -60px;
  right: -60px;
  background: rgb(168 146 255 / 65%);
}
.blob-2 {
  width: 200px;
  height: 200px;
  bottom: 40px;
  left: -80px;
  background: rgb(43 217 160 / 35%);
}
.blob-3 {
  width: 140px;
  height: 140px;
  bottom: -30px;
  right: 20%;
  background: rgb(67 39 214 / 70%);
}
.rise {
  animation: rise 520ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
  animation-delay: calc(var(--i, 0) * 60ms);
}
@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
}
@media (prefers-reduced-motion: reduce) {
  .rise {
    animation: none;
  }
}
</style>
