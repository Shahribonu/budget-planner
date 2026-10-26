// Typed wrapper around the subset of window.Telegram.WebApp we use.
// https://core.telegram.org/bots/webapps

type ImpactStyle = 'light' | 'medium' | 'heavy' | 'rigid' | 'soft'
type NotificationType = 'error' | 'success' | 'warning'

interface BottomButton {
  setText(text: string): BottomButton
  show(): BottomButton
  hide(): BottomButton
  enable(): BottomButton
  disable(): BottomButton
  showProgress(leaveActive?: boolean): BottomButton
  hideProgress(): BottomButton
  onClick(cb: () => void): BottomButton
  offClick(cb: () => void): BottomButton
}

interface BackButtonApi {
  isVisible: boolean
  show(): BackButtonApi
  hide(): BackButtonApi
  onClick(cb: () => void): BackButtonApi
  offClick(cb: () => void): BackButtonApi
}

interface WebApp {
  initData: string
  initDataUnsafe: { user?: { id: number; first_name: string; language_code?: string } }
  platform: string
  colorScheme: 'light' | 'dark'
  isVersionAtLeast(version: string): boolean
  ready(): void
  expand(): void
  disableVerticalSwipes?(): void
  setHeaderColor?(color: string): void
  setBackgroundColor?(color: string): void
  onEvent(event: 'themeChanged', cb: () => void): void
  BackButton: BackButtonApi
  MainButton: BottomButton
  HapticFeedback: {
    impactOccurred(style: ImpactStyle): void
    notificationOccurred(type: NotificationType): void
    selectionChanged(): void
  }
}

declare global {
  interface Window {
    Telegram?: { WebApp?: WebApp }
  }
}

const webApp: WebApp | undefined = window.Telegram?.WebApp

/** True only when running inside a Telegram client (the script also loads in plain browsers). */
export const inTelegram = !!webApp && webApp.platform !== 'unknown' && webApp.initData !== ''

const main = inTelegram ? webApp?.MainButton : undefined

export function initData(): string {
  return webApp?.initData ?? ''
}

export function telegramLanguageCode(): string | undefined {
  return webApp?.initDataUnsafe.user?.language_code
}

export function init(): void {
  applyColorScheme()
  if (!inTelegram || !webApp) return
  webApp.ready()
  webApp.expand()
  if (webApp.isVersionAtLeast('7.7')) webApp.disableVerticalSwipes?.()
  if (webApp.isVersionAtLeast('6.1')) {
    webApp.setHeaderColor?.('secondary_bg_color')
    webApp.setBackgroundColor?.('secondary_bg_color')
  }
  webApp.onEvent('themeChanged', applyColorScheme)
}

function applyColorScheme(): void {
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const scheme = inTelegram && webApp ? webApp.colorScheme : media.matches ? 'dark' : 'light'
  document.documentElement.dataset.scheme = scheme
  if (!inTelegram) media.onchange = applyColorScheme
}

export const backButton = {
  show(onClick: () => void): void {
    if (!inTelegram || !webApp) return
    webApp.BackButton.offClick(onClick).onClick(onClick).show()
  },
  hide(onClick: () => void): void {
    if (!inTelegram || !webApp) return
    webApp.BackButton.offClick(onClick).hide()
  },
}

export const mainButton = {
  available: inTelegram,
  bind(text: string, onClick: () => void): void {
    main?.setText(text).onClick(onClick).show()
  },
  unbind(onClick: () => void): void {
    main?.offClick(onClick).hideProgress().hide()
  },
  setText(text: string): void {
    main?.setText(text)
  },
  setEnabled(enabled: boolean): void {
    if (enabled) main?.enable()
    else main?.disable()
  },
  setProgress(busy: boolean): void {
    if (busy) main?.showProgress(false)
    else main?.hideProgress()
  },
}

export const haptic = {
  success: () => inTelegram && webApp?.HapticFeedback.notificationOccurred('success'),
  error: () => inTelegram && webApp?.HapticFeedback.notificationOccurred('error'),
  warning: () => inTelegram && webApp?.HapticFeedback.notificationOccurred('warning'),
  impact: (style: ImpactStyle = 'light') => inTelegram && webApp?.HapticFeedback.impactOccurred(style),
  selection: () => inTelegram && webApp?.HapticFeedback.selectionChanged(),
}
