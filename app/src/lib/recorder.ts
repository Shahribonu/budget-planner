import { concatChunks, downsample, encodeWav, rms, WAV_RATE } from './wav'

/** The longest recording the server accepts. */
export const MAX_SECONDS = 60

type AudioContextCtor = typeof AudioContext
const Ctx: AudioContextCtor | undefined =
  typeof window === 'undefined'
    ? undefined
    : (window.AudioContext ?? (window as unknown as { webkitAudioContext?: AudioContextCtor }).webkitAudioContext)

// Posts each 128-frame block of channel 0 to the main thread.
const WORKLET = `class Tap extends AudioWorkletProcessor {
  process(inputs) {
    const ch = inputs[0] && inputs[0][0]
    if (ch) this.port.postMessage(ch.slice(0))
    return true
  }
}
registerProcessor('bp-tap', Tap)`

export class MicDeniedError extends Error {}
export class MicUnsupportedError extends Error {}

/**
 * Records mono PCM from the microphone and returns a 16 kHz 16-bit WAV.
 * Not MediaRecorder: Android WebViews only give webm/opus, which providers reject.
 *
 * Create it synchronously inside the tap handler: iOS only lets an AudioContext
 * run when it is created (or resumed) during a user gesture.
 */
export class WavRecorder {
  private ctx: AudioContext | null
  private stream: MediaStream | null = null
  private nodes: AudioNode[] = []
  private chunks: Float32Array[] = []
  private closed = false
  onLevel: (level: number) => void = () => {}

  constructor() {
    if (!Ctx || !navigator.mediaDevices?.getUserMedia) throw new MicUnsupportedError('no audio capture')
    this.ctx = new Ctx()
    void this.ctx.resume?.()
  }

  get sampleRate(): number {
    return this.ctx?.sampleRate ?? WAV_RATE
  }

  async start(): Promise<void> {
    let stream: MediaStream
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true },
      })
    } catch (e) {
      this.close()
      const name = e instanceof DOMException ? e.name : ''
      if (name === 'NotAllowedError' || name === 'SecurityError') throw new MicDeniedError(name)
      throw e
    }
    if (this.closed || !this.ctx) {
      stream.getTracks().forEach((t) => t.stop())
      return
    }
    this.stream = stream
    const ctx = this.ctx
    const source = ctx.createMediaStreamSource(stream)
    const take = (chunk: Float32Array) => {
      if (this.closed) return
      this.chunks.push(chunk)
      this.onLevel(Math.min(1, rms(chunk) * 4))
    }

    if (ctx.audioWorklet) {
      const url = URL.createObjectURL(new Blob([WORKLET], { type: 'application/javascript' }))
      try {
        await ctx.audioWorklet.addModule(url)
        const node = new AudioWorkletNode(ctx, 'bp-tap')
        node.port.onmessage = (e: MessageEvent<Float32Array>) => take(e.data)
        source.connect(node)
        this.nodes = [source, node]
        return
      } catch {
        // Some WebViews expose audioWorklet but fail to load it; fall through.
      } finally {
        URL.revokeObjectURL(url)
      }
    }
    // Fallback: deprecated but still the only option on older WebViews.
    const proc = ctx.createScriptProcessor(4096, 1, 1)
    proc.onaudioprocess = (e) => take(new Float32Array(e.inputBuffer.getChannelData(0)))
    source.connect(proc)
    proc.connect(ctx.destination) // ScriptProcessor only runs while connected
    this.nodes = [source, proc]
  }

  /** Seconds recorded so far. */
  get seconds(): number {
    return this.chunks.reduce((n, c) => n + c.length, 0) / this.sampleRate
  }

  /** Stops and returns the WAV. */
  stop(): Blob {
    const rate = this.sampleRate
    const pcm = concatChunks(this.chunks)
    this.close()
    const wav = encodeWav(rate > WAV_RATE ? downsample(pcm, rate) : pcm, rate > WAV_RATE ? WAV_RATE : rate)
    return new Blob([wav], { type: 'audio/wav' })
  }

  /** Releases the microphone and the audio context. Safe to call more than once. */
  close(): void {
    if (this.closed) return
    this.closed = true
    for (const n of this.nodes) n.disconnect()
    this.nodes = []
    this.stream?.getTracks().forEach((t) => t.stop())
    this.stream = null
    void this.ctx?.close().catch(() => {})
    this.ctx = null
    this.chunks = []
  }
}
