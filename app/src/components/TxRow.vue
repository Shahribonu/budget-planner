<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Transaction } from '@/api/types'
import { categoryLabel } from '@/i18n'
import { useFormat } from '@/composables/useFormat'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import AppIcon, { type IconName } from './AppIcon.vue'

const props = defineProps<{ tx: Transaction }>()
defineEmits<{ open: [id: number] }>()

const { t } = useI18n()
const { money } = useFormat()
const accounts = useAccountsStore()
const categories = useCategoriesStore()

const category = computed(() => (props.tx.category_id ? categories.byId.get(props.tx.category_id) : undefined))
const accountName = (id?: number | null) => (id ? (accounts.byId.get(id)?.name ?? '') : '')

const DIRECTION: Record<Transaction['type'], IconName> = {
  expense: 'out',
  income: 'in',
  transfer: 'swap',
  exchange: 'swap',
  opening: 'flag',
}

const isMove = computed(() => props.tx.type === 'transfer' || props.tx.type === 'exchange')

const title = computed(() => {
  if (isMove.value) return `${accountName(props.tx.account_id)} → ${accountName(props.tx.to_account_id)}`
  return category.value ? categoryLabel(category.value) : t(`types.${props.tx.type}`)
})

const subtitle = computed(() => {
  const parts = isMove.value ? [t(`types.${props.tx.type}`)] : [accountName(props.tx.account_id)]
  if (props.tx.description) parts.unshift(props.tx.description)
  return parts.filter(Boolean).join(' · ')
})

const amount = computed(() => {
  const { type, amount_minor, currency } = props.tx
  if (type === 'expense' || type === 'exchange') return money(-amount_minor, currency)
  if (type === 'transfer') return money(amount_minor, currency)
  return money(amount_minor, currency, { signed: true })
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
    class="flex w-full items-center gap-3 px-4 py-2.5 text-left active:bg-page/60"
    @click="$emit('open', tx.id)"
  >
    <span class="relative flex size-10 shrink-0 items-center justify-center rounded-full bg-page text-xl">
      <span v-if="category" aria-hidden="true">{{ category.icon }}</span>
      <AppIcon v-else :name="DIRECTION[tx.type]" :size="20" class="text-hint" />
      <span
        v-if="category"
        class="absolute -right-0.5 -bottom-0.5 flex size-4.5 items-center justify-center rounded-full border-2 border-bg"
        :class="tx.type === 'income' ? 'bg-income text-white' : 'bg-hint text-white'"
      >
        <AppIcon :name="DIRECTION[tx.type]" :size="10" />
      </span>
    </span>
    <span class="min-w-0 flex-1">
      <span class="block truncate font-medium">{{ title }}</span>
      <span v-if="subtitle" class="block truncate text-sm text-hint">{{ subtitle }}</span>
    </span>
    <span class="shrink-0 text-right">
      <span
        class="tabular block font-semibold whitespace-nowrap"
        :class="tx.type === 'income' || (tx.type === 'opening' && tx.amount_minor > 0) ? 'text-income' : ''"
      >
        <span class="sr-only">{{ $t(`types.${tx.type}`) }}:</span>
        {{ amount }}
      </span>
      <span v-if="toAmount" class="tabular block text-sm whitespace-nowrap text-income">{{ toAmount }}</span>
    </span>
  </button>
</template>
