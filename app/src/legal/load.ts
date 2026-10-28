import type { Language } from '@/api/types'
import type { LegalDoc } from './content'

export type { LegalDoc }

/** The texts are ~33 KB, so they load only when a legal page opens (a separate chunk). */
export async function loadLegal(lang: Language, doc: 'privacy' | 'terms'): Promise<LegalDoc | null> {
  const { legal } = await import('./content')
  return legal[lang]?.[doc] ?? legal['uz-Latn']?.[doc] ?? null
}
