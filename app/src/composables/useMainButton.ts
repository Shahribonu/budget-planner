import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { mainButton } from '@/lib/telegram'

/**
 * Drives Telegram's MainButton while the calling component is mounted.
 * Returns `native`; when false the view renders its own in-page button.
 */
export function useMainButton(
  text: Readonly<Ref<string>>,
  busy: Readonly<Ref<boolean>>,
  onClick: () => void,
  visible: Readonly<Ref<boolean>> = ref(true),
  enabled: Readonly<Ref<boolean>> = ref(true),
) {
  const handler = () => {
    if (!busy.value && visible.value && enabled.value) onClick()
  }
  onMounted(() => {
    if (!mainButton.available) return
    mainButton.bind(text.value, handler)
    if (!visible.value) mainButton.setVisible(false)
    if (!enabled.value) mainButton.setEnabled(false)
  })
  onBeforeUnmount(() => {
    if (!mainButton.available) return
    mainButton.setEnabled(true)
    mainButton.unbind(handler)
  })
  watch(text, (t) => mainButton.setText(t))
  watch(busy, (b) => mainButton.setProgress(b))
  watch(visible, (v) => mainButton.setVisible(v))
  watch(enabled, (e) => mainButton.setEnabled(e))
  return { native: mainButton.available }
}
