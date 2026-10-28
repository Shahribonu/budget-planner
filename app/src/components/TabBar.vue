<script setup lang="ts">
import { useRouter } from 'vue-router'
import { haptic } from '@/lib/telegram'
import { useVoiceStore } from '@/stores/voice'
import AppIcon, { type IconName } from './AppIcon.vue'

const router = useRouter()

const tabs: { name: string; icon: IconName; label: string }[] = [
  { name: 'home', icon: 'home', label: 'tabs.home' },
  { name: 'activity', icon: 'list', label: 'tabs.activity' },
  { name: 'plan', icon: 'target', label: 'tabs.plan' },
  { name: 'profile', icon: 'user', label: 'tabs.profile' },
]

const voice = useVoiceStore()

function go(navigate: (e?: MouseEvent) => unknown, e: MouseEvent) {
  haptic.selection()
  void navigate(e)
}

function capture() {
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
    <!-- items-end: the + lines up with the bar; the mic floats above it in the same column -->
    <div class="mx-auto flex max-w-xl items-end gap-3">
      <div
        class="pointer-events-auto flex h-16 flex-1 items-center justify-around rounded-full bg-surface px-2"
        :style="{ boxShadow: 'var(--shadow-float)' }"
      >
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
      <div class="flex shrink-0 flex-col items-center gap-3">
        <!-- Secondary: type a transaction by hand -->
        <button
          type="button"
          class="pressable pointer-events-auto flex size-13 items-center justify-center rounded-full bg-accent text-white"
          :style="{ boxShadow: '0 6px 18px rgb(91 61 245 / 35%)' }"
          :aria-label="$t('tabs.add')"
          @click="capture"
        >
          <AppIcon name="plus" :size="24" />
        </button>
        <!-- Primary: voice note, in line with the bar -->
        <button
          type="button"
          class="pressable pointer-events-auto flex size-16 items-center justify-center rounded-full bg-mint text-[#0b2b20]"
          :style="{ boxShadow: '0 8px 24px rgb(43 217 160 / 45%)' }"
          :aria-label="$t('voice.mic')"
          @click="voice.begin()"
        >
          <AppIcon name="mic" :size="28" />
        </button>
      </div>
    </div>
  </nav>
</template>
