<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Transaction } from '@/api/types'
import { categoryLabel } from '@/i18n'
import { useFormat } from '@/composables/useFormat'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'
import { amountSign } from '@/lib/transactions'
import AppIcon, { type IconName } from './AppIcon.vue'
import CategoryIcon from './CategoryIcon.vue'

/** `dated`: lead the subtitle with the day, for lists not grouped by date (debt and plan history). */
const props = defineProps<{ tx: Transaction; dated?: boolean }>()
defineEmits<{ open: [tx: Transaction] }>()

const { t } = useI18n()
const { money, dayLabel } = useFormat()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const plans = useInstallmentsStore()

const category = computed(() => (props.tx.category_id ? categories.byId.get(props.tx.category_id) : undefined))
const accountName = (id?: number | null) => (id ? (accounts.byId.get(id)?.name ?? '') : '')

const ICON: Record<Transaction['type'], IconName> = {
  expense: 'out',
  income: 'in',
  transfer: 'swap',
  exchange: 'swap',
  opening: 'flag',
  debt: 'person',
  debt_payment: 'person',
  installment: 'out',
  installment_payment: 'calendar',
}

const isMove = computed(() => props.tx.type === 'transfer' || props.tx.type === 'exchange')
const plan = computed(() => (props.tx.installment_id ? plans.byId.get(props.tx.installment_id) : undefined))
const sign = computed(() => amountSign(props.tx))

/** Debt rows are worded by direction: "Lent to Aziz", "Aziz paid back". */
const debtTitle = computed(() => {
  const { debt_direction: direction, counterparty: name } = props.tx
  if (!direction || !name) return t(`types.${props.tx.type}`)
  const paid = props.tx.type === 'debt_payment' ? '_paid' : ''
  return t(`debt_text.${direction}${paid}`, { name })
})

const title = computed(() => {
  const { type } = props.tx
  if (isMove.value) return `${accountName(props.tx.account_id)} → ${accountName(props.tx.to_account_id)}`
  if (type === 'debt' || type === 'debt_payment') return debtTitle.value
  if (type === 'installment_payment') return plan.value?.title || props.tx.description || t('types.installment_payment')
  return category.value ? categoryLabel(category.value) : t(`types.${type}`)
})

const subtitle = computed(() => {
  const text = baseSubtitle.value
  return props.dated ? [dayLabel(props.tx.occurred_on), text].filter(Boolean).join(' · ') : text
})

const baseSubtitle = computed(() => {
  const { type, description } = props.tx
  const account = accountName(props.tx.account_id)
  if (isMove.value) return [description, t(`types.${type}`)].filter(Boolean).join(' · ')
  if (type === 'debt' || type === 'debt_payment' || type === 'installment_payment') {
    // The description is already the title here; the type label stands in for a missing account.
    return [t(`types.${type}`), account].filter(Boolean).join(' · ')
  }
  if (type === 'installment') return [description, t('types.installment'), account].filter(Boolean).join(' · ')
  return [description, account].filter(Boolean).join(' · ')
})

const amount = computed(() => {
  const { amount_minor, currency } = props.tx
  if (sign.value === 'out') return money(-Math.abs(amount_minor), currency)
  if (sign.value === 'in') return money(Math.abs(amount_minor), currency, { signed: true })
  return money(amount_minor, currency)
})

const toAmount = computed(() =>
  props.tx.type === 'exchange' && props.tx.to_amount_minor && props.tx.to_currency
    ? money(props.tx.to_amount_minor, props.tx.to_currency, { signed: true })
    : '',
)
</script>

<template>
  <button
    type="button"
    class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors active:bg-page/60"
    @click="$emit('open', tx)"
  >
    <span class="relative shrink-0">
      <CategoryIcon v-if="category" :category="category" :size="44" />
      <span v-else class="flex size-11 items-center justify-center rounded-full bg-accent-soft text-accent">
        <AppIcon :name="ICON[tx.type]" :size="20" />
      </span>
      <!-- Direction badge: in/out is never shown by colour alone -->
      <span
        v-if="category"
        class="absolute -right-0.5 -bottom-0.5 flex size-4.5 items-center justify-center rounded-full border-2 border-surface"
        :class="sign === 'in' ? 'bg-income text-white' : 'bg-hint text-white'"
      >
        <AppIcon :name="ICON[tx.type]" :size="10" />
      </span>
    </span>
    <span class="min-w-0 flex-1">
      <span class="block truncate text-[15px] font-semibold">{{ title }}</span>
      <span v-if="subtitle" class="block truncate text-sm text-hint">{{ subtitle }}</span>
    </span>
    <span class="shrink-0 text-right">
      <span
        class="tabular block font-semibold whitespace-nowrap"
        :class="sign === 'in' ? 'text-income' : ''"
      >
        <span class="sr-only">{{ $t(`types.${tx.type}`) }}:</span>
        {{ amount }}
      </span>
      <span v-if="toAmount" class="tabular block text-sm whitespace-nowrap text-income">{{ toAmount }}</span>
    </span>
  </button>
</template>
