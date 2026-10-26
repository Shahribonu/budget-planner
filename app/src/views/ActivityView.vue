<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { api } from '@/api/client'
import { CURRENCIES } from '@/lib/money'
import type { Currency, Transaction, TransactionFilter, TransactionType } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import TxRow from '@/components/TxRow.vue'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { formatDay } from '@/lib/dates'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useLedgerStore } from '@/stores/ledger'

const PAGE_SIZE = 50
const TYPES: TransactionType[] = ['expense', 'income', 'transfer', 'exchange', 'opening']
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

const route = useRoute()
const router = useRouter()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const ledger = useLedgerStore()
const { dayLabel, lang } = useFormat()

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
  const type = str(q.type) as TransactionType | undefined
  const currency = str(q.currency) as Currency | undefined
  const from = str(q.from)
  const to = str(q.to)
  return {
    q: str(q.q)?.trim() || undefined,
    account_id: int(q.account_id),
    category_id: int(q.category_id),
    type: type && TYPES.includes(type) ? type : undefined,
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

const sentinel = ref<HTMLElement>()
let observer: IntersectionObserver | undefined

onMounted(() => {
  void Promise.all([accounts.load(), categories.load()]).catch(() => {})
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

function openTx(id: number) {
  void router.push({ name: 'tx-edit', params: { id } })
}
</script>

<template>
  <PageHeader :title="$t('activity.title')" />

  <div class="sticky top-0 z-10 bg-page px-4 pt-1 pb-2" :style="{ top: 'var(--safe-top)' }">
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

      <label class="chip" :class="{ 'chip-on': filter.type }">
        <select
          :value="filter.type ?? ''"
          :aria-label="$t('activity.type')"
          @change="setQuery('type', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">{{ $t('activity.type') }}: {{ $t('activity.all') }}</option>
          <option v-for="t in TYPES" :key="t" :value="t">{{ $t(`types.${t}`) }}</option>
        </select>
      </label>

      <label class="chip" :class="{ 'chip-on': filter.category_id }">
        <select
          :value="filter.category_id ?? ''"
          :aria-label="$t('activity.category')"
          @change="setQuery('category_id', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">{{ $t('activity.category') }}: {{ $t('activity.all') }}</option>
          <optgroup :label="$t('types.expense')">
            <option v-for="c in categories.ofKind('expense')" :key="c.id" :value="c.id">
              {{ c.icon }} {{ categoryLabel(c) }}
            </option>
          </optgroup>
          <optgroup :label="$t('types.income')">
            <option v-for="c in categories.ofKind('income')" :key="c.id" :value="c.id">
              {{ c.icon }} {{ categoryLabel(c) }}
            </option>
          </optgroup>
        </select>
      </label>

      <label class="chip" :class="{ 'chip-on': filter.account_id }">
        <select
          :value="filter.account_id ?? ''"
          :aria-label="$t('activity.account')"
          @change="setQuery('account_id', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">{{ $t('activity.account') }}: {{ $t('activity.all') }}</option>
          <option v-for="a in accounts.all" :key="a.id" :value="a.id">{{ a.name }}</option>
        </select>
      </label>

      <label class="chip" :class="{ 'chip-on': filter.currency }">
        <select
          :value="filter.currency ?? ''"
          :aria-label="$t('activity.currency')"
          @change="setQuery('currency', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">{{ $t('activity.currency') }}: {{ $t('activity.all') }}</option>
          <option v-for="c in CURRENCIES" :key="c" :value="c">{{ c }}</option>
        </select>
      </label>

      <button v-if="hasFilters" type="button" class="chip text-link" @click="clearFilters">
        {{ $t('activity.clear') }}
      </button>
    </div>
  </div>

  <div class="px-4">
    <section v-for="g in groups" :key="g.date" class="mb-3">
      <h2 class="px-1 pt-2 pb-1 text-sm font-medium text-hint first-letter:uppercase">{{ dayLabel(g.date) }}</h2>
      <div class="overflow-hidden rounded-2xl bg-surface py-1">
        <TxRow v-for="tx in g.items" :key="tx.id" :tx="tx" @open="openTx" />
      </div>
    </section>

    <EmptyState
      v-if="!loading && !failed && items.length === 0"
      icon="🔍"
      :body="hasFilters ? $t('activity.empty_filtered') : $t('activity.empty')"
      :action="hasFilters ? $t('activity.clear') : undefined"
      @action="clearFilters"
    />
    <EmptyState v-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="loadPage(items.length === 0)" />
    <div v-if="loading" class="py-6 text-center text-sm text-hint">{{ $t('common.loading') }}</div>
    <div ref="sentinel" class="h-px" />
  </div>
</template>

<style scoped>
@reference '../style.css';

.chip {
  @apply relative flex shrink-0 items-center gap-1 rounded-full bg-surface px-3 py-1.5 text-sm whitespace-nowrap;
}
.chip-on {
  @apply bg-accent text-on-accent;
}
.chip select {
  @apply max-w-44 appearance-none truncate bg-transparent outline-none;
}
.chip select option,
.chip select optgroup {
  @apply bg-bg text-text;
}
</style>
