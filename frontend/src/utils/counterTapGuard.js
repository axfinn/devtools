/**
 * counterTapGuard —— 只服务「破坏性操作」的长按判定。
 *
 * ⚠️ 设计红线（方案 §D2① / B1 硬口径，jaxiu 2026-09-16 裁决）：
 * 计数面（主敲击按钮）**一击即计** —— `pointerdown` 直接提交，不施加任何门限。
 * 因此本模块**只导出长按守卫**，供「结束训练 / 暂停 / 继续」这类破坏性操作使用。
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
