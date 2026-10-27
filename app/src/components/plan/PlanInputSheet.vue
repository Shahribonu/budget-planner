<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlanPatch } from '@/api/types'
import { useFormat } from '@/composables/useFormat'
import { currencyLabel, parseAmount, toInputValue } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useMonthlyPlanStore } from '@/stores/monthlyPlan'
import { useToastStore } from '@/stores/toast'
import AmountInput from '../AmountInput.vue'
import BottomSheet from '../BottomSheet.vue'

export type PlanInputField = 'income' | 'savings' | 'buffer'

/** Edit one plan input. Income and everyday spending can go back to the server's estimate. */
const props = defineProps<{ field: PlanInputField }>()
const open = defineModel<boolean>('open', { required: true })

const { t } = useI18n()
const { lang } = useFormat()
const plan = useMonthlyPlanStore()
const toast = useToastStore()

const amount = ref('')
const salaryDay = ref<number | ''>('')
const error = ref('')
const busy = ref(false)

const currency = computed(() => plan.view?.currency ?? 'UZS')
const title = computed(
  () => ({ income: t('mplan.edit_income'), savings: t('mplan.edit_savings'), buffer: t('mplan.edit_everyday') })[props.field],
)
const canReset = computed(() => {
  const i = plan.view?.inputs
  return props.field === 'income' ? !!i?.income_override : props.field === 'buffer' ? !!i?.buffer_override : false
})

watch(open, (isOpen) => {
  const i = plan.view?.inputs
  if (!isOpen || !i) return
  error.value = ''
  const value = { income: i.income_minor, savings: i.savings_minor, buffer: i.buffer_minor }[props.field]
  amount.value = value ? toInputValue(value, currency.value) : ''
  salaryDay.value = i.income_day || ''
})

async function submit(patch: PlanPatch) {
  if (busy.value) return
  busy.value = true
  try {
    await plan.patch(patch)
    haptic.success()
    open.value = false
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}

function save() {
  const minor = amount.value.trim() ? parseAmount(amount.value, currency.value) : 0
  const day = salaryDay.value === '' ? null : Number(salaryDay.value)
  if (minor === null) error.value = t('form.err_amount')
  else if (day !== null && (!Number.isInteger(day) || day < 1 || day > 31)) error.value = t('mplan.salary_day')
  if (error.value || minor === null) {
    haptic.error()
    return
  }
  if (props.field === 'income') void submit({ income_override: minor, income_day: day })
  else if (props.field === 'savings') void submit({ savings_minor: minor })
  else void submit({ buffer_override: minor })
}

function reset() {
  void submit(props.field === 'income' ? { income_override: null } : { buffer_override: null })
}
</script>

<template>
  <BottomSheet v-model:open="open" :title="title">
    <form class="space-y-4 px-4 pt-2 pb-4" novalidate @submit.prevent="save">
      <AmountInput
        v-model="amount"
        :label="title"
        :unit="currencyLabel(currency, lang)"
        :prefix="currency === 'USD'"
        :error="error"
        data-autofocus
        @update:model-value="error = ''"
      />
      <label v-if="field === 'income'" class="flex items-center justify-between gap-3">
        <span class="text-sm">{{ $t('mplan.salary_day') }}</span>
        <input
          v-model.number="salaryDay"
          type="number"
          inputmode="numeric"
          min="1"
          max="31"
          class="tabular w-20 rounded-xl bg-page px-3 py-2 text-center outline-none"
        />
      </label>
      <button
        type="submit"
        class="pressable w-full rounded-xl bg-accent py-3.5 font-semibold text-on-accent disabled:opacity-60"
        :disabled="busy"
      >
        {{ busy ? $t('common.loading') : $t('common.save') }}
      </button>
      <button v-if="canReset" type="button" class="w-full py-2 text-sm text-link" :disabled="busy" @click="reset">
        {{ $t('mplan.reset') }}
      </button>
    </form>
  </BottomSheet>
</template>
