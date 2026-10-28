<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { backButton } from '@/lib/telegram'
import AppIcon from './AppIcon.vue'

/**
 * Modal bottom sheet shell: backdrop, slide-up, focus trap, Escape and Telegram
 * BackButton to close, focus returned to the opener. Sheets can stack (a picker
 * inside a form sheet); only the top one reacts to keys.
 */
/** `persistent`: can't be dismissed (no close button, backdrop, Escape or Back) until its own action closes it. */
const props = defineProps<{ title: string; persistent?: boolean }>()
const open = defineModel<boolean>('open', { required: true })
defineEmits<{ closed: [] }>()

const titleId = useId()
const panel = ref<HTMLElement>()
const sheetId = Symbol('sheet')
let returnFocus: HTMLElement | null = null

function close() {
  if (!props.persistent) open.value = false
}

function focusables(): HTMLElement[] {
  return [...(panel.value?.querySelectorAll<HTMLElement>('button, input, select, textarea') ?? [])].filter(
    (el) => !el.hasAttribute('disabled'),
  )
}

function onKeydown(e: KeyboardEvent) {
  if (openSheets[openSheets.length - 1] !== sheetId) return
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
  openSheets.push(sheetId)
  if (!props.persistent) backButton.show(close)
  document.addEventListener('keydown', onKeydown)
  document.body.style.overflow = 'hidden'
  void nextTick(() => {
    // [data-autofocus] lets the content choose (e.g. the selected option, not a search field).
    const marked = panel.value?.querySelector<HTMLElement>('[data-autofocus]')
    const target = marked?.matches('button, input, select, textarea')
      ? marked
      : (marked?.querySelector<HTMLElement>('input, button') ?? focusables()[0])
    target?.focus({ preventScroll: true })
    target?.scrollIntoView({ block: 'center' })
  })
}

function deactivate() {
  const i = openSheets.indexOf(sheetId)
  if (i !== -1) openSheets.splice(i, 1)
  backButton.hide(close)
  document.removeEventListener('keydown', onKeydown)
  if (!openSheets.length) document.body.style.overflow = ''
  returnFocus?.focus({ preventScroll: true })
  returnFocus = null
}

// immediate: a lazily mounted sheet may already be open on its first render.
watch(open, (isOpen, was) => (isOpen ? activate() : was && deactivate()), { immediate: true })
onBeforeUnmount(() => open.value && deactivate())
</script>

<script lang="ts">
const openSheets: symbol[] = []
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet" @after-leave="$emit('closed')">
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
              v-if="!persistent"
              type="button"
              class="pressable -mr-2 rounded-full p-2 text-hint"
              :aria-label="$t('common.close')"
              @click="close"
            >
              <AppIcon name="x" :size="20" />
            </button>
          </div>
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
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
