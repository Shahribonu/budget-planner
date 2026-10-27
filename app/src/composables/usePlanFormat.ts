import { useI18n } from 'vue-i18n'
import type { PlanItem } from '@/api/types'
import { formatMonth } from '@/lib/dates'
import { buildReason } from '@/lib/plan'
import { useMonthlyPlanStore } from '@/stores/monthlyPlan'
import { useFormat } from './useFormat'

/** Formatting shared by plan rows and sheets. */
export function usePlanFormat() {
  const { t } = useI18n()
  const { money, lang } = useFormat()
  const plan = useMonthlyPlanStore()

  function amountText(item: Pick<PlanItem, 'amount_min_minor' | 'amount_max_minor'>): string {
    const cur = plan.view?.currency ?? 'UZS'
    const { amount_min_minor: min, amount_max_minor: max } = item
    if (!max || max <= min) return money(min, cur)
    return `${money(min, cur, { bare: true })}–${money(max, cur)}`
  }

  function reason(item: PlanItem): { text: string; tone: 'ok' | 'neutral' | 'danger' } | null {
    const v = plan.view
    if (!item.decision || !v) return null
    const line = buildReason(item.decision, {
      planMonth: v.month,
      money: (m) => money(m, v.currency),
      month: (ym) => formatMonth(ym, lang.value),
    })
    const text = line.parts.map((p) => t(`mplan_reason.${p.key}`, p.params ?? {})).join(', ')
    return { text: text.charAt(0).toUpperCase() + text.slice(1), tone: line.tone }
  }

  return { amountText, reason }
}
