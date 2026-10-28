<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import type { Debt, DebtPatch, Transaction } from '@/api/types'
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
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { formatDay } from '@/lib/dates'
import { currencyLabel, parseAmount, toInputValue } from '@/lib/money'
import { confirmDialog, haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useDebtsStore } from '@/stores/debts'
import { useLedgerStore } from '@/stores/ledger'
import { useToastStore } from '@/stores/toast'

const route = useRoute()
const { t } = useI18n()
const { money, lang, today } = useFormat()
const goBack = useBack()
const accounts = useAccountsStore()
const debts = useDebtsStore()
const ledger = useLedgerStore()
const toast = useToastStore()

const id = Number(route.params.id)
const debt = ref<Debt | null>(null)
const history = ref<Transaction[]>([])
const loading = ref(true)
const failed = ref(false)
const busy = ref(false)

const payAmount = ref('')
const payAccountId = ref<string | null>(null)
const payDate = ref(today())
const payErrors = reactive<{ amount?: string; account?: string }>({})
const dueOn = ref('')
const note = ref('')

async function load() {
  failed.value = false
  try {
    const [detail] = await Promise.all([api.getDebt(id), accounts.load()])
    apply(detail.debt)
    history.value = detail.transactions
    payAmount.value = detail.debt.remaining_minor ? toInputValue(detail.debt.remaining_minor, detail.debt.currency) : ''
    if (!accountOptions.value.some((o) => o.value === payAccountId.value)) {
      payAccountId.value = accountOptions.value[0]?.value ?? null
    }
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
// Undoing a repayment from the toast changes the debt.
watch(() => ledger.revision, load)

function apply(d: Debt) {
  debt.value = d
  dueOn.value = d.due_on ?? ''
  note.value = d.note
  debts.upsert(d)
}

const accountOptions = computed(() =>
  accounts.active
    .filter((a) => a.currency === debt.value?.currency)
    .map((a) => ({ value: String(a.id), label: a.name, icon: ACCOUNT_ICONS[a.type], hint: money(a.balance_minor, a.currency) })),
)

/** "Lent on 1 Sep from Naqd pul"; a debt from before tracking has no account. */
const origin = computed(() => {
  const d = debt.value
  if (!d) return ''
  const date = formatDay(d.occurred_on, lang.value, d.occurred_on.slice(0, 4) !== today().slice(0, 4))
  const account = d.account_id ? accounts.byId.get(d.account_id)?.name : undefined
  if (!account) {
    return `${t(d.direction === 'lent' ? 'debt.lent_on' : 'debt.borrowed_on', { date })} · ${t('debt.untracked')}`
  }
  return t(d.direction === 'lent' ? 'debt.lent_on_from' : 'debt.borrowed_on_to', { date, account })
})

/** Lent money in violet, borrowed in coral: the direction at a glance (also spelled out in text). */
const heroBg = computed(() =>
  debt.value?.direction === 'borrowed'
    ? 'linear-gradient(135deg, #ff6b5a 0%, #c9372c 100%)'
    : 'linear-gradient(135deg, #5b3df5 0%, #3a1fc9 100%)',
)

const overdue = computed(() => !!debt.value && !debt.value.closed && !!debt.value.due_on && debt.value.due_on < today())

async function patch(p: DebtPatch, revert: () => void) {
  try {
    apply(await api.patchDebt(id, p))
    haptic.success()
  } catch (e) {
    revert()
    haptic.error()
    if (invalidField(e).code === 'invalid_due_date') toast.show(t('debt_form.err_due'))
    else toast.error()
  }
}

function saveDue(value: string) {
  const previous = dueOn.value
  dueOn.value = value
  if ((debt.value?.due_on ?? '') === value) return
  if (value && debt.value && value < debt.value.occurred_on) {
    dueOn.value = previous
    haptic.error()
    toast.show(t('debt_form.err_due'))
    return
  }
  void patch({ due_on: value || null }, () => (dueOn.value = previous))
}

function saveNote() {
  const value = note.value.trim()
  if (!debt.value || value === debt.value.note) return
  const previous = debt.value.note
  void patch({ note: value }, () => (note.value = previous))
}

async function repay() {
  if (!debt.value || busy.value) return
  delete payErrors.amount
  delete payErrors.account
  const amountMinor = parseAmount(payAmount.value, debt.value.currency)
  if (!amountMinor) payErrors.amount = t('form.err_amount')
  else if (amountMinor > debt.value.remaining_minor) payErrors.amount = t('debt.err_overpayment')
  if (!payAccountId.value) payErrors.account = t('debt_form.choose_account')
  if (payErrors.amount || payErrors.account || !amountMinor || !payAccountId.value) {
    haptic.error()
    return
  }
  busy.value = true
  try {
    const tx = await api.payDebt(id, {
      amount_minor: amountMinor,
      account_id: Number(payAccountId.value),
      occurred_on: payDate.value || today(),
    })
    haptic.success()
    ledger.offerUndo(t('debt.repaid'), tx.id)
    ledger.changed() // reloads this view via revision
  } catch (e) {
    haptic.error()
    const { field, code, message } = invalidField(e)
    if (code === 'overpayment') payErrors.amount = t('debt.err_overpayment')
    else if (field === 'amount_minor') payErrors.amount = message
    else if (field === 'account_id') payErrors.account = message
    else toast.error()
  } finally {
    busy.value = false
  }
}

async function removeDebt() {
  if (busy.value || !(await confirmDialog(t('debt.delete_confirm')))) return
  busy.value = true
  try {
    await api.deleteDebt(id)
    debts.drop(id)
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
    await ledger.remove(tx.id) // shows Undo; revision bump reloads
    haptic.impact('medium')
  } catch {
    toast.error()
  }
}

const canRepay = computed(() => !!debt.value && !debt.value.closed)
const { native } = useMainButton(
  computed(() => t('debt.repay')),
  busy,
  () => void repay(),
  canRepay,
)
</script>

<template>
  <PageHeader :title="debt?.counterparty ?? $t('debt.title')" back>
    <button
      v-if="debt"
      type="button"
      class="pressable rounded-full p-2 text-danger"
      :aria-label="$t('debt.delete')"
      :disabled="busy"
      @click="removeDebt"
    >
      <AppIcon name="trash" :size="22" />
    </button>
  </PageHeader>

  <SkeletonList v-if="loading" hero />
  <EmptyState v-else-if="failed || !debt" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

  <div v-else class="space-y-4 px-4 pb-6">
    <section class="detail-hero relative overflow-hidden rounded-[26px] p-5 text-white" :style="{ background: heroBg }">
      <span class="absolute -top-12 -right-10 size-40 rounded-full bg-white/10" aria-hidden="true" />
      <div class="flex items-center gap-1 text-sm text-white/75">
        <AppIcon :name="debt.direction === 'lent' ? 'in' : 'out'" :size="14" />
        {{ debt.direction === 'lent' ? $t('debts.they_owe') : $t('debts.i_owe') }} · {{ $t('debt.remaining') }}
      </div>
      <div class="tabular mt-1 display text-[34px] leading-tight">{{ money(debt.remaining_minor, debt.currency) }}</div>
      <div class="tabular mb-3 text-sm text-white/75">
        <template v-if="debt.closed"><AppIcon name="check" :size="14" class="inline text-mint" /> {{ $t('debts.paid_off') }} · </template>
        {{ $t('debt.principal') }}: {{ money(debt.principal_minor, debt.currency) }}
      </div>
      <ProgressBar :value="debt.principal_minor - debt.remaining_minor" :total="debt.principal_minor" :danger="overdue" on-dark />
      <p class="mt-3 text-sm text-white/75">{{ origin }}</p>
    </section>

    <section class="card divide-y divide-separator">
      <label class="flex items-center gap-3 px-4 py-3">
        <span class="flex-1 text-sm" :class="overdue ? 'font-medium text-danger' : ''">{{ $t('debt.due_date') }}</span>
        <input
          type="date"
          :value="dueOn"
          :min="debt.occurred_on"
          class="rounded-lg bg-page px-2 py-1.5 text-sm outline-none"
          :class="overdue ? 'text-danger' : ''"
          @change="saveDue(($event.target as HTMLInputElement).value)"
        />
        <button v-if="dueOn" type="button" class="text-sm text-link" @click="saveDue('')">{{ $t('debt.clear_due') }}</button>
      </label>
      <label class="block px-4 py-3">
        <span class="mb-1 block text-sm">{{ $t('debt.note') }}</span>
        <input
          v-model="note"
          type="text"
          maxlength="200"
          :placeholder="$t('debt.note_placeholder')"
          class="w-full bg-transparent outline-none placeholder:text-hint"
          @change="saveNote"
        />
      </label>
    </section>

    <section v-if="canRepay" class="card space-y-4 p-4">
      <h2 class="font-semibold">{{ $t('debt.repay') }}</h2>
      <AmountInput
        v-model="payAmount"
        :label="$t('inst.amount')"
        :unit="currencyLabel(debt.currency, lang)"
        :prefix="debt.currency === 'USD'"
        :error="payErrors.amount"
        @update:model-value="delete payErrors.amount"
      />
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

    <section v-if="history.length">
      <h2 class="eyebrow px-1 pb-2">{{ $t('debt.history') }}</h2>
      <div class="card overflow-hidden py-1">
        <div v-for="tx in history" :key="tx.id" class="flex items-center">
          <TxRow :tx="tx" dated class="min-w-0 flex-1" @open="() => {}" />
          <button
            v-if="tx.type === 'debt_payment'"
            type="button"
            class="pressable mr-2 rounded-full p-2 text-hint"
            :aria-label="$t('common.delete')"
            @click="removePayment(tx)"
          >
            <AppIcon name="trash" :size="18" />
          </button>
        </div>
      </div>
    </section>
  </div>

  <FallbackButton v-if="!native && canRepay" :label="$t('debt.repay')" :busy="busy" @click="repay" />
</template>
