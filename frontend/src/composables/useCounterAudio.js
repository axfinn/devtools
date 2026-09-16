import { ref } from 'vue'

/**
 * useCounterAudio —— 敲击音效的生命周期管理（纯前端，0 网络请求）。
 *
 * 解决的真问题（对应需求 2「息屏后再点击页面没有声音」）：
 * iOS 在锁屏 / 切后台后会把 AudioContext 置为 `suspended` **或专有的 `interrupted`**。
 * 旧实现只判断 `'suspended'`，且 `resume()` 的 Promise 没有 await，
 * 于是在 ctx 时钟仍冻结时就把 `source.start()` 排程进去 → 第一次敲击被吞。
 *
 * ── 硬口径（方案 §B2 / B4，code-reviewer 按此核对）──────────────────────────
 * 对 AudioContext 的「恢复 / 关闭 / 新建」只允许出现在这四个函数体内：
 *   1. `unlock()`   —— 用户手势入口：await resume + 静音 buffer 解锁
 *   2. `play()`     —— 用户手势入口：await resume 后再排程音源
 *   3. `rebuild()`  —— 降级阶梯第 3 层，**只能由用户手势**（「点我恢复」胶囊）调用
 *   4. `dispose()`  —— 组件卸载时的资源回收，不参与运行时恢复路径
 * 其余辅助（`safeResume` / `ensureCtx`）只被上面四个调用，自身不出现在前台恢复路径上。
 * `handleForeground()` **绝不碰 ctx**：只置 `needsUnlock = true`。
 * `visibilitychange → visible` / `pageshow` 一律不得裸调恢复接口。
 *
 * 自动化主判据是 `.test.js` 里的 fake-ctx 调用计数断言（L1 #32 / #33 / #34）；
 * L2 #42 的行号 grep 只作辅助 —— 它会命中 4 处（默认 ctx 工厂 / safeResume / rebuild / dispose），
 * 逐条的调用路径见交付评论。
 */

export const DEFAULT_RESUME_TIMEOUT_MS = 250
const DEGRADE_FAIL_THRESHOLD = 2

/**
 * @param {{
 *   ctxFactory?: () => any,
 *   createBuffers?: (ctx: any) => Map<string, any>,
 *   resumeTimeoutMs?: number,
 *   setTimer?: typeof setTimeout,
 *   clearTimer?: typeof clearTimeout
 * }} options
 */
export function createCounterAudio({
  ctxFactory = () => new (window.AudioContext || window.webkitAudioContext)(),
  createBuffers = () => new Map(),
  resumeTimeoutMs = DEFAULT_RESUME_TIMEOUT_MS,
  setTimer = setTimeout,
  clearTimer = clearTimeout
} = {}) {
  let ctx = null
  let buffers = null
  let silentPrimed = false
  let needsUnlock = false
  let failCount = 0

  const isDegraded = ref(false)

  function safeResume(target) {
    try {
      const result = target.resume()
      return result && typeof result.then === 'function' ? result : Promise.resolve()
    } catch (_) {
      return Promise.reject(new Error('resume failed'))
    }
  }

  /** 给 `resume()` 加一个上限，避免 UI 无限等（方案 §R8 同思路） */
  function raceWithTimeout(promise, timeoutMs) {
    return new Promise((resolve) => {
      let settled = false
      const timer = setTimer(() => {
        if (settled) return
        settled = true
        resolve(false)
      }, timeoutMs)

      Promise.resolve(promise).then(
        () => {
          if (settled) return
          settled = true
          clearTimer(timer)
          resolve(true)
        },
        () => {
          if (settled) return
          settled = true
          clearTimer(timer)
          resolve(false)
        }
      )
    })
  }

  /** 层 0：清理 / 重建。ctx 缺失或已 closed → 新建 ctx 并重建全部音色缓冲 */
  function ensureCtx() {
    if (!ctx || ctx.state === 'closed') {
      ctx = ctxFactory()
      buffers = null
      silentPrimed = false
    }
    if (!buffers) {
      buffers = createBuffers(ctx) || new Map()
    }
    return ctx
  }

  /** 层 2：静音 buffer 解锁 —— 把输出管线真正推起来，每个 ctx 只做一次 */
  function primeSilent(target) {
    if (silentPrimed) return
    silentPrimed = true
    try {
      const buffer = target.createBuffer(1, 1, target.sampleRate || 44100)
      const source = target.createBufferSource()
      source.buffer = buffer
      source.connect(target.destination)
      source.start(0)
    } catch (_) {
      // 解锁失败不影响计数
    }
  }

  function scheduleSource(target, soundName, volumeRatio) {
    const buffer = buffers ? buffers.get(soundName) : null
    if (!buffer) return

    const source = target.createBufferSource()
    source.buffer = buffer

    const gain = target.createGain()
    const level = Math.max(0, Math.min(1, Number(volumeRatio) || 0))
    gain.gain.setValueAtTime(level, target.currentTime)

    source.connect(gain)
    gain.connect(target.destination)
    source.start(target.currentTime)
  }

  function registerFailure() {
    failCount += 1
    if (failCount >= DEGRADE_FAIL_THRESHOLD) {
      isDegraded.value = true
    }
  }

  /**
   * 用户手势入口（同步返回、不阻塞调用方）。
   * 必须在 `pointerdown` / `keydown` 这类手势调用栈内同步调用。
   * 返回 Promise<void>，调用方**不要** await —— `increment()` 必须先执行完。
   */
  function unlock() {
    if (isDegraded.value) return Promise.resolve()

    // 异步 IIFE：函数体在第一个 await 之前同步执行 → ctxFactory / resume 都发生在手势内
    return (async () => {
      let target
      try {
        target = ensureCtx()
      } catch (_) {
        return
      }
      if (!target) return

      if (target.state !== 'running') {
        const ok = await raceWithTimeout(safeResume(target), resumeTimeoutMs)
        if (!ok) {
          registerFailure()
          return
        }
      }

      needsUnlock = false
      primeSilent(target)
    })()
  }

  /**
   * 播放一个音色。**永不抛异常**，音频失败绝不冒泡到计数路径。
   * 若 ctx 未 running 则先 await resume（上限 resumeTimeoutMs）再排程 —— 这就是「首击不丢」。
   */
  function play(soundName, volumeRatio = 1) {
    if (isDegraded.value) return Promise.resolve()

    return (async () => {
      let target
      try {
        target = ensureCtx()
      } catch (_) {
        return
      }
      if (!target) return

      try {
        if (target.state !== 'running') {
          const ok = await raceWithTimeout(safeResume(target), resumeTimeoutMs)
          if (!ok) {
            registerFailure()
            return
          }
        }
        needsUnlock = false
        scheduleSource(target, soundName, volumeRatio)
        failCount = 0
      } catch (_) {
        registerFailure()
      }
    })()
  }

  /**
   * 回前台 / bfcache 恢复专用：**只置标志，绝不碰 ctx**。
   * 真正的恢复只能发生在下一次用户手势的 `unlock()` / `play()` 里。
   */
  function handleForeground() {
    needsUnlock = true
  }

  /** bfcache 恢复后补跑 1s 时钟的钩子；本模块不碰音频，只负责把回调跑起来 */
  function handleForegroundClock(runClock) {
    if (typeof runClock === 'function') runClock()
  }

  /**
   * 降级阶梯第 3 层：丢弃旧 ctx + 缓冲并重建。
   * ⚠️ 只能由用户手势（「音效已暂停 · 点我恢复」胶囊）调用 —— 手势外新建的 ctx 在 iOS 上只会是 suspended。
   */
  function rebuild() {
    return (async () => {
      const previous = ctx
      ctx = null
      buffers = null
      silentPrimed = false
      needsUnlock = true

      try {
        if (previous && previous.state !== 'closed') {
          await previous.close()
        }
      } catch (_) {
        // 旧 ctx 关不掉不影响重建
      }

      try {
        const target = ensureCtx()
        if (target.state !== 'running') {
          const ok = await raceWithTimeout(safeResume(target), resumeTimeoutMs)
          if (!ok) {
            registerFailure()
            return false
          }
        }
        primeSilent(target)
        failCount = 0
        isDegraded.value = false
        needsUnlock = false
        return true
      } catch (_) {
        registerFailure()
        return false
      }
    })()
  }

  /** 组件卸载时的资源回收（不参与运行时恢复路径） */
  function dispose() {
    const previous = ctx
    ctx = null
    buffers = null
    silentPrimed = false
    needsUnlock = false
    try {
      if (previous && previous.state !== 'closed') previous.close()
    } catch (_) {
      // ignore
    }
  }

  return {
    unlock,
    play,
    handleForeground,
    handleForegroundClock,
    isDegraded,
    rebuild,
    dispose,
    // 只读诊断信息，供组件 / QA 观察状态，不参与任何音频决策
    get neededUnlock() {
      return needsUnlock
    },
    get resumeTimeoutMs() {
      return resumeTimeoutMs
    }
  }
}
