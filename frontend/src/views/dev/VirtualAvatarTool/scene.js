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

// 身体段定义 — 颜色 + 半径 + 长度
const SEGMENTS = [
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
function buildSkinnedMesh() {
  const { root, byName, world } = collectWorldRestPoses()

  const positions = []
  const colors = []
  const normals = []
  const skinIndices = []
  const skinWeights = []
  const indices = []
  let vOffset = 0

  for (const seg of SEGMENTS) {
    const boneIndex = BONE_DEFS.findIndex((d) => d.name === seg.bone)
    if (boneIndex < 0) continue
    const srcGeom = seg.sphere
      ? new THREE.SphereGeometry(seg.r, 16, 12)
      : new THREE.CapsuleGeometry(seg.r, Math.max(seg.length - 2 * seg.r, 0.001), 4, 10)
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

  const { mesh, skeleton, bones, byName, root } = buildSkinnedMesh()
  // 骨头树单独挂入场景,便于 SkeletonHelper 显示
  scene.add(root)
  scene.add(mesh)

  const skelHelper = new THREE.SkeletonHelper(mesh)
  skelHelper.material.linewidth = 2
  skelHelper.visible = false
  scene.add(skelHelper)

  const controls = new OrbitControls(camera, canvas)
  controls.target.set(0, 1.0, 0)
  controls.enableDamping = true
  controls.update()

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
    skeleton.update()
    return true
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
  let onFrame = null
  function loop() {
    if (!running) return
    resize()
    if (typeof onFrame === 'function') onFrame(skeleton, byName)
    controls.update()
    renderer.render(scene, camera)
    requestAnimationFrame(loop)
  }
  loop()

  function dispose() {
    running = false
    controls.dispose()
    renderer.dispose()
    scene.traverse((obj) => {
      if (obj.geometry) obj.geometry.dispose?.()
      if (obj.material) {
        if (Array.isArray(obj.material)) obj.material.forEach((m) => m.dispose?.())
        else obj.material.dispose?.()
      }
    })
  }

  return {
    renderer,
    scene,
    camera,
    controls,
    mesh,
    skeleton,
    bones,
    byName,
    root,
    skelHelper,
    boneNames: Object.keys(byName),
    poses: Object.keys(POSE_PRESETS),
    applyPose,
    setOnFrame: (fn) => { onFrame = fn },
    setSkeletonVisible: (v) => { skelHelper.visible = !!v },
    dispose,
  }
}
