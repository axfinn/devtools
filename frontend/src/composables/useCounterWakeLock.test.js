import { describe, it, expect, vi } from 'vitest'
import { createCounterWakeLock } from './useCounterWakeLock'

/**
 * fake navigator —— 只实现 wakeLock.request，用来断言「申请 / 释放 / 静默降级」三条契约。
 * `released` 收集从 release 监听器里回调的锁，用来验证 sentinel 被清空。
 */
function makeNav({ supported = true, reject = null } = {}) {
  const locks = []
  const nav = {}
  if (supported) {
    nav.wakeLock = {
      request: vi.fn(() => {
        if (reject) return Promise.reject(new Error(reject))
        const listeners = []
        const lock = {
          released: false,
          listeners,
          addEventListener: (type, handler) => {
            if (type === 'release') listeners.push(handler)
          },
          release() {
            lock.released = true
            listeners.forEach((handler) => handler())
          }
        }
        locks.push(lock)
        return Promise.resolve(lock)
      })
    }
  }
  return { nav, locks }
}

describe('useCounterWakeLock · 屏幕常亮（jaxiu 裁决第 5 条）', () => {
  it('不支持 wakeLock 的设备：supported=false，request() 返回 false 且不抛', async () => {
    const { nav } = makeNav({ supported: false })
    const wakeLock = createCounterWakeLock({ nav })

    expect(wakeLock.supported).toBe(false)
    await expect(wakeLock.request()).resolves.toBe(false)
    expect(wakeLock.active).toBe(false)
  })

  it('申请成功：active 变 true，且同一时刻只持有一把锁', async () => {
    const { nav, locks } = makeNav()
    const wakeLock = createCounterWakeLock({ nav })

    await expect(wakeLock.request()).resolves.toBe(true)
    expect(wakeLock.active).toBe(true)
    expect(locks.length).toBe(1)

    // 已持有 → 重复申请是 no-op（不会申请出第二把）
    await expect(wakeLock.request()).resolves.toBe(false)
    expect(locks.length).toBe(1)
  })

  it('被拒绝 / 非安全上下文：静默降级为 supported=false，本次会话不再重试', async () => {
    const { nav } = makeNav({ reject: 'NotAllowedError' })
    const wakeLock = createCounterWakeLock({ nav })

    await expect(wakeLock.request()).resolves.toBe(false)
    expect(wakeLock.supported).toBe(false)
    expect(wakeLock.active).toBe(false)

    // 不再重试：request 次数保持 1
    await wakeLock.request()
    expect(nav.wakeLock.request).toHaveBeenCalledTimes(1)
  })

  it('release() 释放锁，active 回落，且可以再次申请', async () => {
    const { nav, locks } = makeNav()
    const wakeLock = createCounterWakeLock({ nav })

    await wakeLock.request()
    wakeLock.release()
    expect(wakeLock.active).toBe(false)
    expect(locks[0].released).toBe(true)

    await expect(wakeLock.reacquire()).resolves.toBe(true)
    expect(locks.length).toBe(2)
  })

  it('浏览器自动释放锁（页面 hidden）：sentinel 被清空 → 回前台可 reacquire', async () => {
    const { nav, locks } = makeNav()
    const wakeLock = createCounterWakeLock({ nav })

    await wakeLock.request()
    // 模拟浏览器在 hidden 时自行释放
    locks[0].release()
    expect(wakeLock.active).toBe(false)

    await expect(wakeLock.reacquire()).resolves.toBe(true)
    expect(locks.length).toBe(2)
  })

  it('release() 幂等，未持有时调用不抛', () => {
    const { nav } = makeNav()
    const wakeLock = createCounterWakeLock({ nav })

    expect(() => wakeLock.release()).not.toThrow()
    expect(() => wakeLock.release()).not.toThrow()
    expect(wakeLock.active).toBe(false)
  })

  it('nav 为 null（非浏览器环境）时不抛，supported=false', async () => {
    const wakeLock = createCounterWakeLock({ nav: null })

    expect(wakeLock.supported).toBe(false)
    await expect(wakeLock.request()).resolves.toBe(false)
    await expect(wakeLock.reacquire()).resolves.toBe(false)
  })
})
