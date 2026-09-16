import { describe, it, expect, vi } from 'vitest'
import { createCounterAudio } from './useCounterAudio'

/**
 * fake AudioContext —— jsdom 没有真 AudioContext，所以 ctx 必须由外部工厂注入。
 * 每个 fake ctx 自带调用计数，用来断言「谁在什么时候碰了 ctx」。
 */
function createFakeCtx({ state = 'running', resumeDelay = 0, resumeNever = false } = {}) {
  const ctx = {
    state,
    sampleRate: 44100,
    currentTime: 0,
    destination: { id: 'destination' },
    resumeCalls: 0,
    closeCalls: 0,
    createBufferCalls: [],
    startedSources: [],
    resume() {
      ctx.resumeCalls += 1
      if (resumeNever) return new Promise(() => {})
      return new Promise((resolve) => {
        setTimeout(() => {
          ctx.state = 'running'
          resolve()
        }, resumeDelay)
      })
    },
    close() {
      ctx.closeCalls += 1
      ctx.state = 'closed'
      return Promise.resolve()
    },
    createBuffer(channels, length, sampleRate) {
      ctx.createBufferCalls.push({ channels, length, sampleRate })
      return { channels, length, sampleRate, getChannelData: () => new Float32Array(length) }
    },
    createBufferSource() {
      const source = {
        buffer: null,
        connect() {},
        start() {
          ctx.startedSources.push(source)
        }
      }
      return source
    },
    createGain() {
      return { gain: { setValueAtTime() {} }, connect() {} }
    }
  }
  return ctx
}

function makeAudio(options = {}) {
  const created = []
  const ctxFactory = vi.fn(() => {
    const ctx = createFakeCtx(options.ctxOptions)
    created.push(ctx)
    return ctx
  })
  const createBuffers = vi.fn((ctx) => {
    const map = new Map()
    map.set('mokugyo', ctx.createBuffer(1, 64, ctx.sampleRate || 44100))
    map.set('bell', ctx.createBuffer(1, 64, ctx.sampleRate || 44100))
    return map
  })

  const audio = createCounterAudio({
    ctxFactory,
    createBuffers,
    resumeTimeoutMs: options.resumeTimeoutMs ?? 250
  })

  return { audio, ctxFactory, createBuffers, created }
}

describe('useCounterAudio · 首击不丢（核心回归）', () => {
  it('#26 初始 suspended → play() 调用 resume()，且 resume resolve 之前不 start 音源', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'suspended', resumeDelay: 5 } })
    const pending = audio.play('mokugyo', 1)

    // 同步阶段：resume 已经被调用，但音源还没排程
    expect(created[0].resumeCalls).toBe(1)
    expect(created[0].startedSources.length).toBe(0)

    await pending
    expect(created[0].startedSources.length).toBe(1)
  })

  it('#27 state === interrupted（WebKit 专有）→ 仍然调用 resume()', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'interrupted', resumeDelay: 1 } })
    await audio.play('mokugyo', 1)
    expect(created[0].resumeCalls).toBe(1)
    expect(created[0].startedSources.length).toBe(1)
  })
})

describe('useCounterAudio · 降级阶梯', () => {
  it('#28 resume 超时 → 不抛异常、跳过本次播放，连续 2 次后 isDegraded', async () => {
    const { audio, created } = makeAudio({
      ctxOptions: { state: 'suspended', resumeNever: true },
      resumeTimeoutMs: 15
    })

    await expect(audio.play('mokugyo', 1)).resolves.toBeUndefined()
    expect(created[0].startedSources.length).toBe(0)
    expect(audio.isDegraded.value).toBe(false)

    await audio.play('mokugyo', 1)
    expect(audio.isDegraded.value).toBe(true)
  })

  it('#29 state === closed → 新建 ctx 且缓冲在新 ctx 上重建', async () => {
    const { audio, created, createBuffers } = makeAudio({ ctxOptions: { state: 'running' } })

    await audio.play('mokugyo', 1)
    const firstCtx = created[0]
    expect(firstCtx.createBufferCalls.length).toBe(2)

    firstCtx.state = 'closed'
    await audio.play('mokugyo', 1)

    expect(created.length).toBe(2)
    expect(createBuffers).toHaveBeenCalledTimes(2)
    expect(created[1].createBufferCalls.length).toBe(2)
    expect(created[1].startedSources.length).toBe(1)
  })

  it('#30 ctxFactory 抛异常 → play() 不抛（计数路径不中断）', async () => {
    const audio = createCounterAudio({
      ctxFactory: () => {
        throw new Error('too many AudioContexts')
      },
      createBuffers: () => new Map()
    })

    expect(() => audio.play('mokugyo', 1)).not.toThrow()
    await expect(audio.play('mokugyo', 1)).resolves.toBeUndefined()
  })

  it('#31 rebuild() 关闭旧 ctx 并重建，任何时刻只存在 1 个 ctx', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'running' } })

    await audio.unlock()
    const ok = await audio.rebuild()

    expect(ok).toBe(true)
    expect(created[0].closeCalls).toBe(1)
    expect(created[0].state).toBe('closed')
    expect(created.length).toBe(2)
    expect(created[1].state).toBe('running')
  })

  it('isDegraded 为真时 play() 直接返回，不再触碰 ctx', async () => {
    const { audio, created } = makeAudio({
      ctxOptions: { state: 'suspended', resumeNever: true },
      resumeTimeoutMs: 10
    })

    await audio.play('mokugyo', 1)
    await audio.play('mokugyo', 1)
    expect(audio.isDegraded.value).toBe(true)

    const resumeCallsBefore = created[0].resumeCalls
    await audio.play('mokugyo', 1)
    expect(created[0].resumeCalls).toBe(resumeCallsBefore)
  })
})

describe('useCounterAudio · B2 硬口径（回前台不得裸调 resume）', () => {
  it('#32 handleForeground() 绝不碰 ctx：resume / close / ctxFactory 调用次数全为 0', async () => {
    const { audio, created, ctxFactory } = makeAudio({ ctxOptions: { state: 'running' } })
    await audio.unlock()

    const ctxFactoryCalls = ctxFactory.mock.calls.length
    const resumeCalls = created[0].resumeCalls

    audio.handleForeground()
    audio.handleForeground()
    audio.handleForeground()

    expect(created[0].resumeCalls).toBe(resumeCalls)
    expect(created[0].closeCalls).toBe(0)
    expect(ctxFactory.mock.calls.length).toBe(ctxFactoryCalls)
    expect(audio.neededUnlock).toBe(true)
  })

  it('#33 play() 遇到 closed 只新建 ctx，不 close 旧 ctx', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'running' } })
    await audio.unlock()

    created[0].state = 'closed'
    await audio.play('mokugyo', 1)

    expect(created[0].closeCalls).toBe(0)
    expect(created.length).toBe(2)
  })

  it('#34 unlock() 同步调用 resume()，且不阻塞调用方', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'suspended', resumeDelay: 30 } })

    const returned = audio.unlock()
    expect(typeof returned.then).toBe('function')
    // 同步即可断言 resume 已被调用（ctxFactory / resume 都发生在手势调用栈内）
    expect(created.length).toBe(1)
    expect(created[0].resumeCalls).toBe(1)

    // 调用方在 unlock 的 Promise 未 resolve 前仍能继续执行自己的逻辑
    let sideEffect = 0
    sideEffect += 1
    expect(sideEffect).toBe(1)

    await returned
    expect(audio.neededUnlock).toBe(false)
  })

  it('#35 静音 buffer 解锁只在 unlock() 内发生，play() 路径不产生额外 createBuffer', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'running' } })

    await audio.unlock()
    // createBuffers 2 次 + 静音解锁 1 次
    expect(created[0].createBufferCalls.length).toBe(3)
    expect(created[0].createBufferCalls.some((call) => call.length === 1)).toBe(true)

    const afterUnlock = created[0].createBufferCalls.length
    await audio.play('mokugyo', 1)
    await audio.play('bell', 1)

    expect(created[0].createBufferCalls.length).toBe(afterUnlock)
    // 只有 unlock() 里那次静音解锁 + 这两次 play，play 本身不再新建任何缓冲
    const playedSources = created[0].startedSources.filter(
      (source) => source.buffer && source.buffer.length === 64
    )
    expect(playedSources.length).toBe(2)
  })

  it('unlock() 幂等：静音解锁每个 ctx 只做一次', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'running' } })

    await audio.unlock()
    await audio.unlock()
    await audio.unlock()

    const silentPrimes = created[0].createBufferCalls.filter((call) => call.length === 1)
    expect(silentPrimes.length).toBe(1)
  })

  it('handleForegroundClock() 只跑传入的回调，不碰音频', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'running' } })
    await audio.unlock()

    const runClock = vi.fn()
    const resumeCalls = created[0].resumeCalls
    audio.handleForegroundClock(runClock)

    expect(runClock).toHaveBeenCalledTimes(1)
    expect(created[0].resumeCalls).toBe(resumeCalls)
  })

  it('dispose() 回收 ctx，不抛异常', async () => {
    const { audio, created } = makeAudio({ ctxOptions: { state: 'running' } })
    await audio.unlock()

    expect(() => audio.dispose()).not.toThrow()
    expect(created[0].closeCalls).toBe(1)
  })
})
