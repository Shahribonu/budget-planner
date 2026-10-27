<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Currency, Language, UserPatch } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import PageHeader from '@/components/PageHeader.vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { useFormat } from '@/composables/useFormat'
import { LANGUAGES, setLanguage } from '@/i18n'
import { ACCOUNT_ICONS } from '@/lib/accounts'
import { CURRENCIES } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const { money } = useFormat()
const user = useUserStore()
const accounts = useAccountsStore()
const toast = useToastStore()

const showArchived = ref(false)
const archived = computed(() => accounts.all.filter((a) => a.archived))

onMounted(() => accounts.load(true).catch(() => toast.error()))

const languageOptions = computed(() => LANGUAGES.map((l) => ({ value: l, label: t(`languages.${l}`) })))
const currencyOptions = CURRENCIES.map((c) => ({ value: c, label: c }))

async function setLang(lang: Language) {
  const previous = user.language
  await setLanguage(lang)
  try {
    await user.update({ language: lang })
    haptic.success()
  } catch {
    await setLanguage(previous)
    toast.error()
  }
}

async function setCurrency(currency: Currency) {
  try {
    await user.update({ main_currency: currency })
    haptic.success()
  } catch {
    toast.error()
  }
}

// Local copies so the controls react instantly and can be reverted when PATCH fails.
const notifyEnabled = ref(user.me?.notify_enabled ?? false)
const notifyTime = ref(user.me?.notify_time ?? '21:00')
watch(
  () => user.me,
  (me) => {
    if (!me) return
    notifyEnabled.value = me.notify_enabled
    notifyTime.value = me.notify_time
  },
)

// Only the latest save may revert, so a slow earlier failure can't undo a newer change.
let reminderSeq = 0

async function saveReminder(patch: UserPatch, revert: () => void) {
  const seq = ++reminderSeq
  try {
    await user.update(patch)
    haptic.success()
  } catch {
    if (seq === reminderSeq) revert()
    haptic.error()
    toast.error()
  }
}

function toggleReminder(e: Event) {
  const enabled = (e.target as HTMLInputElement).checked
  const previous = notifyEnabled.value
  notifyEnabled.value = enabled
  void saveReminder({ notify_enabled: enabled }, () => (notifyEnabled.value = previous))
}

function changeReminderTime(e: Event) {
  const input = e.target as HTMLInputElement
  const time = input.value.slice(0, 5) // some browsers append ":SS"
  const previous = notifyTime.value
  if (!/^\d{2}:\d{2}$/.test(time) || time === previous) {
    input.value = previous // cleared or unchanged: nothing to save
    return
  }
  notifyTime.value = time
  void saveReminder({ notify_time: time }, () => (notifyTime.value = previous))
}
</script>

<template>
  <PageHeader :title="user.me?.first_name ?? $t('profile.title')" />

  <div class="space-y-4 px-4">
    <section class="overflow-hidden rounded-2xl bg-surface">
      <h2 class="px-4 pt-3 pb-1 text-sm font-medium text-hint">{{ $t('profile.accounts') }}</h2>
      <RouterLink
        v-for="a in [...accounts.active, ...(showArchived ? archived : [])]"
        :key="a.id"
        :to="{ name: 'account-edit', params: { id: a.id } }"
        class="flex items-center gap-3 px-4 py-2.5 active:bg-page/60"
        :class="{ 'opacity-60': a.archived }"
      >
        <span class="flex size-10 items-center justify-center rounded-full bg-page text-xl" aria-hidden="true">
          {{ ACCOUNT_ICONS[a.type] }}
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate font-medium">{{ a.name }}</span>
          <span class="block text-xs text-hint">
            {{ $t(`account_types.${a.type}`) }}<template v-if="a.archived"> · {{ $t('profile.archived') }}</template>
          </span>
        </span>
        <span class="tabular font-semibold whitespace-nowrap">{{ money(a.balance_minor, a.currency) }}</span>
        <AppIcon name="chevron" :size="18" class="text-hint" />
      </RouterLink>
      <RouterLink :to="{ name: 'account-new' }" class="flex items-center gap-3 px-4 py-3 text-link active:bg-page/60">
        <span class="flex size-10 items-center justify-center rounded-full bg-page"><AppIcon name="plus" :size="20" /></span>
        {{ $t('profile.add_account') }}
      </RouterLink>
      <button
        v-if="archived.length"
        type="button"
        class="w-full border-t border-separator px-4 py-3 text-left text-sm text-hint"
        :aria-pressed="showArchived"
        @click="showArchived = !showArchived"
      >
        {{ $t('profile.show_archived') }} ({{ archived.length }})
      </button>
    </section>

    <section class="space-y-3 rounded-2xl bg-surface p-4">
      <label class="flex items-center justify-between gap-3">
        <span class="font-medium">{{ $t('profile.reminders') }}</span>
        <input
          type="checkbox"
          role="switch"
          class="peer sr-only"
          :checked="notifyEnabled"
          :aria-checked="notifyEnabled"
          @change="toggleReminder"
        />
        <span
          aria-hidden="true"
          class="relative h-7 w-12 shrink-0 rounded-full transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-accent"
          :class="notifyEnabled ? 'bg-accent' : 'bg-hint/40'"
        >
          <span
            class="absolute top-0.5 left-0.5 size-6 rounded-full bg-white shadow transition-transform"
            :class="{ 'translate-x-5': notifyEnabled }"
          />
        </span>
      </label>
      <p class="text-sm text-hint">{{ $t('profile.reminders_hint') }}</p>
      <label class="flex items-center justify-between gap-3" :class="{ 'opacity-50': !notifyEnabled }">
        <span class="text-sm">{{ $t('profile.reminder_time') }}</span>
        <input
          type="time"
          :value="notifyTime"
          :disabled="!notifyEnabled"
          class="tabular rounded-lg bg-page px-3 py-2 outline-none"
          @change="changeReminderTime"
        />
      </label>
    </section>

    <section class="space-y-3 rounded-2xl bg-surface p-4">
      <h2 class="text-sm font-medium text-hint">{{ $t('profile.settings') }}</h2>
      <div>
        <div class="mb-2 text-sm">{{ $t('profile.language') }}</div>
        <div class="grid grid-cols-2 gap-2" role="radiogroup" :aria-label="$t('profile.language')">
          <button
            v-for="opt in languageOptions"
            :key="opt.value"
            type="button"
            role="radio"
            :aria-checked="user.language === opt.value"
            class="pressable rounded-xl border-2 px-3 py-2 text-sm"
            :class="user.language === opt.value ? 'border-accent' : 'border-transparent bg-page'"
            @click="setLang(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
      <div>
        <div class="mb-2 text-sm">{{ $t('profile.main_currency') }}</div>
        <SegmentedControl
          :model-value="user.mainCurrency"
          :options="currencyOptions"
          :label="$t('profile.main_currency')"
          @update:model-value="setCurrency"
        />
      </div>
    </section>
  </div>
</template>
