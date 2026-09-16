/**
 * counterTapGuard —— 「破坏性输入」的门限，两类：
 *   ① 破坏性操作（结束训练 / 暂停 / 继续）的长按判定（v2 起）；
 *   ② 训练层存活期（`prefs.trainingMode === true`）的键盘快捷键门限 —— `resolveShortcut()`（v4 / B10）。
 *
 * ⚠️ 设计红线（方案 §D2① / B1 硬口径，jaxiu 2026-09-16 裁决）：
 * 计数面（主敲击按钮）**一击即计** —— `pointerdown` 直接提交，不施加任何门限；
 * 键盘上的计数键（`Space` / `ArrowUp` / `Equal`）同理，训练层存活时**照常计数**（本模块只锁
 * `Cmd/Ctrl+Z` 与 `ArrowDown` / `Minus` / `Backspace` 这三个破坏性入口，见方案 §D2⑦-b.1）。
 *
 * 历史上存在过的「计数门限四件套」（提交判定函数 / 位移容差 / 最大时长 / 去抖间隔）
 * **已被否决并删除**，不得再加回来（L1 #6 / L2 #40 负向断言会 grep 这四个标识符）。
 * 任何位移 / 时长 / 去抖门限出现在计数面上，都会造成「有效敲击被吞」，属于返工项。
 */

/** 暂停 / 继续 的长按阈值（ms） */
export const LONG_PRESS_MS = 600

/** 结束训练 的长按阈值（ms） */
export const EXIT_LONG_PRESS_MS = 700

/** 手指滑出该像素距离即取消长按 —— 只作用于破坏性操作，与计数面无关 */
export const LONG_PRESS_MOVE_TOLERANCE_PX = 12

/**
 * 纯函数：给定起止时间与阈值，判断是否达成一次长按提交。
 * 边界包含等号（`endAt - startAt === thresholdMs` 视为达成）。
 */
export function isLongPressCommit({ startAt, endAt, thresholdMs }) {
  const start = Number(startAt) || 0
  const end = Number(endAt) || 0
  const threshold = Number(thresholdMs) || 0
  return end - start >= threshold
}

/**
 * 有状态的长按守卫。同一时刻可跟踪多个 pointerId（多指按下时互不干扰）。
 *
 * @param {{ thresholdMs?: number, now?: () => number }} options
 */
export function createLongPressGuard({ thresholdMs = LONG_PRESS_MS, now = () => performance.now() } = {}) {
  const states = new Map()

  return {
    /** pointerdown 时调用，记录起点时间 */
    start(pointerId) {
      states.set(pointerId, { startAt: now() })
    },

    /** pointerup / pointercancel / 位移超阈值时调用；对未知 pointerId 静默无事 */
    cancel(pointerId) {
      states.delete(pointerId)
    },

    /** 供 UI 显示长按进度：是否仍在跟踪且已达到阈值 */
    isArmed(pointerId) {
      const state = states.get(pointerId)
      if (!state) return false
      return now() - state.startAt >= thresholdMs
    },

    /**
     * 提交时调用：返回是否达到阈值，并结束该 pointerId 的跟踪。
     * 未跟踪的 pointerId 一律返回 false（已取消 / 从未开始）。
     */
    takeCommit(pointerId) {
      const state = states.get(pointerId)
      if (!state) return false
      const elapsed = now() - state.startAt
      states.delete(pointerId)
      return elapsed >= thresholdMs
    }
  }
}

// ───────────────────── 键盘快捷键门限（v4 / B10，方案 §D2⑦-b.1） ─────────────────────

export const SHORTCUT_INCREMENT = 'increment' // Space / ArrowUp / Equal
export const SHORTCUT_DECREMENT = 'decrement' // ArrowDown / Minus / Backspace
export const SHORTCUT_UNDO = 'undo' // Cmd/Ctrl + Z
export const SHORTCUT_BLOCKED = 'blocked' // 本模块认识的键，但当前状态下不派发动作

/**
 * 纯函数：把一次 keydown 解析成动作常量。`onKeyDown` 据此**单一派发**（方案 §D2⑦-b.1）。
 *
 *   返回 `null`           = 不是本模块处理的键（既不派发动作，也不 preventDefault）
 *   返回 `SHORTCUT_BLOCKED` = 是本模块处理的键但当前状态禁用（**仍要 preventDefault**，只是不派发动作）
 *
 * 为什么要区分 `null` 与 `blocked`（不是设计洁癖，是防一个真实回归）：
 * `Backspace` / `ArrowDown` / `ArrowUp` 在浏览器里有默认行为（历史后退 / 页面滚动），本模块
 * 对它们一律 `preventDefault()`。若屏蔽时直接 `return`（等价于 `null`），训练层存活时这三个键
 * 就会**漏出浏览器默认行为** —— 把「静默改数」换成「静默跳页」，同样是 B10 要消灭的形态。
 *
 * `trainingMode` 默认值必须是 `false`：日间模式下本函数的返回值必须与阻塞改造前的
 * `onKeyDown` **逐键等价**（方案 §D2⑦-b.1 的「日间模式零变化」口径）。
 */
export function resolveShortcut({ code, ctrlKey, metaKey }, { trainingMode = false } = {}) {
  if ((ctrlKey || metaKey) && code === 'KeyZ') {
    return trainingMode ? SHORTCUT_BLOCKED : SHORTCUT_UNDO
  }
  if (code === 'Space' || code === 'ArrowUp' || code === 'Equal') {
    return SHORTCUT_INCREMENT // 计数面：训练层存活时也照常计数（B1「锁 UI，不锁计数」）
  }
  if (code === 'ArrowDown' || code === 'Minus' || code === 'Backspace') {
    return trainingMode ? SHORTCUT_BLOCKED : SHORTCUT_DECREMENT
  }
  return null
}
