import { ApiError } from '@/api/client'

/** The 422 field and code of an API error, or nulls for anything else. */
export function invalidField(e: unknown): { field: string | null; code: string | null; message: string } {
  if (e instanceof ApiError && e.status === 422) return { field: e.field ?? null, code: e.code, message: e.message }
  return { field: null, code: null, message: '' }
}
