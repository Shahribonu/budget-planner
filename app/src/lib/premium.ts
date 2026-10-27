/** Whole months of the monthly price the yearly plan saves, e.g. 12 × 150 − 1 500 = 300 Stars → 2. */
export function monthsFree(monthlyStars: number, yearlyStars: number): number {
  if (monthlyStars <= 0) return 0
  return Math.max(0, Math.floor((monthlyStars * 12 - yearlyStars) / monthlyStars))
}

/** "1 500" with the same narrow no-break space as money. */
export function formatStars(n: number): string {
  return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ' ')
}

/** Poll until `check` is true: the payment webhook can land after Telegram reports "paid". */
export async function pollUntil(
  check: () => Promise<boolean>,
  { tries = 10, intervalMs = 1000, sleep = (ms: number) => new Promise((r) => setTimeout(r, ms)) } = {},
): Promise<boolean> {
  for (let i = 0; i < tries; i++) {
    if (await check()) return true
    if (i < tries - 1) await sleep(intervalMs)
  }
  return false
}
