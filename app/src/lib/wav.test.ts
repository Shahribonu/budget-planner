import { describe, expect, it } from 'vitest'
import { concatChunks, downsample, encodeWav, rms } from './wav'

const text = (b: Uint8Array, from: number, len: number) => String.fromCharCode(...b.slice(from, from + len))

describe('encodeWav', () => {
  it('writes a 44-byte PCM16 mono header', () => {
    const wav = encodeWav(new Float32Array(10), 16000)
    const v = new DataView(wav.buffer)
    expect(wav.length).toBe(44 + 20)
    expect(text(wav, 0, 4)).toBe('RIFF')
    expect(v.getUint32(4, true)).toBe(36 + 20)
    expect(text(wav, 8, 4)).toBe('WAVE')
    expect(text(wav, 12, 4)).toBe('fmt ')
    expect(v.getUint32(16, true)).toBe(16)
    expect(v.getUint16(20, true)).toBe(1) // PCM
    expect(v.getUint16(22, true)).toBe(1) // mono
    expect(v.getUint32(24, true)).toBe(16000)
    expect(v.getUint32(28, true)).toBe(32000)
    expect(v.getUint16(32, true)).toBe(2)
    expect(v.getUint16(34, true)).toBe(16)
    expect(text(wav, 36, 4)).toBe('data')
    expect(v.getUint32(40, true)).toBe(20)
  })

  it('clamps and scales samples to 16-bit', () => {
    const wav = encodeWav(new Float32Array([1, -1, 0, 2, -2]))
    const v = new DataView(wav.buffer)
    expect(v.getInt16(44, true)).toBe(32767)
    expect(v.getInt16(46, true)).toBe(-32768)
    expect(v.getInt16(48, true)).toBe(0)
    expect(v.getInt16(50, true)).toBe(32767)
    expect(v.getInt16(52, true)).toBe(-32768)
  })
})

describe('downsample', () => {
  it('produces the right number of samples', () => {
    expect(downsample(new Float32Array(48000), 48000).length).toBe(16000)
    expect(downsample(new Float32Array(44100), 44100).length).toBe(16000)
    expect(downsample(new Float32Array(1000), 16000).length).toBe(1000)
  })

  it('averages each window', () => {
    expect(Array.from(downsample(new Float32Array([1, 1, 1, 0, 0, 0]), 48000))).toEqual([1, 0])
  })

  it('refuses to upsample', () => {
    expect(() => downsample(new Float32Array(10), 8000)).toThrow()
  })
})

describe('helpers', () => {
  it('joins chunks and measures level', () => {
    expect(Array.from(concatChunks([new Float32Array([1, 2]), new Float32Array([3])]))).toEqual([1, 2, 3])
    expect(rms(new Float32Array([0.5, -0.5]))).toBeCloseTo(0.5)
    expect(rms(new Float32Array())).toBe(0)
  })
})
