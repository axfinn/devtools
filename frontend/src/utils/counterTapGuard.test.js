import { describe, it, expect } from 'vitest'
import * as tapGuard from './counterTapGuard'
import {
  LONG_PRESS_MS,
  EXIT_LONG_PRESS_MS,
  SHORTCUT_BLOCKED,
  SHORTCUT_DECREMENT,
  SHORTCUT_INCREMENT,
  SHORTCUT_UNDO,
  isLongPressCommit,
  createLongPressGuard,
  resolveShortcut
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

describe('counterTapGuard · resolveShortcut', () => {
  // L1 #6b —— 训练层撤销门限（B10 的自动化主判据）
  describe('#6b 训练层存活期的撤销门限', () => {
    it('Meta+Z 训练中 → blocked', () => {
      expect(resolveShortcut({ code: 'KeyZ', metaKey: true }, { trainingMode: true })).toBe(SHORTCUT_BLOCKED)
    })

    it('Ctrl+Z 训练中 → blocked（Windows 修饰键同样屏蔽）', () => {
      expect(resolveShortcut({ code: 'KeyZ', ctrlKey: true }, { trainingMode: true })).toBe(SHORTCUT_BLOCKED)
    })

    it('Meta+Z 日间模式 → undo（零回归）', () => {
      expect(resolveShortcut({ code: 'KeyZ', metaKey: true }, { trainingMode: false })).toBe(SHORTCUT_UNDO)
    })

    it('Ctrl+Z 日间模式 → undo（零回归）', () => {
      expect(resolveShortcut({ code: 'KeyZ', ctrlKey: true }, { trainingMode: false })).toBe(SHORTCUT_UNDO)
    })
  })

  // L1 #6c —— 其余键的训练层门限
  describe('#6c 其余键的训练层门限', () => {
    it.each(['ArrowDown', 'Minus', 'Backspace'])('%s 训练中 → blocked', (code) => {
      expect(resolveShortcut({ code }, { trainingMode: true })).toBe(SHORTCUT_BLOCKED)
    })

    it.each(['ArrowDown', 'Minus', 'Backspace'])('%s 日间模式 → decrement（零回归）', (code) => {
      expect(resolveShortcut({ code }, { trainingMode: false })).toBe(SHORTCUT_DECREMENT)
    })

    // 反向闸门：防止实现「一刀切屏蔽所有快捷键」把训练中的计数也吞掉（B1 硬口径）
    it.each(['Space', 'ArrowUp', 'Equal'])('%s 训练中 → increment（训练层不吞计数）', (code) => {
      expect(resolveShortcut({ code }, { trainingMode: true })).toBe(SHORTCUT_INCREMENT)
    })

    it('不传第二参时 trainingMode 默认为 false → undo', () => {
      expect(resolveShortcut({ code: 'KeyZ', metaKey: true })).toBe(SHORTCUT_UNDO)
    })

    it.each([
      ['KeyA', { code: 'KeyA', metaKey: true }],
      ['Escape', { code: 'Escape' }],
      ['Enter', { code: 'Enter' }]
    ])('未涉及的键 %s → null（不是 blocked）', (_label, event) => {
      expect(resolveShortcut(event, { trainingMode: true })).toBeNull()
      expect(resolveShortcut(event, { trainingMode: false })).toBeNull()
    })
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

  it('模块只导出长按守卫 + 快捷键门限相关的 10 个符号', () => {
    expect(Object.keys(tapGuard).sort()).toEqual([
      'EXIT_LONG_PRESS_MS',
      'LONG_PRESS_MS',
      'LONG_PRESS_MOVE_TOLERANCE_PX',
      'SHORTCUT_BLOCKED',
      'SHORTCUT_DECREMENT',
      'SHORTCUT_INCREMENT',
      'SHORTCUT_UNDO',
      'createLongPressGuard',
      'isLongPressCommit',
      'resolveShortcut'
    ].sort())
  })
})
