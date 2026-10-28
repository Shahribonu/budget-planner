<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'
import type { Budget, Currency } from '@/api/types'
import { invalidField } from '@/composables/useApiError'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { CURRENCIES, currencyLabel, parseAmount, toInputValue } from '@/lib/money'
import { confirmDialog, haptic } from '@/lib/telegram'
import { useBudgetsStore } from '@/stores/budgets'
import { useCategoriesStore } from '@/stores/categories'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import AmountInput from './AmountInput.vue'
import AppIcon from './AppIcon.vue'
import BottomSheet from './BottomSheet.vue'
import PickerField from './PickerField.vue'
import SegmentedControl from './SegmentedControl.vue'

/** Add a budget, or edit the limit of / delete an existing one (`budget` set). */
const props = defineProps<{ budget: Budget | null }>()
const open = defineModel<boolean>('open', { required: true })

const OVERALL = 'overall'

const { t } = useI18n()
const { lang } = useFormat()
const user = useUserStore()
const budgets = useBudgetsStore()
const categories = useCategoriesStore()
const toast = useToastStore()

const target = ref<string | null>(OVERALL)
const currency = ref<Currency>(user.mainCurrency)
const amount = ref('')
const busy = ref(false)
const targetError = ref('')
const amountError = ref('')



/** Overall plus expense categories, minus those that already have a budget in this currency. */
const targetOptions = computed(() => {
  const taken = new Set(
    budgets.all.filter((b) => b.currency === currency.value).map((b) => (b.category_id === null ? OVERALL : String(b.category_id))),
  )
  const options = [
    { value: OVERALL, label: t('budgets.overall_option'), icon: '🎯' },
    ...categories.ofKind('expense').map((c) => ({ value: String(c.id), label: categoryLabel(c), icon: c.icon })),
  ]
  return options.filter((o) => !taken.has(o.value))
})

function pickTarget() {
  if (!targetOptions.value.some((o) => o.value === target.value)) target.value = targetOptions.value[0]?.value ?? null
}
watch(currency, () => !props.budget && pickTarget())

const title = computed(() => {
  const b = props.budget
  if (!b) return t('budgets.add')
  return b.category_id === null ? t('budgets.overall') : categoryLabel(categories.byId.get(b.category_id))
})

async function save() {
  if (busy.value) return
  targetError.value = ''
  amountError.value = ''
  const amountMinor = parseAmount(amount.value, currency.value)
  if (!amountMinor) amountError.value = t('form.err_amount')
  if (!props.budget && !target.value) targetError.value = t('form.err_category')
  if (amountError.value || targetError.value || !amountMinor) {
    haptic.error()
    return
  }
  busy.value = true
  try {
    budgets.upsert(
      props.budget
        ? await api.patchBudget(props.budget.id, amountMinor)
        : await api.createBudget({
            category_id: target.value === OVERALL ? null : Number(target.value),
            currency: currency.value,
            amount_minor: amountMinor,
          }),
    )
    haptic.success()
    toast.show(t('common.saved'))
    open.value = false
  } catch (e) {
    haptic.error()
    const { field, code, message } = invalidField(e)
    if (code === 'budget_exists') targetError.value = t('budgets.err_exists', { currency: currency.value })
    else if (field === 'amount_minor') amountError.value = message
    else if (field === 'category_id') targetError.value = message
    else toast.error()
  } finally {
    busy.value = false
  }
}

async function remove() {
  const b = props.budget
  if (!b || busy.value || !(await confirmDialog(t('budgets.delete_confirm')))) return
  busy.value = true
  try {
    await api.deleteBudget(b.id)
    budgets.drop(b.id)
    haptic.success()
    toast.show(t('common.deleted'))
    open.value = false
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}

const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))

// Last, so everything it uses is defined when it runs immediately (sheet mounted already open).
watch(open, (isOpen) => {
  if (!isOpen) return
  targetError.value = ''
  amountError.value = ''
  const b = props.budget
  currency.value = b?.currency ?? user.mainCurrency
  amount.value = b ? toInputValue(b.amount_minor, b.currency) : ''
  target.value = b ? (b.category_id === null ? OVERALL : String(b.category_id)) : null
  if (!b) pickTarget()
}, { immediate: true })
</script>

<template>
  <BottomSheet v-model:open="open" :title="title">
    <form class="min-h-0 flex-1 space-y-5 overflow-y-auto px-4 pt-2 pb-4" novalidate @submit.prevent="save">
      <template v-if="!budget">
        <PickerField
          v-model="target"
          :label="$t('budgets.target')"
          :options="targetOptions"
          :placeholder="$t('form.err_category')"
          :error="targetError"
          searchable
          @update:model-value="targetError = ''"
        />
        <SegmentedControl v-model="currency" :options="currencyOptions" :label="$t('debt_form.currency')" />
      </template>

      <div>
        <span class="form-label">{{ $t('budgets.limit') }}</span>
        <AmountInput
          v-model="amount"
          :label="$t('budgets.limit')"
          :unit="currencyLabel(currency, lang)"
          :prefix="currency === 'USD'"
          :error="amountError"
          data-autofocus
          @update:model-value="amountError = ''"
        />
      </div>

      <button
        type="submit"
        class="pressable w-full rounded-xl bg-accent py-3.5 font-semibold text-on-accent disabled:opacity-60"
        :disabled="busy"
      >
        {{ busy ? $t('common.loading') : $t('common.save') }}
      </button>
      <button
        v-if="budget"
        type="button"
        class="pressable flex w-full items-center justify-center gap-2 rounded-xl py-3 text-danger"
        :disabled="busy"
        @click="remove"
      >
        <AppIcon name="trash" :size="18" />{{ $t('budgets.delete') }}
      </button>
    </form>
  </BottomSheet>
</template>
