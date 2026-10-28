<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Currency, Debt, DebtDirection } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import ProgressBar from '@/components/ProgressBar.vue'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { formatDay } from '@/lib/dates'
import { useDebtsStore } from '@/stores/debts'

const { t } = useI18n()
const router = useRouter()
const debts = useDebtsStore()
const { money, lang, today } = useFormat()

const loading = ref(!debts.loaded)
const failed = ref(false)
const showClosed = ref(false)

async function load() {
  failed.value = false
  try {
    await debts.load(true)
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const groups = computed(() =>
  (['lent', 'borrowed'] as DebtDirection[])
    .map((direction) => {
      const items = debts.all.filter((d) => !d.closed && d.direction === direction)
      const totals = new Map<Currency, number>()
      for (const d of items) totals.set(d.currency, (totals.get(d.currency) ?? 0) + d.remaining_minor)
      return {
        direction,
        title: direction === 'lent' ? t('debts.they_owe') : t('debts.i_owe'),
        items,
        totals: [...totals].map(([currency, amount]) => ({ currency, amount })),
      }
    })
    .filter((g) => g.items.length),
)
const closed = computed(() => debts.all.filter((d) => d.closed))

function isOverdue(d: Debt) {
  return !d.closed && !!d.due_on && d.due_on < today()
}

const add = () => void router.push({ name: 'debt-new' })
const { native } = useMainButton(
  computed(() => t('debts.add')),
  ref(false),
  add,
)
</script>

<template>
  <PageHeader :title="$t('debts.title')" back>
    <button type="button" class="pressable rounded-full p-2 text-link" :aria-label="$t('debts.add')" @click="add">
      <AppIcon name="plus" />
    </button>
  </PageHeader>

  <div class="space-y-4 px-4 pb-4">
    <div v-if="loading" class="py-20 text-center text-hint">{{ $t('common.loading') }}</div>
    <EmptyState v-else-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />
    <EmptyState
      v-else-if="!debts.all.length"
      icon="🤝"
      :body="$t('debts.empty')"
      :action="$t('debts.add')"
      @action="add"
    />

    <section v-for="g in groups" :key="g.direction">
      <div class="flex items-baseline justify-between gap-2 px-1 pb-1">
        <h2 class="flex items-center gap-1 text-sm font-medium text-hint">
          <AppIcon :name="g.direction === 'lent' ? 'in' : 'out'" :size="14" />{{ g.title }}
        </h2>
        <span class="tabular truncate text-sm font-semibold">
          {{ g.totals.map((x) => money(x.amount, x.currency)).join(' · ') }}
        </span>
      </div>
      <div class="card overflow-hidden py-1">
        <RouterLink
          v-for="d in g.items"
          :key="d.id"
          :to="{ name: 'debt', params: { id: d.id } }"
          class="block px-4 py-2.5 active:bg-page/60"
        >
          <span class="flex items-center gap-3">
            <span
              class="flex size-11 shrink-0 items-center justify-center rounded-full bg-accent-soft font-bold text-accent uppercase"
              aria-hidden="true"
            >
              {{ d.counterparty.slice(0, 1) }}
            </span>
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium">{{ d.counterparty }}</span>
              <span
                v-if="d.due_on"
                class="block truncate text-sm"
                :class="isOverdue(d) ? 'font-medium text-danger' : 'text-hint'"
              >
                {{ isOverdue(d) ? $t('debts.overdue', { date: formatDay(d.due_on, lang) }) : $t('debts.due', { date: formatDay(d.due_on, lang) }) }}
              </span>
              <span v-else-if="d.note" class="block truncate text-sm text-hint">{{ d.note }}</span>
            </span>
            <span class="shrink-0 text-right">
              <span class="tabular block font-semibold whitespace-nowrap">{{ money(d.remaining_minor, d.currency) }}</span>
              <span v-if="d.remaining_minor !== d.principal_minor" class="tabular block text-xs whitespace-nowrap text-hint">
                {{ $t('debts.of', { principal: money(d.principal_minor, d.currency) }) }}
              </span>
            </span>
          </span>
          <ProgressBar
            class="mt-2 ml-13"
            :value="d.principal_minor - d.remaining_minor"
            :total="d.principal_minor"
            :danger="isOverdue(d)"
          />
        </RouterLink>
      </div>
    </section>

    <section v-if="closed.length">
      <button
        type="button"
        class="flex w-full items-center justify-between px-1 py-1 text-sm font-medium text-hint"
        :aria-expanded="showClosed"
        @click="showClosed = !showClosed"
      >
        {{ $t('debts.closed') }} ({{ closed.length }})
        <AppIcon name="down" :size="16" class="transition-transform" :class="{ 'rotate-180': showClosed }" />
      </button>
      <div v-if="showClosed" class="card mt-1 overflow-hidden py-1">
        <RouterLink
          v-for="d in closed"
          :key="d.id"
          :to="{ name: 'debt', params: { id: d.id } }"
          class="flex items-center gap-3 px-4 py-2.5 opacity-70 active:bg-page/60"
        >
          <AppIcon name="check" :size="18" class="shrink-0 text-income" />
          <span class="min-w-0 flex-1 truncate">{{ d.counterparty }}</span>
          <span class="tabular shrink-0 text-sm text-hint">{{ money(d.principal_minor, d.currency) }}</span>
        </RouterLink>
      </div>
    </section>
  </div>

  <FallbackButton v-if="!native && debts.all.length" :label="$t('debts.add')" @click="add" />
</template>
