<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ApiError, api } from '@/api/client'
import type { Account, InputTransactionType, Transaction, TransactionInput } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import TxRow from '@/components/TxRow.vue'
import { useBack } from '@/composables/useBack'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { categoryLabel } from '@/i18n'
import { addDays } from '@/lib/dates'
import { currencyLabel, formatMoney, parseAmount, toInputValue } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { isManaged, txRoute } from '@/lib/transactions'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useLedgerStore } from '@/stores/ledger'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const LAST_ACCOUNT_KEY = 'bp.lastAccount'
const INPUT_TYPES: InputTransactionType[] = ['expense', 'income', 'transfer', 'exchange']

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { money, lang, today } = useFormat()
const goBack = useBack()
const user = useUserStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const ledger = useLedgerStore()
const toast = useToastStore()

const editId = route.name === 'tx-edit' ? Number(route.params.id) : null
const queryType = route.query.type as InputTransactionType | undefined

const type = ref<InputTransactionType>(queryType && INPUT_TYPES.includes(queryType) ? queryType : 'expense')
const amount = ref('')
const toAmount = ref('')
const accountId = ref<number | null>(null)
const toAccountId = ref<number | null>(null)
const categoryId = ref<number | null>(null)
const date = ref(today())
const description = ref('')
const errors = reactive<Partial<Record<'amount' | 'account' | 'to_account' | 'to_amount' | 'category', string>>>({})

const original = ref<Transaction | null>(null)
const loading = ref(true)
const loadFailed = ref(false)
const saving = ref(false)
// Idempotency key: a double tap on Save records one transaction.
const sourceRef = crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`

// Opening balances (and any managed row without a debt/plan link) can only be deleted here.
const readOnly = computed(() => !!original.value && (original.value.type === 'opening' || isManaged(original.value.type)))

const accountOptions = computed<Account[]>(() => {
  const list = [...accounts.active]
  for (const id of [original.value?.account_id, original.value?.to_account_id]) {
    const a = id ? accounts.byId.get(id) : undefined
    if (a?.archived) list.push(a)
  }
  return list
})

const fromAccount = computed(() => (accountId.value ? accounts.byId.get(accountId.value) : undefined))
const currency = computed(() => fromAccount.value?.currency ?? user.mainCurrency)

const toOptions = computed(() =>
  accountOptions.value.filter((a) => {
    if (a.id === accountId.value) return false
    return type.value === 'exchange' ? a.currency !== currency.value : a.currency === currency.value
  }),
)
const toAccount = computed(() => (toAccountId.value ? accounts.byId.get(toAccountId.value) : undefined))

const categoryOptions = computed(() =>
  type.value === 'expense' || type.value === 'income' ? categories.ofKind(type.value) : [],
)

const rate = computed(() => {
  if (type.value !== 'exchange' || !toAccount.value) return ''
  const from = parseAmount(amount.value, currency.value)
  const to = parseAmount(toAmount.value, toAccount.value.currency)
  if (!from || !to) return ''
  // Show as "1 USD = N UZS" whichever way the money moves.
  const usdFirst = currency.value === 'USD'
  // Both currencies have 2 minor digits, so the minor-unit ratio is the rate; keep 2 decimals for display.
  const rateHundredths = Math.round(((usdFirst ? to / from : from / to) * 100))
  return t('form.rate', { from: 'USD', to: 'UZS', rate: formatMoney(rateHundredths, 'UZS', lang.value, { bare: true }) })
})

function pickDefaultAccount() {
  if (accountId.value && accountOptions.value.some((a) => a.id === accountId.value)) return
  let last: number | undefined
  try {
    last = Number(localStorage.getItem(LAST_ACCOUNT_KEY)) || undefined
  } catch {
    /* storage unavailable */
  }
  const preferred =
    accounts.active.find((a) => a.id === last) ??
    accounts.active.find((a) => a.currency === user.mainCurrency) ??
    accounts.active[0]
  accountId.value = preferred?.id ?? null
}

function fixDependents() {
  if (toAccountId.value && !toOptions.value.some((a) => a.id === toAccountId.value)) toAccountId.value = null
  if ((type.value === 'transfer' || type.value === 'exchange') && !toAccountId.value) {
    toAccountId.value = toOptions.value[0]?.id ?? null
  }
  if (categoryId.value && !categoryOptions.value.some((c) => c.id === categoryId.value)) categoryId.value = null
}

watch([type, accountId], () => {
  if (!loading.value) fixDependents()
})

function fill(tx: Transaction) {
  original.value = tx
  if (tx.type !== 'opening' && !isManaged(tx.type)) type.value = tx.type
  amount.value = toInputValue(tx.amount_minor, tx.currency)
  accountId.value = tx.account_id
  toAccountId.value = tx.to_account_id ?? null
  toAmount.value = tx.to_amount_minor && tx.to_currency ? toInputValue(tx.to_amount_minor, tx.to_currency) : ''
  categoryId.value = tx.category_id ?? null
  date.value = tx.occurred_on
  description.value = tx.description
}

async function load() {
  loading.value = true
  loadFailed.value = false
  try {
    const [, , tx] = await Promise.all([
      accounts.load(),
      categories.load(),
      editId ? api.getTransaction(editId) : Promise.resolve(null),
    ])
    // Debts and plans are edited on their own screens (PUT would return managed_elsewhere).
    if (tx && isManaged(tx.type) && (tx.debt_id || tx.installment_id)) {
      await router.replace(txRoute(tx))
      return
    }
    if (tx) fill(tx)
    else pickDefaultAccount()
    loading.value = false
    fixDependents()
  } catch {
    loadFailed.value = true
    loading.value = false
  }
}
onMounted(load)

function setType(next: InputTransactionType) {
  type.value = next
  for (const k of Object.keys(errors)) delete errors[k as keyof typeof errors]
}

function selectAccount(id: number) {
  haptic.selection()
  accountId.value = id
  delete errors.account
}

function selectToAccount(id: number) {
  haptic.selection()
  toAccountId.value = id
  delete errors.to_account
}

function selectCategory(id: number) {
  haptic.selection()
  categoryId.value = id
  delete errors.category
}

function buildInput(): TransactionInput | null {
  for (const k of Object.keys(errors)) delete errors[k as keyof typeof errors]
  const amountMinor = parseAmount(amount.value, currency.value)
  if (!amountMinor) errors.amount = t('form.err_amount')
  if (!accountId.value) errors.account = t('form.err_account')

  const input: TransactionInput = {
    type: type.value,
    occurred_on: date.value || today(),
    amount_minor: amountMinor ?? 0,
    account_id: accountId.value ?? 0,
    description: description.value.trim(),
    source: original.value?.source ?? 'manual',
  }
  if (type.value === 'transfer' || type.value === 'exchange') {
    if (!toAccountId.value) errors.to_account = t('form.err_to_account')
    else input.to_account_id = toAccountId.value
  }
  if (type.value === 'exchange' && toAccount.value) {
    const to = parseAmount(toAmount.value, toAccount.value.currency)
    if (!to) errors.to_amount = t('form.err_to_amount')
    else input.to_amount_minor = to
  }
  if (type.value === 'expense' || type.value === 'income') {
    if (!categoryId.value) errors.category = t('form.err_category')
    else input.category_id = categoryId.value
  }
  if (!editId) input.source_ref = sourceRef
  return Object.keys(errors).length ? null : input
}

async function save() {
  if (readOnly.value || saving.value) return
  const input = buildInput()
  if (!input) {
    haptic.error()
    return
  }
  saving.value = true
  try {
    await ledger.save(editId, input)
    haptic.success()
    try {
      localStorage.setItem(LAST_ACCOUNT_KEY, String(input.account_id))
    } catch {
      /* storage unavailable */
    }
    toast.show(t('common.saved'))
    goBack()
  } catch (e) {
    haptic.error()
    const field = e instanceof ApiError && e.status === 422 ? e.field : undefined
    const key = field === 'amount_minor' ? 'amount' : field === 'to_amount_minor' ? 'to_amount' : field?.replace(/_id$/, '')
    if (key && key in { amount: 1, account: 1, to_account: 1, to_amount: 1, category: 1 } && e instanceof ApiError) {
      errors[key as keyof typeof errors] = e.message
    } else {
      toast.error()
    }
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!editId || saving.value) return
  saving.value = true
  try {
    await ledger.remove(editId)
    haptic.impact('medium')
    goBack()
  } catch {
    haptic.error()
    toast.error()
  } finally {
    saving.value = false
  }
}

const saveLabel = computed(() => t('common.save'))
const { native } = useMainButton(saveLabel, saving, save)

const typeOptions = computed(() => INPUT_TYPES.map((v) => ({ value: v, label: t(`types.${v}`) })))
const yesterday = computed(() => addDays(today(), -1))
</script>

<template>
  <PageHeader :title="editId ? $t('form.edit') : $t('form.new')" back>
    <button
      v-if="editId && original"
      type="button"
      class="pressable rounded-full p-2 text-danger"
      :aria-label="$t('common.delete')"
      :disabled="saving"
      @click="remove"
    >
      <AppIcon name="trash" :size="22" />
    </button>
  </PageHeader>

  <div v-if="loading" class="py-20 text-center text-hint">{{ $t('common.loading') }}</div>
  <EmptyState v-else-if="loadFailed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

  <div v-else-if="readOnly && original" class="px-4">
    <div class="card py-1"><TxRow :tx="original" /></div>
    <p class="mt-3 px-1 text-sm text-hint">{{ $t('form.opening_readonly') }}</p>
  </div>

  <EmptyState
    v-else-if="!accountOptions.length"
    icon="💳"
    :body="$t('form.no_accounts')"
    :action="$t('form.add_account')"
    @action="router.replace({ name: 'account-new' })"
  />

  <form v-else class="space-y-5 px-4 pb-6" novalidate @submit.prevent="save">
    <SegmentedControl :model-value="type" :options="typeOptions" @update:model-value="setType" />

    <AmountInput
      v-model="amount"
      :label="$t('form.amount')"
      :unit="currencyLabel(currency, lang)"
      :prefix="currency === 'USD'"
      :error="errors.amount"
      :autofocus="!editId"
      @update:model-value="delete errors.amount"
    />

    <section>
      <h2 class="field-label">{{ type === 'income' ? $t('form.income_account') : $t('form.account') }}</h2>
      <div class="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1 [scrollbar-width:none]">
        <button
          v-for="a in accountOptions"
          :key="a.id"
          type="button"
          class="account-chip"
          :class="{ 'account-chip-on': a.id === accountId }"
          :aria-pressed="a.id === accountId"
          @click="selectAccount(a.id)"
        >
          <span class="block truncate font-medium">{{ a.name }}</span>
          <span class="tabular block truncate text-xs opacity-75">{{ money(a.balance_minor, a.currency) }}</span>
        </button>
      </div>
      <p v-if="errors.account" class="field-error">{{ errors.account }}</p>
    </section>

    <section v-if="type === 'transfer' || type === 'exchange'">
      <h2 class="field-label">{{ $t('form.to_account') }}</h2>
      <p v-if="!toOptions.length" class="text-sm text-hint">
        {{ type === 'exchange' ? $t('form.need_other_currency') : $t('form.need_same_currency', { currency }) }}
        <RouterLink :to="{ name: 'account-new' }" class="text-link">{{ $t('form.add_account') }}</RouterLink>
      </p>
      <div v-else class="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1 [scrollbar-width:none]">
        <button
          v-for="a in toOptions"
          :key="a.id"
          type="button"
          class="account-chip"
          :class="{ 'account-chip-on': a.id === toAccountId }"
          :aria-pressed="a.id === toAccountId"
          @click="selectToAccount(a.id)"
        >
          <span class="block truncate font-medium">{{ a.name }}</span>
          <span class="tabular block truncate text-xs opacity-75">{{ money(a.balance_minor, a.currency) }}</span>
        </button>
      </div>
      <p v-if="errors.to_account" class="field-error">{{ errors.to_account }}</p>
    </section>

    <section v-if="type === 'exchange' && toAccount">
      <h2 class="field-label">{{ $t('form.to_amount') }}</h2>
      <AmountInput
        v-model="toAmount"
        :label="$t('form.to_amount')"
        :unit="currencyLabel(toAccount.currency, lang)"
        :prefix="toAccount.currency === 'USD'"
        :error="errors.to_amount"
        @update:model-value="delete errors.to_amount"
      />
      <p v-if="rate" class="mt-2 text-center text-sm text-hint">{{ rate }}</p>
    </section>

    <section v-if="categoryOptions.length">
      <h2 class="field-label">{{ $t('form.category') }}</h2>
      <div class="grid grid-cols-4 gap-2">
        <button
          v-for="c in categoryOptions"
          :key="c.id"
          type="button"
          class="pressable flex min-w-0 flex-col items-center gap-1 rounded-xl border-2 px-1 py-2"
          :class="c.id === categoryId ? 'border-accent bg-accent/10' : 'border-transparent bg-surface'"
          :aria-pressed="c.id === categoryId"
          @click="selectCategory(c.id)"
        >
          <span class="text-2xl" aria-hidden="true">{{ c.icon }}</span>
          <span class="line-clamp-2 w-full text-center text-[11px] leading-tight">{{ categoryLabel(c) }}</span>
        </button>
      </div>
      <p v-if="errors.category" class="field-error">{{ errors.category }}</p>
    </section>

    <section>
      <h2 class="field-label">{{ $t('form.date') }}</h2>
      <div class="flex flex-wrap items-center gap-2">
        <button type="button" class="date-chip" :class="{ 'date-chip-on': date === today() }" @click="date = today()">
          {{ $t('common.today') }}
        </button>
        <button type="button" class="date-chip" :class="{ 'date-chip-on': date === yesterday }" @click="date = yesterday">
          {{ $t('common.yesterday') }}
        </button>
        <input
          v-model="date"
          type="date"
          :max="today()"
          :aria-label="$t('form.date')"
          class="min-w-0 flex-1 rounded-xl bg-surface px-3 py-2 text-sm outline-none"
        />
      </div>
    </section>

    <label class="block">
      <span class="field-label">{{ $t('form.description') }}</span>
      <input
        v-model="description"
        type="text"
        maxlength="200"
        enterkeyhint="done"
        :placeholder="$t('form.description_placeholder')"
        class="w-full rounded-xl bg-surface px-3 py-3 outline-none placeholder:text-hint"
      />
    </label>
  </form>

  <FallbackButton v-if="!native && !loading && !loadFailed && !readOnly && accountOptions.length" :label="$t('common.save')" :busy="saving" @click="save" />
</template>

<style scoped>
@reference '../style.css';

.field-label {
  @apply mb-2 block text-sm font-medium text-hint;
}
.field-error {
  @apply mt-1 text-sm text-danger;
}
.account-chip {
  @apply pressable w-32 shrink-0 rounded-xl border-2 border-transparent bg-surface px-3 py-2 text-left;
}
.account-chip-on {
  @apply border-accent;
}
.date-chip {
  @apply pressable rounded-full bg-surface px-3 py-2 text-sm;
}
.date-chip-on {
  @apply bg-accent text-on-accent;
}
</style>
