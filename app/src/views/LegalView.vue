<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import SkeletonList from '@/components/SkeletonList.vue'
import { currentLanguage } from '@/i18n'
import { loadLegal, type LegalDoc } from '@/legal/load'
import { formatDay } from '@/lib/dates'

const route = useRoute()
const kind = computed(() => (route.params.doc === 'terms' ? 'terms' : 'privacy'))
const doc = ref<LegalDoc | null>(null)
const loading = ref(true)
// `updated` is an ISO date; show it in the reader's language.
const updated = computed(() => {
  const u = doc.value?.updated ?? ''
  return /^\d{4}-\d{2}-\d{2}$/.test(u) ? formatDay(u, currentLanguage(), true) : u
})

watch(
  kind,
  async (k) => {
    loading.value = true
    try {
      doc.value = await loadLegal(currentLanguage(), k)
    } catch {
      doc.value = null
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
</script>

<template>
  <PageHeader :title="doc?.title ?? $t(kind === 'terms' ? 'legal.terms' : 'legal.privacy')" back />

  <SkeletonList v-if="loading" :rows="5" />
  <EmptyState v-else-if="!doc" icon="📄" :body="$t('legal.unavailable')" />

  <article v-else class="px-4 pb-10">
    <p class="eyebrow mb-4 px-1">{{ $t('legal.updated', { date: updated }) }}</p>
    <div class="card space-y-6 px-5 py-6">
      <section v-for="(s, i) in doc.sections" :key="i">
        <h2 class="mb-2 text-[17px] font-bold">{{ s.heading }}</h2>
        <p v-for="(para, j) in s.paragraphs" :key="j" class="mb-3 text-[15px] leading-relaxed text-text/85 last:mb-0">{{ para }}</p>
      </section>
    </div>
  </article>
</template>
