<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { AccountType, Currency, Language } from '@/api/types'
import AmountInput from '@/components/AmountInput.vue'
import AppIcon from '@/components/AppIcon.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { useMainButton } from '@/composables/useMainButton'
import { LANGUAGES, detectLanguage, setLanguage } from '@/i18n'
import { ACCOUNT_ICONS, ACCOUNT_TYPES } from '@/lib/accounts'
import { CURRENCIES, currencyLabel, parseAmount } from '@/lib/money'
import { backButton, haptic, telegramLanguageCode } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

interface Draft {
  uid: number
  type: AccountType
  name: string
  currency: Currency
  opening: string
  error?: string
  /** Set once created, so a retry after a partial failure does not duplicate accounts. */
  createdId?: number
}

const { t } = useI18n()
const router = useRouter()
const user = useUserStore()
const accounts = useAccountsStore()
const toast = useToastStore()

const step = ref<1 | 2>(1)
// Default to the Telegram client language, not the server default.
const language = ref<Language>(detectLanguage(telegramLanguageCode() ?? navigator.language))
void setLanguage(language.value)
const mainCurrency = ref<Currency>(user.me?.main_currency ?? 'UZS')
const busy = ref(false)

watch(language, (l) => void setLanguage(l))

let uid = 0
const drafts = reactive<Draft[]>([])
const showMoreTypes = ref(false)
const PRIMARY: AccountType[] = ['cash', 'humo', 'uzcard']
const presetTypes = computed(() => (showMoreTypes.value ? ACCOUNT_TYPES : PRIMARY))

function addDraft(type: AccountType) {
  haptic.selection()
  drafts.push({ uid: ++uid, type, name: t(`account_types.${type}`), currency: mainCurrency.value, opening: '' })
}

function removeDraft(d: Draft) {
  drafts.splice(drafts.indexOf(d), 1)
}

async function next() {
  busy.value = true
  try {
    let timezone: string | undefined
    try {
      timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
    } catch {
      /* keep server default */
    }
    await user.update({ language: language.value, main_currency: mainCurrency.value, timezone })
    if (!drafts.length) addDraft('cash')
    for (const d of drafts) if (!d.createdId) d.currency = mainCurrency.value
    step.value = 2
    window.scrollTo(0, 0)
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}

async function finish() {
  if (!drafts.length) {
    haptic.error()
    toast.show(t('onboarding.need_account'))
    return
  }
  let valid = true
  for (const d of drafts) {
    const opening = d.opening.trim() ? parseAmount(d.opening, d.currency) : 0
    d.error = !d.name.trim() ? t('account.err_name') : opening === null ? t('form.err_amount') : undefined
    if (d.error) valid = false
  }
  if (!valid) {
    haptic.error()
    return
  }
  busy.value = true
  try {
    for (const d of drafts) {
      if (d.createdId) continue
      const created = await accounts.create({
        name: d.name.trim(),
        type: d.type,
        currency: d.currency,
        opening_balance_minor: (d.opening.trim() && parseAmount(d.opening, d.currency)) || 0,
      })
      d.createdId = created.id
    }
    await user.load(true)
    haptic.success()
    await router.replace({ name: 'home' })
  } catch {
    haptic.error()
    toast.error()
  } finally {
    busy.value = false
  }
}

const buttonText = computed(() => (step.value === 1 ? t('common.next') : t('onboarding.finish')))
const { native } = useMainButton(buttonText, busy, () => (step.value === 1 ? next() : finish()))

const toStep1 = () => (step.value = 1)
watch(step, (s) => (s === 2 ? backButton.show(toStep1) : backButton.hide(toStep1)))
onBeforeUnmount(() => backButton.hide(toStep1))

const currencyOptions = computed(() => CURRENCIES.map((c) => ({ value: c, label: t(`currencies.${c}`) })))
</script>

<template>
  <!-- Step 1 opens on the brand hero -->
  <section v-if="step === 1" class="onb-hero relative overflow-hidden px-6 pt-10 pb-24 text-center text-white">
    <span class="blob blob-1" aria-hidden="true" />
    <span class="blob blob-2" aria-hidden="true" />
    <svg viewBox="0 0 120 90" class="wave-in relative mx-auto h-24 w-auto" aria-hidden="true">
      <rect x="14" y="22" width="92" height="60" rx="16" fill="#fff" opacity=".95" />
      <rect x="14" y="22" width="92" height="20" rx="10" fill="#cfc4ff" />
      <rect x="72" y="46" width="38" height="22" rx="11" fill="#2bd9a0" />
      <circle cx="83" cy="57" r="4.5" fill="#fff" />
      <path d="M42 10l5 9m14-12v11m16-7-5 9" stroke="#2bd9a0" stroke-width="4.5" stroke-linecap="round" />
    </svg>
    <h1 class="display relative mt-4 text-[30px]">{{ $t('onboarding.welcome') }}</h1>
    <p class="relative mx-auto mt-1 max-w-xs text-white/85">{{ $t('onboarding.welcome_body') }}</p>
  </section>

  <div class="px-4 pb-6" :class="step === 1 ? '-mt-16 relative' : 'pt-8'">
    <template v-if="step === 1">
      <h2 class="eyebrow mb-2 px-1 text-white/80">{{ $t('onboarding.language') }}</h2>
      <div class="mb-6 overflow-hidden card" role="radiogroup" :aria-label="$t('onboarding.language')">
        <button
          v-for="l in LANGUAGES"
          :key="l"
          type="button"
          role="radio"
          :aria-checked="language === l"
          class="flex w-full items-center justify-between border-b border-separator px-4 py-4 text-left font-medium last:border-b-0 active:bg-page/60"
          @click="language = l"
        >
          {{ $t(`languages.${l}`) }}
          <span
            class="flex size-5 items-center justify-center rounded-full border-2"
            :class="language === l ? 'border-accent bg-accent' : 'border-hint/50'"
          >
            <span v-if="language === l" class="size-2 rounded-full bg-on-accent" />
          </span>
        </button>
      </div>

      <h2 class="eyebrow mb-2 px-1">{{ $t('onboarding.main_currency') }}</h2>
      <SegmentedControl v-model="mainCurrency" :options="currencyOptions" :label="$t('onboarding.main_currency')" />
    </template>

    <template v-else>
      <div class="mb-5">
        <h1 class="text-2xl font-bold">{{ $t('onboarding.accounts_title') }}</h1>
        <p class="mt-1 text-hint">{{ $t('onboarding.accounts_body') }}</p>
      </div>

      <div class="mb-4 flex flex-wrap gap-2">
        <button
          v-for="at in presetTypes"
          :key="at"
          type="button"
          class="pressable flex items-center gap-1 rounded-full bg-surface px-3 py-2 text-sm"
          @click="addDraft(at)"
        >
          <AppIcon name="plus" :size="16" class="text-link" />
          {{ ACCOUNT_ICONS[at] }} {{ $t(`account_types.${at}`) }}
        </button>
        <button
          v-if="!showMoreTypes"
          type="button"
          class="rounded-full px-3 py-2 text-sm text-link"
          @click="showMoreTypes = true"
        >
          {{ $t('onboarding.add_more') }}…
        </button>
      </div>

      <div class="space-y-3">
        <div v-for="d in drafts" :key="d.uid" class="space-y-3 card p-4">
          <div class="flex items-center gap-2">
            <span class="text-xl" aria-hidden="true">{{ ACCOUNT_ICONS[d.type] }}</span>
            <input
              v-model="d.name"
              type="text"
              maxlength="64"
              :aria-label="$t('account.name')"
              :disabled="!!d.createdId"
              class="min-w-0 flex-1 rounded-lg bg-page px-3 py-2 font-medium outline-none"
            />
            <button
              v-if="!d.createdId"
              type="button"
              class="rounded-full p-2 text-hint"
              :aria-label="$t('onboarding.remove')"
              @click="removeDraft(d)"
            >
              <AppIcon name="x" :size="18" />
            </button>
          </div>
          <SegmentedControl
            v-if="!d.createdId"
            v-model="d.currency"
            :options="CURRENCIES.map((c) => ({ value: c, label: c }))"
            :label="$t('account.currency')"
          />
          <div>
            <div class="mb-1 text-xs text-hint">{{ $t('onboarding.opening_balance') }}</div>
            <AmountInput
              v-model="d.opening"
              :label="$t('onboarding.opening_balance')"
              :unit="currencyLabel(d.currency, language)"
              :prefix="d.currency === 'USD'"
            />
          </div>
          <p v-if="d.error" class="text-sm text-danger">{{ d.error }}</p>
        </div>
      </div>
    </template>
  </div>

  <FallbackButton v-if="!native" :label="buttonText" :busy="busy" @click="step === 1 ? next() : finish()" />
</template>

<style scoped>
.onb-hero {
  margin-top: calc(-1 * var(--safe-top));
  padding-top: calc(var(--safe-top) + 2.5rem);
  background: linear-gradient(165deg, #5b3df5 0%, #6a4bfa 55%, #7b5cff 100%);
  border-bottom-left-radius: 32px;
  border-bottom-right-radius: 32px;
}
.blob {
  position: absolute;
  border-radius: 9999px;
  filter: blur(30px);
}
.blob-1 {
  width: 220px;
  height: 220px;
  top: -70px;
  right: -60px;
  background: rgb(168 146 255 / 65%);
}
.blob-2 {
  width: 180px;
  height: 180px;
  bottom: -40px;
  left: -60px;
  background: rgb(43 217 160 / 35%);
}
.wave-in {
  animation: pop 700ms cubic-bezier(0.2, 1.4, 0.4, 1) both;
}
@keyframes pop {
  from {
    opacity: 0;
    transform: scale(0.6) rotate(-8deg);
  }
}
@media (prefers-reduced-motion: reduce) {
  .wave-in {
    animation: none;
  }
}
</style>
