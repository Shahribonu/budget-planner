<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ApiError, api } from '@/api/client'
import type { VoiceItem } from '@/api/types'
import { useFormat } from '@/composables/useFormat'
import { categoryLabel } from '@/i18n'
import { chooseAccount, lastAccountId } from '@/lib/accounts'
import { allResolved, needsPick, reviewOptions } from '@/lib/voiceReview'
import { MAX_SECONDS, MicDeniedError, type WavRecorder } from '@/lib/recorder'
import { haptic } from '@/lib/telegram'
import { useAccountsStore } from '@/stores/accounts'
import { useCategoriesStore } from '@/stores/categories'
import { useLedgerStore } from '@/stores/ledger'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import { useVoiceStore } from '@/stores/voice'
import AppIcon from './AppIcon.vue'
import BottomSheet from './BottomSheet.vue'
import CategoryIcon from './CategoryIcon.vue'

/** Record → recognise → one item opens the form, several can be saved at once, none asks to retry. */
const { t } = useI18n()
const router = useRouter()
const { money } = useFormat()
const voice = useVoiceStore()
const accounts = useAccountsStore()
const categories = useCategoriesStore()
const ledger = useLedgerStore()
const toast = useToastStore()
const user = useUserStore()

type State = 'recording' | 'uploading' | 'many' | 'empty' | 'denied' | 'error'
const state = ref<State>('recording')
const seconds = ref(0)
const level = ref(0)
const heard = ref('')
// Each suggestion plus whether its flagged amount has been confirmed (see lib/voiceReview).
const items = ref<{ item: VoiceItem; resolved: boolean }[]>([])
const canSaveAll = computed(() => items.value.length > 0 && allResolved(items.value))
const errorText = ref('')
const saving = ref(false)
let rec: WavRecorder | null = null
let timer: ReturnType<typeof setInterval> | undefined

const open = computed({
  get: () => voice.open,
  set: (v) => {
    if (!v) cancel()
  },
})

const clock = computed(() => {
  const s = Math.floor(seconds.value)
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
})

async function record(r: WavRecorder) {
  rec = r
  state.value = 'recording'
  seconds.value = 0
  level.value = 0
  r.onLevel = (l) => (level.value = level.value * 0.6 + l * 0.4)
  try {
    await r.start()
    haptic.impact('medium')
    timer = setInterval(() => {
      seconds.value = r.seconds
      if (seconds.value >= MAX_SECONDS) void stop()
    }, 200)
  } catch (e) {
    release()
    if (e instanceof MicDeniedError) state.value = 'denied'
    else {
      errorText.value = t('voice.unsupported')
      state.value = 'error'
    }
  }
}

function release() {
  clearInterval(timer)
  rec?.close()
  rec = null
}

async function stop() {
  if (!rec || state.value !== 'recording') return
  clearInterval(timer)
  const blob = rec.stop()
  rec = null
  haptic.impact('light')
  state.value = 'uploading'
  try {
    const res = await api.voice(blob)
    heard.value = res.text
    if (res.items.length === 1) {
      openInForm(res.items[0]!)
    } else if (res.items.length > 1) {
      items.value = res.items.map((item) => ({ item, resolved: false }))
      state.value = 'many'
      haptic.success()
    } else {
      state.value = 'empty'
    }
  } catch (e) {
    haptic.error()
    const code = e instanceof ApiError ? e.status : 0
    if (code === 503) {
      toast.show(t('voice.coming_soon'))
      close()
      return
    }
    errorText.value = code === 429 ? t('voice.too_many') : code === 502 || code === 400 ? t('voice.failed') : t('common.error')
    state.value = 'error'
  }
}

/** Retry is a fresh tap, so the new recorder is created inside the gesture as well. */
function retry() {
  const r = voice.newRecorder()
  if (r) void record(r)
}

function close() {
  release()
  voice.open = false
}

function cancel() {
  if (state.value === 'recording') haptic.impact('light')
  close()
}

function openInForm(item: VoiceItem) {
  voice.prefill = { item, text: heard.value }
  close()
  void router.push({ name: 'tx-new', query: { voice: '1' } })
}

function edit(i: number) {
  const [row] = items.value.splice(i, 1)
  if (row) openInForm(row.item)
}

function pick(i: number, amountMinor: number) {
  const row = items.value[i]
  if (!row) return
  haptic.selection()
  row.item = { ...row.item, amount_minor: amountMinor }
  row.resolved = true
}

/** Saves every item; one ledger refresh at the end keeps this to a handful of requests. */
async function saveAll() {
  if (saving.value || !canSaveAll.value) return
  saving.value = true
  let saved = 0
  try {
    await accounts.load()
    for (const { item } of [...items.value]) {
      const account = chooseAccount(accounts.active, {
        currency: item.currency,
        preferredId: item.account_id,
        lastId: lastAccountId(),
        mainCurrency: user.mainCurrency,
      })
      if (!account) {
        toast.show(t('voice.no_account', { currency: item.currency }))
        break
      }
      await api.createTransaction({
        type: item.type,
        occurred_on: item.occurred_on,
        amount_minor: item.amount_minor,
        account_id: account.id,
        category_id: item.category_id,
        description: item.description,
        source: 'voice',
        source_ref: crypto.randomUUID?.() ?? `${Date.now()}-${Math.random()}`,
      })
      items.value.shift()
      saved++
    }
  } catch {
    haptic.error()
    toast.error()
  } finally {
    saving.value = false
    if (saved) {
      ledger.changed()
      haptic.success()
      toast.show(t('voice.saved_n', { n: saved }))
    }
    if (!items.value.length) close()
  }
}

onMounted(() => {
  void categories.load().catch(() => {})
  const r = voice.recorder
  voice.recorder = null
  if (r) void record(r)
})
onBeforeUnmount(release)
</script>

<template>
  <BottomSheet v-model:open="open" :title="$t('voice.title')">
    <div class="px-4 pt-2 pb-6">
      <!-- Recording -->
      <div v-if="state === 'recording' || state === 'uploading'" class="flex flex-col items-center gap-4 py-2 text-center">
        <div class="relative flex size-36 items-center justify-center" aria-hidden="true">
          <span
            class="pulse absolute inset-0 rounded-full bg-accent/15"
            :style="{ transform: `scale(${0.75 + level * 0.6})` }"
          />
          <span class="absolute inset-4 rounded-full bg-accent/20" />
          <span class="relative flex size-20 items-center justify-center rounded-full bg-accent text-white shadow-lg shadow-accent/40">
            <AppIcon name="mic" :size="36" />
          </span>
        </div>
        <p class="display text-3xl tabular-nums" aria-live="off">{{ clock }}</p>
        <p class="text-sm text-hint" role="status">
          {{ state === 'uploading' ? $t('voice.recognizing') : $t('voice.listening') }}
        </p>
        <p v-if="state === 'recording'" class="max-w-xs text-xs text-hint">{{ $t('voice.hint') }}</p>
        <div v-if="state === 'recording'" class="grid w-full grid-cols-2 gap-3 pt-2">
          <button type="button" class="pressable rounded-2xl bg-page py-3.5 font-semibold" @click="cancel">
            {{ $t('common.cancel') }}
          </button>
          <button
            type="button"
            class="pressable flex items-center justify-center gap-2 rounded-2xl bg-accent py-3.5 font-semibold text-white shadow-lg shadow-accent/30"
            data-autofocus
            @click="stop"
          >
            <AppIcon name="stop" :size="18" />{{ $t('voice.stop') }}
          </button>
        </div>
      </div>

      <!-- Several items -->
      <div v-else-if="state === 'many'" class="space-y-3">
        <p class="text-sm text-hint">{{ $t('voice.heard', { text: heard }) }}</p>
        <ul class="card divide-y divide-separator overflow-hidden">
          <li v-for="(row, i) in items" :key="i" class="px-3 py-2.5">
            <div class="flex items-center gap-3">
            <button type="button" class="flex min-w-0 flex-1 items-center gap-3 text-left" :aria-label="$t('voice.edit')" @click="edit(i)">
              <CategoryIcon :category="categories.byId.get(row.item.category_id)" :size="40" />
              <span class="min-w-0 flex-1">
                <span class="block truncate font-semibold">{{ row.item.description || categoryLabel(categories.byId.get(row.item.category_id)) }}</span>
                <span class="block truncate text-xs text-hint">{{ categoryLabel(categories.byId.get(row.item.category_id)) }}</span>
              </span>
              <span
                class="tabular shrink-0 rounded-lg px-1.5 font-bold whitespace-nowrap"
                :class="row.item.needs_review && !row.resolved ? 'bg-warning/15 text-warning' : row.item.type === 'income' ? 'text-income' : ''"
                :title="row.item.needs_review && !row.resolved ? $t('voice.check_amount') : undefined"
              >
                <AppIcon v-if="row.item.needs_review && !row.resolved" name="alert" :size="12" class="mr-0.5 inline" />
                {{ money(row.item.type === 'income' ? row.item.amount_minor : -row.item.amount_minor, row.item.currency, { signed: row.item.type === 'income' }) }}
              </span>
            </button>
            <button type="button" class="pressable shrink-0 rounded-full p-2 text-hint" :aria-label="$t('voice.remove')" @click="items.splice(i, 1)">
              <AppIcon name="x" :size="16" />
            </button>
            </div>
            <!-- Flagged so'm amount: pick as heard or × 1000 -->
            <div v-if="needsPick(row.item, row.resolved)" class="mt-2 flex gap-2 pl-[52px]" role="group" :aria-label="$t('voice.check_amount')">
              <button
                v-for="opt in reviewOptions(row.item)"
                :key="opt"
                type="button"
                class="pressable rounded-full border-2 border-warning/60 px-3 py-1 text-sm font-semibold"
                @click="pick(i, opt)"
              >
                {{ money(opt, row.item.currency) }}
              </button>
            </div>
          </li>
        </ul>
        <p v-if="!canSaveAll && items.length" class="flex items-center justify-center gap-1 text-sm font-medium text-warning" role="status">
          {{ $t('voice.check_marked') }}
        </p>
        <button
          type="button"
          class="pressable w-full rounded-2xl bg-accent py-3.5 font-semibold text-white shadow-lg shadow-accent/30 disabled:opacity-50"
          :disabled="saving || !canSaveAll"
          @click="saveAll"
        >
          {{ saving ? $t('common.loading') : $t('voice.save_all', { n: items.length }) }}
        </button>
      </div>

      <!-- Nothing found, errors, mic off -->
      <div v-else class="flex flex-col items-center gap-3 py-4 text-center">
        <span class="flex size-16 items-center justify-center rounded-full bg-page text-hint" aria-hidden="true">
          <AppIcon :name="state === 'denied' ? 'mic' : 'alert'" :size="28" />
        </span>
        <template v-if="state === 'empty'">
          <p v-if="heard" class="text-sm">{{ $t('voice.heard', { text: heard }) }}</p>
          <p class="text-sm text-hint">{{ $t('voice.none') }}</p>
        </template>
        <template v-else-if="state === 'denied'">
          <p class="font-semibold">{{ $t('voice.denied_title') }}</p>
          <p class="text-sm text-hint">{{ $t('voice.denied_body') }}</p>
        </template>
        <p v-else class="text-sm">{{ errorText }}</p>
        <button
          type="button"
          class="pressable mt-2 flex items-center gap-2 rounded-2xl bg-accent px-6 py-3 font-semibold text-white"
          data-autofocus
          @click="retry"
        >
          <AppIcon name="mic" :size="18" />{{ $t('voice.retry') }}
        </button>
      </div>
    </div>
  </BottomSheet>
</template>

<style scoped>
.pulse {
  transition: transform 120ms linear;
  animation: breathe 1.6s ease-in-out infinite;
}
@keyframes breathe {
  50% {
    opacity: 0.6;
  }
}
@media (prefers-reduced-motion: reduce) {
  .pulse {
    transition: none;
    animation: none;
    transform: none !important;
  }
}
</style>
