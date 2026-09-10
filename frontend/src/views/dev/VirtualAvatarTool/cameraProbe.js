// cameraProbe.js — getUserMedia 包装 + 权限状态机
// 不依赖 three.js,可独立使用。失败路径永远不抛,只把状态推到 ERROR/UNSUPPORTED 等。

export const CameraState = Object.freeze({
  IDLE: 'idle',            // 未启动
  REQUESTING: 'requesting',// 浏览器正在弹权限框
  GRANTED: 'granted',      // 已拿到流
  DENIED: 'denied',        // 用户拒绝 / SecurityError
  NO_DEVICE: 'no_device',  // 没有摄像头 / OverconstrainedError
  UNSUPPORTED: 'unsupported',// 浏览器没有 mediaDevices
  ERROR: 'error',          // 其它错误
})

export function isCameraSupported() {
  return typeof navigator !== 'undefined'
    && !!navigator.mediaDevices
    && typeof navigator.mediaDevices.getUserMedia === 'function'
}

/**
 * @param {{ facingMode?: 'user'|'environment', audio?: boolean }} [options]
 */
export function createCameraProbe(options = {}) {
  const { facingMode = 'user', audio = false } = options
  let stream = null
  let state = CameraState.IDLE
  let lastError = null
  const listeners = new Set()

  function emit() {
    listeners.forEach((fn) => {
      try { fn({ state, error: lastError, stream }) } catch (_) { /* listener 异常不影响主流程 */ }
    })
  }

  async function request() {
    if (!isCameraSupported()) {
      state = CameraState.UNSUPPORTED
      lastError = new Error('当前浏览器不支持 getUserMedia (需要 HTTPS / localhost / secure context)')
      emit()
      return null
    }
    if (stream) return stream
    state = CameraState.REQUESTING
    lastError = null
    emit()
    try {
      const s = await navigator.mediaDevices.getUserMedia({ video: { facingMode }, audio })
      stream = s
      state = CameraState.GRANTED
      emit()
      return s
    } catch (err) {
      // err.name: NotAllowedError / NotFoundError / NotReadableError / OverconstrainedError / SecurityError / TypeError
      const map = {
        NotAllowedError: CameraState.DENIED,
        SecurityError: CameraState.DENIED,
        NotFoundError: CameraState.NO_DEVICE,
        OverconstrainedError: CameraState.NO_DEVICE,
        NotReadableError: CameraState.ERROR,
        TypeError: CameraState.UNSUPPORTED,
      }
      state = map[err && err.name] || CameraState.ERROR
      lastError = err
      emit()
      return null
    }
  }

  function stop() {
    if (!stream) return
    try { stream.getTracks().forEach((t) => t.stop()) } catch (_) { /* noop */ }
    stream = null
    state = CameraState.IDLE
    lastError = null
    emit()
  }

  function subscribe(fn) {
    listeners.add(fn)
    fn({ state, error: lastError, stream })
    return () => listeners.delete(fn)
  }

  /**
   * 绑定到一个 <video> 元素;后续 stream 变化会自动同步。
   * 返回 unsubscribe 函数。
   */
  function attach(videoEl) {
    if (!videoEl) return () => {}
    const apply = (s) => {
      if (videoEl.srcObject !== s) {
        videoEl.srcObject = s || null
        if (s) videoEl.play().catch(() => { /* autoplay 受限也无所谓 */ })
      }
    }
    apply(stream)
    return subscribe(({ stream: s }) => apply(s))
  }

  return {
    request,
    stop,
    subscribe,
    attach,
    getState: () => state,
    getError: () => lastError,
    isSupported: isCameraSupported(),
  }
}

// 状态机 → 人类可读的中文描述
export function describeState(state, error) {
  switch (state) {
    case CameraState.IDLE: return '尚未请求摄像头'
    case CameraState.REQUESTING: return '等待浏览器权限框…'
    case CameraState.GRANTED: return '已授权,摄像头画面可显示'
    case CameraState.DENIED: return '权限被拒绝 — 请在浏览器站点设置里重新允许'
    case CameraState.NO_DEVICE: return '未检测到可用摄像头 (NotFoundError)'
    case CameraState.UNSUPPORTED: return '浏览器不支持 getUserMedia,需要 HTTPS / localhost 环境'
    case CameraState.ERROR: return `摄像头错误: ${error && error.message ? error.message : '未知错误'}`
    default: return state
  }
}
