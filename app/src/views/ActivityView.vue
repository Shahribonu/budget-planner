<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { api } from '@/api/client'
import { CURRENCIES } from '@/lib/money'
import type { Currency, Transaction, TransactionFilter, TransactionType } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import PullIndicator from '@/components/PullIndicator.vue'
import OptionSheet, { type SheetOption } from '@/components/OptionSheet.vue'
import TxRow from '@/components/TxRow.vue'
import { useFormat } from '@/composables/useFormat'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { categoryLabel } from '@/i18n'
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { formatDay } from '@/lib/dates'
import { txRoute } from '@/lib/transactions'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'
import { useLedgerStore } from '@/stores/ledger'

const PAGE_SIZE = 50
const TYPES: TransactionType[] = [
  'expense',
  'income',
  'transfer',
  'exchange',
  'debt',
  'debt_payment',
  'installment',
  'installment_payment',
  'opening',
]
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

const route = useRoute()
const router = useRouter()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const ledger = useLedgerStore()
const plans = useInstallmentsStore()
const { dayLabel, lang } = useFormat()
const { t } = useI18n()

function str(v: LocationQuery[string] | undefined): string | undefined {
  const s = Array.isArray(v) ? v[0] : v
  return s ?? undefined
}

function int(v: LocationQuery[string] | undefined): number | undefined {
  const n = Number(str(v))
  return Number.isInteger(n) && n > 0 ? n : undefined
}

const filter = computed<TransactionFilter>(() => {
  const q = route.query
  // One type, or several comma-separated (the overall budget drills into expense,installment).
  const types = (str(q.type) ?? '').split(',').filter((x): x is TransactionType => TYPES.includes(x as TransactionType))
  const currency = str(q.currency) as Currency | undefined
  const from = str(q.from)
  const to = str(q.to)
  return {
    q: str(q.q)?.trim() || undefined,
    account_id: int(q.account_id),
    category_id: int(q.category_id),
    type: types.length > 1 ? types : types[0],
    currency: currency && CURRENCIES.includes(currency) ? currency : undefined,
    from: from && DATE_RE.test(from) ? from : undefined,
    to: to && DATE_RE.test(to) ? to : undefined,
  }
})

const hasFilters = computed(() => Object.values(filter.value).some((v) => v !== undefined))

function setQuery(key: string, value: string | number | undefined) {
  const query = { ...route.query }
  if (value === undefined || value === '') delete query[key]
  else query[key] = String(value)
  void router.replace({ query })
}

function clearFilters() {
  search.value = ''
  void router.replace({ query: {} })
}

// Search box: debounce into the URL so back/forward and drill-downs keep it.
const search = ref(filter.value.q ?? '')
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, (v) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => setQuery('q', v.trim() || undefined), 300)
})
watch(
  () => filter.value.q,
  (q) => {
    if ((q ?? '') !== search.value.trim()) search.value = q ?? ''
  },
)

const items = ref<Transaction[]>([])
const cursor = ref<string | null>(null)
const loading = ref(false)
const failed = ref(false)
const done = ref(false)
let generation = 0

async function loadPage(reset: boolean) {
  if (!reset && (loading.value || done.value)) return
  const gen = reset ? ++generation : generation
  loading.value = true
  failed.value = false
  try {
    const page = await api.listTransactions({
      ...filter.value,
      cursor: reset ? undefined : (cursor.value ?? undefined),
      limit: PAGE_SIZE,
    })
    if (gen !== generation) return
    items.value = reset ? page.transactions : [...items.value, ...page.transactions]
    cursor.value = page.next_cursor
    done.value = !page.next_cursor
  } catch {
    if (gen === generation) failed.value = true
  } finally {
    if (gen === generation) loading.value = false
  }
}

function reload() {
  if (route.name !== 'activity') return // filter recomputes while navigating away
  done.value = false
  cursor.value = null
  void loadPage(true)
}

watch(filter, reload, { deep: true })
watch(() => ledger.revision, reload)

const { pull, refreshing, threshold } = usePullToRefresh(async () => reload())

const sentinel = ref<HTMLElement>()
let observer: IntersectionObserver | undefined

onMounted(() => {
  // Plans only name their payment rows, so failures are ignored.
  void Promise.all([accounts.load(), categories.load(), plans.load()]).catch(() => {})
  reload()
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting) && !failed.value) void loadPage(false)
    },
    { rootMargin: '400px' },
  )
  if (sentinel.value) observer.observe(sentinel.value)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  clearTimeout(searchTimer)
})

const groups = computed(() => {
  const out: { date: string; items: Transaction[] }[] = []
  for (const tx of items.value) {
    const last = out[out.length - 1]
    if (last?.date === tx.occurred_on) last.items.push(tx)
    else out.push({ date: tx.occurred_on, items: [tx] })
  }
  return out
})

const period = computed(() => {
  const { from, to } = filter.value
  if (!from && !to) return ''
  const f = from ? formatDay(from, lang.value) : '…'
  const t = to ? formatDay(to, lang.value) : '…'
  return f === t ? f : `${f} – ${t}`
})

function clearPeriod() {
  const query = { ...route.query }
  delete query.from
  delete query.to
  void router.replace({ query })
}

type FilterKey = 'type' | 'category_id' | 'account_id' | 'currency'

const TYPE_ICONS: Record<TransactionType, string> = {
  expense: '💸',
  income: '💰',
  transfer: '🔁',
  exchange: '💱',
  opening: '🏁',
  debt: '🤝',
  debt_payment: '↩️',
  installment: '🛍️',
  installment_payment: '🗓️',
}

const sheet = ref<FilterKey | null>(null)

/** Query value of a filter as a string (option values are strings), or null when unset. */
function filterValue(key: FilterKey): string | null {
  const v = filter.value[key]
  if (v === undefined) return null
  return Array.isArray(v) ? v.join(',') : String(v)
}

const FILTERS = computed<{ key: FilterKey; title: string; options: SheetOption<string>[] }[]>(() => [
  {
    key: 'type',
    title: t('activity.type'),
    options: TYPES.map((v) => ({ value: v, label: t(`types.${v}`), icon: TYPE_ICONS[v] })),
  },
  {
    key: 'category_id',
    title: t('activity.category'),
    options: (['expense', 'income'] as const).flatMap((kind) =>
      categories.ofKind(kind).map((c) => ({
        value: String(c.id),
        label: categoryLabel(c),
        icon: c.icon,
        group: t(`types.${kind}`),
      })),
    ),
  },
  {
    key: 'account_id',
    title: t('activity.account'),
    options: accounts.all.map((a) => ({
      value: String(a.id),
      label: a.name,
      icon: ACCOUNT_ICONS[a.type],
      hint: [t(`account_types.${a.type}`), a.currency, a.archived ? t('profile.archived') : '']
        .filter(Boolean)
        .join(' · '),
    })),
  },
  {
    key: 'currency',
    title: t('activity.currency'),
    options: CURRENCIES.map((c) => ({ value: c, label: c, hint: t(`currencies.${c}`) })),
  },
])

function chipValue(key: FilterKey): string {
  const v = filterValue(key)
  if (v === null) return t('activity.all')
  if (key === 'type' && v.includes(',')) return v.split(',').map((x) => t(`types.${x}`)).join(', ')
  const opt = FILTERS.value.find((f) => f.key === key)?.options.find((o) => o.value === v)
  if (!opt) return '…' // accounts/categories still loading
  return key === 'category_id' && opt.icon ? `${opt.icon} ${opt.label}` : opt.label
}

function openTx(tx: Transaction) {
  void router.push(txRoute(tx))
}
</script>

<template>
  <PullIndicator :pull="pull" :refreshing="refreshing" :threshold="threshold" />
  <PageHeader :title="$t('activity.title')" />

  <div class="sticky top-0 z-10 bg-canvas px-4 pt-1 pb-2" :style="{ top: 'var(--safe-top)' }">
    <label class="flex items-center gap-2 rounded-xl bg-surface px-3 py-2.5">
      <AppIcon name="search" :size="18" class="text-hint" />
      <input
        v-model="search"
        type="search"
        enterkeyhint="search"
        :placeholder="$t('activity.search')"
        class="min-w-0 flex-1 bg-transparent outline-none placeholder:text-hint"
      />
      <button v-if="search" type="button" class="text-hint" :aria-label="$t('activity.clear')" @click="search = ''">
        <AppIcon name="x" :size="18" />
      </button>
    </label>

    <div class="-mx-4 mt-2 flex gap-2 overflow-x-auto px-4 pb-1 [scrollbar-width:none]">
      <button v-if="period" type="button" class="chip chip-on" @click="clearPeriod">
        {{ period }} <AppIcon name="x" :size="14" />
      </button>

      <button
        v-for="f in FILTERS"
        :key="f.key"
        type="button"
        class="chip pressable"
        :class="{ 'chip-on': filterValue(f.key) !== null }"
        aria-haspopup="dialog"
        @click="sheet = f.key"
      >
        <span class="max-w-44 truncate">{{ f.title }}: {{ chipValue(f.key) }}</span>
        <AppIcon name="down" :size="14" class="shrink-0 opacity-70" />
      </button>

      <button v-if="hasFilters" type="button" class="chip text-link" @click="clearFilters">
        {{ $t('activity.clear') }}
      </button>
    </div>
  </div>

  <div class="px-4">
    <section v-for="g in groups" :key="g.date" class="mb-3">
      <h2 class="eyebrow px-1 pt-2 pb-2">{{ dayLabel(g.date) }}</h2>
      <div class="overflow-hidden card py-1">
        <TxRow v-for="tx in g.items" :key="tx.id" :tx="tx" @open="openTx" />
      </div>
    </section>

    <EmptyState
      v-if="!loading && !failed && items.length === 0"
      art="search"
      :body="hasFilters ? $t('activity.empty_filtered') : $t('activity.empty')"
      :action="hasFilters ? $t('activity.clear') : undefined"
      @action="clearFilters"
    />
    <EmptyState v-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="loadPage(items.length === 0)" />
    <div v-if="loading" class="card space-y-4 p-4" aria-busy="true">
      <div v-for="i in (items.length ? 2 : 6)" :key="i" class="flex items-center gap-3">
        <div class="skeleton size-11 rounded-full" />
        <div class="flex-1 space-y-2"><div class="skeleton h-4 w-2/3" /><div class="skeleton h-3 w-1/3" /></div>
      </div>
    </div>
    <div ref="sentinel" class="h-px" />
  </div>

  <OptionSheet
    v-for="f in FILTERS"
    :key="f.key"
    :open="sheet === f.key"
    :title="f.title"
    :options="f.options"
    :all-label="$t('activity.all')"
    :searchable="f.key === 'category_id'"
    :model-value="filterValue(f.key)"
    @update:open="(v) => !v && sheet === f.key && (sheet = null)"
    @update:model-value="(v) => setQuery(f.key, v ?? undefined)"
  />
</template>

<style scoped>
@reference '../style.css';

.chip {
  @apply relative flex shrink-0 items-center gap-1 rounded-full bg-surface px-3 py-1.5 text-sm whitespace-nowrap;
}
.chip-on {
  @apply bg-accent text-on-accent;
}
</style>
