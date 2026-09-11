// scene.js — 虚拟形象 three.js 场景工厂
// 用 SkinnedMesh + Skeleton 装配一个简化的类人骨架,所有骨骼有命名 + parent 链,
// 既能直观调试,又能被 GLTFExporter 正确导出 glTF skeleton(便于后续动捕管线消费)。

import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'

// --- 骨骼定义 ---
// pos = 在 parent 骨骼坐标系下的偏移(length + 方向)
// 每根骨骼对应一段身体:head / 躯干 / 上下臂 / 上下腿 / 手脚。
const BONE_DEFS = [
  { name: 'root',        parent: null,         pos: [0, 0, 0]    },
  { name: 'hips',        parent: 'root',       pos: [0, 0.85, 0] },
  { name: 'spine',       parent: 'hips',       pos: [0, 0.25, 0] },
  { name: 'chest',       parent: 'spine',      pos: [0, 0.25, 0] },
  { name: 'neck',        parent: 'chest',      pos: [0, 0.20, 0] },
  { name: 'head',        parent: 'neck',       pos: [0, 0.15, 0] },

  { name: 'shoulder_l',  parent: 'chest',      pos: [0.18, 0.12, 0] },
  { name: 'upperarm_l',  parent: 'shoulder_l', pos: [0.22, 0, 0]   },
  { name: 'forearm_l',   parent: 'upperarm_l', pos: [0.26, 0, 0]   },
  { name: 'hand_l',      parent: 'forearm_l',  pos: [0.22, 0, 0]   },

  { name: 'shoulder_r',  parent: 'chest',      pos: [-0.18, 0.12, 0] },
  { name: 'upperarm_r',  parent: 'shoulder_r', pos: [-0.22, 0, 0]    },
  { name: 'forearm_r',   parent: 'upperarm_r', pos: [-0.26, 0, 0]    },
  { name: 'hand_r',      parent: 'forearm_r',  pos: [-0.22, 0, 0]    },

  { name: 'upperleg_l',  parent: 'hips',       pos: [0.10, -0.05, 0] },
  { name: 'lowerleg_l',  parent: 'upperleg_l', pos: [0, -0.45, 0]    },
  { name: 'foot_l',      parent: 'lowerleg_l', pos: [0, -0.45, 0]    },

  { name: 'upperleg_r',  parent: 'hips',       pos: [-0.10, -0.05, 0] },
  { name: 'lowerleg_r',  parent: 'upperleg_r', pos: [0, -0.45, 0]    },
  { name: 'foot_r',      parent: 'lowerleg_r', pos: [0, -0.45, 0]    },
]

// 把骨骼搭成树;bone.quaternion 让局部 +Y 对齐到 limb 方向(limb mesh 沿 +Y 延伸)
function buildBoneTree() {
  const byName = {}
  for (const def of BONE_DEFS) {
    const b = new THREE.Bone()
    b.name = def.name
    b.userData.boneName = def.name
    b.position.set(...def.pos)
    const dir = new THREE.Vector3(def.pos[0], def.pos[1], def.pos[2])
    if (dir.lengthSq() > 1e-6) {
      const target = dir.clone().normalize()
      b.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), target)
    }
    byName[def.name] = b
  }
  for (const def of BONE_DEFS) {
    if (def.parent) byName[def.parent].add(byName[def.name])
  }
  return { root: byName.root, byName }
}

// 身体段定义 — 颜色 + 半径 + 长度(给 human 预设用)
const SEGMENTS_HUMAN = [
  { bone: 'spine',       color: 0xb3c0d8, r: 0.10, length: 0.25 },
  { bone: 'chest',       color: 0xb3c0d8, r: 0.14, length: 0.25 },
  { bone: 'neck',        color: 0xe6c7a8, r: 0.04, length: 0.18 },
  { bone: 'head',        color: 0xe6c7a8, sphere: true, r: 0.13 },
  { bone: 'upperarm_l',  color: 0xe6c7a8, r: 0.05, length: 0.22 },
  { bone: 'forearm_l',   color: 0xe6c7a8, r: 0.045, length: 0.24 },
  { bone: 'hand_l',      color: 0xe6c7a8, r: 0.04, length: 0.18 },
  { bone: 'upperarm_r',  color: 0xe6c7a8, r: 0.05, length: 0.22 },
  { bone: 'forearm_r',   color: 0xe6c7a8, r: 0.045, length: 0.24 },
  { bone: 'hand_r',      color: 0xe6c7a8, r: 0.04, length: 0.18 },
  { bone: 'upperleg_l',  color: 0x6b7a99, r: 0.07, length: 0.40 },
  { bone: 'lowerleg_l',  color: 0x6b7a99, r: 0.055, length: 0.42 },
  { bone: 'foot_l',      color: 0x6b7a99, r: 0.05, length: 0.20 },
  { bone: 'upperleg_r',  color: 0x6b7a99, r: 0.07, length: 0.40 },
  { bone: 'lowerleg_r',  color: 0x6b7a99, r: 0.055, length: 0.42 },
  { bone: 'foot_r',      color: 0x6b7a99, r: 0.05, length: 0.20 },
]

// 4 个内置"形象" — 共享 BONE_DEFS 骨骼,只是几何体不同
// 用途:让用户一眼看出"这页是改 3D 形象"的,而不是一个空 demo
const VISUAL_PRESETS = {
  // 默认人形 — 胶囊 + 球头 + 肤色
  human: {
    label: '经典人形',
    desc: '经典胶囊身形,头部/躯干/四肢分明',
    accent: '#b3c0d8',
    segments: SEGMENTS_HUMAN,
    geomType: 'capsule',
  },
  // 机器人 — 钢色 Box,硬边
  robot: {
    label: '硬核机器人',
    desc: '钢蓝方块,几何感强',
    accent: '#5c8ed6',
    geomType: 'box',
    segments: SEGMENTS_HUMAN.map((s, i) => ({
      ...s,
      // 头:改成钢蓝立方体
      ...(s.bone === 'head' ? { color: 0x5c8ed6, sphere: false, r: 0.13, length: 0.22 } : {}),
      // 躯干:加深
      ...(['spine', 'chest'].includes(s.bone) ? { color: 0x4a6fa0, r: 0.12 } : {}),
      // 四肢:钢灰
      ...(s.bone.startsWith('upperarm') || s.bone.startsWith('forearm') || s.bone.startsWith('hand')
        ? { color: 0x6a7e9c, r: s.r * 1.1 } : {}),
      ...(s.bone.startsWith('upperleg') || s.bone.startsWith('lowerleg') || s.bone.startsWith('foot')
        ? { color: 0x445672, r: s.r * 1.05 } : {}),
      // 脖子:小金属
      ...(s.bone === 'neck' ? { color: 0x7a8aac, r: 0.045 } : {}),
    })),
  },
  // 圆滚滚 — 全身 Sphere
  sphere: {
    label: '圆滚滚',
    desc: '全身圆球,卡通可爱',
    accent: '#ff7eb6',
    geomType: 'sphere',
    segments: SEGMENTS_HUMAN.map((s) => ({
      bone: s.bone,
      color: s.color === 0x6b7a99 ? 0xc8a4d6
        : s.color === 0xb3c0d8 ? 0xffb3d1
        : s.color === 0xe6c7a8 ? 0xfff0e1
        : s.color,
      sphere: true,
      r: s.sphere ? s.r * 1.1 : s.r * 1.15,
      length: s.length,
    })),
  },
  // 体素 — 多色 Box,像素感
  voxel: {
    label: '体素风',
    desc: '方块拼接,体素风',
    accent: '#7bc96f',
    geomType: 'box',
    segments: SEGMENTS_HUMAN.map((s) => ({
      ...s,
      color: s.color === 0xb3c0d8 ? 0x7bc96f
        : s.color === 0xe6c7a8 ? 0xffd166
        : s.color === 0x6b7a99 ? 0x06a77d
        : s.color,
      r: s.sphere ? 0.16 : s.r * 1.15,
      length: s.sphere ? undefined : (s.length || 0.2) * 1.05,
    })),
  },
}

// 计算所有骨骼在静止姿态下的世界变换 — 给 SkinnedMesh 的 skinIndex/skinWeight 做参考系
function collectWorldRestPoses() {
  const { root, byName } = buildBoneTree()
  // 给骨头一个临时 Group 触发 updateWorldMatrix
  const tmp = new THREE.Group()
  tmp.add(root)
  root.updateMatrixWorld(true)
  const out = {}
  for (const [name, bone] of Object.entries(byName)) {
    bone.updateWorldMatrix(true, false)
    out[name] = {
      bone,
      worldPos: new THREE.Vector3().setFromMatrixPosition(bone.matrixWorld),
      worldQuat: new THREE.Quaternion().setFromRotationMatrix(bone.matrixWorld),
      invWorldQuat: new THREE.Quaternion().setFromRotationMatrix(bone.matrixWorld).invert(),
    }
  }
  return { root, byName, world: out }
}

// 构造 SkinnedMesh —— 整身一个 mesh,每根骨骼绑一段顶点(skinIndex/skinWeight)
// segments: 视觉预设里的段定义;geomType: 'capsule' | 'box' | 'sphere'(box/sphere 都不依赖 seg.sphere)
function buildSkinnedMesh(segments, geomType = 'capsule') {
  const { root, byName, world } = collectWorldRestPoses()

  const positions = []
  const colors = []
  const normals = []
  const skinIndices = []
  const skinWeights = []
  const indices = []
  let vOffset = 0

  for (const seg of segments) {
    const boneIndex = BONE_DEFS.findIndex((d) => d.name === seg.bone)
    if (boneIndex < 0) continue
    // 几何选择:capsule 用 length;box/sphere 不用 length
    let srcGeom
    if (geomType === 'sphere') {
      srcGeom = new THREE.SphereGeometry(seg.r, 16, 12)
    } else if (geomType === 'box') {
      // 头用球,其余用方块
      if (seg.sphere || seg.bone === 'head') {
        srcGeom = new THREE.BoxGeometry(seg.r * 2, seg.r * 2, seg.r * 2)
      } else {
        const len = seg.length || 0.2
        srcGeom = new THREE.BoxGeometry(seg.r * 2, len, seg.r * 2)
      }
    } else {
      // capsule(默认)
      srcGeom = seg.sphere
        ? new THREE.SphereGeometry(seg.r, 16, 12)
        : new THREE.CapsuleGeometry(seg.r, Math.max(seg.length - 2 * seg.r, 0.001), 4, 10)
    }
    const flat = srcGeom.toNonIndexed()
    const posAttr = flat.getAttribute('position').array
    const normAttr = flat.getAttribute('normal').array
    const idxAttr = flat.index ? flat.index.array : null
    const vCount = posAttr.length / 3

    const boneWorld = world[seg.bone]
    const r = (seg.color >> 16) & 0xff
    const g = (seg.color >> 8) & 0xff
    const b = seg.color & 0xff
    const colorSRGB = [r / 255, g / 255, b / 255]

    for (let v = 0; v < vCount; v++) {
      const lx = posAttr[v * 3], ly = posAttr[v * 3 + 1], lz = posAttr[v * 3 + 2]
      // limb 局部 → 世界(锚点 = bone origin)
      const worldV = new THREE.Vector3(lx, ly, lz)
        .applyQuaternion(boneWorld.worldQuat)
        .add(boneWorld.worldPos)
      // 世界 → 该骨骼的局部(绑定的目标空间)
      const bindV = worldV.clone().sub(boneWorld.worldPos).applyQuaternion(boneWorld.invWorldQuat)
      positions.push(bindV.x, bindV.y, bindV.z)
      // 法线:同样变换
      const nWorld = new THREE.Vector3(normAttr[v*3], normAttr[v*3+1], normAttr[v*3+2])
        .applyQuaternion(boneWorld.worldQuat)
        .applyQuaternion(boneWorld.invWorldQuat)
      normals.push(nWorld.x, nWorld.y, nWorld.z)
      colors.push(colorSRGB[0], colorSRGB[1], colorSRGB[2])
      skinIndices.push(boneIndex, 0, 0, 0)
      skinWeights.push(1, 0, 0, 0)
    }
    if (idxAttr) {
      for (let k = 0; k < idxAttr.length; k++) indices.push(idxAttr[k] + vOffset)
    } else {
      for (let k = 0; k < vCount; k++) indices.push(k + vOffset)
    }
    vOffset += vCount
    srcGeom.dispose()
    flat.dispose()
  }

  const geom = new THREE.BufferGeometry()
  geom.setAttribute('position',   new THREE.Float32BufferAttribute(positions, 3))
  geom.setAttribute('normal',     new THREE.Float32BufferAttribute(normals, 3))
  geom.setAttribute('color',      new THREE.Float32BufferAttribute(colors, 3))
  geom.setAttribute('skinIndex',  new THREE.Uint16BufferAttribute(skinIndices, 4))
  geom.setAttribute('skinWeight', new THREE.Float32BufferAttribute(skinWeights, 4))
  geom.setIndex(indices)
  geom.computeBoundingSphere()

  const mat = new THREE.MeshStandardMaterial({
    vertexColors: true,
    roughness: 0.55,
    metalness: 0.05,
  })
  const mesh = new THREE.SkinnedMesh(geom, mat)
  mesh.castShadow = true
  mesh.receiveShadow = true
  mesh.frustumCulled = false

  const bonesArr = []
  const walk = (b) => { bonesArr.push(b); b.children.forEach(walk) }
  walk(root)
  const skeleton = new THREE.Skeleton(bonesArr)
  mesh.bind(skeleton)

  return { mesh, skeleton, bones: bonesArr, byName, root }
}

// 几个常用 mocap 关键姿态预设 —— 后续可以扩展成 .json 导入
const POSE_PRESETS = {
  'T-pose': {},
  'Wave': {
    'forearm_r': { rx: -2.4 },
    'hand_r': { rx: -0.4 },
  },
  'Salute': {
    'upperarm_r': { rx: -1.2, rz: -0.3 },
    'forearm_r': { rx: -1.9 },
    'hand_r': { rx: 0.2 },
  },
  'Crouch': {
    'hips': { rx: 0.4 },
    'upperleg_l': { rx: -1.1, rz: 0.05 },
    'upperleg_r': { rx: -1.1, rz: -0.05 },
    'lowerleg_l': { rx: 1.4 },
    'lowerleg_r': { rx: 1.4 },
    'spine': { rx: 0.15 },
  },
  'Walk-L': {
    'upperleg_l': { rx: -0.6 },
    'lowerleg_l': { rx: 0.3 },
    'foot_l': { rx: 0.4 },
    'upperleg_r': { rx: 0.4 },
    'lowerleg_r': { rx: -0.1 },
    'upperarm_l': { rx: 0.4 },
    'upperarm_r': { rx: -0.4 },
  },
}

/**
 * @param {HTMLCanvasElement} canvas
 */
export function createAvatarScene(canvas) {
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
  renderer.outputColorSpace = THREE.SRGBColorSpace

  const scene = new THREE.Scene()
  scene.background = new THREE.Color('#eef2f7')

  const camera = new THREE.PerspectiveCamera(40, 1, 0.1, 100)
  camera.position.set(2.5, 1.4, 3.2)
  camera.lookAt(0, 1.0, 0)

  scene.add(new THREE.AmbientLight(0xffffff, 0.55))
  const dir = new THREE.DirectionalLight(0xffffff, 0.7)
  dir.position.set(3, 5, 4)
  scene.add(dir)
  const fill = new THREE.DirectionalLight(0xc4d4ff, 0.35)
  fill.position.set(-3, 2, -2)
  scene.add(fill)

  const grid = new THREE.GridHelper(8, 16, '#bcc6d6', '#dde3ec')
  scene.add(grid)

  const initial = buildSkinnedMesh(
    VISUAL_PRESETS.human.segments,
    VISUAL_PRESETS.human.geomType
  )
  let { mesh, skeleton, bones, byName, root } = initial
  // 骨头树单独挂入场景,便于 SkeletonHelper 显示
  scene.add(root)
  scene.add(mesh)

  let skelHelper = new THREE.SkeletonHelper(mesh)
  skelHelper.material.linewidth = 2
  skelHelper.visible = false
  scene.add(skelHelper)

  const controls = new OrbitControls(camera, canvas)
  controls.target.set(0, 1.0, 0)
  controls.enableDamping = true
  controls.update()

  // 当前选中的 visual preset / pose —— applyVisualPreset 会切 preset 但保留 pose
  let currentVisualPreset = 'human'
  let currentPose = 'T-pose'

  // 切换 visual preset —— 重建 SkinnedMesh,几何 + 颜色换,骨骼树形状不变
  function applyVisualPreset(name) {
    const preset = VISUAL_PRESETS[name]
    if (!preset) return false
    if (currentVisualPreset === name && mesh) return true
    // 卸下旧 mesh + helper
    scene.remove(mesh)
    scene.remove(skelHelper)
    mesh.geometry?.dispose?.()
    if (Array.isArray(mesh.material)) mesh.material.forEach((m) => m.dispose?.())
    else mesh.material?.dispose?.()
    skelHelper.dispose?.()
    // 重建
    const built = buildSkinnedMesh(preset.segments, preset.geomType)
    scene.add(built.root)
    scene.add(built.mesh)
    const newHelper = new THREE.SkeletonHelper(built.mesh)
    newHelper.material.linewidth = 2
    newHelper.visible = skelHelper.visible
    scene.add(newHelper)
    // 替换闭包引用
    mesh = built.mesh
    skeleton = built.skeleton
    bones = built.bones
    byName = built.byName
    root = built.root
    skelHelper = newHelper
    currentVisualPreset = name
    // 应用当前姿态
    applyPose(currentPose)
    return true
  }

  function applyPose(presetName) {
    const def = POSE_PRESETS[presetName]
    if (!def) return false
    for (const b of bones) b.rotation.set(0, 0, 0)
    for (const [boneName, r] of Object.entries(def)) {
      const b = byName[boneName]
      if (!b) continue
      if (typeof r.rx === 'number') b.rotation.x = r.rx
      if (typeof r.ry === 'number') b.rotation.y = r.ry
      if (typeof r.rz === 'number') b.rotation.z = r.rz
    }
    currentPose = presetName
    skeleton.update()
    return true
  }

  // 动捕桥接 —— 接 kalidokit Pose.solve 输出,直接把 quaternion 写到 byName 骨骼
  // kalidokit 没有 chest/neck/head/foot —— 这几个由 spine 补偿
  function applyLandmarks(rigged) {
    if (!rigged || !mesh) return
    const M = {
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
    for (const k of Object.keys(M)) {
      const slot = rigged[k]
      if (!slot || !slot.rotation) continue
      const b = byName[M[k]]
      if (!b) continue
      const q = slot.rotation
      if (typeof q.x === 'number') {
        b.quaternion.set(q.x, q.y, q.z, q.w)
        b.rotation.set(0, 0, 0)
      }
    }
    // chest / neck / head 从 spine 平滑过渡
    const spineSlot = rigged.Spine?.rotation
    const sBone = byName['spine']
    const cBone = byName['chest']
    const nBone = byName['neck']
    const hBone = byName['head']
    if (spineSlot && sBone && cBone && nBone && typeof spineSlot.x === 'number') {
      const sq = sBone.quaternion
      // chest = spine × 70%,neck = spine × 35%,head = spine(简化)
      const qChest = sq.clone().slerp(new THREE.Quaternion(), 0.30)
      const qNeck = sq.clone().slerp(new THREE.Quaternion(), 0.65)
      cBone.quaternion.copy(qChest)
      nBone.quaternion.copy(qNeck)
      if (hBone) hBone.quaternion.copy(sq)
    }
    skeleton.update()
  }

  function resetMocap() {
    applyPose(currentPose)
  }
  applyPose('T-pose')

  function resize() {
    const w = canvas.clientWidth || 1
    const h = canvas.clientHeight || 1
    if (canvas.width !== w || canvas.height !== h) {
      renderer.setSize(w, h, false)
      camera.aspect = w / h
      camera.updateProjectionMatrix()
    }
  }
  let running = true
  let rafHandle = 0
  let disposed = false
  let onFrame = null
  function loop() {
    // 已 dispose / 已停 → 不再排下一帧;若已被 cancelAnimationFrame,也不再排。
    if (!running || disposed) return
    resize()
    if (typeof onFrame === 'function') onFrame(skeleton, byName)
    controls.update()
    renderer.render(scene, camera)
    rafHandle = requestAnimationFrame(loop)
  }
  rafHandle = requestAnimationFrame(loop)

  // 释放单个材质 + 它附带的纹理(R8,准备给 GLTFLoader)
  function disposeMaterial(m) {
    if (!m) return
    // 释放材质引用的所有纹理(map / normalMap / roughnessMap 等)
    for (const key in m) {
      const v = m[key]
      if (v && typeof v === 'object' && typeof v.dispose === 'function' && v.isTexture) {
        v.dispose?.()
      }
    }
    m.dispose?.()
  }

  function dispose() {
    if (disposed) return
    disposed = true
    running = false
    // 取消已 enqueue 的那一帧,避免 dispose 后仍执行 renderer.render 持强引用(R3)。
    if (rafHandle) {
      cancelAnimationFrame(rafHandle)
      rafHandle = 0
    }
    controls.dispose()
    scene.traverse((obj) => {
      if (obj.geometry) obj.geometry.dispose?.()
      if (obj.material) {
        if (Array.isArray(obj.material)) obj.material.forEach(disposeMaterial)
        else disposeMaterial(obj.material)
      }
    })
    renderer.dispose()
    // 清掉内部引用,帮 GC。
    onFrame = null
  }

  return {
    renderer,
    scene,
    camera,
    controls,
    // 用 getter 让外部总是读到最新引用(applyVisualPreset 重建后)
    get mesh() { return mesh },
    get skeleton() { return skeleton },
    get bones() { return bones },
    get byName() { return byName },
    get root() { return root },
    get skelHelper() { return skelHelper },
    get boneNames() { return Object.keys(byName) },
    poses: Object.keys(POSE_PRESETS),
    visualPresets: Object.keys(VISUAL_PRESETS).map((k) => ({ name: k, label: VISUAL_PRESETS[k].label, desc: VISUAL_PRESETS[k].desc, accent: VISUAL_PRESETS[k].accent })),
    get currentVisualPreset() { return currentVisualPreset },
    get currentPose() { return currentPose },
    applyPose,
    applyVisualPreset,
    applyLandmarks,
    resetMocap,
    setOnFrame: (fn) => { onFrame = fn },
    setSkeletonVisible: (v) => { skelHelper.visible = !!v },
    dispose,
  }
}
