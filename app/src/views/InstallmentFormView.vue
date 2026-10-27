<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Currency } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import AppIcon from '@/components/AppIcon.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import PickerField from '@/components/PickerField.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import { invalidField } from '@/composables/useApiError'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { categoryLabel } from '@/i18n'
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { formatDay } from '@/lib/dates'
import { addMonthsClamped, splitPayments, tooSmallForMonths } from '@/lib/installments'
import { CURRENCIES, currencyLabel, parseAmount } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'
import { useLedgerStore } from '@/stores/ledger'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const PROVIDERS = ['Uzum Nasiya', 'Alif Nasiya', 'Anor', 'Openshop']
const MIN_MONTHS = 1
const MAX_MONTHS = 60

const { t } = useI18n()
const router = useRouter()
const { money, lang, today } = useFormat()
const user = useUserStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const plans = useInstallmentsStore()
const ledger = useLedgerStore()
const toast = useToastStore()

const title = ref('')
const provider = ref('')
const categoryId = ref<string | null>(null)
const currency = ref<Currency>(user.mainCurrency)
const total = ref('')
const months = ref(12)
const purchaseDate = ref(today())
const firstDue = ref(addMonthsClamped(today(), 1))
const accountId = ref<string | null>(null)
const recordPurchase = ref(true)
const saving = ref(false)
const errors = reactive<{ title?: string; category?: string; total?: string; months?: string; account?: string; date?: string }>({})

onMounted(() => {
  void Promise.all([accounts.load(), categories.load()])
    .then(() => {
      pickAccount()
      // Most nasiya purchases are electronics or home goods; fall back to the first expense category.
      categoryId.value ??= String(categories.ofKind('expense').find((c) => c.key === 'home')?.id ?? '') || null
    })
    .catch(() => toast.error())
})

const categoryOptions = computed(() =>
  categories.ofKind('expense').map((c) => ({ value: String(c.id), label: categoryLabel(c), icon: c.icon })),
)
const accountOptions = computed(() =>
  accounts.active
    .filter((a) => a.currency === currency.value)
    .map((a) => ({ value: String(a.id), label: a.name, icon: ACCOUNT_ICONS[a.type], hint: money(a.balance_minor, a.currency) })),
)
function pickAccount() {
  if (!accountOptions.value.some((o) => o.value === accountId.value)) accountId.value = accountOptions.value[0]?.value ?? null
}
watch(currency, pickAccount)
watch(purchaseDate, (d, old) => {
  // Keep "first payment one month after purchase" unless the user set it themselves.
  if (old && firstDue.value === addMonthsClamped(old, 1)) firstDue.value = addMonthsClamped(d, 1)
})

const totalMinor = computed(() => parseAmount(total.value, currency.value) ?? 0)
const payments = computed(() => splitPayments(totalMinor.value, months.value))
const tooSmall = computed(() => totalMinor.value > 0 && tooSmallForMonths(totalMinor.value, months.value))
/** Server rule: first payment within a year after the purchase. */
const firstDueMax = computed(() => addMonthsClamped(purchaseDate.value || today(), 12))
const lastDue = computed(() => addMonthsClamped(firstDue.value || today(), months.value - 1))

function setMonths(n: number) {
  const next = Math.min(MAX_MONTHS, Math.max(MIN_MONTHS, Math.round(n) || MIN_MONTHS))
  if (next !== months.value) haptic.selection()
  months.value = next
  delete errors.months
}

const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))

async function save() {
  if (saving.value) return
  for (const k of Object.keys(errors)) delete errors[k as keyof typeof errors]
  if (!title.value.trim()) errors.title = t('inst_form.err_title')
  if (!categoryId.value) errors.category = t('inst_form.err_category')
  if (!totalMinor.value) errors.total = t('form.err_amount')
  else if (tooSmall.value) errors.total = t('inst_form.err_small')
  const purchased = purchaseDate.value || today()
  if (!firstDue.value || firstDue.value < purchased || firstDue.value > firstDueMax.value) errors.date = t('inst_form.err_first_due')
  if (Object.keys(errors).length || !categoryId.value) {
    haptic.error()
    return
  }
  saving.value = true
  try {
    const plan = await api.createInstallment({
      title: title.value.trim(),
      provider: provider.value.trim(),
      category_id: Number(categoryId.value),
      currency: currency.value,
      total_minor: totalMinor.value,
      months: months.value,
      first_due_on: firstDue.value,
      pay_from_account_id: accountId.value ? Number(accountId.value) : null,
      occurred_on: purchaseDate.value || today(),
      record_purchase: recordPurchase.value,
    })
    plans.upsert(plan)
    ledger.changed()
    haptic.success()
    toast.show(t('common.saved'))
    // The detail screen shows the server's schedule.
    await router.replace({ name: 'installment', params: { id: plan.id } })
  } catch (e) {
    haptic.error()
    const { field, message } = invalidField(e)
    if (field === 'title') errors.title = message
    else if (field === 'category_id') errors.category = message
    else if (field === 'total_minor') errors.total = message
    else if (field === 'months') errors.months = message
    else if (field === 'pay_from_account_id') errors.account = message
    else if (field === 'occurred_on' || field === 'first_due_on') errors.date = message
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
  <PageHeader :title="$t('inst_form.title')" back />

  <form class="space-y-5 px-4 pb-6" novalidate @submit.prevent="save">
    <label class="block">
      <span class="form-label">{{ $t('inst_form.item') }}</span>
      <input
        v-model="title"
        type="text"
        maxlength="64"
        :placeholder="$t('inst_form.item_placeholder')"
        class="form-input"
        :aria-invalid="!!errors.title"
        @input="delete errors.title"
      />
      <span v-if="errors.title" class="form-error">{{ errors.title }}</span>
    </label>

    <div>
      <label class="block">
        <span class="form-label">{{ $t('inst_form.provider') }}</span>
        <input
          v-model="provider"
          type="text"
          maxlength="64"
          :placeholder="$t('inst_form.provider_placeholder')"
          class="form-input"
        />
      </label>
      <div class="mt-2 flex flex-wrap gap-2">
        <button
          v-for="p in PROVIDERS"
          :key="p"
          type="button"
          class="pressable rounded-full px-3 py-1.5 text-sm"
          :class="provider === p ? 'bg-accent text-on-accent' : 'bg-surface'"
          :aria-pressed="provider === p"
          @click="provider = provider === p ? '' : p"
        >
          {{ p }}
        </button>
      </div>
    </div>

    <PickerField
      v-model="categoryId"
      :label="$t('inst_form.category')"
      :options="categoryOptions"
      :placeholder="$t('form.err_category')"
      :error="errors.category"
      searchable
    />

    <div class="space-y-3">
      <span class="form-label">{{ $t('inst_form.total') }}</span>
      <SegmentedControl v-model="currency" :options="currencyOptions" :label="$t('debt_form.currency')" />
      <AmountInput
        v-model="total"
        :label="$t('inst_form.total')"
        :unit="currencyLabel(currency, lang)"
        :prefix="currency === 'USD'"
        :error="errors.total"
        @update:model-value="delete errors.total"
      />
    </div>

    <div>
      <span class="form-label">{{ $t('inst_form.months') }}</span>
      <div class="card flex items-center justify-between p-1.5">
        <button
          type="button"
          class="pressable flex size-11 items-center justify-center rounded-xl bg-page text-link disabled:opacity-40"
          :aria-label="$t('inst_form.decrease')"
          :disabled="months <= MIN_MONTHS"
          @click="setMonths(months - 1)"
        >
          <AppIcon name="minus" />
        </button>
        <input
          :value="months"
          type="number"
          inputmode="numeric"
          :min="MIN_MONTHS"
          :max="MAX_MONTHS"
          :aria-label="$t('inst_form.months')"
          class="tabular w-20 bg-transparent text-center text-2xl font-bold outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none"
          @change="setMonths(Number(($event.target as HTMLInputElement).value))"
        />
        <button
          type="button"
          class="pressable flex size-11 items-center justify-center rounded-xl bg-page text-link disabled:opacity-40"
          :aria-label="$t('inst_form.increase')"
          :disabled="months >= MAX_MONTHS"
          @click="setMonths(months + 1)"
        >
          <AppIcon name="plus" />
        </button>
      </div>
      <span v-if="errors.months" class="form-error">{{ errors.months }}</span>
    </div>

    <!-- Live preview with the server's rounding -->
    <div v-if="payments.length" class="card p-4" aria-live="polite">
      <div class="text-sm text-hint">{{ $t('inst_form.monthly') }}</div>
      <div v-if="tooSmall" class="mt-1 text-sm text-danger">{{ $t('inst_form.err_small') }}</div>
      <template v-else>
        <div class="tabular text-2xl font-bold">{{ money(payments[0]!, currency) }}</div>
        <div class="tabular text-sm text-hint">
          {{ $t('inst_form.preview', { n: months, amount: money(payments[0]!, currency) }) }}
          <template v-if="payments.length > 1 && payments[payments.length - 1] !== payments[0]">
            · {{ $t('inst_form.preview_last', { amount: money(payments[payments.length - 1]!, currency) }) }}
          </template>
        </div>
        <div v-if="firstDue" class="text-sm text-hint">{{ formatDay(firstDue, lang, true) }} → {{ formatDay(lastDue, lang, true) }}</div>
      </template>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <label class="block min-w-0">
        <span class="form-label truncate">{{ $t('inst_form.purchase_date') }}</span>
        <input v-model="purchaseDate" type="date" :max="today()" class="form-input py-2.5" />
      </label>
      <label class="block min-w-0">
        <span class="form-label truncate">{{ $t('inst_form.first_due') }}</span>
        <input v-model="firstDue" type="date" required :min="purchaseDate" :max="firstDueMax" class="form-input py-2.5" @input="delete errors.date" />
      </label>
    </div>
    <span v-if="errors.date" class="form-error -mt-3">{{ errors.date }}</span>

    <PickerField
      v-model="accountId"
      :label="$t('inst_form.account')"
      :options="accountOptions"
      :none-label="$t('inst_form.account_later')"
      :error="errors.account"
    />

    <div class="card space-y-1 p-4">
      <ToggleSwitch v-model="recordPurchase" :label="$t('inst_form.record')" />
      <p class="text-sm text-hint">{{ $t('inst_form.record_hint') }}</p>
    </div>
  </form>

  <FallbackButton v-if="!native" :label="$t('common.save')" :busy="saving" @click="save" />
</template>
