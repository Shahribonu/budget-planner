import type { VoiceItem } from '@/api/types'

/**
 * Amounts the server flags as shorthand ("kofe 18") get two quick picks in so'm:
 * as heard, and × 1000 (the usual meaning). USD gets no picks, only the highlight.
 */
export function reviewOptions(item: Pick<VoiceItem, 'amount_minor' | 'currency' | 'needs_review'>): number[] {
  if (!item.needs_review || item.currency !== 'UZS') return []
  return [item.amount_minor, item.amount_minor * 1000]
}

/** A row still waiting for a pick: flagged, has picks, not resolved yet. */
export function needsPick(item: Pick<VoiceItem, 'amount_minor' | 'currency' | 'needs_review'>, resolved: boolean): boolean {
  return !resolved && reviewOptions(item).length > 0
}

/** "Save all" is allowed only when no row is waiting for a pick. */
export function allResolved(rows: { item: Pick<VoiceItem, 'amount_minor' | 'currency' | 'needs_review'>; resolved: boolean }[]): boolean {
  return rows.every((r) => !needsPick(r.item, r.resolved))
}
