import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { mainButton } from '@/lib/telegram'

/**
 * Drives Telegram's MainButton while the calling component is mounted.
 * Returns `native`; when false the view renders its own in-page button.
 */
export function useMainButton(text: Ref<string>, busy: Ref<boolean>, onClick: () => void, visible: Ref<boolean> = ref(true)) {
  const handler = () => {
    if (!busy.value && visible.value) onClick()
  }
  onMounted(() => {
    if (!mainButton.available) return
    mainButton.bind(text.value, handler)
    if (!visible.value) mainButton.setVisible(false)
  })
  onBeforeUnmount(() => mainButton.available && mainButton.unbind(handler))
  watch(text, (t) => mainButton.setText(t))
  watch(busy, (b) => mainButton.setProgress(b))
  watch(visible, (v) => mainButton.setVisible(v))
  return { native: mainButton.available }
}
