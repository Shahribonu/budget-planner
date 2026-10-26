<script setup lang="ts">
import { haptic } from '@/lib/telegram'
import AppIcon, { type IconName } from './AppIcon.vue'

const left: { name: string; icon: IconName; label: string }[] = [
  { name: 'home', icon: 'home', label: 'tabs.home' },
  { name: 'activity', icon: 'list', label: 'tabs.activity' },
]
const right: { name: string; icon: IconName; label: string }[] = [
  { name: 'plan', icon: 'target', label: 'tabs.plan' },
  { name: 'insights', icon: 'chart', label: 'tabs.insights' },
  { name: 'profile', icon: 'user', label: 'tabs.profile' },
]
</script>

<template>
  <nav
    class="fixed inset-x-0 bottom-0 z-20 border-t border-separator bg-bg"
    :style="{ paddingBottom: 'var(--safe-bottom)' }"
  >
    <div class="mx-auto grid h-(--tabbar-h) max-w-xl grid-cols-6 items-center">
      <RouterLink
        v-for="tab in left"
        :key="tab.name"
        :to="{ name: tab.name }"
        class="tab"
        active-class="tab-active"
        @click="haptic.selection()"
      >
        <AppIcon :name="tab.icon" :size="22" />
        <span class="tab-label">{{ $t(tab.label) }}</span>
      </RouterLink>
      <div class="flex justify-center">
        <RouterLink
          :to="{ name: 'tx-new' }"
          class="pressable flex size-13 items-center justify-center rounded-full bg-accent text-on-accent shadow-lg"
          :aria-label="$t('tabs.add')"
          @click="haptic.impact('medium')"
        >
          <AppIcon name="plus" :size="28" />
        </RouterLink>
      </div>
      <RouterLink
        v-for="tab in right"
        :key="tab.name"
        :to="{ name: tab.name }"
        class="tab"
        active-class="tab-active"
        @click="haptic.selection()"
      >
        <AppIcon :name="tab.icon" :size="22" />
        <span class="tab-label">{{ $t(tab.label) }}</span>
      </RouterLink>
    </div>
  </nav>
</template>

<style scoped>
@reference '../style.css';

.tab {
  @apply flex h-full min-w-0 flex-col items-center justify-center gap-0.5 text-hint;
}
.tab-active {
  @apply text-accent;
}
.tab-label {
  @apply w-full truncate px-0.5 text-center text-[10px] leading-tight font-medium;
}
</style>
