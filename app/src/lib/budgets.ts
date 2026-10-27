export type BudgetLevel = 'ok' | 'warn' | 'over'

/** Whole percent spent, rounded down so 99.6% doesn't read as "100%". */
export function budgetPercent(spentMinor: number, limitMinor: number): number {
  if (limitMinor <= 0) return 0
  return Math.floor((Math.max(0, spentMinor) * 100) / limitMinor)
}

/** Same thresholds as the bot's alerts: 80% and 100%. */
export function budgetLevel(spentMinor: number, limitMinor: number): BudgetLevel {
  if (spentMinor >= limitMinor) return 'over'
  if (spentMinor * 5 >= limitMinor * 4) return 'warn'
  return 'ok'
}

/**
 * A suggested daily spend that keeps the month within budget: remaining split over
 * today plus the days left, in whole so'm/dollars. 0 when nothing is left.
 */
export function dailyPace(remainingMinor: number, daysLeft: number): number {
  if (remainingMinor <= 0) return 0
  const days = Math.max(daysLeft + 1, 1)
  return Math.floor(remainingMinor / days / 100) * 100
}
