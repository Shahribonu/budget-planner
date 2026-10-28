<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AccountCard from '@/components/AccountCard.vue'
import AppIcon from '@/components/AppIcon.vue'
import PageHeader from '@/components/PageHeader.vue'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'

const accounts = useAccountsStore()
const toast = useToastStore()
const showArchived = ref(false)
const archived = computed(() => accounts.all.filter((a) => a.archived))

onMounted(() => accounts.load(true).catch(() => toast.error()))
</script>

<template>
  <PageHeader :title="$t('profile.accounts')" back>
    <RouterLink :to="{ name: 'account-new' }" class="pressable rounded-full p-2 text-link" :aria-label="$t('profile.add_account')">
      <AppIcon name="plus" />
    </RouterLink>
  </PageHeader>

  <div class="grid grid-cols-2 gap-3 px-4 pb-6 [&>*]:w-auto">
    <AccountCard v-for="a in accounts.active" :key="a.id" :account="a" />
    <RouterLink
      :to="{ name: 'account-new' }"
      class="pressable flex h-32 flex-col items-center justify-center gap-1 rounded-3xl border-2 border-dashed border-accent/40 bg-surface text-accent"
    >
      <AppIcon name="plus" :size="26" />
      <span class="text-xs font-semibold">{{ $t('profile.add_account') }}</span>
    </RouterLink>
    <template v-if="showArchived">
      <AccountCard v-for="a in archived" :key="a.id" :account="a" class="opacity-60" />
    </template>
  </div>
  <button
    v-if="archived.length"
    type="button"
    class="mx-auto block px-4 pb-6 text-sm text-hint"
    :aria-pressed="showArchived"
    @click="showArchived = !showArchived"
  >
    {{ $t('profile.show_archived') }} ({{ archived.length }})
  </button>
</template>
