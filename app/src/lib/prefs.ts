import { reactive, watch } from 'vue'

/** Per-device display preferences, kept in localStorage (never sent to the server). */
export interface Prefs {
  theme: 'system' | 'light' | 'dark'
  categoryStyle: 'icons' | 'emoji'
  hideAmounts: boolean
}

const KEY = 'bp.prefs'
const DEFAULTS: Prefs = { theme: 'system', categoryStyle: 'icons', hideAmounts: false }

function read(): Prefs {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) ?? '{}') as Partial<Prefs>
    return {
      theme: ['system', 'light', 'dark'].includes(raw.theme as string) ? (raw.theme as Prefs['theme']) : DEFAULTS.theme,
      categoryStyle: raw.categoryStyle === 'emoji' ? 'emoji' : 'icons',
      hideAmounts: raw.hideAmounts === true,
    }
  } catch {
    return { ...DEFAULTS }
  }
}

export const prefs = reactive<Prefs>(read())

watch(
  prefs,
  (p) => {
    try {
      localStorage.setItem(KEY, JSON.stringify(p))
    } catch {
      /* storage unavailable: prefs last for this session */
    }
  },
  { deep: true },
)

/** Forget everything this app stored on the device (after the account is deleted). */
export function clearLocalData(): void {
  try {
    for (const k of Object.keys(localStorage)) if (k.startsWith('bp.')) localStorage.removeItem(k)
  } catch {
    /* storage unavailable */
  }
}
