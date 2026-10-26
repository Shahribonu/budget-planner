import { computed } from 'vue'
import type { Currency } from '@/api/types'
import { i18n } from '@/i18n'
import { formatDay, todayISO, addDays } from '@/lib/dates'
import { formatMoney, type FormatMoneyOptions } from '@/lib/money'
import { useUserStore } from '@/stores/user'

export function useFormat() {
  const user = useUserStore()
  const lang = computed(() => i18n.global.locale.value)

  function money(amountMinor: number, currency: Currency, opts?: FormatMoneyOptions): string {
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
