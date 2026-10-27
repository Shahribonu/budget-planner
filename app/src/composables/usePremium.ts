import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError, api } from '@/api/client'
import type { BillingPlans, PremiumPlan } from '@/api/types'
import { pollUntil } from '@/lib/premium'
import { canPay, haptic, openInvoice } from '@/lib/telegram'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

// Prices are the same everywhere, so load them once per session.
const prices = ref<BillingPlans | null>(null)
const pricesState = ref<'idle' | 'loading' | 'ready' | 'unavailable'>('idle')

async function loadPrices(): Promise<void> {
  if (pricesState.value === 'ready' || pricesState.value === 'loading') return
  pricesState.value = 'loading'
  try {
    prices.value = await api.getBillingPlans()
    pricesState.value = 'ready'
  } catch {
    // 503: payments not configured. Anything else: also hide the buttons rather than guess prices.
    pricesState.value = 'unavailable'
  }
}

/** 'active': Premium is on. 'pending': paid, but the server hasn't recorded it yet. */
export type BuyResult = 'active' | 'pending' | null

/** Buy Premium with Telegram Stars. */
export function usePremium() {
  const { t } = useI18n()
  const user = useUserStore()
  const toast = useToastStore()
  const buying = ref<PremiumPlan | null>(null)

  async function buy(plan: PremiumPlan): Promise<BuyResult> {
    if (buying.value || !canPay) return null
    buying.value = plan
    try {
      const { url } = await api.createInvoice(plan)
      const status = await openInvoice(url)
      // cancelled / failed / pending: Telegram's own sheet already told the user.
      if (status !== 'paid') return null
      haptic.success()
      const active = await pollUntil(async () => (await user.load(true)).premium)
      toast.show(active ? t('premium.thanks') : t('premium.pending'))
      return active ? 'active' : 'pending'
    } catch (e) {
      haptic.error()
      toast.show(e instanceof ApiError && e.status === 503 ? t('premium.unavailable') : t('common.error'))
      return null
    } finally {
      buying.value = null
    }
  }

  return { buy, buying, canPay, prices, pricesState, loadPrices }
}
