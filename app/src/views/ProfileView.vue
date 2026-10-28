<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ApiError, api } from '@/api/client'
import type { Currency, Language, UserPatch } from '@/api/types'
import AppIcon from '@/components/AppIcon.vue'
import BottomSheet from '@/components/BottomSheet.vue'
import OptionSheet from '@/components/OptionSheet.vue'
import HoldButton from '@/components/HoldButton.vue'
import SettingsRow from '@/components/SettingsRow.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import { useFormat } from '@/composables/useFormat'
import { LANGUAGES, setLanguage } from '@/i18n'
import { formatMonth, todayISO } from '@/lib/dates'
import { clearLocalData, prefs, type Prefs } from '@/lib/prefs'
import { CURRENCIES } from '@/lib/money'
import { haptic } from '@/lib/telegram'
import { useCategoriesStore } from '@/stores/categories'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const router = useRouter()
const { lang } = useFormat()
const categories = useCategoriesStore()
const user = useUserStore()
const accounts = useAccountsStore()
const toast = useToastStore()

onMounted(() => {
  void accounts.load(true).catch(() => toast.error())
  void categories.load().catch(() => {})
})

const since = computed(() => (user.me?.created_at ? formatMonth(todayISO(user.me.timezone, new Date(user.me.created_at)).slice(0, 7), lang.value) : ''))
const daysWithUs = computed(() => {
  if (!user.me?.created_at) return 0
  const start = todayISO(user.me.timezone, new Date(user.me.created_at))
  const today = todayISO(user.me.timezone)
  return Math.max(1, Math.round((Date.parse(today) - Date.parse(start)) / 86_400_000) + 1)
})
const stats = computed(() => [
  { label: t('profile.stat_accounts'), value: accounts.active.length },
  { label: t('profile.stat_categories'), value: categories.all.length },
  { label: t('profile.stat_days'), value: daysWithUs.value },
])

const FLAGS: Record<Currency, string> = { UZS: '🇺🇿', USD: '🇺🇸' }
const languageOptions = computed(() => LANGUAGES.map((l) => ({ value: l, label: t(`languages.${l}`) })))
const currencyOptions = computed(() => CURRENCIES.map((c) => ({ value: c, label: c, icon: FLAGS[c], hint: t(`currencies.${c}`) })))
const themeOptions = computed(() => (['system', 'light', 'dark'] as const).map((v) => ({ value: v, label: t(`profile.theme_${v}`) })))
const styleOptions = computed(() => (['icons', 'emoji'] as const).map((v) => ({ value: v, label: t(`profile.style_${v}`) })))

// One sheet open at a time.
type Sheet = 'language' | 'currency' | 'theme' | 'style' | 'notify' | 'about' | 'support' | 'delete' | null
const sheet = ref<Sheet>(null)
const sheetOpen = (s: Exclude<Sheet, null>) => computed({ get: () => sheet.value === s, set: (v) => (sheet.value = v ? s : null) })
const languageOpen = sheetOpen('language')
const currencyOpen = sheetOpen('currency')
const themeOpen = sheetOpen('theme')
const styleOpen = sheetOpen('style')
const notifyOpen = sheetOpen('notify')
const aboutOpen = sheetOpen('about')
const supportOpen = sheetOpen('support')
const deleteOpen = sheetOpen('delete')

function setTheme(v: Prefs['theme'] | null) {
  if (v) prefs.theme = v
}
function setStyle(v: Prefs['categoryStyle'] | null) {
  if (v) prefs.categoryStyle = v
}
const version = __APP_VERSION__

const exporting = ref(false)
async function exportData() {
  if (exporting.value) return
  exporting.value = true
  try {
    await api.exportData()
    haptic.success()
    toast.show(t('privacy.export_sent'))
  } catch (e) {
    haptic.error()
    if (e instanceof ApiError && e.code === 'bot_blocked') toast.show(t('privacy.export_bot_blocked'))
    else if (!(e instanceof ApiError && e.status === 429)) toast.error() // 429 already has its toast
  } finally {
    exporting.value = false
  }
}

const deleting = ref(false)
async function deleteAccount() {
  if (deleting.value) return
  deleting.value = true
  try {
    await api.deleteMe()
    // Forget everything locally; the guard then allows only the goodbye screen.
    clearLocalData()
    user.deleted = true
    user.me = null
    sheet.value = null
    haptic.success()
    await router.replace({ name: 'deleted' })
  } catch {
    haptic.error()
    toast.error()
  } finally {
    deleting.value = false
  }
}

async function setLang(lang: Language | null) {
  if (!lang) return
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

async function setCurrency(currency: Currency | null) {
  if (!currency) return
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

function toggleReminder(enabled: boolean) {
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
  <div class="space-y-6 px-4 pt-6 pb-4">
    <!-- Who -->
    <header class="flex flex-col items-center text-center">
      <span class="avatar-ring rounded-full p-1">
        <UserAvatar :size="96" ring="transparent" />
      </span>
      <h1 class="display mt-3 text-2xl">{{ user.me?.first_name }}</h1>
      <p v-if="user.me?.username" class="text-sm text-hint">@{{ user.me.username }}</p>
      <p v-if="since" class="mt-1 text-xs text-hint">{{ $t('profile.since', { date: since }) }}</p>
    </header>

    <div class="grid grid-cols-3 gap-3">
      <div v-for="s in stats" :key="s.label" class="card px-2 py-3 text-center">
        <span class="display block text-2xl">{{ s.value }}</span>
        <span class="block text-xs text-hint">{{ s.label }}</span>
      </div>
    </div>

    <section>
      <h2 class="eyebrow mb-2 px-1">{{ $t('profile.group_manage') }}</h2>
      <div class="card divide-y divide-separator overflow-hidden">
        <SettingsRow icon="card" :label="$t('profile.accounts')" :value="String(accounts.active.length)" :to="{ name: 'accounts' }" />
        <SettingsRow icon="grid" tone="#f5a524" :label="$t('profile.categories')" :value="String(categories.all.length)" :to="{ name: 'categories' }" />
        <SettingsRow icon="repeat" tone="#0f9e6e" :label="$t('mplan.regular')" :to="{ name: 'recurring' }" />
      </div>
    </section>

    <section>
      <h2 class="eyebrow mb-2 px-1">{{ $t('profile.group_app') }}</h2>
      <div class="card divide-y divide-separator overflow-hidden">
        <SettingsRow icon="cash" tone="#0f9e6e" :label="$t('profile.main_currency')" @click="sheet = 'currency'">
          <span class="rounded-full bg-page px-2.5 py-1 text-sm font-medium">{{ FLAGS[user.mainCurrency] }} {{ user.mainCurrency }}</span>
        </SettingsRow>
        <SettingsRow icon="sun" tone="#f5a524" :label="$t('profile.theme')" :value="$t(`profile.theme_${prefs.theme}`)" @click="sheet = 'theme'" />
        <SettingsRow icon="palette" tone="#ec4899" :label="$t('profile.category_style')" :value="$t(`profile.style_${prefs.categoryStyle}`)" @click="sheet = 'style'" />
        <SettingsRow icon="globe" tone="#3b82f6" :label="$t('profile.language')" :value="$t(`languages.${user.language}`)" @click="sheet = 'language'" />
        <SettingsRow
          icon="bell"
          tone="#ff6b5a"
          :label="$t('profile.reminders')"
          :value="notifyEnabled ? notifyTime : $t('profile.off')"
          @click="sheet = 'notify'"
        />
      </div>
    </section>

    <section>
      <h2 class="eyebrow mb-2 px-1">{{ $t('privacy.group') }}</h2>
      <div class="card divide-y divide-separator overflow-hidden">
        <SettingsRow icon="download" tone="#0f9e6e" :label="$t('privacy.export')" :value="exporting ? $t('common.loading') : undefined" @click="exportData" />
        <SettingsRow icon="shield" tone="#5b3df5" :label="$t('legal.privacy')" :to="{ name: 'legal', params: { doc: 'privacy' } }" />
        <SettingsRow icon="info" tone="#5b3df5" :label="$t('legal.terms')" :to="{ name: 'legal', params: { doc: 'terms' } }" />
        <button type="button" class="flex min-h-14 w-full items-center gap-3 px-4 py-2.5 text-left font-semibold text-danger active:bg-page/60" @click="sheet = 'delete'">
          <span class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-danger/12"><AppIcon name="trash" :size="18" /></span>
          {{ $t('privacy.delete') }}
        </button>
      </div>
      <p class="mt-2 px-1 text-xs text-hint">{{ $t('privacy.export_hint') }}</p>
    </section>

    <section>
      <h2 class="eyebrow mb-2 px-1">{{ $t('profile.group_about') }}</h2>
      <div class="card divide-y divide-separator overflow-hidden">
        <SettingsRow icon="info" tone="#8a87a0" :label="$t('profile.about')" :value="version" @click="sheet = 'about'" />
        <SettingsRow icon="help" tone="#8a87a0" :label="$t('profile.support')" @click="sheet = 'support'" />
      </div>
    </section>
  </div>

  <OptionSheet v-model:open="currencyOpen" :model-value="user.mainCurrency" :title="$t('profile.main_currency')" :options="currencyOptions" @update:model-value="setCurrency" />
  <OptionSheet v-model:open="languageOpen" :model-value="user.language" :title="$t('profile.language')" :options="languageOptions" @update:model-value="setLang" />
  <OptionSheet v-model:open="themeOpen" :model-value="prefs.theme" :title="$t('profile.theme')" :options="themeOptions" @update:model-value="setTheme" />
  <OptionSheet v-model:open="styleOpen" :model-value="prefs.categoryStyle" :title="$t('profile.category_style')" :options="styleOptions" @update:model-value="setStyle" />

  <BottomSheet v-model:open="notifyOpen" :title="$t('profile.reminders')">
    <div class="space-y-4 px-4 pt-2 pb-6">
      <ToggleSwitch :model-value="notifyEnabled" :label="$t('profile.reminders')" @update:model-value="toggleReminder" />
      <p class="text-sm text-hint">{{ $t('profile.reminders_hint') }}</p>
      <label class="flex items-center justify-between gap-3" :class="{ 'opacity-50': !notifyEnabled }">
        <span class="text-sm">{{ $t('profile.reminder_time') }}</span>
        <input
          type="time"
          :value="notifyTime"
          :disabled="!notifyEnabled"
          class="tabular rounded-xl bg-page px-3 py-2 outline-none"
          @change="changeReminderTime"
        />
      </label>
    </div>
  </BottomSheet>
  <BottomSheet v-model:open="aboutOpen" :title="$t('profile.about')">
    <div class="space-y-2 px-4 pt-2 pb-6 text-sm">
      <p>{{ $t('profile.about_body') }}</p>
      <p class="text-hint">{{ $t('profile.version', { v: version }) }}</p>
    </div>
  </BottomSheet>
  <BottomSheet v-model:open="deleteOpen" :title="$t('privacy.delete_title')">
    <div class="space-y-4 px-4 pt-1 pb-6">
      <p class="text-[15px] font-medium">{{ $t('privacy.delete_lead') }}</p>
      <ul class="space-y-2 text-sm">
        <li v-for="k in ['delete_balances', 'delete_history', 'delete_plans']" :key="k" class="flex items-start gap-2">
          <AppIcon name="x" :size="16" class="mt-0.5 shrink-0 text-danger" />{{ $t(`privacy.${k}`) }}
        </li>
      </ul>
      <p class="text-xs text-hint">{{ $t('privacy.delete_export_first') }}</p>
      <HoldButton :label="deleting ? $t('common.loading') : $t('privacy.delete_hold')" :disabled="deleting" @confirm="deleteAccount" />
      <p class="text-center text-xs text-hint">{{ $t('privacy.delete_hold_hint') }}</p>
    </div>
  </BottomSheet>
  <BottomSheet v-model:open="supportOpen" :title="$t('profile.support')">
    <p class="px-4 pt-2 pb-6 text-sm">{{ $t('profile.support_body') }}</p>
  </BottomSheet>
</template>

<style scoped>
.avatar-ring {
  background: linear-gradient(135deg, var(--brand) 0%, var(--mint) 100%);
  box-shadow: 0 8px 24px rgb(91 61 245 / 25%);
}
</style>
