<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import type { Installment, Transaction } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import SkeletonList from '@/components/SkeletonList.vue'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import PickerField from '@/components/PickerField.vue'
import ProgressBar from '@/components/ProgressBar.vue'
import TxRow from '@/components/TxRow.vue'
import { invalidField } from '@/composables/useApiError'
import { useBack } from '@/composables/useBack'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { categoryLabel } from '@/i18n'
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { darken } from '@/lib/accountStyle'
import { categoryStyle } from '@/lib/categoryStyle'
import { formatDay } from '@/lib/dates'
import { currencyLabel, parseAmount, toInputValue } from '@/lib/money'
import { confirmDialog, haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'
import { useLedgerStore } from '@/stores/ledger'
import { useToastStore } from '@/stores/toast'

const HISTORY_PAGE = 20

const route = useRoute()
const { t } = useI18n()
const { money, lang, today } = useFormat()
const goBack = useBack()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const plans = useInstallmentsStore()
const ledger = useLedgerStore()
const toast = useToastStore()

const id = Number(route.params.id)
const plan = ref<Installment | null>(null)
// The plan's transactions (purchase and payments), paged with the cursor.
const payments = ref<Transaction[]>([])
const paymentsCursor = ref<string | null>(null)
const paymentsLoading = ref(false)
const loading = ref(true)
const failed = ref(false)
const busy = ref(false)

const payAmount = ref('')
const payAccountId = ref<string | null>(null)
const payDate = ref(today())
const payErrors = reactive<{ amount?: string; account?: string }>({})

async function loadPayments(more = false) {
  if (more && !paymentsCursor.value) return
  paymentsLoading.value = true
  try {
    const page = await api.listTransactions({
      installment_id: id,
      cursor: more ? (paymentsCursor.value ?? undefined) : undefined,
      limit: HISTORY_PAGE,
    })
    payments.value = more ? [...payments.value, ...page.transactions] : page.transactions
    paymentsCursor.value = page.next_cursor
  } finally {
    paymentsLoading.value = false
  }
}

async function load() {
  failed.value = false
  try {
    const [p] = await Promise.all([api.getInstallment(id), accounts.load(), categories.load()])
    plan.value = p
    plans.upsert(p)
    payAmount.value = p.next ? toInputValue(p.next.amount_minor, p.currency) : ''
    if (!accountOptions.value.some((o) => o.value === payAccountId.value)) {
      const preferred = p.pay_from_account_id ? String(p.pay_from_account_id) : null
      payAccountId.value = accountOptions.value.find((o) => o.value === preferred)?.value ?? accountOptions.value[0]?.value ?? null
    }
    void loadPayments().catch(() => {})
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch(() => ledger.revision, load)

const category = computed(() => (plan.value ? categories.byId.get(plan.value.category_id) : undefined))
const accountOptions = computed(() =>
  accounts.active
    .filter((a) => a.currency === plan.value?.currency)
    .map((a) => ({ value: String(a.id), label: a.name, icon: ACCOUNT_ICONS[a.type], hint: money(a.balance_minor, a.currency) })),
)
const heroBg = computed(() => {
  const c = categoryStyle(category.value).color
  return `linear-gradient(135deg, ${c} 0%, ${darken(c, 0.35)} 100%)`
})

const overdue = computed(() => !!plan.value?.next && plan.value.next.due_on < today())

async function pay() {
  const p = plan.value
  if (!p?.next || busy.value) return
  delete payErrors.amount
  delete payErrors.account
  const amountMinor = parseAmount(payAmount.value, p.currency)
  if (!amountMinor) payErrors.amount = t('form.err_amount')
  else if (amountMinor > p.remaining_minor) payErrors.amount = t('inst.err_overpayment')
  if (!payAccountId.value) payErrors.account = t('debt_form.choose_account')
  if (payErrors.amount || payErrors.account || !amountMinor || !payAccountId.value) {
    haptic.error()
    return
  }
  busy.value = true
  try {
    const tx = await api.payInstallment(id, {
      // 0 = "the rest of the next payment", so an untouched amount always matches the server.
      amount_minor: amountMinor === p.next.amount_minor ? 0 : amountMinor,
      account_id: Number(payAccountId.value),
      occurred_on: payDate.value || today(),
    })
    haptic.success()
    ledger.offerUndo(t('inst.paid_ok'), tx.id)
    ledger.changed()
  } catch (e) {
    haptic.error()
    const { field, code, message } = invalidField(e)
    if (code === 'overpayment') payErrors.amount = t('inst.err_overpayment')
    else if (field === 'amount_minor') payErrors.amount = message
    else if (field === 'account_id') payErrors.account = message
    else toast.error()
  } finally {
    busy.value = false
  }
}

async function removePlan() {
  if (busy.value || !(await confirmDialog(t('inst.delete_confirm')))) return
  busy.value = true
  try {
    await api.deleteInstallment(id)
    plans.drop(id)
    ledger.changed()
    haptic.success()
    toast.show(t('common.deleted'))
    goBack()
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}

async function removePayment(tx: Transaction) {
  try {
    await ledger.remove(tx.id)
    haptic.impact('medium')
  } catch {
    toast.error()
  }
}

const canPay = computed(() => !!plan.value?.next)
const { native } = useMainButton(
  computed(() => t('inst.pay')),
  busy,
  () => void pay(),
  canPay,
)
</script>

<template>
  <PageHeader :title="plan?.title ?? $t('inst.title')" back>
    <button
      v-if="plan"
      type="button"
      class="pressable rounded-full p-2 text-danger"
      :aria-label="$t('inst.delete')"
      :disabled="busy"
      @click="removePlan"
    >
      <AppIcon name="trash" :size="22" />
    </button>
  </PageHeader>

  <SkeletonList v-if="loading" hero />
  <EmptyState v-else-if="failed || !plan" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

  <div v-else class="space-y-4 px-4 pb-6">
    <section class="detail-hero relative overflow-hidden rounded-[26px] p-5 text-white" :style="{ background: heroBg }">
      <span class="absolute -top-12 -right-10 size-40 rounded-full bg-white/10" aria-hidden="true" />
      <div class="truncate text-sm text-white/75">
        {{ category?.icon }} {{ categoryLabel(category) }}<template v-if="plan.provider"> · {{ plan.provider }}</template>
      </div>
      <div class="mt-1 text-sm text-white/75">{{ $t('inst.remaining') }}</div>
      <div class="tabular display text-[34px] leading-tight">{{ money(plan.remaining_minor, plan.currency) }}</div>
      <div class="tabular mb-3 text-sm text-white/75">
        <template v-if="plan.closed"><AppIcon name="check" :size="14" class="inline text-mint" /> {{ $t('installments.paid_off') }} · </template>
        {{ $t('installments.paid_of', { paid: money(plan.paid_minor, plan.currency), total: money(plan.total_minor, plan.currency) }) }}
      </div>
      <ProgressBar :value="plan.paid_minor" :total="plan.total_minor" :danger="overdue" on-dark />
    </section>

    <section v-if="plan.next" class="card space-y-4 p-4">
      <div>
        <h2 class="font-semibold">{{ $t('inst.pay_next') }}</h2>
        <span class="block text-sm" :class="overdue ? 'font-medium text-danger' : 'text-hint'">
          <template v-if="overdue">{{ $t('installments.overdue') }} · </template>{{ formatDay(plan.next.due_on, lang) }}
        </span>
      </div>
      <div>
        <AmountInput
          v-model="payAmount"
          :label="$t('inst.amount')"
          :unit="currencyLabel(plan.currency, lang)"
          :prefix="plan.currency === 'USD'"
          :error="payErrors.amount"
          @update:model-value="delete payErrors.amount"
        />
        <p class="mt-1 text-center text-xs text-hint">{{ $t('inst.amount_hint') }}</p>
      </div>
      <PickerField
        v-model="payAccountId"
        :label="$t('debt.account')"
        :options="accountOptions"
        :placeholder="$t('debt_form.choose_account')"
        :error="payErrors.account"
      />
      <label class="block">
        <span class="form-label">{{ $t('debt_form.date') }}</span>
        <input v-model="payDate" type="date" :max="today()" class="form-input bg-page py-2.5" />
      </label>
    </section>

    <section>
      <h2 class="eyebrow px-1 pb-2">{{ $t('inst.schedule') }}</h2>
      <ol class="card overflow-hidden py-1">
        <li
          v-for="row in plan.schedule"
          :key="row.n"
          class="flex items-center gap-3 px-4 py-2.5"
          :class="{ 'bg-accent/10': row.n === plan.next?.n }"
        >
          <span
            class="flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
            :class="row.paid ? 'bg-income text-white' : !row.paid && row.due_on < today() ? 'bg-danger text-white' : 'bg-page text-hint'"
          >
            <AppIcon v-if="row.paid" name="check" :size="14" />
            <template v-else>{{ row.n }}</template>
          </span>
          <span class="flex-1 text-sm" :class="{ 'text-hint': row.paid, 'font-medium text-danger': !row.paid && row.due_on < today() }">
            {{ formatDay(row.due_on, lang, row.due_on.slice(0, 4) !== today().slice(0, 4)) }}
            <span v-if="row.paid" class="sr-only">· {{ $t('inst.paid') }}</span>
          </span>
          <span class="tabular text-sm font-semibold whitespace-nowrap" :class="{ 'text-hint line-through': row.paid }">
            {{ money(row.amount_minor, plan.currency) }}
          </span>
        </li>
      </ol>
      <p v-if="plan.next && plan.schedule.find((r) => r.n === plan!.next!.n)?.amount_minor !== plan.next.amount_minor" class="mt-1 px-1 text-xs text-hint">
        {{ $t('inst.remaining') }} #{{ plan.next.n }}: {{ money(plan.next.amount_minor, plan.currency) }}
      </p>
    </section>

    <section v-if="payments.length">
      <h2 class="eyebrow px-1 pb-2">{{ $t('debt.history') }}</h2>
      <div class="card overflow-hidden py-1">
        <div v-for="tx in payments" :key="tx.id" class="flex items-center">
          <TxRow :tx="tx" dated class="min-w-0 flex-1" @open="() => {}" />
          <button
            v-if="tx.type === 'installment_payment'"
            type="button"
            class="pressable mr-2 rounded-full p-2 text-hint"
            :aria-label="$t('common.delete')"
            @click="removePayment(tx)"
          >
            <AppIcon name="trash" :size="18" />
          </button>
        </div>
        <button
          v-if="paymentsCursor"
          type="button"
          class="w-full border-t border-separator py-3 text-sm text-link disabled:opacity-60"
          :disabled="paymentsLoading"
          @click="loadPayments(true).catch(() => toast.error())"
        >
          {{ paymentsLoading ? $t('common.loading') : $t('common.load_more') }}
        </button>
      </div>
    </section>
  </div>

  <FallbackButton v-if="!native && canPay" :label="$t('inst.pay')" :busy="busy" @click="pay" />
</template>
