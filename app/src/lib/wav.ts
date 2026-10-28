/** Target format for voice uploads: what speech-to-text providers accept everywhere. */
export const WAV_RATE = 16000

/** Joins recorded PCM chunks into one buffer. */
export function concatChunks(chunks: Float32Array[]): Float32Array {
  const out = new Float32Array(chunks.reduce((n, c) => n + c.length, 0))
  let offset = 0
  for (const c of chunks) {
    out.set(c, offset)
    offset += c.length
  }
  return out
}

/**
 * Downsamples mono PCM by averaging each output sample's source window (a cheap
 * low-pass, enough for speech). Output length is floor(input * to / from).
 */
export function downsample(input: Float32Array, fromRate: number, toRate = WAV_RATE): Float32Array {
  if (fromRate === toRate) return input.slice()
  if (fromRate < toRate) throw new Error(`can't upsample ${fromRate} → ${toRate}`)
  const ratio = fromRate / toRate
  const out = new Float32Array(Math.floor(input.length / ratio))
  for (let i = 0; i < out.length; i++) {
    const start = Math.floor(i * ratio)
    const end = Math.min(input.length, Math.floor((i + 1) * ratio))
    let sum = 0
    for (let j = start; j < end; j++) sum += input[j]!
    out[i] = sum / Math.max(1, end - start)
  }
  return out
}

/** 16-bit PCM mono WAV with the standard 44-byte RIFF header. */
export function encodeWav(samples: Float32Array, sampleRate = WAV_RATE): Uint8Array<ArrayBuffer> {
  const dataBytes = samples.length * 2
  const buf = new ArrayBuffer(44 + dataBytes)
  const v = new DataView(buf)
  const ascii = (offset: number, s: string) => {
    for (let i = 0; i < s.length; i++) v.setUint8(offset + i, s.charCodeAt(i))
  }
  ascii(0, 'RIFF')
  v.setUint32(4, 36 + dataBytes, true)
  ascii(8, 'WAVE')
  ascii(12, 'fmt ')
  v.setUint32(16, 16, true) // fmt chunk size
  v.setUint16(20, 1, true) // PCM
  v.setUint16(22, 1, true) // mono
  v.setUint32(24, sampleRate, true)
  v.setUint32(28, sampleRate * 2, true) // byte rate
  v.setUint16(32, 2, true) // block align
  v.setUint16(34, 16, true) // bits per sample
  ascii(36, 'data')
  v.setUint32(40, dataBytes, true)
  for (let i = 0; i < samples.length; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]!))
    v.setInt16(44 + i * 2, s < 0 ? s * 0x8000 : s * 0x7fff, true)
  }
  return new Uint8Array(buf)
}

/** Root-mean-square level of a chunk, 0–1, for the recording indicator. */
export function rms(chunk: Float32Array): number {
  if (!chunk.length) return 0
  let sum = 0
  for (const s of chunk) sum += s * s
  return Math.sqrt(sum / chunk.length)
}
