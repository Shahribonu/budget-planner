<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Budget, Currency, Installment } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import BudgetBar from '@/components/BudgetBar.vue'
import BudgetRing from '@/components/BudgetRing.vue'
import CategoryIcon from '@/components/CategoryIcon.vue'
import BudgetSheet from '@/components/BudgetSheet.vue'
import MonthlyPlanSection from '@/components/plan/MonthlyPlanSection.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { dailyPace } from '@/lib/budgets'
import { formatDay, monthRange } from '@/lib/dates'
import { useBudgetsStore } from '@/stores/budgets'
import { useCategoriesStore } from '@/stores/categories'
import { useDebtsStore } from '@/stores/debts'
import { useInstallmentsStore } from '@/stores/installments'

const router = useRouter()
const debts = useDebtsStore()
const plans = useInstallmentsStore()
const budgets = useBudgetsStore()
const categories = useCategoriesStore()
const { money, lang, today } = useFormat()
const failed = ref(false)
const budgetsFailed = ref(false)

async function load() {
  failed.value = false
  budgetsFailed.value = false
  // Budgets and debts fail independently so one section can still show.
  await Promise.all([
    Promise.all([budgets.load(true), categories.load()]).catch(() => (budgetsFailed.value = true)),
    Promise.all([debts.load(true), plans.load(true)]).catch(() => (failed.value = true)),
  ])
}
onMounted(load)

const sheetOpen = ref(false)
const editing = ref<Budget | null>(null)
function openSheet(b: Budget | null) {
  editing.value = b
  sheetOpen.value = true
}

/** Activity for what the budget counts this month. */
function drill(b: Budget) {
  const range = budgets.data ? monthRange(budgets.data.month) : {}
  const query = b.category_id === null ? { type: 'expense,installment', currency: b.currency } : { category_id: b.category_id, currency: b.currency }
  void router.push({ name: 'activity', query: { ...query, ...range } })
}

function sumByCurrency<T>(items: T[], currency: (x: T) => Currency, amount: (x: T) => number) {
  const totals = new Map<Currency, number>()
  for (const x of items) totals.set(currency(x), (totals.get(currency(x)) ?? 0) + amount(x))
  return [...totals].map(([c, a]) => ({ currency: c, amount_minor: a }))
}

const openDebts = computed(() => debts.all.filter((d) => !d.closed))
const owedToMe = computed(() =>
  sumByCurrency(openDebts.value.filter((d) => d.direction === 'lent'), (d) => d.currency, (d) => d.remaining_minor),
)
const iOwe = computed(() =>
  sumByCurrency(openDebts.value.filter((d) => d.direction === 'borrowed'), (d) => d.currency, (d) => d.remaining_minor),
)

const openPlans = computed(() => plans.all.filter((p) => !p.closed))
const planRemaining = computed(() => sumByCurrency(openPlans.value, (p) => p.currency, (p) => p.remaining_minor))
const soonest = computed<Installment | undefined>(() =>
  openPlans.value
    .filter((p) => p.next)
    .sort((a, b) => (a.next!.due_on < b.next!.due_on ? -1 : 1))[0],
)
</script>

<template>
  <PageHeader :title="$t('plan.title')" />

  <div class="space-y-4 px-4">
    <MonthlyPlanSection />

    <!-- Budgets -->
    <div class="flex items-center justify-between px-1 pt-1">
      <h2 class="eyebrow">{{ $t('budgets.title') }}</h2>
      <button
        v-if="budgets.all.length"
        type="button"
        class="pressable -my-1 flex items-center gap-1 rounded-full px-2 py-1 text-sm text-link"
        @click="openSheet(null)"
      >
        <AppIcon name="plus" :size="16" />{{ $t('budgets.add') }}
      </button>
    </div>

    <EmptyState v-if="budgetsFailed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />
    <div v-else-if="budgets.data && !budgets.all.length" class="card">
      <EmptyState icon="🎯" :body="$t('budgets.empty')" :action="$t('budgets.add')" class="py-6" @action="openSheet(null)" />
    </div>

    <!-- Overall monthly budget, one card per currency -->
    <section v-for="b in budgets.overall" :key="b.id" class="card p-4">
      <div class="flex items-start gap-2">
        <button type="button" class="pressable min-w-0 flex-1 text-left" @click="drill(b)">
          <span class="block text-sm text-hint">{{ $t('budgets.overall') }} · {{ b.currency }}</span>
          <span class="tabular block truncate text-2xl font-bold">{{ money(b.spent_minor, b.currency) }}</span>
          <span class="tabular block truncate text-sm text-hint">
            {{ $t('budgets.limit') }}: {{ money(b.amount_minor, b.currency) }}
          </span>
        </button>
        <button
          type="button"
          class="pressable -mt-1 -mr-2 rounded-full p-2 text-hint"
          :aria-label="$t('budgets.edit')"
          @click="openSheet(b)"
        >
          <AppIcon name="edit" :size="18" />
        </button>
      </div>
      <BudgetBar class="mt-3" :spent="b.spent_minor" :limit="b.amount_minor" thick />
      <div class="mt-2 flex flex-wrap items-baseline justify-between gap-x-3 text-sm">
        <span :class="b.remaining_minor < 0 ? 'font-semibold text-danger' : 'font-medium'">
          {{
            b.remaining_minor < 0
              ? $t('budgets.over', { amount: money(-b.remaining_minor, b.currency) })
              : $t('budgets.left', { amount: money(b.remaining_minor, b.currency) })
          }}
        </span>
        <span class="text-hint">{{ $t('budgets.days_left', { n: budgets.data?.days_left ?? 0 }) }}</span>
      </div>
      <p v-if="b.remaining_minor > 0" class="mt-1 text-xs text-hint">
        {{ $t('budgets.pace', { amount: money(dailyPace(b.remaining_minor, budgets.data?.days_left ?? 0), b.currency) }) }}
      </p>
    </section>

    <!-- Category budgets: rings in a 2-column grid -->
    <section v-if="budgets.byCategory.length" class="grid grid-cols-2 gap-3">
      <div v-for="b in budgets.byCategory" :key="b.id" class="card relative">
        <button type="button" class="pressable flex w-full flex-col items-center gap-2 px-3 pt-4 pb-3 text-center" @click="drill(b)">
          <BudgetRing :spent="b.spent_minor" :limit="b.amount_minor" />
          <span class="flex max-w-full items-center gap-1.5">
            <CategoryIcon :category="categories.byId.get(b.category_id!)" :size="20" />
            <span class="truncate text-sm font-semibold">{{ categoryLabel(categories.byId.get(b.category_id!)) }}</span>
          </span>
          <span class="tabular text-xs text-hint">
            {{ money(b.spent_minor, b.currency, { bare: true }) }} / {{ money(b.amount_minor, b.currency) }}
          </span>
        </button>
        <button
          type="button"
          class="pressable absolute top-1.5 right-1.5 rounded-full p-2 text-hint"
          :aria-label="`${$t('budgets.edit')}: ${categoryLabel(categories.byId.get(b.category_id!))}`"
          @click="openSheet(b)"
        >
          <AppIcon name="edit" :size="14" />
        </button>
      </div>
    </section>

    <h2 class="eyebrow px-1 pt-3">{{ $t('plan.debts_section') }}</h2>

    <EmptyState v-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

    <template v-else>
      <RouterLink :to="{ name: 'debts' }" class="card pressable flex items-start gap-3 p-4">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-page text-xl" aria-hidden="true">🤝</span>
        <span class="min-w-0 flex-1">
          <span class="block font-semibold">{{ $t('plan.debts') }}</span>
          <span class="block text-sm text-hint">
            {{ openDebts.length ? $t('plan.open_count', { n: openDebts.length }) : $t('plan.none') }}
          </span>
          <span v-for="m in owedToMe" :key="`in-${m.currency}`" class="mt-1 flex items-center gap-1 text-sm">
            <AppIcon name="in" :size="14" class="text-income" />
            <span class="min-w-0 flex-1 truncate text-hint">{{ $t('debts.they_owe') }}</span>
            <span class="tabular shrink-0 font-semibold">{{ money(m.amount_minor, m.currency) }}</span>
          </span>
          <span v-for="m in iOwe" :key="`out-${m.currency}`" class="mt-1 flex items-center gap-1 text-sm">
            <AppIcon name="out" :size="14" />
            <span class="min-w-0 flex-1 truncate text-hint">{{ $t('debts.i_owe') }}</span>
            <span class="tabular shrink-0 font-semibold">{{ money(m.amount_minor, m.currency) }}</span>
          </span>
        </span>
        <AppIcon name="chevron" :size="18" class="mt-2.5 shrink-0 text-hint" />
      </RouterLink>

      <RouterLink :to="{ name: 'installments' }" class="card pressable flex items-start gap-3 p-4">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-page text-xl" aria-hidden="true">🛍️</span>
        <span class="min-w-0 flex-1">
          <span class="block font-semibold">{{ $t('plan.installments') }}</span>
          <span class="block text-sm text-hint">
            {{ openPlans.length ? $t('plan.open_count', { n: openPlans.length }) : $t('plan.none') }}
          </span>
          <span v-for="m in planRemaining" :key="m.currency" class="mt-1 flex items-center gap-1 text-sm">
            <span class="min-w-0 flex-1 truncate text-hint">{{ $t('inst.remaining') }}</span>
            <span class="tabular shrink-0 font-semibold">{{ money(m.amount_minor, m.currency) }}</span>
          </span>
          <span
            v-if="soonest?.next"
            class="mt-1 block truncate text-sm"
            :class="soonest.next.due_on < today() ? 'font-medium text-danger' : 'text-hint'"
          >
            {{
              $t('plan.next_payment', {
                amount: money(soonest.next.amount_minor, soonest.currency),
                date: formatDay(soonest.next.due_on, lang),
              })
            }}
            · {{ soonest.title }}
          </span>
        </span>
        <AppIcon name="chevron" :size="18" class="mt-2.5 shrink-0 text-hint" />
      </RouterLink>
    </template>

  </div>

  <BudgetSheet v-model:open="sheetOpen" :budget="editing" />
</template>
