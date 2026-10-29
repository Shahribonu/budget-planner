<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Installment } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import CategoryIcon from '@/components/CategoryIcon.vue'
import SkeletonList from '@/components/SkeletonList.vue'
import EmptyState from '@/components/EmptyState.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import ProgressBar from '@/components/ProgressBar.vue'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { formatDay } from '@/lib/dates'
import { useCategoriesStore } from '@/stores/categories'
import { useInstallmentsStore } from '@/stores/installments'

const { t } = useI18n()
const router = useRouter()
const plans = useInstallmentsStore()
const categories = useCategoriesStore()
const { money, lang, today } = useFormat()

const loading = ref(!plans.loaded)
const failed = ref(false)
const showClosed = ref(false)

async function load() {
  failed.value = false
  try {
    await Promise.all([plans.load(true), categories.load()])
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const open = computed(() => plans.all.filter((p) => !p.closed))
const closed = computed(() => plans.all.filter((p) => p.closed))

function isOverdue(p: Installment) {
  return !!p.next && p.next.due_on < today()
}

const add = () => void router.push({ name: 'installment-new' })
// The one add action on this screen: Telegram's MainButton (FallbackButton outside Telegram).
const { native } = useMainButton(
  computed(() => t('installments.add')),
  ref(false),
  add,
  computed(() => !loading.value && !failed.value),
)
</script>

<template>
  <PageHeader :title="$t('installments.title')" back />

  <div class="space-y-4 px-4 pb-4">
    <SkeletonList v-if="loading" class="-mx-4" />
    <EmptyState v-else-if="failed" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />
    <EmptyState
      v-else-if="!plans.all.length"
      icon="🛍️"
      :body="$t('installments.empty')"
    />

    <div v-if="open.length" class="card overflow-hidden py-1">
      <RouterLink
        v-for="p in open"
        :key="p.id"
        :to="{ name: 'installment', params: { id: p.id } }"
        class="block px-4 py-3 active:bg-page/60"
      >
        <span class="flex items-start gap-3">
          <CategoryIcon :category="categories.byId.get(p.category_id)" :size="44" />
          <span class="min-w-0 flex-1">
            <span class="block truncate font-medium">{{ p.title }}</span>
            <span class="tabular block truncate text-sm text-hint">
              <template v-if="p.provider">{{ p.provider }} · </template>
              {{ $t('installments.paid_of', { paid: money(p.paid_minor, p.currency), total: money(p.total_minor, p.currency) }) }}
            </span>
            <ProgressBar class="mt-2" :value="p.paid_minor" :total="p.total_minor" :danger="isOverdue(p)" />
            <span
              v-if="p.next"
              class="mt-1.5 flex items-center gap-1 text-sm"
              :class="isOverdue(p) ? 'font-medium text-danger' : 'text-hint'"
            >
              <AppIcon name="calendar" :size="14" class="shrink-0" />
              <span class="min-w-0 flex-1 truncate">
                {{ formatDay(p.next.due_on, lang) }}<template v-if="isOverdue(p)"> · {{ $t('installments.overdue') }}</template>
              </span>
              <span class="tabular shrink-0 font-semibold">{{ money(p.next.amount_minor, p.currency) }}</span>
            </span>
          </span>
        </span>
      </RouterLink>
    </div>

    <section v-if="closed.length">
      <button
        type="button"
        class="flex w-full items-center justify-between px-1 py-1 text-sm font-medium text-hint"
        :aria-expanded="showClosed"
        @click="showClosed = !showClosed"
      >
        {{ $t('installments.closed') }} ({{ closed.length }})
        <AppIcon name="down" :size="16" class="transition-transform" :class="{ 'rotate-180': showClosed }" />
      </button>
      <div v-if="showClosed" class="card mt-1 overflow-hidden py-1">
        <RouterLink
          v-for="p in closed"
          :key="p.id"
          :to="{ name: 'installment', params: { id: p.id } }"
          class="flex items-center gap-3 px-4 py-2.5 opacity-70 active:bg-page/60"
        >
          <AppIcon name="check" :size="18" class="shrink-0 text-income" />
          <span class="min-w-0 flex-1 truncate">{{ p.title }}</span>
          <span class="tabular shrink-0 text-sm text-hint">{{ money(p.total_minor, p.currency) }}</span>
        </RouterLink>
      </div>
    </section>
  </div>

  <FallbackButton v-if="!native && !loading && !failed" :label="$t('installments.add')" @click="add" />
</template>
