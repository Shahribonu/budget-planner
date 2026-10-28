<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'
import type { NewPlanItem, PlanPriority } from '@/api/types'
import { invalidField } from '@/composables/useApiError'
import { categoryLabel } from '@/i18n'
import { toInputValue } from '@/lib/money'
import { nextPriority, parseRange } from '@/lib/plan'
import { haptic } from '@/lib/telegram'
import { useCategoriesStore } from '@/stores/categories'
import { useMonthlyPlanStore } from '@/stores/monthlyPlan'
import { useToastStore } from '@/stores/toast'
import AppIcon from '../AppIcon.vue'
import OptionSheet from '../OptionSheet.vue'
import PriorityChip from './PriorityChip.vue'

/** Free text → parsed items the user can fix → POST /plan/items. */
defineEmits<{ manual: [] }>()

const { t } = useI18n()
const plan = useMonthlyPlanStore()
const categories = useCategoriesStore()
const toast = useToastStore()

interface Draft {
  uid: number
  title: string
  amount: string
  priority: PlanPriority
  categoryId: string | null
  dueOn: string | null
  note: string
  error?: string
}

const text = ref('')
const drafts = ref<Draft[]>([])
const parsing = ref(false)
const adding = ref(false)
const nothing = ref(false)
let uid = 0

const currency = computed(() => plan.view?.currency ?? 'UZS')
const placeholder = computed(() => t(currency.value === 'USD' ? 'mplan.input_placeholder_usd' : 'mplan.input_placeholder'))
defineExpose({
  fill(example: string) {
    text.value = example
    void parse()
  },
})

function toDraft(item: NewPlanItem): Draft {
  const min = toInputValue(item.amount_min_minor, currency.value)
  const max = item.amount_max_minor && item.amount_max_minor > item.amount_min_minor ? item.amount_max_minor : 0
  return {
    uid: ++uid,
    title: item.title,
    amount: max ? `${min}-${toInputValue(max, currency.value)}` : min,
    priority: item.priority,
    categoryId: item.category_id ? String(item.category_id) : null,
    dueOn: item.due_on ?? null,
    note: item.note ?? '',
  }
}

async function parse() {
  const value = text.value.trim()
  if (!value || parsing.value) return
  parsing.value = true
  nothing.value = false
  try {
    const items = await api.parsePlan(value, currency.value)
    drafts.value = [...drafts.value, ...items.map(toDraft)]
    nothing.value = items.length === 0
    if (items.length) {
      text.value = ''
      haptic.impact('light')
    }
  } catch {
    toast.error()
  } finally {
    parsing.value = false
  }
}

async function add() {
  if (adding.value || !drafts.value.length) return
  const items: NewPlanItem[] = []
  for (const d of drafts.value) {
    const range = parseRange(d.amount, currency.value)
    d.error = !d.title.trim() ? t('mplan.err_title') : range ? undefined : t('mplan.err_amount')
    if (range && !d.error) {
      items.push({
        title: d.title.trim(),
        amount_min_minor: range.min,
        amount_max_minor: range.max,
        priority: d.priority,
        category_id: d.categoryId ? Number(d.categoryId) : null,
        due_on: d.dueOn,
        note: d.note,
      })
    }
  }
  if (items.length !== drafts.value.length) {
    haptic.error()
    return
  }
  adding.value = true
  try {
    await plan.addItems(items)
    drafts.value = []
    haptic.success()
  } catch (e) {
    haptic.error()
    const { message } = invalidField(e)
    toast.show(message || t('common.error'))
  } finally {
    adding.value = false
  }
}

// One category sheet shared by all draft rows.
const catSheet = ref(false)
const catValue = ref<string | null>(null)
let catTarget: Draft | null = null
const categoryOptions = computed(() =>
  categories.ofKind('expense').map((c) => ({ value: String(c.id), label: categoryLabel(c), icon: c.icon })),
)
function pickCategory(d: Draft) {
  catTarget = d
  catValue.value = d.categoryId
  catSheet.value = true
}
function setCategory(v: string | null) {
  if (catTarget) catTarget.categoryId = v
}
function categoryOf(d: Draft) {
  return d.categoryId ? categories.byId.get(Number(d.categoryId)) : undefined
}
</script>

<template>
  <div class="space-y-3">
    <!-- Parsed items to check before adding -->
    <section v-if="drafts.length" class="card space-y-1 p-3" aria-live="polite">
      <h3 class="px-1 pb-1 text-sm font-semibold">{{ $t('mplan.parsed_title') }}</h3>
      <div v-for="d in drafts" :key="d.uid" class="rounded-xl bg-page p-2">
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="pressable flex size-9 shrink-0 items-center justify-center rounded-lg bg-surface text-lg"
            :aria-label="`${$t('mplan.category')}: ${categoryOf(d) ? categoryLabel(categoryOf(d)) : $t('mplan.no_category')}`"
            @click="pickCategory(d)"
          >
            {{ categoryOf(d)?.icon ?? '🏷️' }}
          </button>
          <input
            v-model="d.title"
            type="text"
            maxlength="64"
            :aria-label="$t('mplan.item_title')"
            class="min-w-0 flex-1 bg-transparent font-medium outline-none"
            @input="d.error = undefined"
          />
          <button
            type="button"
            class="shrink-0 rounded-full p-1.5 text-hint"
            :aria-label="$t('onboarding.remove')"
            @click="drafts = drafts.filter((x) => x !== d)"
          >
            <AppIcon name="x" :size="16" />
          </button>
        </div>
        <div class="mt-1 flex items-center gap-2 pl-11">
          <input
            v-model="d.amount"
            type="text"
            inputmode="decimal"
            :aria-label="$t('mplan.amount')"
            class="tabular min-w-0 flex-1 rounded-lg bg-surface px-2 py-1 text-sm outline-none"
            @input="d.error = undefined"
          />
          <PriorityChip :priority="d.priority" @cycle="d.priority = nextPriority(d.priority)" />
        </div>
        <p v-if="d.error" class="form-error pl-11">{{ d.error }}</p>
      </div>
      <div class="flex gap-2 pt-1">
        <button
          type="button"
          class="pressable flex-1 rounded-xl bg-accent py-2.5 font-semibold text-on-accent disabled:opacity-60"
          :disabled="adding"
          @click="add"
        >
          {{ adding ? $t('common.loading') : $t('mplan.add_n', { n: drafts.length }) }}
        </button>
        <button type="button" class="rounded-xl px-4 py-2.5 text-hint" @click="drafts = []">{{ $t('common.cancel') }}</button>
      </div>
    </section>

    <p v-if="nothing" class="px-1 text-sm text-hint">{{ $t('mplan.nothing_parsed') }}</p>

    <!-- Sticky input above the tab bar while the plan section is on screen -->
    <form
      class="sticky z-10 flex items-center gap-2 card p-2 shadow-lg ring-1 ring-separator"
      :style="{ bottom: 'calc(var(--safe-bottom) + var(--tabbar-h) + 8px)' }"
      @submit.prevent="parse"
    >
      <input
        v-model="text"
        type="text"
        enterkeyhint="send"
        :placeholder="placeholder"
        :aria-label="placeholder"
        class="h-10 min-w-0 flex-1 bg-transparent px-2 text-sm outline-none placeholder:text-hint"
      />
      <button
        type="button"
        class="pressable flex size-10 shrink-0 items-center justify-center rounded-full bg-page text-link"
        :aria-label="$t('mplan.add_one')"
        @click="$emit('manual')"
      >
        <AppIcon name="plus" :size="20" />
      </button>
      <button
        type="submit"
        class="pressable flex size-10 shrink-0 items-center justify-center rounded-full bg-accent text-on-accent disabled:opacity-50"
        :aria-label="$t('mplan.send')"
        :disabled="!text.trim() || parsing"
      >
        <AppIcon name="chevron" :size="20" :class="{ 'animate-pulse': parsing }" />
      </button>
    </form>
  </div>

  <OptionSheet
    v-model:open="catSheet"
    v-model="catValue"
    :title="$t('mplan.category')"
    :options="categoryOptions"
    :all-label="$t('mplan.no_category')"
    searchable
    @update:model-value="setCategory"
  />
</template>
