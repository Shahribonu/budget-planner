<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PremiumPlan } from '@/api/types'
import AppIcon, { type IconName } from '@/components/AppIcon.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import { useMainButton } from '@/composables/useMainButton'
import { usePremium } from '@/composables/usePremium'
import { formatDateNumeric } from '@/lib/dates'
import { formatStars, monthsFree } from '@/lib/premium'
import { haptic } from '@/lib/telegram'
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const user = useUserStore()
const { buy, buying, canPay, prices, pricesState, loadPrices } = usePremium()
onMounted(loadPrices)

// Monthly is selected first: the cheaper, no-commitment option. Nothing else is pre-checked.
const plan = ref<PremiumPlan>('monthly')
const celebrate = ref(false)
const pending = ref(false)

const free = computed(() => prices.value?.free_budgets ?? 1)
const saved = computed(() => (prices.value ? monthsFree(prices.value.monthly_stars, prices.value.yearly_stars) : 0))
const priceOf = (p: PremiumPlan) => (p === 'monthly' ? prices.value?.monthly_stars : prices.value?.yearly_stars) ?? 0
const perMonth = computed(() => (prices.value ? Math.round(prices.value.yearly_stars / 12) : 0))

const benefits = computed<{ icon: IconName; title: string; sub: string; soon?: boolean }[]>(() => [
  { icon: 'target', title: t('premium.b_budgets'), sub: t('premium.b_budgets_sub', { n: free.value }) },
  { icon: 'star', title: t('premium.b_support'), sub: t('premium.b_support_sub') },
  { icon: 'wand', title: t('premium.soon_ai'), sub: t('premium.soon_ai_sub'), soon: true },
  { icon: 'mic', title: t('premium.soon_voice'), sub: t('premium.soon_voice_sub'), soon: true },
])

function pick(p: PremiumPlan) {
  if (plan.value !== p) haptic.selection()
  plan.value = p
}

async function getPremium() {
  const result = await buy(plan.value)
  if (result === 'active') celebrate.value = true
  else if (result === 'pending') pending.value = true
}

const canBuy = computed(() => !user.me?.premium && canPay && pricesState.value === 'ready')
const buttonText = computed(() => t('premium.get', { stars: formatStars(priceOf(plan.value)) }))
const { native } = useMainButton(buttonText, computed(() => !!buying.value), () => void getPremium(), canBuy)

// Confetti pieces: positions and delays fixed per piece, colours from the brand.
const CONFETTI = Array.from({ length: 24 }, (_, i) => ({
  left: `${(i * 37) % 100}%`,
  delay: `${(i % 6) * 0.08}s`,
  color: ['#2bd9a0', '#ffffff', '#ffc861', '#ff6b5a', '#b9a8ff'][i % 5],
  rotate: `${(i * 47) % 360}deg`,
}))
</script>

<template>
  <PageHeader :title="$t('premium.title')" back />

  <div class="space-y-5 px-4 pb-8">
    <!-- Hero -->
    <section class="premium-hero relative overflow-hidden rounded-[28px] px-5 pt-8 pb-6 text-center text-white">
      <span v-for="i in 6" :key="i" class="sparkle" :class="`sparkle-${i}`" aria-hidden="true">✦</span>
      <div v-if="celebrate" class="confetti" aria-hidden="true">
        <span
          v-for="(c, i) in CONFETTI"
          :key="i"
          :style="{ left: c.left, animationDelay: c.delay, background: c.color, transform: `rotate(${c.rotate})` }"
        />
      </div>
      <div class="crown-glow relative mx-auto flex size-24 items-center justify-center rounded-full">
        <svg viewBox="0 0 64 64" class="size-16" aria-hidden="true">
          <defs>
            <linearGradient id="crown" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stop-color="#ffe9a8" />
              <stop offset="1" stop-color="#f5a524" />
            </linearGradient>
          </defs>
          <path d="M8 22l12 10 12-18 12 18 12-10-5 28H13z" fill="url(#crown)" stroke="#fff6d8" stroke-width="2" stroke-linejoin="round" />
          <rect x="13" y="50" width="38" height="6" rx="3" fill="#f5a524" />
          <circle cx="32" cy="38" r="3.5" fill="#2bd9a0" />
        </svg>
      </div>
      <h1 class="display relative mt-4 text-3xl">
        {{ user.me?.premium ? $t('premium.active_title') : $t('premium.title') }}
      </h1>
      <p class="relative mt-1 text-sm text-white/80">
        <template v-if="user.me?.premium && user.me.premium_until">
          {{ $t('premium.until', { date: formatDateNumeric(user.me.premium_until, user.me.timezone) }) }}
        </template>
        <template v-else>{{ $t('premium.hero_sub') }}</template>
      </p>
    </section>

    <!-- Active: thanks and how to manage -->
    <section v-if="user.me?.premium" class="card space-y-2 p-5 text-center">
      <p class="font-semibold">{{ $t('premium.active_thanks') }}</p>
      <p class="text-sm text-hint">{{ $t('premium.manage_hint') }}</p>
    </section>

    <template v-else>
      <p v-if="!user.me?.trial_used" class="card flex items-start gap-3 p-4 text-sm">
        <span class="text-xl" aria-hidden="true">🎁</span>{{ $t('premium.trial_hint') }}
      </p>

      <!-- Benefits: only what's true today, the rest clearly "coming soon" -->
      <section>
        <h2 class="eyebrow mb-2 px-1">{{ $t('premium.benefits') }}</h2>
        <ul class="card divide-y divide-separator">
          <li v-for="b in benefits" :key="b.title" class="flex items-center gap-3 px-4 py-3.5" :class="{ 'opacity-70': b.soon }">
            <span
              class="flex size-11 shrink-0 items-center justify-center rounded-2xl"
              :class="b.soon ? 'bg-page text-hint' : 'bg-accent-soft text-accent'"
            >
              <AppIcon :name="b.icon" :size="22" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex items-center gap-2 font-semibold">
                {{ b.title }}
                <span v-if="b.soon" class="rounded-full bg-page px-2 py-0.5 text-[10px] font-semibold tracking-wide text-hint uppercase">
                  {{ $t('premium.soon') }}
                </span>
              </span>
              <span class="block text-sm text-hint">{{ b.sub }}</span>
            </span>
          </li>
        </ul>
      </section>

      <!-- Plans -->
      <section>
        <h2 class="eyebrow mb-2 px-1">{{ $t('premium.choose') }}</h2>
        <p v-if="!canPay" class="card p-4 text-sm text-hint">{{ $t('premium.open_in_telegram') }}</p>
        <p v-else-if="pricesState === 'unavailable'" class="card p-4 text-sm text-hint">{{ $t('premium.unavailable') }}</p>
        <div v-else-if="!prices" class="space-y-3"><div class="skeleton h-20" /><div class="skeleton h-20" /></div>
        <div v-else role="radiogroup" :aria-label="$t('premium.choose')" class="space-y-3">
          <button
            v-for="p in (['monthly', 'yearly'] as const)"
            :key="p"
            type="button"
            role="radio"
            :aria-checked="plan === p"
            class="pressable relative flex w-full items-center gap-3 rounded-2xl border-2 bg-surface px-4 py-4 text-left transition-colors"
            :class="plan === p ? 'border-accent' : 'border-transparent'"
            :style="{ boxShadow: 'var(--shadow-card)' }"
            @click="pick(p)"
          >
            <span
              v-if="p === 'yearly' && saved > 0"
              class="absolute -top-2.5 right-4 rounded-full bg-mint px-2.5 py-0.5 text-[11px] font-bold text-[#0b2b20]"
            >
              {{ $t('premium.yearly_badge', { n: saved }) }}
            </span>
            <span
              class="flex size-6 shrink-0 items-center justify-center rounded-full border-2"
              :class="plan === p ? 'border-accent bg-accent' : 'border-hint/50'"
            >
              <span v-if="plan === p" class="size-2.5 rounded-full bg-white" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="block font-semibold">{{ p === 'monthly' ? $t('premium.month_label') : $t('premium.year_label') }}</span>
              <span class="block text-xs text-hint">
                {{ p === 'monthly' ? $t('premium.monthly_note') : `${$t('premium.yearly_note')} · ${$t('premium.per_month', { stars: formatStars(perMonth) })}` }}
              </span>
            </span>
            <span class="display shrink-0 text-xl">{{ formatStars(priceOf(p)) }} ⭐</span>
          </button>
        </div>
      </section>

      <p v-if="pending" class="card p-4 text-sm" role="status">{{ $t('premium.pending') }}</p>
      <p class="px-1 text-center text-xs text-hint">{{ $t('premium.free_forever') }}</p>
    </template>
  </div>

  <FallbackButton v-if="!native && canBuy" :label="buttonText" :busy="!!buying" @click="getPremium" />
</template>

<style scoped>
.premium-hero {
  background:
    radial-gradient(120% 80% at 50% 0%, rgb(123 98 255 / 60%) 0%, transparent 60%),
    linear-gradient(160deg, #5b3df5 0%, #2c1a9e 60%, #1b1060 100%);
  box-shadow: 0 16px 40px rgb(67 39 214 / 35%);
}
.crown-glow {
  background: radial-gradient(circle, rgb(255 214 120 / 45%) 0%, rgb(255 214 120 / 0%) 70%);
}
.sparkle {
  position: absolute;
  color: #fff;
  opacity: 0.7;
  animation: twinkle 2.4s ease-in-out infinite;
}
.sparkle-1 { top: 14%; left: 12%; font-size: 14px; }
.sparkle-2 { top: 24%; right: 14%; font-size: 18px; animation-delay: 0.5s; }
.sparkle-3 { top: 52%; left: 8%; font-size: 10px; animation-delay: 1s; }
.sparkle-4 { top: 8%; right: 34%; font-size: 10px; animation-delay: 1.5s; }
.sparkle-5 { bottom: 18%; right: 9%; font-size: 12px; animation-delay: 0.8s; }
.sparkle-6 { bottom: 26%; left: 22%; font-size: 9px; animation-delay: 1.8s; }
@keyframes twinkle {
  50% {
    opacity: 0.15;
    transform: scale(0.7);
  }
}
.confetti span {
  position: absolute;
  top: -10px;
  width: 8px;
  height: 12px;
  border-radius: 2px;
  animation: fall 1.6s ease-in forwards;
}
@keyframes fall {
  to {
    top: 110%;
    transform: rotate(540deg);
    opacity: 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .sparkle,
  .confetti span {
    animation: none;
  }
  .confetti {
    display: none;
  }
}
</style>
