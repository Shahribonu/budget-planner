<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'
import type { PlanItem, PlanItemPatch, PlanItemStatus, PlanPriority, Transaction } from '@/api/types'
import { invalidField } from '@/composables/useApiError'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { formatDay, monthRange } from '@/lib/dates'
import { PRIORITIES, parseRange } from '@/lib/plan'
import { toInputValue } from '@/lib/money'
import { confirmDialog, haptic } from '@/lib/telegram'
import { useCategoriesStore } from '@/stores/categories'
import { useMonthlyPlanStore } from '@/stores/monthlyPlan'
import { useToastStore } from '@/stores/toast'
import AppIcon from '../AppIcon.vue'
import BottomSheet from '../BottomSheet.vue'
import OptionSheet, { type SheetOption } from '../OptionSheet.vue'
import PickerField from '../PickerField.vue'
import SegmentedControl from '../SegmentedControl.vue'

/** Edit an item and change its status; with `item` null, add one item by hand. */
const props = defineProps<{ item: PlanItem | null }>()
const open = defineModel<boolean>('open', { required: true })

const { t } = useI18n()
const { money, lang } = useFormat()
const plan = useMonthlyPlanStore()
const categories = useCategoriesStore()
const toast = useToastStore()

const title = ref('')
const amount = ref('')
const priority = ref<PlanPriority>('should')
const categoryId = ref<string | null>(null)
const dueOn = ref('')
const note = ref('')
const busy = ref(false)
const titleError = ref('')
const amountError = ref('')

const currency = computed(() => plan.view?.currency ?? 'UZS')
const isDone = computed(() => !!props.item && props.item.status !== 'planned')

watch(open, (isOpen) => {
  if (!isOpen) return
  const i = props.item
  titleError.value = ''
  amountError.value = ''
  title.value = i?.title ?? ''
  priority.value = i?.priority ?? 'should'
  categoryId.value = i?.category_id ? String(i.category_id) : null
  dueOn.value = i?.due_on ?? ''
  note.value = i?.note ?? ''
  amount.value = i ? rangeText(i.amount_min_minor, i.amount_max_minor) : ''
})

function rangeText(min: number, max: number): string {
  const a = toInputValue(min, currency.value)
  return max > min ? `${a}-${toInputValue(max, currency.value)}` : a
}

const categoryOptions = computed(() =>
  categories.ofKind('expense').map((c) => ({ value: String(c.id), label: categoryLabel(c), icon: c.icon })),
)
const priorityOptions = computed(() => PRIORITIES.map((p) => ({ value: p, label: t(`mplan.${p}`) })))

async function run(action: () => Promise<unknown>, done?: string) {
  if (busy.value) return
  busy.value = true
  try {
    await action()
    haptic.success()
    if (done) toast.show(done)
    open.value = false
  } catch (e) {
    haptic.error()
    const { field, message } = invalidField(e)
    if (field === 'title') titleError.value = message
    else if (field?.startsWith('amount')) amountError.value = message
    else toast.error()
  } finally {
    busy.value = false
  }
}

function save() {
  titleError.value = title.value.trim() ? '' : t('mplan.err_title')
  const range = parseRange(amount.value, currency.value)
  amountError.value = range ? '' : t('mplan.err_amount')
  if (titleError.value || !range) {
    haptic.error()
    return
  }
  const fields = {
    title: title.value.trim(),
    category_id: categoryId.value ? Number(categoryId.value) : null,
    amount_min_minor: range.min,
    amount_max_minor: range.max,
    priority: priority.value,
    due_on: dueOn.value || null,
    note: note.value.trim(),
  }
  const item = props.item
  if (!item) return run(() => plan.addItems([fields]), t('common.saved'))
  return run(() => plan.patchItem(item.id, fields satisfies PlanItemPatch), t('common.saved'))
}

function setStatus(status: PlanItemStatus) {
  const item = props.item
  if (item) void run(() => plan.patchItem(item.id, { status }))
}

async function remove() {
  const item = props.item
  if (!item || !(await confirmDialog(t('mplan.delete_confirm')))) return
  void run(() => plan.deleteItem(item.id), t('common.deleted'))
}

// "Bought it": optionally link the purchase from this month's expenses in the item's category.
const pickTx = ref(false)
const txs = ref<Transaction[]>([])
const txChoice = ref<string | null>(null)

async function bought() {
  const v = plan.view
  const item = props.item
  if (!v || !item) return
  try {
    const page = await api.listTransactions({
      type: ['expense', 'installment'],
      currency: v.currency,
      category_id: item.category_id ?? undefined,
      ...monthRange(v.month),
      limit: 30,
    })
    txs.value = page.transactions
  } catch {
    txs.value = []
  }
  if (!txs.value.length) return setStatus('bought')
  // Not null: picking "don't link" (null) must still count as a change.
  txChoice.value = ''
  pickTx.value = true
}

const txOptions = computed<SheetOption<string>[]>(() =>
  txs.value.map((tx) => {
    const cat = tx.category_id ? categories.byId.get(tx.category_id) : undefined
    return {
      value: String(tx.id),
      label: tx.description || categoryLabel(cat),
      icon: cat?.icon,
      hint: `${formatDay(tx.occurred_on, lang.value)} · ${money(tx.amount_minor, tx.currency)}`,
    }
  }),
)

function linkTx(value: string | null) {
  const item = props.item
  if (!item) return
  void run(() => plan.patchItem(item.id, { status: 'bought', transaction_id: value ? Number(value) : null }))
}
</script>

<template>
  <BottomSheet v-model:open="open" :title="item ? item.title : $t('mplan.new_item')">
    <form class="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 pt-2 pb-4" novalidate @submit.prevent="save">
      <template v-if="!isDone">
        <label class="block">
          <span class="form-label">{{ $t('mplan.item_title') }}</span>
          <input
            v-model="title"
            type="text"
            maxlength="64"
            :placeholder="$t('mplan.item_title_placeholder')"
            class="form-input bg-page"
            :aria-invalid="!!titleError"
            :data-autofocus="!item || undefined"
            @input="titleError = ''"
          />
          <span v-if="titleError" class="form-error">{{ titleError }}</span>
        </label>

        <label class="block">
          <span class="form-label">{{ $t('mplan.amount') }}</span>
          <input
            v-model="amount"
            type="text"
            inputmode="decimal"
            autocomplete="off"
            :placeholder="$t('mplan.amount_hint')"
            class="form-input tabular bg-page"
            :aria-invalid="!!amountError"
            @input="amountError = ''"
          />
          <span v-if="amountError" class="form-error">{{ amountError }}</span>
        </label>

        <div>
          <span class="form-label">{{ $t('mplan.priority') }}</span>
          <SegmentedControl v-model="priority" :options="priorityOptions" :label="$t('mplan.priority')" />
        </div>

        <PickerField
          v-model="categoryId"
          :label="$t('mplan.category')"
          :options="categoryOptions"
          :none-label="$t('mplan.no_category')"
          searchable
        />

        <div class="grid grid-cols-2 gap-3">
          <label class="block min-w-0">
            <span class="form-label truncate">{{ $t('mplan.due') }}</span>
            <input v-model="dueOn" type="date" class="form-input bg-page py-2.5" />
          </label>
          <label class="block min-w-0">
            <span class="form-label">{{ $t('mplan.note') }}</span>
            <input v-model="note" type="text" maxlength="200" class="form-input bg-page py-2.5" />
          </label>
        </div>

        <button
          type="submit"
          class="pressable w-full rounded-xl bg-accent py-3.5 font-semibold text-on-accent disabled:opacity-60"
          :disabled="busy"
        >
          {{ busy ? $t('common.loading') : $t('common.save') }}
        </button>
      </template>

      <div v-if="item" class="divide-y divide-separator overflow-hidden rounded-xl bg-page">
        <template v-if="!isDone">
          <button type="button" class="sheet-action text-income" :disabled="busy" @click="bought">
            <AppIcon name="check" :size="18" />{{ $t('mplan.bought') }}
          </button>
          <button type="button" class="sheet-action" :disabled="busy" @click="setStatus('postponed')">
            <AppIcon name="calendar" :size="18" />{{ $t('mplan.postpone') }}
          </button>
          <button type="button" class="sheet-action" :disabled="busy" @click="setStatus('dropped')">
            <AppIcon name="x" :size="18" />{{ $t('mplan.drop') }}
          </button>
        </template>
        <button v-else type="button" class="sheet-action text-link" :disabled="busy" @click="setStatus('planned')">
          <AppIcon name="back" :size="18" />{{ $t('mplan.back_to_plan') }}
        </button>
        <button type="button" class="sheet-action text-danger" :disabled="busy" @click="remove">
          <AppIcon name="trash" :size="18" />{{ $t('mplan.delete') }}
        </button>
      </div>
    </form>
  </BottomSheet>

  <OptionSheet
    v-model:open="pickTx"
    v-model="txChoice"
    :title="$t('mplan.link_tx')"
    :options="txOptions"
    :all-label="$t('mplan.no_link')"
    @update:model-value="linkTx"
  />
</template>

<style scoped>
@reference '../../style.css';

.sheet-action {
  @apply flex min-h-12 w-full items-center gap-3 px-4 py-3 text-left font-medium disabled:opacity-60;
}
</style>
