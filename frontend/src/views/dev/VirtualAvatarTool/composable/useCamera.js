// useCamera.js —— 摄像头状态机 composable(P2)
//
// 包 cameraProbe.js,暴露 Vue-friendly 响应式状态(state / error / stream / retry):
//   const cam = useCamera()
//   cam.state.value     // 'idle' | 'requesting' | 'granted' | 'denied' | 'no_device' | 'unsupported' | 'error'
//   cam.error.value     // Error | null
//   cam.stream.value    // MediaStream | null
//   cam.request()       // 触发授权
//   cam.retry()         // 复位后重新请求(用户点"再试")
//   cam.stop()          // 关闭摄像头
//   cam.attach(videoEl) // 绑定到 <video> 自动同步
//
// 文案与图标由 describeState + stateIcon 提供,组件层直接消费。
// 不抛错到上层 —— state 就是状态机的对外契约。

import { ref, onUnmounted } from 'vue'
import { createCameraProbe, CameraState, describeState } from '../cameraProbe.js'

export function useCamera(options = {}) {
  const probe = createCameraProbe(options)

  const state = ref(probe.getState())
  const error = ref(probe.getError())
  const stream = ref(probe.getState() === CameraState.GRANTED ? null : null)
  let unsub = null

  function syncFromProbe({ state: s, error: e, stream: st }) {
    state.value = s
    error.value = e
    stream.value = st
  }

  // 订阅 probe —— 立即同步一次当前快照,后续变化推送
  unsub = probe.subscribe(syncFromProbe)

  async function request() {
    const s = await probe.request()
    // 同步一次(请求后 probe 已 emit,但 subscribe 也已收到,这里再覆盖保险)
    syncFromProbe({ state: probe.getState(), error: probe.getError(), stream: probe.getState() === CameraState.GRANTED ? s : null })
    return s
  }

  function stop() {
    probe.stop()
  }

  // 再试:reset 不存在(NotAllowedError 不能 reset),只能 stop 后再 request;
  // 这是浏览器 mediaDevices 的设计,我们照做即可。
  async function retry() {
    stop()
    // 给浏览器一拍时间回收 track
    await new Promise((r) => setTimeout(r, 50))
    return request()
  }

  function attach(videoEl) {
    return probe.attach(videoEl)
  }

  onUnmounted(() => {
    if (unsub) unsub()
    stop()
  })

  return {
    state,
    error,
    stream,
    isSupported: probe.isSupported,
    request,
    retry,
    stop,
    attach,
    describe: () => describeState(state.value, error.value),
    CameraState,
  }
}

// 给四态画 el-icon 用的 emoji —— 不引入图标组件,组件层 switch case 即可。
export function stateEmoji(s) {
  switch (s) {
    case CameraState.GRANTED: return '✅'
    case CameraState.DENIED: return '🚫'
    case CameraState.NO_DEVICE: return '📷'
    case CameraState.UNSUPPORTED: return '⚠️'
    case CameraState.REQUESTING: return '⏳'
    case CameraState.ERROR: return '❌'
    default: return '📹'
  }
}
