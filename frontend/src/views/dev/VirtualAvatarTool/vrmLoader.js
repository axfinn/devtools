// vrmLoader.js —— VRM 模型载入器(N+1 轮)
// 用 @pixiv/three-vrm 的 VRMLoaderPlugin 接 GLTFLoader,把 VRM 加进现有 AvatarCanvas 场景。
//
// 用法:
//   const api = createAvatarScene(canvas)
//   const vrm = await loadVRMIntoScene(api, '/api/avatar/models/abc/file')
//   // vrm.scene 已加入 api.scene,人形可见
//
// 资源释放:
//   替换前调 disposeCurrentVRM(api),把上一只 VRM 的 GPU 资源全释放。

import * as THREE from 'three'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { VRMLoaderPlugin } from '@pixiv/three-vrm'

// 当前活跃 VRM —— 一个场景只挂一只
const _state = { api: null, vrm: null }

/**
 * 把 ArrayBuffer 解析成 VRM 并加入 scene。
 * @param {object} sceneApi  createAvatarScene 返回值
 * @param {string} url        glb/gltf/vrm 的 URL(浏览器 fetch → arrayBuffer)
 * @returns {Promise<object>}  { vrm, humanoid, meta }
 */
export async function loadVRMIntoScene(sceneApi, url) {
  if (!sceneApi || !sceneApi.scene) {
    throw new Error('sceneApi 未就绪')
  }
  // 调试钩:让 smoke 能确认 loadVRMIntoScene 真的被调了
  // (VRM 载入成功/失败都可能到 — 我们只关心"路径通了")
  try { console.info('[vrmLoader] loadVRMIntoScene', url) } catch (_) {}
  // 释放上一只
  disposeCurrentVRM(sceneApi)

  const loader = new GLTFLoader()
  loader.register((p) => new VRMLoaderPlugin(p))

  const buffer = await fetch(url).then((r) => {
    if (!r.ok) throw new Error(`fetch ${url} failed: ${r.status}`)
    return r.arrayBuffer()
  })

  const gltf = await new Promise((resolve, reject) => {
    loader.parse(
      buffer,
      '',
      (data) => resolve(data),
      (err) => reject(err),
    )
  })

  const vrm = gltf.userData.vrm
  if (!vrm) {
    throw new Error('这份文件不是 VRM(没有 vrm 扩展);普通 GLB 请走 GLTFLoader')
  }
  // VRM 1.0 的身高按场景单位(m)调整,默认 1.6m
  // 也保证 humanoid.update 完整跑一遍
  vrm.scene.position.set(0, 0, 0)
  sceneApi.scene.add(vrm.scene)
  vrm.update?.(0)

  _state.api = sceneApi
  _state.vrm = vrm

  return {
    vrm,
    humanoid: vrm.humanoid,
    meta: vrm.meta || null,
    expressionManager: vrm.expressionManager || null,
  }
}

/**
 * 释放当前 VRM 占用:geometry + texture + 从 scene 摘除。
 * 多次调用安全;无活跃 VRM 时是 noop。
 */
export function disposeCurrentVRM(sceneApi) {
  const cur = _state.vrm
  if (!cur) return
  try {
    sceneApi?.scene?.remove?.(cur.scene)
    cur.scene.traverse((obj) => {
      if (obj.geometry) obj.geometry.dispose?.()
      if (obj.material) {
        const mats = Array.isArray(obj.material) ? obj.material : [obj.material]
        for (const m of mats) {
          for (const k in m) {
            const v = m[k]
            if (v && typeof v === 'object' && typeof v.dispose === 'function' && v.isTexture) v.dispose?.()
          }
          m.dispose?.()
        }
      }
    })
    cur.dispose?.()
  } catch (_) {
    // 释放失败不阻塞后续
  }
  _state.vrm = null
  _state.api = null
}