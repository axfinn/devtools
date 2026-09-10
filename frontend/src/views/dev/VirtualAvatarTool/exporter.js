// exporter.js — glTF / 姿态 JSON 导出
// glTF 走 three 自带 GLTFExporter(只支持二进制 .glb 与文本 .gltf;FBX 没有官方实现,
// 需要外部库,这里只输出 glTF)。姿态 JSON 用 poseData.js,便于直接对接动捕管线。

import * as THREE from 'three'
import { GLTFExporter } from 'three/examples/jsm/exporters/GLTFExporter.js'
import { capturePose, serializePoseClip, buildClip } from './poseData.js'

/**
 * @param {THREE.Object3D} node - 要导出的根节点(我们的 SkinnedMesh 或 root Bone)
 * @param {'glb'|'gltf'} [format='glb']
 * @returns {Promise<{ format: string, blob?: Blob, json?: object, filename: string }>}
 */
export function exportGLTF(node, format = 'glb') {
  return new Promise((resolve, reject) => {
    const exp = new GLTFExporter()
    exp.parse(
      node,
      (result) => {
        const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
        if (format === 'glb') {
          const blob = new Blob([result], { type: 'model/gltf-binary' })
          resolve({ format, blob, filename: `avatar-${ts}.glb` })
        } else {
          resolve({ format, json: result, filename: `avatar-${ts}.gltf` })
        }
      },
      (err) => reject(err),
      {
        binary: format === 'glb',
        embedImages: true,
        onlyVisible: true,
        maxTextureSize: 1024,
      },
    )
  })
}

/**
 * 触发浏览器下载。返回值仅用于测试断言。
 */
export function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  // 给 Safari 一点时间触发下载,再回收 URL
  setTimeout(() => URL.revokeObjectURL(url), 2000)
  return { ok: true, filename }
}

/**
 * 一站式:从 scene API 直接导出 + 下载 glb
 * @param {ReturnType<typeof import('./scene.js').createAvatarScene>} sceneApi
 */
export async function exportSceneAsGLB(sceneApi) {
  // 导出 root Bone Group —— 这样 SkinnedMesh + Skeleton + 所有 Bone 都会被包含。
  const out = await exportGLTF(sceneApi.root, 'glb')
  downloadBlob(out.blob, out.filename)
  return out
}

/**
 * 单独导出 SkinnedMesh(只想要 mesh,不要别的 group/网格)。
 */
export async function exportMeshAsGLB(sceneApi) {
  const out = await exportGLTF(sceneApi.mesh, 'glb')
  downloadBlob(out.blob, out.filename)
  return out
}

/**
 * 把当前 pose 作为单帧 JSON 下载(动捕管线友好)。
 */
export function exportCurrentPose(sceneApi) {
  // 懒加载避免循环依赖
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const { capturePose, serializePoseClip, buildClip } = require('./poseData.js')
  const frame = capturePose(sceneApi.skeleton, 0)
  const clip = buildClip('avatar-current-pose', [frame], 30, {
    source: 'VirtualAvatarTool',
    note: '当前姿态的单帧快照;后续接入动捕时可改为多帧 BVH/CSV。',
  })
  const text = serializePoseClip(clip)
  const blob = new Blob([text], { type: 'application/json' })
  const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  const filename = `avatar-pose-${ts}.json`
  downloadBlob(blob, filename)
  return { filename, size: blob.size, boneCount: clip.meta.boneCount }
}
