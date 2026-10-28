<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { haptic } from '@/lib/telegram'
import { useToastStore } from '@/stores/toast'
import AppIcon, { type IconName } from './AppIcon.vue'

const { t } = useI18n()
const router = useRouter()
const toast = useToastStore()

const tabs: { name: string; icon: IconName; label: string }[] = [
  { name: 'home', icon: 'home', label: 'tabs.home' },
  { name: 'activity', icon: 'list', label: 'tabs.activity' },
  { name: 'plan', icon: 'target', label: 'tabs.plan' },
  { name: 'profile', icon: 'user', label: 'tabs.profile' },
]

// Capture button: tap adds a transaction; a long press hints at voice notes in the chat.
let pressTimer: ReturnType<typeof setTimeout> | undefined
let longPressed = false

function pressStart() {
  longPressed = false
  pressTimer = setTimeout(() => {
    longPressed = true
    haptic.impact('heavy')
    toast.show(t('tabs.voice_hint'))
  }, 550)
}

function pressEnd() {
  clearTimeout(pressTimer)
}

function go(navigate: (e?: MouseEvent) => unknown, e: MouseEvent) {
  haptic.selection()
  void navigate(e)
}

function capture() {
  if (longPressed) return
  haptic.impact('medium')
  void router.push({ name: 'tx-new' })
}
</script>

<template>
  <nav
    class="pointer-events-none fixed inset-x-0 bottom-0 z-20 px-4"
    :style="{ paddingBottom: 'calc(var(--safe-bottom) + 12px)' }"
    :aria-label="$t('tabs.nav')"
  >
    <div class="pointer-events-auto mx-auto flex max-w-xl items-center gap-3">
      <div class="flex h-16 flex-1 items-center justify-around rounded-full bg-surface px-2" :style="{ boxShadow: 'var(--shadow-float)' }">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.name"
          v-slot="{ isActive, isExactActive, navigate, href }"
          :to="{ name: tab.name }"
          custom
        >
          <a
            :href="href"
            class="pressable flex size-12 items-center justify-center rounded-full transition-colors"
            :class="(tab.name === 'home' ? isExactActive : isActive) ? 'bg-text text-surface' : 'text-hint'"
            :aria-label="$t(tab.label)"
            :aria-current="(tab.name === 'home' ? isExactActive : isActive) ? 'page' : undefined"
            @click="(e: MouseEvent) => go(navigate, e)"
          >
            <AppIcon :name="tab.icon" :size="22" />
          </a>
        </RouterLink>
      </div>
      <button
        type="button"
        class="pressable flex size-16 shrink-0 items-center justify-center rounded-full bg-mint text-[#0b2b20]"
        :style="{ boxShadow: '0 8px 24px rgb(43 217 160 / 45%)' }"
        :aria-label="$t('tabs.add')"
        @pointerdown="pressStart"
        @pointerup="pressEnd"
        @pointerleave="pressEnd"
        @pointercancel="pressEnd"
        @contextmenu.prevent
        @click="capture"
      >
        <AppIcon name="plus" :size="30" />
      </button>
    </div>
  </nav>
</template>
