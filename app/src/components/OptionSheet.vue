<script setup lang="ts" generic="T extends string | number">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { backButton, haptic } from '@/lib/telegram'
import AppIcon from './AppIcon.vue'

export interface SheetOption<V> {
  value: V
  label: string
  /** Emoji or short text shown before the label. */
  icon?: string
  /** Secondary line under the label. */
  hint?: string
  /** Options with the same group are listed under one heading, in order. */
  group?: string
}

const props = defineProps<{
  title: string
  options: SheetOption<T>[]
  /** When set, an extra first row selects `null` (e.g. "All"). */
  allLabel?: string
  searchable?: boolean
}>()
const open = defineModel<boolean>('open', { required: true })
const model = defineModel<T | null>({ required: true })

const titleId = useId()
const panel = ref<HTMLElement>()
const query = ref('')
let returnFocus: HTMLElement | null = null

const sections = computed(() => {
  const q = query.value.trim().toLowerCase()
  const out: { group?: string; items: SheetOption<T>[] }[] = []
  for (const opt of props.options) {
    if (q && !`${opt.label} ${opt.hint ?? ''}`.toLowerCase().includes(q)) continue
    const last = out[out.length - 1]
    if (last && last.group === opt.group) last.items.push(opt)
    else out.push({ group: opt.group, items: [opt] })
  }
  return out
})

function close() {
  open.value = false
}

function select(value: T | null) {
  haptic.impact('light')
  model.value = value
  close()
}

function focusables(): HTMLElement[] {
  return [...(panel.value?.querySelectorAll<HTMLElement>('button, input') ?? [])].filter((el) => !el.hasAttribute('disabled'))
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
  } else if (e.key === 'Tab') {
    const els = focusables()
    const first = els[0]
    const last = els[els.length - 1]
    if (!first || !last) return
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }
}

function activate() {
  returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  backButton.show(close)
  document.addEventListener('keydown', onKeydown)
  document.body.style.overflow = 'hidden'
  void nextTick(() => {
    // Focus the selected row, not the search field, so mobile keyboards stay closed.
    const target =
      panel.value?.querySelector<HTMLElement>('[aria-pressed="true"]') ?? panel.value?.querySelector<HTMLElement>('.option')
    target?.focus({ preventScroll: true })
    target?.scrollIntoView({ block: 'center' })
  })
}

function deactivate() {
  backButton.hide(close)
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
  query.value = ''
  returnFocus?.focus({ preventScroll: true })
  returnFocus = null
}

watch(open, (isOpen) => (isOpen ? activate() : deactivate()))
onBeforeUnmount(() => open.value && deactivate())
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet">
      <div v-if="open" class="fixed inset-0 z-40 flex items-end justify-center bg-black/40" @click.self="close">
        <div
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          class="panel flex max-h-[85dvh] w-full max-w-xl flex-col rounded-t-2xl bg-bg text-text shadow-2xl"
          :style="{ paddingBottom: 'var(--safe-bottom)' }"
        >
          <div class="mx-auto mt-2 h-1 w-10 shrink-0 rounded-full bg-hint/40" aria-hidden="true" />
          <div class="flex shrink-0 items-center gap-2 px-4 pt-2 pb-2">
            <h2 :id="titleId" class="min-w-0 flex-1 truncate text-lg font-semibold">{{ title }}</h2>
            <button
              type="button"
              class="pressable -mr-2 rounded-full p-2 text-hint"
              :aria-label="$t('common.close')"
              @click="close"
            >
              <AppIcon name="x" :size="20" />
            </button>
          </div>

          <label v-if="searchable" class="mx-4 mb-2 flex shrink-0 items-center gap-2 rounded-xl bg-page px-3 py-2.5">
            <AppIcon name="search" :size="18" class="text-hint" />
            <input
              v-model="query"
              type="search"
              enterkeyhint="search"
              :placeholder="$t('common.search')"
              :aria-label="$t('common.search')"
              class="min-w-0 flex-1 bg-transparent outline-none placeholder:text-hint"
            />
          </label>

          <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain pb-2">
            <button
              v-if="allLabel && !query"
              type="button"
              class="option"
              :aria-pressed="model === null"
              @click="select(null)"
            >
              <span class="min-w-0 flex-1 truncate font-medium">{{ allLabel }}</span>
              <AppIcon v-if="model === null" name="check" :size="20" class="shrink-0 text-accent" />
            </button>

            <template v-for="section in sections" :key="section.group ?? ''">
              <h3 v-if="section.group" class="px-4 pt-3 pb-1 text-xs font-semibold tracking-wide text-hint uppercase">
                {{ section.group }}
              </h3>
              <button
                v-for="opt in section.items"
                :key="opt.value"
                type="button"
                class="option"
                :aria-pressed="model === opt.value"
                @click="select(opt.value)"
              >
                <span v-if="opt.icon" class="w-7 shrink-0 text-center text-xl" aria-hidden="true">{{ opt.icon }}</span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-medium">{{ opt.label }}</span>
                  <span v-if="opt.hint" class="block truncate text-sm text-hint">{{ opt.hint }}</span>
                </span>
                <AppIcon v-if="model === opt.value" name="check" :size="20" class="shrink-0 text-accent" />
              </button>
            </template>

            <p v-if="!sections.length" class="px-4 py-6 text-center text-sm text-hint">{{ $t('common.no_results') }}</p>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
@reference '../style.css';

.option {
  @apply flex min-h-12 w-full items-center gap-3 px-4 py-2 text-left outline-none active:bg-page focus-visible:bg-page;
}
.option[aria-pressed='true'] {
  @apply bg-accent/10;
}

.sheet-enter-active,
.sheet-leave-active {
  transition: opacity 250ms ease;
}
.sheet-enter-active .panel,
.sheet-leave-active .panel {
  transition: transform 250ms cubic-bezier(0.2, 0.8, 0.2, 1);
}
.sheet-enter-from,
.sheet-leave-to {
  opacity: 0;
}
.sheet-enter-from .panel,
.sheet-leave-to .panel {
  transform: translateY(100%);
}

@media (prefers-reduced-motion: reduce) {
  .sheet-enter-active,
  .sheet-leave-active,
  .sheet-enter-active .panel,
  .sheet-leave-active .panel {
    transition-duration: 1ms;
  }
  .sheet-enter-from .panel,
  .sheet-leave-to .panel {
    transform: none;
  }
}
</style>
