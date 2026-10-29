<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Currency } from '@/api/types'
import { invalidField } from '@/composables/useApiError'
import { CURRENCIES } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { rememberSavings, useSavingsStore } from '@/stores/savings'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import AppIcon from './AppIcon.vue'
import BottomSheet from './BottomSheet.vue'
import SegmentedControl from './SegmentedControl.vue'

/** First savings account: name and currency, kept out of the total balance; then on to the transfer. */
const { t } = useI18n()
const router = useRouter()
const savings = useSavingsStore()
const accounts = useAccountsStore()
const user = useUserStore()
const toast = useToastStore()

const name = ref('')
const currency = ref<Currency>(user.mainCurrency)
const busy = ref(false)
const nameError = ref('')

watch(
  () => savings.createOpen,
  (open) => {
    if (!open) return
    name.value = t('savings.default_name')
    currency.value = user.mainCurrency
    nameError.value = ''
  },
  { immediate: true },
)

async function create() {
  if (busy.value) return
  if (!name.value.trim()) {
    nameError.value = t('account.err_name')
    haptic.error()
    return
  }
  busy.value = true
  try {
    const account = await accounts.create({
      name: name.value.trim(),
      type: 'savings',
      currency: currency.value,
      include_in_total: false,
    })
    rememberSavings(account.id)
    haptic.success()
    savings.createOpen = false
    savings.start(savings.pending, router)
  } catch (e) {
    haptic.error()
    const { field, message } = invalidField(e)
    if (field === 'name') nameError.value = message
    else toast.error()
  } finally {
    busy.value = false
  }
}

const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))
</script>

<template>
  <BottomSheet v-model:open="savings.createOpen" :title="$t('savings.create_title')">
    <form class="space-y-4 px-4 pt-1 pb-6" novalidate @submit.prevent="create">
      <p class="flex items-start gap-3 text-sm text-hint">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-mint/20 text-income"><AppIcon name="piggy" :size="20" /></span>
        {{ $t('savings.create_body') }}
      </p>
      <label class="block">
        <span class="form-label">{{ $t('account.name') }}</span>
        <input
          v-model="name"
          type="text"
          maxlength="64"
          class="form-input bg-page"
          :aria-invalid="!!nameError"
          data-autofocus
          @input="nameError = ''"
        />
        <span v-if="nameError" class="form-error">{{ nameError }}</span>
      </label>
      <div>
        <span class="form-label">{{ $t('account.currency') }}</span>
        <SegmentedControl v-model="currency" :options="currencyOptions" :label="$t('account.currency')" />
      </div>
      <button
        type="submit"
        class="pressable w-full rounded-2xl bg-accent py-3.5 font-semibold text-on-accent shadow-lg shadow-accent/25 disabled:opacity-60"
        :disabled="busy"
      >
        {{ busy ? $t('common.loading') : $t('savings.create_continue') }}
      </button>
    </form>
  </BottomSheet>
</template>
