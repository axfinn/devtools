import { describe, it, expect, vi } from 'vitest'
import * as tapGuard from './counterTapGuard'
import {
  LONG_PRESS_MS,
  EXIT_LONG_PRESS_MS,
  isLongPressCommit,
  createLongPressGuard
} from './counterTapGuard'

describe('counterTapGuard · isLongPressCommit', () => {
  it('未达阈值返回 false', () => {
    expect(isLongPressCommit({ startAt: 0, endAt: 599, thresholdMs: LONG_PRESS_MS })).toBe(false)
  })

  it('恰好等于阈值返回 true（边界含等号）', () => {
    expect(isLongPressCommit({ startAt: 0, endAt: 600, thresholdMs: LONG_PRESS_MS })).toBe(true)
  })

  it('超过阈值返回 true', () => {
    expect(isLongPressCommit({ startAt: 100, endAt: 900, thresholdMs: EXIT_LONG_PRESS_MS })).toBe(true)
  })
})

describe('counterTapGuard · createLongPressGuard', () => {
  function makeGuard(thresholdMs = LONG_PRESS_MS) {
    let clock = 0
    const guard = createLongPressGuard({ thresholdMs, now: () => clock })
    return { guard, advance: (ms) => { clock += ms } }
  }

  it('start 后立刻 cancel → takeCommit 返回 false', () => {
    const { guard } = makeGuard()
    guard.start(1)
    guard.cancel(1)
    expect(guard.takeCommit(1)).toBe(false)
  })

  it('start → 700ms → takeCommit 返回 true', () => {
    const { guard, advance } = makeGuard()
    guard.start(1)
    advance(700)
    expect(guard.takeCommit(1)).toBe(true)
  })

  it('cancel 未知 pointerId 不抛异常', () => {
    const { guard } = makeGuard()
    expect(() => guard.cancel(42)).not.toThrow()
    expect(() => guard.cancel(undefined)).not.toThrow()
  })

  it('takeCommit 未跟踪的 pointerId 返回 false 且只消费一次', () => {
    const { guard, advance } = makeGuard()
    guard.start(7)
    advance(800)
    expect(guard.takeCommit(7)).toBe(true)
    expect(guard.takeCommit(7)).toBe(false)
  })

  it('isArmed 反映长按进度，未 start 时为 false', () => {
    const { guard, advance } = makeGuard()
    expect(guard.isArmed(3)).toBe(false)
    guard.start(3)
    expect(guard.isArmed(3)).toBe(false)
    advance(LONG_PRESS_MS)
    expect(guard.isArmed(3)).toBe(true)
  })

  it('多指各自独立跟踪', () => {
    const { guard, advance } = makeGuard()
    guard.start(1)
    advance(300)
    guard.start(2)
    advance(400)
    // 1 号已 700ms，2 号只有 400ms
    expect(guard.isArmed(1)).toBe(true)
    expect(guard.isArmed(2)).toBe(false)
    expect(guard.takeCommit(2)).toBe(false)
    expect(guard.takeCommit(1)).toBe(true)
  })
})

describe('counterTapGuard · 负向断言（B1 闸门：计数面不得有任何门限）', () => {
  it('不得导出 shouldCommitTap', () => {
    expect(tapGuard.shouldCommitTap).toBeUndefined()
  })

  it('不得导出 TAP_MOVE_TOLERANCE_PX', () => {
    expect(tapGuard.TAP_MOVE_TOLERANCE_PX).toBeUndefined()
  })

  it('不得导出 TAP_MAX_DURATION_MS', () => {
    expect(tapGuard.TAP_MAX_DURATION_MS).toBeUndefined()
  })

  it('不得导出 TAP_DEBOUNCE_MS', () => {
    expect(tapGuard.TAP_DEBOUNCE_MS).toBeUndefined()
  })

  it('模块只导出长按守卫相关的 6 个符号', () => {
    expect(Object.keys(tapGuard).sort()).toEqual([
      'EXIT_LONG_PRESS_MS',
      'LONG_PRESS_MS',
      'LONG_PRESS_MOVE_TOLERANCE_PX',
      'createLongPressGuard',
      'isLongPressCommit'
    ].sort())
  })
})
