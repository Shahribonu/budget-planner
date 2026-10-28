<script setup lang="ts">
import { onMounted } from 'vue'
import CategoryIcon from '@/components/CategoryIcon.vue'
import PageHeader from '@/components/PageHeader.vue'
import { categoryLabel } from '@/i18n'
import { useCategoriesStore } from '@/stores/categories'

const categories = useCategoriesStore()
onMounted(() => void categories.load().catch(() => {}))
</script>

<template>
  <PageHeader :title="$t('profile.categories')" back />
  <div class="space-y-5 px-4 pb-6">
    <section v-for="kind in (['expense', 'income'] as const)" :key="kind">
      <h2 class="eyebrow mb-2 px-1">{{ $t(`types.${kind}`) }}</h2>
      <div class="card grid grid-cols-4 gap-y-4 px-2 py-4">
        <div v-for="c in categories.ofKind(kind)" :key="c.id" class="flex min-w-0 flex-col items-center gap-1.5">
          <CategoryIcon :category="c" :size="44" />
          <span class="line-clamp-2 w-full text-center text-[11px] leading-tight">{{ categoryLabel(c) }}</span>
        </div>
      </div>
    </section>
    <p class="px-1 text-xs text-hint">{{ $t('profile.categories_hint') }}</p>
  </div>
</template>
