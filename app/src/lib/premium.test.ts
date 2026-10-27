import { describe, expect, it } from 'vitest'
import { formatStars, monthsFree, pollUntil } from './premium'

describe('premium', () => {
  it('works out the months the yearly plan saves', () => {
    expect(monthsFree(150, 1500)).toBe(2)
    expect(monthsFree(150, 1800)).toBe(0)
    expect(monthsFree(150, 2000)).toBe(0)
    expect(monthsFree(0, 1500)).toBe(0)
  })

  it('groups star counts like money', () => {
    expect(formatStars(1500)).toBe('1 500')
    expect(formatStars(150)).toBe('150')
  })

  it('polls until the check passes, then stops', async () => {
    let calls = 0
    const ok = await pollUntil(async () => ++calls === 3, { sleep: async () => {} })
    expect(ok).toBe(true)
    expect(calls).toBe(3)
  })

  it('gives up after the last try', async () => {
    let calls = 0
    const slept: number[] = []
    const ok = await pollUntil(async () => (calls++, false), { tries: 4, intervalMs: 50, sleep: async (ms) => void slept.push(ms) })
    expect(ok).toBe(false)
    expect(calls).toBe(4)
    expect(slept).toEqual([50, 50, 50])
  })
})
