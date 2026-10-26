import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { mainButton } from '@/lib/telegram'

/**
 * Drives Telegram's MainButton while the calling component is mounted.
 * Returns `native`; when false the view renders its own in-page button.
 */
export function useMainButton(text: Ref<string>, busy: Ref<boolean>, onClick: () => void) {
  const handler = () => {
    if (!busy.value) onClick()
  }
  onMounted(() => mainButton.available && mainButton.bind(text.value, handler))
  onBeforeUnmount(() => mainButton.available && mainButton.unbind(handler))
  watch(text, (t) => mainButton.setText(t))
  watch(busy, (b) => mainButton.setProgress(b))
  return { native: mainButton.available }
}
