<script setup lang="ts">
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'

defineProps<{ title: string; icon: string }>()
</script>

<template>
  <PageHeader :title="$t(title)" />
  <EmptyState :icon="icon" :title="$t('common.coming_soon')" :body="$t('common.coming_soon_body')" />
</template>
