<script setup lang="ts">
import { computed } from 'vue'
import type { PlanItem } from '@/api/types'
import { usePlanFormat } from '@/composables/usePlanFormat'
import { useFormat } from '@/composables/useFormat'
import { formatDay } from '@/lib/dates'
import { useCategoriesStore } from '@/stores/categories'
import AppIcon from '../AppIcon.vue'
import PriorityChip from './PriorityChip.vue'

const props = defineProps<{ item: PlanItem }>()
defineEmits<{ open: [item: PlanItem]; cycle: [item: PlanItem] }>()

const { lang } = useFormat()
const { amountText, reason } = usePlanFormat()
const categories = useCategoriesStore()

const category = computed(() => (props.item.category_id ? categories.byId.get(props.item.category_id) : undefined))
const why = computed(() => reason(props.item))
const done = computed(() => props.item.status !== 'planned')
</script>

<template>
  <div
    role="button"
    tabindex="0"
    class="block w-full px-4 py-2.5 text-left active:bg-page/60"
    @click="$emit('open', item)"
    @keydown.enter.prevent="$emit('open', item)"
  >
    <span class="flex items-start gap-3">
      <span class="mt-0.5 w-6 shrink-0 text-center text-lg" aria-hidden="true">{{ category?.icon ?? '🛍️' }}</span>
      <span class="min-w-0 flex-1">
        <span class="flex items-baseline gap-2">
          <span class="min-w-0 flex-1 truncate font-medium" :class="{ 'text-hint line-through': item.status === 'dropped' }">
            {{ item.title }}
          </span>
          <span class="tabular shrink-0 text-sm font-semibold whitespace-nowrap" :class="{ 'text-hint': done }">
            {{ amountText(item) }}
          </span>
        </span>
        <span class="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-hint">
          <PriorityChip v-if="!done" :priority="item.priority" @cycle="$emit('cycle', item)" />
          <span v-else class="rounded-full bg-page px-2 py-0.5 font-medium">{{ $t(`mplan.st_${item.status}`) }}</span>
          <span v-if="item.due_on" class="flex items-center gap-0.5">
            <AppIcon name="calendar" :size="12" />{{ formatDay(item.due_on, lang) }}
          </span>
          <span v-if="item.postponed_count > 0" class="rounded-full bg-page px-1.5 py-0.5">
            {{ $t('mplan.postponed_n', { n: item.postponed_count }) }}
          </span>
        </span>
        <span
          v-if="why"
          class="mt-1 flex items-start gap-1 text-xs"
          :class="{ 'text-income': why.tone === 'ok', 'text-hint': why.tone === 'neutral', 'font-medium text-danger': why.tone === 'danger' }"
        >
          <AppIcon v-if="why.tone === 'danger'" name="alert" :size="13" class="mt-px shrink-0" />
          <AppIcon v-else-if="why.tone === 'ok'" name="check" :size="13" class="mt-px shrink-0" />
          <span>{{ why.text }}</span>
        </span>
      </span>
    </span>
  </div>
</template>
