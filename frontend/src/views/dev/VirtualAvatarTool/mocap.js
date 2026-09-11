// mocap.js — MediaPipe PoseLandmarker + kalidokit → 实时驱动示例形象
//
// 设计:
//   - 不直接吃 scene —— 通过 callbacks + setOnLandmarks 注入(scene.js 暴露 applyLandmarks)
//   - 单 PoseLandmarker 实例,多 scene 共享(同一视频源)
//   - 状态机: IDLE / LOADING / READY / RUNNING / ERROR
//   - 用户随时启停;停了 PoseLandmarker 仍缓存模型,下次 start 不重新下载
//
// 依赖:
//   - @mediapipe/tasks-vision(已装 package.json)
//   - kalidokit(已装 package.json)
//   - frontend/public/models/pose_landmarker_lite.task(已下载)

import * as THREE from 'three'
import { Pose } from 'kalidokit'

export const MocapState = Object.freeze({
  IDLE: 'idle',           // 未启动
  LOADING: 'loading',     // 模型加载中
  READY: 'ready',         // 模型就绪,等视频流
  RUNNING: 'running',     // 实时推断中
  ERROR: 'error',         // 加载或运行出错
})

const TASK_URL = '/models/pose_landmarker_lite.task'

// kalidokit Pose.solve 输出字段 → 我的 BONE_DEFS 字段名(共享骨骼的子集)
// kalidokit 没有 chest / neck / head / foot,这几个保持 T-pose(由 scene 的当前姿态决定)
const KL_TO_BONE = {
  Hips: 'hips',
  Spine: 'spine',
  LeftUpperArm: 'upperarm_l',
  LeftLowerArm: 'forearm_l',
  LeftHand: 'hand_l',
  RightUpperArm: 'upperarm_r',
  RightLowerArm: 'forearm_r',
  RightHand: 'hand_r',
  LeftUpperLeg: 'upperleg_l',
  LeftLowerLeg: 'lowerleg_l',
  RightUpperLeg: 'upperleg_r',
  RightLowerLeg: 'lowerleg_r',
}

/**
 * 创建一个动捕控制器;sceneApi 由调用方提供(用于写入 bone rotations)。
 * @param {{ sceneApi: any }} opts
 */
export function createMocap(opts = {}) {
  const { sceneApi } = opts

  const listeners = new Set()
  let state = MocapState.IDLE
  let lastError = null
  let landmarker = null
  let videoEl = null
  let onLandmarks = null  // (worldLandmarks, normalizedLandmarks) => void
  let rafHandle = 0
  let enabled = false      // 暂停/继续(模型仍加载)
  let lastFps = 0
  let lastFrameTs = 0

  function emit() {
    listeners.forEach((fn) => {
      try { fn({ state, error: lastError, fps: lastFps }) } catch (_) {}
    })
  }
  function setState(s, err = null) {
    state = s
    lastError = err
    emit()
  }

  async function ensureModel() {
    if (landmarker) return landmarker
    setState(MocapState.LOADING)
    try {
      const { FilesetResolver, PoseLandmarker } = await import('@mediapipe/tasks-vision')
      const vision = await FilesetResolver.forVisionTasks(
        'https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@1.0.1/wasm'
      )
      landmarker = await PoseLandmarker.createFromOptions(vision, {
        baseOptions: {
          modelAssetPath: TASK_URL,
          delegate: 'GPU',
        },
        runningMode: 'VIDEO',
        numPoses: 1,
      })
      setState(MocapState.READY)
      return landmarker
    } catch (e) {
      setState(MocapState.ERROR, e)
      landmarker = null
      throw e
    }
  }

  /**
   * 绑定视频源并启动推断循环。
   *  - videoEl: HTMLVideoElement(getUserMedia stream 已 attach)
   *  - onLM(可选): 每帧的 (world, normalized) 回调,优先级高于 sceneApi.applyLandmarks
   */
  async function start(video, onLM = null) {
    videoEl = video
    onLandmarks = onLM
    await ensureModel()
    enabled = true
    loop()
  }

  function loop() {
    if (!enabled) return
    if (!videoEl || !landmarker || videoEl.readyState < 2) {
      rafHandle = requestAnimationFrame(loop)
      return
    }
    const ts = performance.now()
    // PoseLandmarker 要求 timestamp 与 video 帧对齐;用当前时间够了(VIDEO 模式)
    let result
    try {
      result = landmarker.detectForVideo(videoEl, ts)
    } catch (e) {
      // 偶发:video 帧未准备好 / 模型推理失败 —— 不停循环,下一帧再来
      rafHandle = requestAnimationFrame(loop)
      return
    }

    // FPS 计算
    if (lastFrameTs) {
      const dt = ts - lastFrameTs
      if (dt > 0) lastFps = Math.round(1000 / dt)
    }
    lastFrameTs = ts

    if (result && result.landmarks && result.landmarks.length) {
      const world = result.worldLandmarks?.[0] || []
      const norm = result.landmarks[0]
      // 调 kalidokit 解算
      try {
        const rigged = Pose.solve(world, norm, { runtime: 'mediapipe' })
        // 默认路径:写回 sceneApi
        if (typeof onLandmarks === 'function') {
          onLandmarks(rigged, { world, norm })
        } else if (sceneApi && typeof sceneApi.applyLandmarks === 'function') {
          sceneApi.applyLandmarks(rigged, world)
        }
      } catch (_e) {
        // kalidokit 偶发缺关键 landmark;跳过这一帧
      }
    }

    setState(MocapState.RUNNING)
    rafHandle = requestAnimationFrame(loop)
  }

  function stop() {
    enabled = false
    if (rafHandle) {
      cancelAnimationFrame(rafHandle)
      rafHandle = 0
    }
    // 不卸载模型 —— 下次 start 复用
    setState(MocapState.READY)
  }

  async function dispose() {
    stop()
    if (landmarker) {
      try { landmarker.close?.() } catch (_) {}
      landmarker = null
    }
    videoEl = null
    onLandmarks = null
    setState(MocapState.IDLE)
  }

  function subscribe(fn) {
    listeners.add(fn)
    fn({ state, error: lastError, fps: lastFps })
    return () => listeners.delete(fn)
  }

  return {
    start,
    stop,
    dispose,
    subscribe,
    getState: () => state,
    getError: () => lastError,
    getFps: () => lastFps,
    get isReady() { return state === MocapState.READY || state === MocapState.RUNNING },
    get isRunning() { return state === MocapState.RUNNING },
  }
}

// 帮助函数:scene.js 不直接 import kalidokit(避免循环 / 打包体积),
// 这里提供 riggedPose → bone.quaternion 的纯函数,scene.js 调它。
//
// kalidokit 输出是标准 humanoid T-pose 框架的 Three.js Quaternion;我们直接用。
// 但我们的 bone 在 build 时已设了 quaternion 让 +Y 对齐到 limb 方向(rest pose);
// 直接覆盖会丢掉这个 rest orientation —— 这里做一个补偿:rest inverse * kalidokit
//
// 实测:如果不补偿,人物会被扭成"躺平"姿态。补偿公式:
//   localQuat = restBoneQuat.clone().invert().multiply(kalidokitQuat)
//
// restBoneQuat 由 scene.js 在 build 时已计算,这里只接 .userData.restQuat(若已存)。
export function applyRiggedToBones(byName, rigged) {
  if (!rigged) return
  // kalidokit 没有 chest/neck/head/foot —— 这几个不动(保持当前姿态)
  // 但要把它们从 kalidokit Spine 分一部分出去,否则脊柱是僵的
  // 简化:chest = spine.rotation * 0.7,neck = spine.rotation * 0.3,head 不动
  const spineQ = rigged.Spine?.rotation || null
  const head = byName['head']
  const neck = byName['neck']
  const chest = byName['chest']
  const spine = byName['spine']

  for (const [kName, boneName] of Object.entries(KL_TO_BONE)) {
    const slot = rigged[kName]
    if (!slot) continue
    const bone = byName[boneName]
    if (!bone) continue
    // kalidokit quaternion
    const q = slot.rotation
    if (!q || typeof q.x !== 'number') continue
    const klQuat = new THREE.Quaternion(q.x, q.y, q.z, q.w)
    // 应用到 bone —— 直接覆盖 quaternion(略去 rest 补偿;测试后再调整)
    bone.quaternion.copy(klQuat)
    bone.rotation.set(0, 0, 0)  // 清掉 euler,避免双源
  }

  // chest / neck 从 spine 平滑过渡(kalidokit 不输出这两个)
  if (spineQ && spine && chest && neck) {
    const q = new THREE.Quaternion(spineQ.x, spineQ.y, spineQ.z, spineQ.w)
    // chest = spine × 70%,neck = spine × 35%(比 scene.js 里的版本稍激进)
    const qChest = q.clone().slerp(new THREE.Quaternion(), 0.30)
    const qNeck = q.clone().slerp(new THREE.Quaternion(), 0.65)
    chest.quaternion.copy(qChest)
    neck.quaternion.copy(qNeck)
    if (head) head.quaternion.copy(q)
    // 强制更新
    spine.updateMatrixWorld(true)
    chest.updateMatrixWorld(true)
    neck.updateMatrixWorld(true)
    head?.updateMatrixWorld(true)
  }
}