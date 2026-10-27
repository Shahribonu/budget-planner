<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { Currency, PlanGroup, PlanItem } from '@/api/types'
import { useFormat } from '@/composables/useFormat'
import { formatMonth } from '@/lib/dates'
import { CURRENCIES } from '@/lib/money'
import { addMonthsYM, nextPriority } from '@/lib/plan'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useMonthlyPlanStore } from '@/stores/monthlyPlan'
import { useToastStore } from '@/stores/toast'
import AppIcon from '../AppIcon.vue'
import EmptyState from '../EmptyState.vue'
import SegmentedControl from '../SegmentedControl.vue'
import PlanComposer from './PlanComposer.vue'
import PlanItemRow from './PlanItemRow.vue'
import PlanItemSheet from './PlanItemSheet.vue'
import PlanSummaryCard from './PlanSummaryCard.vue'

const { lang } = useFormat()
const plan = useMonthlyPlanStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const toast = useToastStore()

const loading = ref(!plan.view)
const failed = ref(false)
const showDone = ref(false)

async function load() {
  failed.value = false
  try {
    await Promise.all([plan.load(), categories.load(), accounts.load()])
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

function shiftMonth(n: number) {
  if (!plan.view) return
  haptic.selection()
  plan.month = addMonthsYM(plan.view.month, n)
  void load()
}

const hasUsd = computed(() => accounts.active.some((a) => a.currency === 'USD'))
const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))
function setCurrency(c: Currency) {
  plan.currency = c
  void load()
}

const GROUPS: { key: PlanGroup; hint?: string }[] = [
  { key: 'fits' },
  { key: 'later' },
  { key: 'reconsider', hint: 'mplan.reconsider_hint' },
]
const groups = computed(() =>
  GROUPS.map((g) => ({ ...g, items: plan.planned.filter((i) => i.decision?.group === g.key) })).filter((g) => g.items.length),
)
const isEmpty = computed(() => !!plan.view && plan.view.items.length === 0)

async function cycle(item: PlanItem) {
  haptic.selection()
  try {
    await plan.patchItem(item.id, { priority: nextPriority(item.priority) })
  } catch {
    toast.error()
  }
}

const sheetOpen = ref(false)
const editing = ref<PlanItem | null>(null)
function openItem(item: PlanItem | null) {
  editing.value = item
  sheetOpen.value = true
}

const composer = ref<InstanceType<typeof PlanComposer>>()
// Plan text is read as so'm unless written in dollars, so a USD plan needs dollar examples.
const examples = computed(() =>
  plan.view?.currency === 'USD' ? ['mplan.ex1_usd', 'mplan.ex2_usd', 'mplan.ex3_usd'] : ['mplan.ex1', 'mplan.ex2', 'mplan.ex3'],
)
</script>

<template>
  <section class="space-y-3" :aria-label="$t('mplan.title')">
    <div class="flex items-center gap-1 px-1">
      <h2 class="min-w-0 flex-1 truncate text-sm font-medium text-hint">{{ $t('mplan.title') }}</h2>
      <template v-if="plan.view">
        <button type="button" class="pressable rounded-full p-1.5 text-link" :aria-label="$t('mplan.month_prev')" @click="shiftMonth(-1)">
          <AppIcon name="back" :size="18" />
        </button>
        <span class="min-w-24 text-center text-sm font-semibold first-letter:uppercase" aria-live="polite">
          {{ formatMonth(plan.view.month, lang) }}
        </span>
        <button type="button" class="pressable rounded-full p-1.5 text-link" :aria-label="$t('mplan.month_next')" @click="shiftMonth(1)">
          <AppIcon name="chevron" :size="18" />
        </button>
      </template>
    </div>

    <SegmentedControl
      v-if="hasUsd && plan.view"
      :model-value="plan.view.currency"
      :options="currencyOptions"
      :label="$t('debt_form.currency')"
      @update:model-value="setCurrency"
    />

    <div v-if="loading" class="py-10 text-center text-hint">{{ $t('common.loading') }}</div>
    <EmptyState v-else-if="failed || !plan.view" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

    <template v-else>
      <PlanSummaryCard :view="plan.view" />

      <!-- Empty plan: the promise and examples -->
      <div v-if="isEmpty" class="card px-4 py-5 text-center">
        <div class="mb-2 text-4xl" aria-hidden="true">🗓️</div>
        <p class="text-sm">{{ $t('mplan.promise') }}</p>
        <div class="mt-3 flex flex-wrap justify-center gap-2">
          <button
            v-for="ex in examples"
            :key="ex"
            type="button"
            class="pressable rounded-full bg-page px-3 py-1.5 text-sm"
            @click="composer?.fill($t(ex))"
          >
            {{ $t(ex) }}
          </button>
        </div>
        <RouterLink
          v-if="plan.view.inputs.income_source === 'none'"
          :to="{ name: 'recurring', query: { new: 'income' } }"
          class="mt-3 block text-sm text-link"
        >
          {{ $t('mplan.add_salary') }}
        </RouterLink>
      </div>

      <section v-for="g in groups" :key="g.key">
        <h3 class="px-1 pb-1 text-sm font-semibold">
          {{ $t(`mplan.${g.key}`) }}
          <span class="font-normal text-hint">· {{ g.items.length }}</span>
        </h3>
        <p v-if="g.hint" class="-mt-1 px-1 pb-1 text-xs text-hint">{{ $t(g.hint) }}</p>
        <div class="card divide-y divide-separator overflow-hidden">
          <PlanItemRow v-for="item in g.items" :key="item.id" :item="item" @open="openItem" @cycle="cycle" />
        </div>
      </section>

      <PlanComposer ref="composer" @manual="openItem(null)" />

      <section v-if="plan.done.length">
        <button
          type="button"
          class="flex w-full items-center justify-between px-1 py-1 text-sm font-medium text-hint"
          :aria-expanded="showDone"
          @click="showDone = !showDone"
        >
          {{ $t('mplan.done') }} ({{ plan.done.length }})
          <AppIcon name="down" :size="16" class="transition-transform" :class="{ 'rotate-180': showDone }" />
        </button>
        <div v-if="showDone" class="card mt-1 divide-y divide-separator overflow-hidden">
          <PlanItemRow v-for="item in plan.done" :key="item.id" :item="item" @open="openItem" />
        </div>
      </section>
    </template>
  </section>

  <PlanItemSheet v-model:open="sheetOpen" :item="editing" />
</template>
