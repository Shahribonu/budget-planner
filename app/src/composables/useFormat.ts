import { computed } from 'vue'
import type { Currency } from '@/api/types'
import { i18n } from '@/i18n'
import { formatDay, todayISO, addDays } from '@/lib/dates'
import { formatMoney, maskMoney, type FormatMoneyOptions } from '@/lib/money'
import { prefs } from '@/lib/prefs'
import { useUserStore } from '@/stores/user'

export function useFormat() {
  const user = useUserStore()
  const lang = computed(() => i18n.global.locale.value)

  /** Every displayed amount goes through here, so "hide balance" masks the whole app. */
  function money(amountMinor: number, currency: Currency, opts?: FormatMoneyOptions): string {
    if (prefs.hideAmounts) return maskMoney(currency, lang.value, opts)
    return formatMoney(amountMinor, currency, lang.value, opts)
  }

  function today(): string {
    return todayISO(user.me?.timezone)
  }

  function dayLabel(iso: string): string {
    const t = today()
    if (iso === t) return i18n.global.t('common.today')
    if (iso === addDays(t, -1)) return i18n.global.t('common.yesterday')
    return formatDay(iso, lang.value, iso.slice(0, 4) !== t.slice(0, 4))
  }

  return { lang, money, today, dayLabel }
}
