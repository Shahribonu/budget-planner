<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AccountCard from '@/components/AccountCard.vue'
import FallbackButton from '@/components/FallbackButton.vue'
import PageHeader from '@/components/PageHeader.vue'
import { useMainButton } from '@/composables/useMainButton'
import { useAccountsStore } from '@/stores/accounts'
import { useToastStore } from '@/stores/toast'

const { t } = useI18n()
const router = useRouter()
const accounts = useAccountsStore()
const toast = useToastStore()
const showArchived = ref(false)
const archived = computed(() => accounts.all.filter((a) => a.archived))

onMounted(() => accounts.load(true).catch(() => toast.error()))

// The one add action on this screen: Telegram's MainButton (FallbackButton outside Telegram).
const add = () => void router.push({ name: 'account-new' })
const { native } = useMainButton(
  computed(() => t('profile.add_account')),
  ref(false),
  add,
)
</script>

<template>
  <PageHeader :title="$t('profile.accounts')" back />

  <div class="grid grid-cols-2 gap-3 px-4 pb-6 [&>*]:w-auto">
    <AccountCard v-for="a in accounts.active" :key="a.id" :account="a" />
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

  <FallbackButton v-if="!native" :label="$t('profile.add_account')" @click="add" />
</template>
