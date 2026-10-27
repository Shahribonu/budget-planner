<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Currency, DebtDirection } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import PickerField from '@/components/PickerField.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { invalidField } from '@/composables/useApiError'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { CURRENCIES, currencyLabel, parseAmount } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useDebtsStore } from '@/stores/debts'
import { useLedgerStore } from '@/stores/ledger'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const router = useRouter()
const { money, lang, today } = useFormat()
const user = useUserStore()
const accounts = useAccountsStore()
const debts = useDebtsStore()
const ledger = useLedgerStore()
const toast = useToastStore()

const direction = ref<DebtDirection>('lent')
const counterparty = ref('')
const currency = ref<Currency>(user.mainCurrency)
const amount = ref('')
const accountId = ref<string | null>(null)
const date = ref(today())
const dueOn = ref('')
const note = ref('')
const saving = ref(false)
const errors = reactive<{ counterparty?: string; amount?: string; account?: string; date?: string; due?: string }>({})

onMounted(() => {
  void accounts.load().then(pickAccount).catch(() => {})
  void debts.load().catch(() => {})
})

const accountOptions = computed(() =>
  accounts.active
    .filter((a) => a.currency === currency.value)
    .map((a) => ({
      value: String(a.id),
      label: a.name,
      icon: ACCOUNT_ICONS[a.type],
      hint: money(a.balance_minor, a.currency),
    })),
)

// Default to the first account in the debt's currency; none left means an old debt.
function pickAccount() {
  if (!accountOptions.value.some((o) => o.value === accountId.value)) accountId.value = accountOptions.value[0]?.value ?? null
}
watch(currency, pickAccount)

/** People already in the debt list, for one-tap entry. */
const people = computed(() => [...new Set(debts.all.map((d) => d.counterparty))].slice(0, 6))

const directionOptions = computed(() => [
  { value: 'lent' as const, label: t('debt_form.lent') },
  { value: 'borrowed' as const, label: t('debt_form.borrowed') },
])
const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))

async function save() {
  if (saving.value) return
  for (const k of Object.keys(errors)) delete errors[k as keyof typeof errors]
  const amountMinor = parseAmount(amount.value, currency.value)
  if (!counterparty.value.trim()) errors.counterparty = t('debt_form.err_person')
  if (!amountMinor) errors.amount = t('form.err_amount')
  if (dueOn.value && dueOn.value < (date.value || today())) errors.due = t('debt_form.err_due')
  if (Object.keys(errors).length || !amountMinor) {
    haptic.error()
    return
  }
  saving.value = true
  try {
    const debt = await api.createDebt({
      direction: direction.value,
      counterparty: counterparty.value.trim(),
      currency: currency.value,
      amount_minor: amountMinor,
      account_id: accountId.value ? Number(accountId.value) : undefined,
      occurred_on: date.value || today(),
      due_on: dueOn.value || null,
      note: note.value.trim(),
    })
    debts.upsert(debt)
    ledger.changed()
    haptic.success()
    toast.show(t('common.saved'))
    await router.replace({ name: 'debt', params: { id: debt.id } })
  } catch (e) {
    haptic.error()
    const { field, code, message } = invalidField(e)
    if (code === 'invalid_due_date' || field === 'due_on') errors.due = t('debt_form.err_due')
    else if (field === 'counterparty') errors.counterparty = message
    else if (field === 'amount_minor') errors.amount = message
    else if (field === 'account_id') errors.account = message
    else if (field === 'occurred_on') errors.date = message
    else toast.error()
  } finally {
    saving.value = false
  }
}

const { native } = useMainButton(
  computed(() => t('common.save')),
  saving,
  save,
)
</script>

<template>
  <PageHeader :title="$t('debt_form.title')" back />

  <form class="space-y-5 px-4 pb-6" novalidate @submit.prevent="save">
    <SegmentedControl v-model="direction" :options="directionOptions" :label="$t('debt_form.title')" />

    <label class="block">
      <span class="form-label">{{ $t('debt_form.person') }}</span>
      <input
        v-model="counterparty"
        type="text"
        maxlength="64"
        autocomplete="off"
        :placeholder="$t('debt_form.person_placeholder')"
        class="form-input"
        :aria-invalid="!!errors.counterparty"
        @input="delete errors.counterparty"
      />
      <span v-if="errors.counterparty" class="form-error">{{ errors.counterparty }}</span>
    </label>
    <div v-if="people.length" class="-mt-3 flex flex-wrap gap-2">
      <button
        v-for="p in people"
        :key="p"
        type="button"
        class="pressable rounded-full bg-surface px-3 py-1.5 text-sm"
        :class="{ 'bg-accent text-on-accent': p === counterparty }"
        @click="counterparty = p"
      >
        {{ p }}
      </button>
    </div>

    <div class="space-y-3">
      <SegmentedControl v-model="currency" :options="currencyOptions" :label="$t('debt_form.currency')" />
      <AmountInput
        v-model="amount"
        :label="$t('debt_form.amount')"
        :unit="currencyLabel(currency, lang)"
        :prefix="currency === 'USD'"
        :error="errors.amount"
        autofocus
        @update:model-value="delete errors.amount"
      />
    </div>

    <div>
      <PickerField
        v-model="accountId"
        :label="direction === 'lent' ? $t('debt_form.account_lent') : $t('debt_form.account_borrowed')"
        :options="accountOptions"
        :none-label="$t('debt_form.no_account')"
        :error="errors.account"
      />
      <p class="mt-1 px-1 text-xs text-hint">{{ $t('debt_form.no_account_hint') }}</p>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <label class="block min-w-0">
        <span class="form-label">{{ $t('debt_form.date') }}</span>
        <input v-model="date" type="date" :max="today()" class="form-input py-2.5" />
        <span v-if="errors.date" class="form-error">{{ errors.date }}</span>
      </label>
      <label class="block min-w-0">
        <span class="form-label truncate">{{ $t('debt_form.due') }}</span>
        <input v-model="dueOn" type="date" :min="date" class="form-input py-2.5" :aria-invalid="!!errors.due" @input="delete errors.due" />
      </label>
    </div>

    <span v-if="errors.due" class="form-error -mt-3">{{ errors.due }}</span>

    <label class="block">
      <span class="form-label">{{ $t('debt_form.note') }}</span>
      <input v-model="note" type="text" maxlength="200" class="form-input" />
    </label>
  </form>

  <FallbackButton v-if="!native" :label="$t('common.save')" :busy="saving" @click="save" />
</template>
