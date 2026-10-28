<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import type { Currency, Recurring } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import SkeletonList from '@/components/SkeletonList.vue'
import AppIcon from '@/components/AppIcon.vue'
import CategoryIcon from '@/components/CategoryIcon.vue'
import BottomSheet from '@/components/BottomSheet.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import PickerField from '@/components/PickerField.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { invalidField } from '@/composables/useApiError'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { CURRENCIES, currencyLabel, parseAmount, toInputValue } from '@/lib/money'
import { confirmDialog, haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useMonthlyPlanStore } from '@/stores/monthlyPlan'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

type Kind = Recurring['kind']

const route = useRoute()
const { t } = useI18n()
const { money, lang } = useFormat()
const user = useUserStore()
const categories = useCategoriesStore()
const accounts = useAccountsStore()
const plan = useMonthlyPlanStore()
const toast = useToastStore()

const items = ref<Recurring[]>([])
const loading = ref(true)
const failed = ref(false)

async function load() {
  failed.value = false
  try {
    const [list] = await Promise.all([api.listRecurring(), categories.load(), accounts.load()])
    items.value = list
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(async () => {
  await load()
  // Deep link from the plan's "add salary" nudge.
  if (route.query.new === 'income' || route.query.new === 'expense') openSheet(null, route.query.new)
})

const groups = computed(() =>
  (['income', 'expense'] as Kind[])
    .map((kind) => ({ kind, items: items.value.filter((r) => r.kind === kind).sort((a, b) => a.day_of_month - b.day_of_month) }))
    .filter((g) => g.items.length),
)

// Sheet: add (kind, category and currency are fixed after creation) or edit title, amount and day.
const sheetOpen = ref(false)
const editing = ref<Recurring | null>(null)
const kind = ref<Kind>('income')
const title = ref('')
const currency = ref<Currency>(user.mainCurrency)
const amount = ref('')
const day = ref<number | ''>(1)
const categoryId = ref<string | null>(null)
const accountId = ref<string | null>(null)
const busy = ref(false)
const errors = ref<{ title?: string; amount?: string; day?: string }>({})

function openSheet(r: Recurring | null, newKind: Kind = 'income') {
  editing.value = r
  kind.value = r?.kind ?? newKind
  title.value = r?.title ?? ''
  currency.value = r?.currency ?? user.mainCurrency
  amount.value = r ? toInputValue(r.amount_minor, r.currency) : ''
  day.value = r?.day_of_month ?? 1
  categoryId.value = r?.category_id ? String(r.category_id) : null
  accountId.value = r?.account_id ? String(r.account_id) : null
  errors.value = {}
  sheetOpen.value = true
}
watch(kind, () => {
  if (!editing.value) categoryId.value = null
})
watch(currency, () => {
  if (!editing.value) accountId.value = null
})

const accountOptions = computed(() =>
  accounts.active
    .filter((a) => a.currency === currency.value)
    .map((a) => ({ value: String(a.id), label: a.name, icon: ACCOUNT_ICONS[a.type] })),
)

const kindOptions = computed(() => [
  { value: 'income' as const, label: t('recurring.kind_income') },
  { value: 'expense' as const, label: t('recurring.kind_expense') },
])
const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))
const categoryOptions = computed(() =>
  categories.ofKind(kind.value).map((c) => ({ value: String(c.id), label: categoryLabel(c), icon: c.icon })),
)

async function save() {
  if (busy.value) return
  const e: typeof errors.value = {}
  const minor = parseAmount(amount.value, currency.value)
  const d = Number(day.value)
  if (!title.value.trim()) e.title = t('recurring.err_title')
  if (!minor) e.amount = t('form.err_amount')
  if (!Number.isInteger(d) || d < 1 || d > 31) e.day = t('recurring.day')
  errors.value = e
  if (Object.keys(e).length || !minor) {
    haptic.error()
    return
  }
  busy.value = true
  try {
    const r = editing.value
    if (r) {
      const saved = await api.patchRecurring(r.id, {
        title: title.value.trim(),
        amount_minor: minor,
        day_of_month: d,
        category_id: categoryId.value ? Number(categoryId.value) : null,
        account_id: accountId.value ? Number(accountId.value) : null,
      })
      items.value = items.value.map((x) => (x.id === saved.id ? saved : x))
    } else {
      const saved = await api.createRecurring({
        kind: kind.value,
        title: title.value.trim(),
        currency: currency.value,
        amount_minor: minor,
        day_of_month: d,
        category_id: categoryId.value ? Number(categoryId.value) : null,
        account_id: accountId.value ? Number(accountId.value) : null,
      })
      items.value = [...items.value, saved]
    }
    refreshPlan()
    haptic.success()
    toast.show(t('common.saved'))
    sheetOpen.value = false
  } catch (err) {
    haptic.error()
    const { field, message } = invalidField(err)
    if (field === 'title') errors.value = { title: message }
    else if (field === 'amount_minor') errors.value = { amount: message }
    else if (field === 'day_of_month') errors.value = { day: message }
    else if (field === 'category_id' || field === 'account_id') toast.show(message)
    else toast.error()
  } finally {
    busy.value = false
  }
}

async function remove() {
  const r = editing.value
  if (!r || busy.value || !(await confirmDialog(t('recurring.delete_confirm')))) return
  busy.value = true
  try {
    await api.deleteRecurring(r.id)
    items.value = items.value.filter((x) => x.id !== r.id)
    refreshPlan()
    haptic.success()
    toast.show(t('common.deleted'))
    sheetOpen.value = false
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}

/** Income and bills feed the plan, so it has to recalculate. */
function refreshPlan() {
  if (plan.view) void plan.load().catch(() => {})
}
</script>

<template>
  <PageHeader :title="$t('recurring.title')" back>
    <button type="button" class="pressable rounded-full p-2 text-link" :aria-label="$t('recurring.add')" @click="openSheet(null)">
      <AppIcon name="plus" />
    </button>
  </PageHeader>

  <div class="space-y-4 px-4 pb-6">
    <SkeletonList v-if="loading" class="-mx-4" />
    <EmptyState v-else-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />
    <EmptyState
      v-else-if="!items.length"
      icon="🔁"
      :body="$t('recurring.empty')"
      :action="$t('mplan.add_salary_btn')"
      @action="openSheet(null, 'income')"
    />

    <section v-for="g in groups" :key="g.kind">
      <h2 class="eyebrow flex items-center gap-1 px-1 pb-2">
        <AppIcon :name="g.kind === 'income' ? 'in' : 'out'" :size="14" />
        {{ g.kind === 'income' ? $t('recurring.income') : $t('recurring.bills') }}
      </h2>
      <div class="card divide-y divide-separator overflow-hidden">
        <button
          v-for="r in g.items"
          :key="r.id"
          type="button"
          class="flex w-full items-center gap-3 px-4 py-2.5 text-left active:bg-page/60"
          @click="openSheet(r)"
        >
          <CategoryIcon :category="r.category_id ? categories.byId.get(r.category_id) : undefined" :size="44" />
          <span class="min-w-0 flex-1">
            <span class="block truncate font-medium">{{ r.title }}</span>
            <span class="block text-sm text-hint">{{ $t('recurring.day_n', { n: r.day_of_month }) }}</span>
          </span>
          <span class="tabular shrink-0 font-semibold" :class="{ 'text-income': r.kind === 'income' }">
            {{ money(r.kind === 'income' ? r.amount_minor : -r.amount_minor, r.currency, { signed: r.kind === 'income' }) }}
          </span>
        </button>
      </div>
    </section>
  </div>

  <BottomSheet v-model:open="sheetOpen" :title="editing ? editing.title : $t('recurring.new')">
    <form class="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 pt-2 pb-4" novalidate @submit.prevent="save">
      <SegmentedControl v-if="!editing" v-model="kind" :options="kindOptions" :label="$t('recurring.new')" />

      <label class="block">
        <span class="form-label">{{ $t('recurring.name') }}</span>
        <input
          v-model="title"
          type="text"
          maxlength="64"
          :placeholder="kind === 'income' ? $t('recurring.ph_income') : $t('recurring.ph_expense')"
          class="form-input bg-page"
          :aria-invalid="!!errors.title"
          data-autofocus
        />
        <span v-if="errors.title" class="form-error">{{ errors.title }}</span>
      </label>

      <SegmentedControl v-if="!editing" v-model="currency" :options="currencyOptions" :label="$t('debt_form.currency')" />
      <AmountInput
        v-model="amount"
        :label="$t('recurring.amount')"
        :unit="currencyLabel(currency, lang)"
        :prefix="currency === 'USD'"
        :error="errors.amount"
      />

      <label class="flex items-center justify-between gap-3">
        <span class="text-sm">{{ $t('recurring.day') }}</span>
        <input
          v-model.number="day"
          type="number"
          inputmode="numeric"
          min="1"
          max="31"
          class="tabular w-20 rounded-xl bg-page px-3 py-2 text-center outline-none"
          :aria-invalid="!!errors.day"
        />
      </label>

      <PickerField
        v-model="categoryId"
        :label="$t('recurring.category')"
        :options="categoryOptions"
        :none-label="$t('mplan.no_category')"
      />
      <PickerField
        v-model="accountId"
        :label="$t('recurring.account')"
        :options="accountOptions"
        :none-label="$t('recurring.no_account')"
      />
      <p v-if="editing" class="text-xs text-hint">{{ $t('recurring.fixed_hint') }}</p>

      <button
        type="submit"
        class="pressable w-full rounded-xl bg-accent py-3.5 font-semibold text-on-accent disabled:opacity-60"
        :disabled="busy"
      >
        {{ busy ? $t('common.loading') : $t('common.save') }}
      </button>
      <button
        v-if="editing"
        type="button"
        class="pressable flex w-full items-center justify-center gap-2 rounded-xl py-3 text-danger"
        :disabled="busy"
        @click="remove"
      >
        <AppIcon name="trash" :size="18" />{{ $t('recurring.delete') }}
      </button>
    </form>
  </BottomSheet>
</template>
