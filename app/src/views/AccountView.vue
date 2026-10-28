<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ApiError } from '@/api/client'
import type { AccountPatch, AccountType, Currency } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import { useBack } from '@/composables/useBack'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { ACCOUNT_SWATCHES, accountColor, darken, isCard } from '@/lib/accountStyle'
import { CURRENCIES, currencyLabel, parseAmount } from '@/lib/money'
import { confirmDialog, haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const CARD_TYPES: AccountType[] = ['humo', 'uzcard', 'visa', 'mastercard']
const OTHER_TYPES: AccountType[] = ['savings', 'other']

const route = useRoute()
const { t } = useI18n()
const { money, lang } = useFormat()
const goBack = useBack()
const accounts = useAccountsStore()
const user = useUserStore()
const toast = useToastStore()

const editId = route.name === 'account-edit' ? Number(route.params.id) : null
const account = computed(() => (editId ? accounts.byId.get(editId) : undefined))

const name = ref('')
const type = ref<AccountType>('humo')
const currency = ref<Currency>(user.mainCurrency)
const opening = ref('')
const mask = ref('')
const color = ref<string | null>(null)
const includeInTotal = ref(true)
const showAppearance = ref(false)
const errors = ref<{ name?: string; opening?: string; mask?: string }>({})
const saving = ref(false)
const loading = ref(true)
const failed = ref(false)

async function load() {
  loading.value = true
  failed.value = false
  try {
    await accounts.load()
    const a = account.value
    if (a) {
      name.value = a.name
      type.value = a.type
      currency.value = a.currency
      mask.value = a.card_mask ?? ''
      color.value = a.color
      includeInTotal.value = a.include_in_total
    }
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

// The big card at the top previews colour, name and number as they're edited.
const preview = computed(() => accountColor({ type: type.value, color: color.value }))
const previewBg = computed(() => `linear-gradient(135deg, ${preview.value} 0%, ${darken(preview.value, 0.35)} 100%)`)
const kind = computed<'card' | 'cash' | 'other'>(() => (isCard(type.value) ? 'card' : type.value === 'cash' ? 'cash' : 'other'))

function pickType(next: AccountType) {
  if (editId) return // type is fixed after creation (the API can't change it)
  haptic.selection()
  if (!name.value.trim() || name.value === t(`account_types.${type.value}`)) name.value = t(`account_types.${next}`)
  type.value = next
}

function pickColor(c: string | null) {
  haptic.selection()
  color.value = c
}

/** The styling fields, sent with PATCH (also right after create: POST doesn't take them). */
function stylePatch(): AccountPatch {
  return {
    color: color.value,
    include_in_total: includeInTotal.value,
    card_mask: isCard(type.value) && mask.value ? mask.value : null,
  }
}

async function save() {
  if (saving.value) return
  const e: typeof errors.value = {}
  if (!name.value.trim()) e.name = t('account.err_name')
  const openingMinor = opening.value.trim() ? parseAmount(opening.value, currency.value) : 0
  if (openingMinor === null) e.opening = t('form.err_amount')
  if (mask.value && !/^\d{4}$/.test(mask.value)) e.mask = t('account.err_mask')
  errors.value = e
  if (Object.keys(e).length) {
    haptic.error()
    return
  }
  saving.value = true
  try {
    if (editId) {
      await accounts.update(editId, { name: name.value.trim(), ...stylePatch() })
    } else {
      const created = await accounts.create({
        name: name.value.trim(),
        type: type.value,
        currency: currency.value,
        opening_balance_minor: openingMinor ?? 0,
      })
      const style = stylePatch()
      if (style.color || style.card_mask || !style.include_in_total) await accounts.update(created.id, style)
    }
    haptic.success()
    toast.show(t('common.saved'))
    goBack()
  } catch (err) {
    haptic.error()
    if (err instanceof ApiError && err.field === 'card_mask') errors.value = { mask: err.message }
    else toast.error()
  } finally {
    saving.value = false
  }
}

async function toggleArchive() {
  const a = account.value
  if (!a || saving.value) return
  if (!a.archived && !(await confirmDialog(t('account.archive_confirm')))) return
  saving.value = true
  try {
    await accounts.update(a.id, { archived: !a.archived })
    haptic.success()
    toast.show(t('common.saved'))
    goBack()
  } catch {
    haptic.error()
    toast.error()
  } finally {
    saving.value = false
  }
}

const { native } = useMainButton(
  computed(() => t('common.save')),
  saving,
  save,
)
const currencyOptions = computed(() => CURRENCIES.map((c) => ({ value: c, label: c })))

if (!editId) name.value = t('account_types.humo')
</script>

<template>
  <PageHeader :title="editId ? $t('account.edit') : $t('account.new')" back />

  <div v-if="loading" class="space-y-4 px-4"><div class="skeleton h-44 rounded-[28px]" /><div class="skeleton h-24" /></div>
  <EmptyState v-else-if="failed || (editId && !account)" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

  <form v-else class="space-y-6 px-4 pb-6" novalidate @submit.prevent="save">
    <!-- Big card preview -->
    <section class="relative overflow-hidden rounded-[28px] p-5 text-white" :style="{ background: previewBg, boxShadow: 'var(--shadow-float)' }">
      <span class="absolute -top-10 -right-10 size-40 rounded-full bg-white/10" aria-hidden="true" />
      <span class="absolute -bottom-16 -left-6 size-40 rounded-full bg-white/5" aria-hidden="true" />
      <div class="relative flex items-center justify-between">
        <span class="truncate font-semibold">{{ name || $t(`account_types.${type}`) }}</span>
        <span class="rounded-full bg-white/15 px-2.5 py-0.5 text-xs font-semibold">{{ $t(`account_types.${type}`) }}</span>
      </div>
      <p class="relative mt-8 text-xs tracking-[0.12em] text-white/70 uppercase">{{ $t('account.balance') }}</p>
      <p class="display relative text-[34px] leading-tight">
        {{ account ? money(account.balance_minor, account.currency) : money(parseAmount(opening, currency) ?? 0, currency) }}
      </p>
      <p v-if="kind === 'card'" class="relative mt-2 text-sm tracking-[0.2em] text-white/80">•••• {{ mask || '0000' }}</p>
      <RouterLink
        v-if="account"
        :to="{ name: 'activity', query: { account_id: account.id } }"
        class="relative mt-3 inline-flex items-center gap-0.5 rounded-full bg-white/15 px-3 py-1 text-xs font-semibold"
      >
        {{ $t('activity.title') }}<AppIcon name="chevron" :size="14" />
      </RouterLink>
    </section>

    <!-- Type: two big tiles, card networks underneath -->
    <section>
      <h2 class="eyebrow mb-2 px-1">{{ $t('account.type') }}</h2>
      <div class="grid grid-cols-2 gap-3" role="radiogroup" :aria-label="$t('account.type')">
        <button
          v-for="k in (['card', 'cash'] as const)"
          :key="k"
          type="button"
          role="radio"
          :aria-checked="kind === k"
          :disabled="!!editId && kind !== k"
          class="card pressable flex flex-col items-start gap-2 border-2 p-4 text-left disabled:opacity-40"
          :class="kind === k ? 'border-accent' : 'border-transparent'"
          @click="pickType(k === 'card' ? 'humo' : 'cash')"
        >
          <span class="flex size-10 items-center justify-center rounded-xl bg-accent-soft text-accent">
            <AppIcon :name="k" :size="22" />
          </span>
          <span class="font-semibold">{{ $t(`account.kind_${k}`) }}</span>
          <span class="text-xs text-hint">{{ $t(`account.kind_${k}_sub`) }}</span>
        </button>
      </div>
      <div v-if="!editId" class="mt-3 flex flex-wrap gap-2">
        <button
          v-for="at in (kind === 'card' ? CARD_TYPES : OTHER_TYPES)"
          :key="at"
          type="button"
          class="pressable rounded-full px-3 py-1.5 text-sm font-medium"
          :class="at === type ? 'bg-accent text-on-accent' : 'bg-surface'"
          :aria-pressed="at === type"
          @click="pickType(at)"
        >
          {{ $t(`account_types.${at}`) }}
        </button>
        <button
          v-if="kind === 'card'"
          type="button"
          class="rounded-full px-3 py-1.5 text-sm text-hint"
          @click="pickType('savings')"
        >
          {{ $t('account.more_types') }}
        </button>
      </div>
    </section>

    <!-- Basics -->
    <section>
      <h2 class="eyebrow mb-2 px-1">{{ $t('account.basics') }}</h2>
      <div class="card divide-y divide-separator">
        <label class="flex items-center gap-3 px-4 py-3">
          <span class="w-28 shrink-0 text-sm text-hint">{{ $t('account.name') }}</span>
          <input v-model="name" type="text" maxlength="64" class="min-w-0 flex-1 bg-transparent text-right font-medium outline-none" @input="errors.name = undefined" />
        </label>
        <p v-if="errors.name" class="form-error px-4 pb-2">{{ errors.name }}</p>
        <label v-if="kind === 'card'" class="flex items-center gap-3 px-4 py-3">
          <span class="w-28 shrink-0 text-sm text-hint">{{ $t('account.last4') }}</span>
          <input
            v-model="mask"
            type="text"
            inputmode="numeric"
            maxlength="4"
            placeholder="0000"
            class="tabular min-w-0 flex-1 bg-transparent text-right font-medium tracking-[0.2em] outline-none placeholder:text-hint/50"
            @input="mask = mask.replace(/\D/g, '').slice(0, 4); errors.mask = undefined"
          />
        </label>
        <p v-if="errors.mask" class="form-error px-4 pb-2">{{ errors.mask }}</p>
        <div class="flex items-center gap-3 px-4 py-3">
          <span class="w-28 shrink-0 text-sm text-hint">{{ $t('account.currency') }}</span>
          <span class="flex-1" />
          <SegmentedControl v-if="!editId" v-model="currency" :options="currencyOptions" :label="$t('account.currency')" class="w-36" />
          <span v-else class="rounded-full bg-page px-3 py-1 text-sm font-semibold">{{ currency }}</span>
        </div>
        <div v-if="!editId" class="px-4 py-3">
          <span class="mb-1 block text-sm text-hint">{{ $t('account.opening_balance') }}</span>
          <AmountInput
            v-model="opening"
            :label="$t('account.opening_balance')"
            :unit="currencyLabel(currency, lang)"
            :prefix="currency === 'USD'"
            :error="errors.opening"
          />
        </div>
      </div>
    </section>

    <!-- Appearance and total -->
    <section class="card divide-y divide-separator">
      <button
        type="button"
        class="flex w-full items-center gap-3 px-4 py-3.5 text-left"
        :aria-expanded="showAppearance"
        @click="showAppearance = !showAppearance"
      >
        <span class="size-6 shrink-0 rounded-full ring-2 ring-surface" :style="{ background: preview, boxShadow: '0 0 0 1px var(--separator)' }" />
        <span class="flex-1 font-medium">{{ $t('account.appearance') }}</span>
        <AppIcon name="down" :size="16" class="text-hint transition-transform" :class="{ 'rotate-180': showAppearance }" />
      </button>
      <div v-if="showAppearance" class="flex flex-wrap gap-3 px-4 py-4" role="radiogroup" :aria-label="$t('account.appearance')">
        <button
          v-for="c in ACCOUNT_SWATCHES"
          :key="c"
          type="button"
          role="radio"
          :aria-checked="color === c"
          :aria-label="c"
          class="pressable flex size-10 items-center justify-center rounded-full text-white"
          :class="{ 'ring-2 ring-offset-2 ring-offset-surface': color === c }"
          :style="{ background: c, '--tw-ring-color': c }"
          @click="pickColor(c)"
        >
          <AppIcon v-if="color === c" name="check" :size="18" />
        </button>
        <button
          type="button"
          role="radio"
          :aria-checked="color === null"
          class="pressable rounded-full bg-page px-3 text-sm"
          :class="{ 'ring-2 ring-accent': color === null }"
          @click="pickColor(null)"
        >
          {{ $t('account.color_auto') }}
        </button>
      </div>
      <div class="px-4 py-3.5">
        <ToggleSwitch v-model="includeInTotal" :label="$t('account.include_total')" />
        <p class="mt-1 text-xs text-hint">{{ $t('account.include_total_hint') }}</p>
      </div>
    </section>

    <button
      v-if="account"
      type="button"
      class="block w-full py-2 text-center font-semibold"
      :class="account.archived ? 'text-link' : 'text-danger'"
      :disabled="saving"
      @click="toggleArchive"
    >
      {{ account.archived ? $t('account.unarchive') : $t('account.archive') }}
    </button>
    <p v-if="account" class="-mt-4 text-center text-xs text-hint">{{ $t('account.archive_hint') }}</p>
  </form>

  <FallbackButton v-if="!native && !loading && !failed" :label="$t('common.save')" :busy="saving" @click="save" />
</template>
