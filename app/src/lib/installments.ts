/** Same rule as the server: equal payments in whole so'm/dollars, the last one takes the remainder. */
export function splitPayments(totalMinor: number, months: number): number[] {
  if (!Number.isInteger(months) || months < 1 || totalMinor <= 0) return []
  const unit = 100
  const each = Math.floor(totalMinor / months / unit) * unit
  const rows = Array<number>(months).fill(each)
  rows[months - 1] = totalMinor - each * (months - 1)
  return rows
}

/** Server rejects a plan that can't pay at least one whole unit per month. */
export function tooSmallForMonths(totalMinor: number, months: number): boolean {
  return totalMinor < months * 100
}

/** first + n months, keeping the day where possible (Jan 31 + 1 month = Feb 28), like the server. */
export function addMonthsClamped(iso: string, n: number): string {
  const [y, m, d] = iso.split('-').map(Number)
  const first = new Date(Date.UTC(y!, m! - 1 + n, 1))
  const lastDay = new Date(Date.UTC(first.getUTCFullYear(), first.getUTCMonth() + 1, 0)).getUTCDate()
  first.setUTCDate(Math.min(d!, lastDay))
  return first.toISOString().slice(0, 10)
}
