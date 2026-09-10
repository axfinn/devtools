// poseData.js — 骨骼姿态数据序列化 / 反序列化
// 目标:输出格式能被任何主流动捕管线消费(blender、mediapipe-3d、custom rig)。

/**
 * 把 Skeleton 拍成一帧姿态数据:
 *   { frame, timestamp, bones: [{ name, parent, position:[x,y,z], quaternion:[x,y,z,w], worldPosition:[x,y,z], worldQuaternion:[x,y,z,w] }] }
 * @param {import('three').Skeleton} skeleton
 * @param {number} [frame=0]
 * @param {number} [timestamp=Date.now()]
 */
export function capturePose(skeleton, frame = 0, timestamp = Date.now()) {
  // 强制刷新一次,确保 boneMatrixWorld 是当前帧的
  skeleton.update()
  const out = {
    frame,
    timestamp,
    bones: [],
  }
  for (const bone of skeleton.bones) {
    bone.updateWorldMatrix(true, false)
    out.bones.push({
      name: bone.name,
      parent: bone.parent && bone.parent.isBone ? bone.parent.name : null,
      position: [bone.position.x, bone.position.y, bone.position.z],
      quaternion: [bone.quaternion.x, bone.quaternion.y, bone.quaternion.z, bone.quaternion.w],
      worldPosition: [
        bone.matrixWorld.elements[12],
        bone.matrixWorld.elements[13],
        bone.matrixWorld.elements[14],
      ],
      worldQuaternion: extractQuaternion(bone.matrixWorld),
    })
  }
  return out
}

export function serializePoseClip(clip) {
  return JSON.stringify(clip, null, 2)
}

/**
 * 从一组 pose frames 构造完整 clip:
 *   { meta: { boneNames, parents, fps }, frames: Pose[] }
 */
export function buildClip(name, frames, fps = 30, meta = {}) {
  const boneNames = []
  const parents = []
  if (frames.length > 0) {
    for (const b of frames[0].bones) {
      boneNames.push(b.name)
      parents.push(b.parent)
    }
  }
  return {
    name,
    meta: {
      fps,
      boneCount: boneNames.length,
      boneNames,
      parents,
      createdAt: new Date().toISOString(),
      ...meta,
    },
    frames,
  }
}

/**
 * 把单帧 pose 应用回 Skeleton(预览 / 调试回放)。
 * @param {import('three').Skeleton} skeleton
 * @param {{ bones: Array<{ name: string, position?: number[], quaternion?: number[] }> }} frame
 */
export function applyFrame(skeleton, frame) {
  if (!frame || !frame.bones) return
  for (const boneData of frame.bones) {
    const bone = skeleton.getBoneByName(boneData.name)
    if (!bone) continue
    if (boneData.position) bone.position.fromArray(boneData.position)
    if (boneData.quaternion) bone.quaternion.fromArray(boneData.quaternion)
  }
  skeleton.update()
}

// --- 内部:从 4x4 matrix 提取 rotation quaternion ---
function extractQuaternion(m) {
  const t = m.elements
  const trace = t[0] + t[5] + t[10]
  let qx, qy, qz, qw
  if (trace > 0) {
    const s = 0.5 / Math.sqrt(trace + 1)
    qw = 0.25 / s
    qx = (t[6] - t[9]) * s
    qy = (t[8] - t[2]) * s
    qz = (t[1] - t[4]) * s
  } else if (t[0] > t[5] && t[0] > t[10]) {
    const s = 2 * Math.sqrt(1 + t[0] - t[5] - t[10])
    qw = (t[6] - t[9]) / s
    qx = 0.25 * s
    qy = (t[4] + t[1]) / s
    qz = (t[8] + t[2]) / s
  } else if (t[5] > t[10]) {
    const s = 2 * Math.sqrt(1 + t[5] - t[0] - t[10])
    qw = (t[8] - t[2]) / s
    qx = (t[4] + t[1]) / s
    qy = 0.25 * s
    qz = (t[9] + t[6]) / s
  } else {
    const s = 2 * Math.sqrt(1 + t[10] - t[0] - t[5])
    qw = (t[1] - t[4]) / s
    qx = (t[8] + t[2]) / s
    qy = (t[9] + t[6]) / s
    qz = 0.25 * s
  }
  return [qx, qy, qz, qw]
}
