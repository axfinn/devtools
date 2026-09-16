/**
 * useCounterWakeLock —— 训练期间保持屏幕常亮（Screen Wake Lock API，纯前端 0 网络请求）。
 *
 * 为什么必须有（jaxiu 2026-09-16 裁决第 5 条）：真机上练 30 分钟屏幕必然熄灭，
 * 「息屏后没声音」会持续复现 —— 音频恢复修得再好也只能保证「第一次敲击可能有声」。
 *
 * 降级原则（方案 §D4）：任何失败都**静默降级** —— 不弹提示、不报错、不影响训练。
 * 不支持 / 被拒绝 / 非安全上下文 → 记 `supported = false`，本次会话内不再重试。
 */

function defaultNavigator() {
  try {
    return typeof navigator === 'undefined' ? null : navigator
  } catch (_) {
    return null
  }
}

export function createCounterWakeLock({ nav = defaultNavigator() } = {}) {
  let supported = typeof nav?.wakeLock?.request === 'function'
  let sentinel = null

  return {
    get supported() {
      return supported
    },

    get active() {
      return sentinel !== null
    },

    /**
     * 申请屏幕常亮。**必须在用户手势内调用**（训练层「开始训练」的点击）。
     * 返回 Promise<boolean>，调用方不需要（也不应该）阻塞在该 Promise 上。
     */
    async request() {
      if (!supported || sentinel) return false
      try {
        const lock = await nav.wakeLock.request('screen')
        sentinel = lock
        if (lock && typeof lock.addEventListener === 'function') {
          lock.addEventListener('release', () => {
            if (sentinel === lock) sentinel = null
          })
        }
        return true
      } catch (_) {
        // 被拒绝 / 非安全上下文 → 当天不再重试
        supported = false
        sentinel = null
        return false
      }
    },

    /**
     * 回前台后重新申请 —— **不需要用户手势**，这是 Wake Lock API 的既定行为；
     * 页面 hidden 时浏览器会自动释放锁，回到 visible 且会话仍 active 时要补回来。
     */
    async reacquire() {
      if (!supported || sentinel) return false
      return this.request()
    },

    /** 会话结束 / 暂停 / 组件离开时释放 */
    release() {
      const current = sentinel
      sentinel = null
      try {
        current?.release?.()
      } catch (_) {
        // 忽略释放失败
      }
    }
  }
}
