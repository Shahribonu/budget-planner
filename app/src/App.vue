<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '@/api/client'
import AppToast from '@/components/AppToast.vue'
import ConsentSheet from '@/components/ConsentSheet.vue'
import EmptyState from '@/components/EmptyState.vue'
import TabBar from '@/components/TabBar.vue'
import { useBack } from '@/composables/useBack'
import { backButton } from '@/lib/telegram'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const user = useUserStore()
const goBack = useBack()

watch(
  () => route.meta.sub,
  (sub) => (sub ? backButton.show(goBack) : backButton.hide(goBack)),
)

const authFailed = computed(() => user.error instanceof ApiError && user.error.status === 401)

function retry() {
  window.location.reload()
}
</script>

<template>
  <div
    class="mx-auto min-h-dvh max-w-xl"
    :style="{
      paddingTop: 'var(--safe-top)',
      paddingBottom: route.meta.tabs ? 'calc(var(--safe-bottom) + var(--tabbar-h) + 16px)' : undefined,
    }"
  >
    <EmptyState
      v-if="user.error && !user.me"
      icon="⚠️"
      :body="authFailed ? $t('common.auth_error') : $t('common.error')"
      :action="authFailed ? undefined : $t('common.retry')"
      class="pt-24"
      @action="retry"
    />
    <RouterView v-else />
  </div>
  <TabBar v-if="route.meta.tabs && user.me" />
  <AppToast />
  <ConsentSheet v-if="user.me" />
</template>
