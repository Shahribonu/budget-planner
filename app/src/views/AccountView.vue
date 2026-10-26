<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import type { AccountType, Currency } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import AppIcon from '@/components/AppIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { useBack } from '@/composables/useBack'
import { useFormat } from '@/composables/useFormat'
import { useMainButton } from '@/composables/useMainButton'
import { ACCOUNT_TYPES, ACCOUNT_ICONS } from '@/lib/accounts'
import { CURRENCIES, currencyLabel, parseAmount } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

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
const type = ref<AccountType>('cash')
const currency = ref<Currency>(user.mainCurrency)
const opening = ref('')
const nameError = ref('')
const openingError = ref('')
const saving = ref(false)
const loading = ref(true)
const failed = ref(false)

async function load() {
  loading.value = true
  failed.value = false
  try {
    await accounts.load()
    if (account.value) name.value = account.value.name
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

function pickType(next: AccountType) {
  haptic.selection()
  // Keep a custom name; replace one that was just the previous type's label.
  if (!name.value.trim() || name.value === t(`account_types.${type.value}`)) name.value = t(`account_types.${next}`)
  type.value = next
}

async function save() {
  if (saving.value) return
  nameError.value = name.value.trim() ? '' : t('account.err_name')
  const openingMinor = opening.value.trim() ? parseAmount(opening.value, currency.value) : 0
  openingError.value = openingMinor === null ? t('form.err_amount') : ''
  if (nameError.value || openingError.value) {
    haptic.error()
    return
  }
  saving.value = true
  try {
    if (editId) {
      await accounts.update(editId, { name: name.value.trim() })
    } else {
      await accounts.create({
        name: name.value.trim(),
        type: type.value,
        currency: currency.value,
        opening_balance_minor: openingMinor ?? 0,
      })
    }
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

async function toggleArchive() {
  if (!account.value || saving.value) return
  saving.value = true
  try {
    await accounts.update(account.value.id, { archived: !account.value.archived })
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
const currencyOptions = computed(() => CURRENCIES.map((c) => ({ value: c, label: t(`currencies.${c}`) })))

if (!editId) name.value = t('account_types.cash')
</script>

<template>
  <PageHeader :title="editId ? $t('account.edit') : $t('account.new')" back />

  <div v-if="loading" class="py-20 text-center text-hint">{{ $t('common.loading') }}</div>
  <EmptyState v-else-if="failed || (editId && !account)" icon="⚠️" :body="$t('common.error')" :action="$t('common.retry')" @action="load" />

  <form v-else class="space-y-5 px-4 pb-6" novalidate @submit.prevent="save">
    <div v-if="account" class="rounded-2xl bg-surface p-4">
      <div class="text-sm text-hint">
        {{ ACCOUNT_ICONS[account.type] }} {{ $t(`account_types.${account.type}`) }} · {{ account.currency }}
      </div>
      <div class="tabular mt-1 text-3xl font-bold">{{ money(account.balance_minor, account.currency) }}</div>
      <RouterLink :to="{ name: 'activity', query: { account_id: account.id } }" class="mt-2 inline-flex items-center text-sm text-link">
        {{ $t('activity.title') }}<AppIcon name="chevron" :size="16" />
      </RouterLink>
    </div>

    <label class="block">
      <span class="field-label">{{ $t('account.name') }}</span>
      <input
        v-model="name"
        type="text"
        maxlength="64"
        class="w-full rounded-xl bg-surface px-3 py-3 outline-none"
        :aria-invalid="!!nameError"
        @input="nameError = ''"
      />
      <span v-if="nameError" class="mt-1 block text-sm text-danger">{{ nameError }}</span>
    </label>

    <template v-if="!editId">
      <section>
        <h2 class="field-label">{{ $t('account.type') }}</h2>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="at in ACCOUNT_TYPES"
            :key="at"
            type="button"
            class="pressable rounded-full px-3 py-2 text-sm"
            :class="at === type ? 'bg-accent text-on-accent' : 'bg-surface'"
            :aria-pressed="at === type"
            @click="pickType(at)"
          >
            {{ ACCOUNT_ICONS[at] }} {{ $t(`account_types.${at}`) }}
          </button>
        </div>
      </section>

      <section>
        <h2 class="field-label">{{ $t('account.currency') }}</h2>
        <SegmentedControl v-model="currency" :options="currencyOptions" :label="$t('account.currency')" />
      </section>

      <section>
        <h2 class="field-label">{{ $t('account.opening_balance') }}</h2>
        <AmountInput
          v-model="opening"
          :label="$t('account.opening_balance')"
          :unit="currencyLabel(currency, lang)"
          :prefix="currency === 'USD'"
          :error="openingError"
        />
      </section>
    </template>

    <section v-if="account">
      <button
        type="button"
        class="pressable flex w-full items-center gap-2 rounded-xl bg-surface px-4 py-3"
        :class="account.archived ? 'text-link' : 'text-danger'"
        :disabled="saving"
        @click="toggleArchive"
      >
        <AppIcon name="archive" :size="20" />
        {{ account.archived ? $t('account.unarchive') : $t('account.archive') }}
      </button>
      <p class="mt-1 px-1 text-xs text-hint">{{ $t('account.archive_hint') }}</p>
    </section>
  </form>

  <FallbackButton v-if="!native && !loading && !failed" :label="$t('common.save')" :busy="saving" @click="save" />
</template>

<style scoped>
@reference '../style.css';

.field-label {
  @apply mb-2 block text-sm font-medium text-hint;
}
</style>
