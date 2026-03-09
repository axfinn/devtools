<template>
  <div class="space3d-page">
    <div v-if="!profileId && !shareId" class="space3d-empty">
      <el-empty description="请先在整理模块登录档案" />
    </div>
    <div v-else class="space3d-layout">
      <section class="space3d-sidebar">
        <template v-if="readOnly">
          <div class="sidebar-header">
            <div class="profile-tag">
              <el-tag type="info">分享预览</el-tag>
            </div>
            <div class="sidebar-actions">
              <el-button size="small" @click="captureSnapshot">截图</el-button>
            </div>
          </div>

          <div class="sidebar-card">
            <div class="card-title">布局摘要</div>
            <div class="summary-item">模板：{{ templateLabel(selectedTemplate) || '未设置' }}</div>
            <div class="summary-item">标注数量：{{ markers.length }}</div>
          </div>

          <div class="sidebar-card">
            <div class="card-title">位置清单</div>
            <el-table :data="markers" height="320">
              <el-table-column prop="name" label="位置" />
              <el-table-column label="物品">
                <template #default="{ row }">
                  <span>{{ row.itemName || '-' }}</span>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-if="markers.length === 0" description="暂无标注" />
          </div>
        </template>

        <template v-else>
        <div class="sidebar-header">
          <div class="profile-tag">
            <el-tag type="success">{{ profileName || '我的家庭物品' }}</el-tag>
          </div>
          <div class="sidebar-actions">
            <el-button size="small" @click="loadLocationLibrary">刷新位置</el-button>
            <el-button size="small" type="primary" @click="saveLayout">保存布局</el-button>
          </div>
        </div>

        <div class="sidebar-card">
          <div class="card-title">位置库</div>
          <div class="location-form">
            <el-input v-model="newLocationName" placeholder="新增位置，如：厨房水槽下" />
            <el-button type="primary" @click="createLocation">新增</el-button>
          </div>
          <el-select v-model="selectedLocation" placeholder="选择要标注的位置" style="width: 100%;">
            <el-option v-for="loc in locationLibrary" :key="loc.id" :label="loc.name" :value="loc.name" />
          </el-select>
          <div class="location-actions">
            <el-button type="success" :disabled="!selectedLocation" @click="startPlacing">
              开始标注
            </el-button>
            <el-button @click="stopPlacing" :disabled="!placing">取消标注</el-button>
          </div>
        </div>

        <div class="sidebar-card">
          <div class="card-title">模板规划</div>
          <el-select v-model="selectedTemplate" placeholder="选择模板" style="width: 100%;">
            <el-option v-for="tpl in templates" :key="tpl.value" :label="tpl.label" :value="tpl.value" />
          </el-select>
          <div class="template-actions">
            <el-button type="primary" :disabled="!selectedTemplate" @click="applyTemplate">应用模板</el-button>
            <el-button @click="clearTemplate">清空模板</el-button>
          </div>
        </div>

        <div class="sidebar-card">
          <div class="card-title">标注列表</div>
          <el-table :data="markers" height="220">
            <el-table-column prop="name" label="位置" />
            <el-table-column label="绑定物品" min-width="140">
              <template #default="{ row }">
                <el-select v-model="row.itemId" placeholder="选择物品" size="small" style="width: 100%;" @change="resolveMarkerItemName(row)">
                  <el-option v-for="item in items" :key="item.id" :label="item.name" :value="item.id" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button size="small" type="primary" text @click="bindMarkerItem(row)">绑定</el-button>
                <el-button size="small" type="danger" text @click="removeMarker(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="marker-actions">
            <el-button size="small" @click="clearMarkers">清空标注</el-button>
          </div>
        </div>

        <div class="sidebar-card">
          <div class="card-title">导入设置</div>
          <el-switch v-model="backupBeforeImport" active-text="导入前自动备份" />
        </div>

        <div class="sidebar-tip">
          <div class="tip-title">操作提示</div>
          <ul>
            <li>滚轮缩放，左键拖动旋转，右键拖动平移</li>
            <li>选择位置后点击“开始标注”，再点击地面放置标记</li>
          </ul>
        </div>
        </template>
      </section>

      <section class="space3d-canvas">
        <div class="canvas-header">
          <div class="canvas-status">
            <el-tag v-if="readOnly" type="info">分享预览</el-tag>
            <el-tag v-else-if="placing" type="warning">标注中：{{ selectedLocation }}</el-tag>
            <el-tag v-else type="info">视图模式</el-tag>
          </div>
          <div class="canvas-actions">
            <el-button size="small" @click="resetCamera">重置视角</el-button>
            <el-button v-if="!readOnly" size="small" @click="exportLayout">导出布局</el-button>
            <el-button v-if="!readOnly" size="small" @click="importLayout">导入布局</el-button>
            <el-button size="small" @click="captureSnapshot">截图</el-button>
            <el-button v-if="!readOnly" size="small" type="primary" @click="shareSnapshot">分享</el-button>
          </div>
        </div>
        <div ref="canvasRef" class="canvas-area"></div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'

const API_BASE = '/api'
const canvasRef = ref(null)
const profileId = ref('')
const creatorKey = ref('')
const profileName = ref('')
const shareId = ref('')
const readOnly = computed(() => !profileId.value && !!shareId.value)

const locationLibrary = ref([])
const newLocationName = ref('')
const selectedLocation = ref('')
const placing = ref(false)
const markers = ref([])
const items = ref([])
const backupBeforeImport = ref(true)

const templates = [
  { value: 'studio', label: '单间工作室' },
  { value: 'one-bedroom', label: '一室一厅' },
  { value: 'kitchen-living', label: '开放式客厅+厨房' }
]
const selectedTemplate = ref('')

let scene = null
let camera = null
let renderer = null
let controls = null
let raycaster = null
let floorMesh = null
let templateGroup = null
let markerGroup = null
let animationId = null
let isLoadingLayout = false

function layoutKey() {
  return `household_3d_layout_${profileId.value}`
}

function initScene() {
  if (!canvasRef.value) return

  scene = new THREE.Scene()
  scene.background = new THREE.Color('#f6f8fb')

  camera = new THREE.PerspectiveCamera(45, canvasRef.value.clientWidth / canvasRef.value.clientHeight, 0.1, 1000)
  camera.position.set(12, 12, 12)

  renderer = new THREE.WebGLRenderer({ antialias: true })
  renderer.setSize(canvasRef.value.clientWidth, canvasRef.value.clientHeight)
  renderer.setPixelRatio(window.devicePixelRatio || 1)
  canvasRef.value.appendChild(renderer.domElement)

  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true

  const ambient = new THREE.AmbientLight(0xffffff, 0.7)
  scene.add(ambient)

  const dirLight = new THREE.DirectionalLight(0xffffff, 0.8)
  dirLight.position.set(10, 15, 8)
  scene.add(dirLight)

  const grid = new THREE.GridHelper(30, 30, '#d0d7e2', '#e3e7ee')
  scene.add(grid)

  const floorGeom = new THREE.PlaneGeometry(30, 30)
  const floorMat = new THREE.MeshStandardMaterial({ color: '#ffffff', transparent: true, opacity: 0.7 })
  floorMesh = new THREE.Mesh(floorGeom, floorMat)
  floorMesh.rotation.x = -Math.PI / 2
  floorMesh.name = 'floor'
  scene.add(floorMesh)

  templateGroup = new THREE.Group()
  scene.add(templateGroup)

  markerGroup = new THREE.Group()
  scene.add(markerGroup)

  raycaster = new THREE.Raycaster()
  renderer.domElement.addEventListener('pointerdown', handlePointerDown)
  window.addEventListener('resize', handleResize)

  animate()
}

function animate() {
  animationId = requestAnimationFrame(animate)
  if (controls) controls.update()
  if (renderer && scene && camera) renderer.render(scene, camera)
}

function handleResize() {
  if (!canvasRef.value || !camera || !renderer) return
  const { clientWidth, clientHeight } = canvasRef.value
  camera.aspect = clientWidth / clientHeight
  camera.updateProjectionMatrix()
  renderer.setSize(clientWidth, clientHeight)
}

function handlePointerDown(event) {
  if (!placing.value || !selectedLocation.value || !renderer || !camera) return

  const rect = renderer.domElement.getBoundingClientRect()
  const x = ((event.clientX - rect.left) / rect.width) * 2 - 1
  const y = -((event.clientY - rect.top) / rect.height) * 2 + 1

  raycaster.setFromCamera({ x, y }, camera)
  const intersects = raycaster.intersectObject(floorMesh)
  if (intersects.length === 0) return

  const point = intersects[0].point
  addMarker(selectedLocation.value, point.x, point.z)
}

function makeTextSprite(text) {
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  const padding = 8
  ctx.font = '14px sans-serif'
  const textWidth = ctx.measureText(text).width
  canvas.width = textWidth + padding * 2
  canvas.height = 28
  ctx.font = '14px sans-serif'
  ctx.fillStyle = '#1f2d3d'
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  ctx.fillStyle = '#ffffff'
  ctx.textBaseline = 'middle'
  ctx.fillText(text, padding, canvas.height / 2)

  const texture = new THREE.CanvasTexture(canvas)
  const material = new THREE.SpriteMaterial({ map: texture })
  const sprite = new THREE.Sprite(material)
  sprite.scale.set(canvas.width / 50, canvas.height / 50, 1)
  return sprite
}

function addMarker(name, x, z, markerId, skipSave, itemId, itemName) {
  const id = markerId || `${Date.now()}-${Math.random().toString(16).slice(2)}`
  const marker = new THREE.Mesh(
    new THREE.SphereGeometry(0.25, 16, 16),
    new THREE.MeshStandardMaterial({ color: '#409eff' })
  )
  marker.userData = { id, name, isMarker: true, originalColor: '#409eff' }
  marker.position.set(x, 0.25, z)

  const label = makeTextSprite(name)
  label.position.set(x, 1, z)
  label.userData = { id, name, isLabel: true }

  markerGroup.add(marker)
  markerGroup.add(label)

  markers.value.push({ id, name, x, z, itemId: itemId || '', itemName: itemName || '' })
  if (!skipSave) {
    saveLayout()
  }
}

function removeMarker(id) {
  markers.value = markers.value.filter(m => m.id !== id)
  const objectsToRemove = markerGroup.children.filter(obj => obj.userData && obj.userData.id === id)
  objectsToRemove.forEach(obj => markerGroup.remove(obj))
  saveLayout()
}

function clearMarkers() {
  markers.value = []
  markerGroup.clear()
  saveLayout()
}

function startPlacing() {
  if (!selectedLocation.value) {
    ElMessage.warning('请选择位置')
    return
  }
  if (readOnly.value) return
  placing.value = true
}

function stopPlacing() {
  if (readOnly.value) return
  placing.value = false
}

function resetCamera() {
  if (!camera || !controls) return
  camera.position.set(12, 12, 12)
  controls.target.set(0, 0, 0)
  controls.update()
}

function highlightLocation(name) {
  if (!markerGroup || !name) return
  markerGroup.children.forEach(child => {
    if (child.userData && child.userData.isMarker) {
      const target = child.userData.name === name
      const material = child.material
      material.color.set(target ? '#f59e0b' : child.userData.originalColor)
      child.scale.setScalar(target ? 1.4 : 1)
    }
  })
}

function highlightFromQuery() {
  const params = new URLSearchParams(window.location.search)
  const target = params.get('highlight')
  if (target) {
    highlightLocation(target)
  }
}

async function loadSharedLayout() {
  const params = new URLSearchParams(window.location.search)
  const share = params.get('share')
  if (!share) return false
  shareId.value = share
  try {
    const res = await fetch(`${API_BASE}/household/space/share/${share}`)
    const data = await res.json()
    if (data.code === 0 && data.data && data.data.content) {
      loadLayoutFromContent(data.data.content)
      return true
    }
  } catch (e) {
    console.error('分享布局加载失败:', e)
  }
  return false
}

function buildSnapshotCanvas() {
  if (!renderer) return null
  const source = renderer.domElement
  const canvas = document.createElement('canvas')
  canvas.width = source.width
  canvas.height = source.height
  const ctx = canvas.getContext('2d')
  if (!ctx) return null

  ctx.drawImage(source, 0, 0)

  const barHeight = Math.max(40, Math.floor(canvas.height * 0.08))
  ctx.fillStyle = 'rgba(0, 0, 0, 0.5)'
  ctx.fillRect(0, canvas.height - barHeight, canvas.width, barHeight)

  const timestamp = new Date().toLocaleString()
  const title = `整理空间3D · ${profileName.value || '我的家庭物品'}`
  ctx.fillStyle = '#ffffff'
  ctx.font = `${Math.max(12, Math.floor(barHeight * 0.35))}px sans-serif`
  ctx.textBaseline = 'middle'
  ctx.fillText(title, 16, canvas.height - barHeight / 2)

  const meta = `标注 ${markers.value.length} 个 · ${timestamp}`
  const metaWidth = ctx.measureText(meta).width
  ctx.fillText(meta, canvas.width - metaWidth - 16, canvas.height - barHeight / 2)

  return canvas
}

function captureCanvasDataURL() {
  const canvas = buildSnapshotCanvas()
  if (!canvas) return ''
  return canvas.toDataURL('image/png')
}

function captureSnapshot() {
  const dataUrl = captureCanvasDataURL()
  if (!dataUrl) {
    ElMessage.error('截图失败')
    return
  }
  const link = document.createElement('a')
  link.href = dataUrl
  link.download = `space3d_${Date.now()}.png`
  link.click()
  ElMessage.success('截图已下载')
}

async function shareSnapshot() {
  const dataUrl = captureCanvasDataURL()
  if (!dataUrl) {
    ElMessage.error('截图失败')
    return
  }

  if (!navigator.share) {
    window.open(dataUrl, '_blank')
    ElMessage.info('已打开图片，可手动保存分享')
    return
  }

  try {
    const res = await fetch(dataUrl)
    const blob = await res.blob()
    const file = new File([blob], `space3d_${Date.now()}.png`, { type: 'image/png' })
    if (navigator.canShare && !navigator.canShare({ files: [file] })) {
      window.open(dataUrl, '_blank')
      ElMessage.info('当前设备不支持分享文件')
      return
    }
    await navigator.share({
      title: '整理空间3D',
      text: '我的整理空间标注',
      files: [file]
    })
  } catch (e) {
    ElMessage.error('分享失败')
  }
}

function exportLayout() {
  if (!profileId.value) return
  const payload = JSON.stringify({
    template: selectedTemplate.value,
    markers: markers.value
  }, null, 2)
  const blob = new Blob([payload], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `space3d_layout_${profileId.value}.json`
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('布局已导出')
  shareLayoutLink()
}

function backupLayout() {
  const payload = JSON.stringify({
    template: selectedTemplate.value,
    markers: markers.value
  }, null, 2)
  const timestamp = Date.now()
  localStorage.setItem(`${layoutKey()}_backup_${timestamp}`, payload)
  const blob = new Blob([payload], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `space3d_layout_backup_${timestamp}.json`
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('已备份当前布局')
}

async function shareLayoutLink() {
  if (!profileId.value) return
  try {
    const shareRes = await fetch(`${API_BASE}/household/profile/${profileId.value}/space/share?creator_key=${creatorKey.value}`, {
      method: 'POST'
    })
    const shareData = await shareRes.json()
    if (shareData.code !== 0 || !shareData.share_url) {
      ElMessage.error(shareData.error || '创建分享失败')
      return
    }
    const shareUrl = `${window.location.origin}${shareData.share_url}`
    const res = await fetch(`${API_BASE}/shorturl`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ original_url: shareUrl })
    })
    const data = await res.json()
    if (data.id) {
      const shortUrl = `${window.location.origin}/s/${data.id}`
      await navigator.clipboard.writeText(shortUrl)
      ElMessage.success('分享短链已复制到剪贴板')
      return
    }
    await navigator.clipboard.writeText(shareUrl)
    ElMessage.success('分享链接已复制到剪贴板')
  } catch (e) {
    ElMessage.error('生成分享链接失败')
  }
}

function importLayout() {
  if (readOnly.value) return
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'application/json'
  input.onchange = (event) => {
    const file = event.target.files && event.target.files[0]
    if (!file) return
    if (backupBeforeImport.value) {
      backupLayout()
    }
    const reader = new FileReader()
    reader.onload = () => {
      try {
        const content = typeof reader.result === 'string' ? reader.result : ''
        markerGroup.clear()
        markers.value = []
        templateGroup.clear()
        selectedTemplate.value = ''
        loadLayoutFromContent(content)
        saveLayout()
        ElMessage.success('布局已导入')
      } catch (e) {
        ElMessage.error('布局解析失败')
      }
    }
    reader.readAsText(file)
  }
  input.click()
}

function templateWalls(segments) {
  templateGroup.clear()
  segments.forEach(seg => {
    const geom = new THREE.BoxGeometry(seg.w, seg.h, seg.d)
    const mat = new THREE.MeshStandardMaterial({ color: '#e6ebf2' })
    const mesh = new THREE.Mesh(geom, mat)
    mesh.position.set(seg.x, seg.h / 2, seg.z)
    templateGroup.add(mesh)
  })
  saveLayout()
}

function applyTemplate() {
  if (!selectedTemplate.value) return
  if (selectedTemplate.value === 'studio') {
    templateWalls([
      { x: 0, z: -7, w: 12, h: 2.4, d: 0.2 },
      { x: 0, z: 7, w: 12, h: 2.4, d: 0.2 },
      { x: -6, z: 0, w: 0.2, h: 2.4, d: 14 },
      { x: 6, z: 0, w: 0.2, h: 2.4, d: 14 }
    ])
  }
  if (selectedTemplate.value === 'one-bedroom') {
    templateWalls([
      { x: 0, z: -8, w: 14, h: 2.4, d: 0.2 },
      { x: 0, z: 8, w: 14, h: 2.4, d: 0.2 },
      { x: -7, z: 0, w: 0.2, h: 2.4, d: 16 },
      { x: 7, z: 0, w: 0.2, h: 2.4, d: 16 },
      { x: 0, z: 0, w: 0.2, h: 2.4, d: 16 }
    ])
  }
  if (selectedTemplate.value === 'kitchen-living') {
    templateWalls([
      { x: 0, z: -6, w: 14, h: 2.4, d: 0.2 },
      { x: 0, z: 6, w: 14, h: 2.4, d: 0.2 },
      { x: -7, z: 0, w: 0.2, h: 2.4, d: 12 },
      { x: 7, z: 0, w: 0.2, h: 2.4, d: 12 },
      { x: -2.5, z: 0, w: 0.2, h: 2.4, d: 12 }
    ])
  }
}

function clearTemplate() {
  templateGroup.clear()
  selectedTemplate.value = ''
  saveLayout()
}

function templateLabel(value) {
  const tpl = templates.find(t => t.value === value)
  return tpl ? tpl.label : ''
}

function saveLayout() {
  if (!profileId.value || isLoadingLayout || readOnly.value) return
  const layout = {
    markers: markers.value,
    template: selectedTemplate.value
  }
  const payload = JSON.stringify(layout)
  localStorage.setItem(layoutKey(), payload)
  saveLayoutToServer(payload)
}

function loadLayoutFromContent(raw) {
  if (!raw) return
  isLoadingLayout = true
  try {
    const layout = JSON.parse(raw)
    if (layout.template) {
      selectedTemplate.value = layout.template
      applyTemplate()
    }
    if (Array.isArray(layout.markers)) {
      layout.markers.forEach(m => addMarker(m.name, m.x, m.z, m.id, true, m.itemId, m.itemName))
    }
  } catch (e) {
    console.error('布局加载失败', e)
  } finally {
    isLoadingLayout = false
  }
}

function loadLayout() {
  const raw = localStorage.getItem(layoutKey())
  if (!raw) return
  loadLayoutFromContent(raw)
}

async function loadLocationLibrary() {
  if (!profileId.value) return
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/locations/library?creator_key=${creatorKey.value}`)
    const data = await res.json()
    if (data.code === 0) {
      locationLibrary.value = data.data || []
    }
  } catch (e) {
    console.error('位置库加载失败:', e)
  }
}

async function loadItems() {
  if (!profileId.value) return
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items?creator_key=${creatorKey.value}`)
    const data = await res.json()
    if (data.code === 0) {
      items.value = data.data || []
      markers.value.forEach(resolveMarkerItemName)
    }
  } catch (e) {
    console.error('物品加载失败:', e)
  }
}

function resolveMarkerItemName(marker) {
  if (!marker || !marker.itemId) return
  const item = items.value.find(i => i.id === marker.itemId)
  if (item) {
    marker.itemName = item.name
  }
}

async function bindMarkerItem(marker) {
  if (!marker || !marker.itemId || !profileId.value) return
  const item = items.value.find(i => i.id === marker.itemId)
  if (!item) {
    ElMessage.warning('未找到物品')
    return
  }
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}?creator_key=${creatorKey.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ location: marker.name })
    })
    const data = await res.json()
    if (data.code === 0) {
      marker.itemName = item.name
      ElMessage.success(`已绑定并更新位置为：${marker.name}`)
      saveLayout()
    } else {
      ElMessage.error(data.error || '绑定失败')
    }
  } catch (e) {
    ElMessage.error('绑定失败')
  }
}

async function loadLayoutFromServer() {
  if (!profileId.value) return
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/space?creator_key=${creatorKey.value}`)
    const data = await res.json()
    if (data.code === 0 && data.data && data.data.content) {
      loadLayoutFromContent(data.data.content)
      return true
    }
  } catch (e) {
    console.error('加载空间布局失败:', e)
  }
  return false
}

async function saveLayoutToServer(payload) {
  if (!profileId.value) return
  try {
    await fetch(`${API_BASE}/household/profile/${profileId.value}/space?creator_key=${creatorKey.value}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: payload })
    })
  } catch (e) {
    console.error('保存空间布局失败:', e)
  }
}

async function createLocation() {
  const name = newLocationName.value.trim()
  if (!name) return
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/locations/library?creator_key=${creatorKey.value}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name })
    })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('已添加位置')
      newLocationName.value = ''
      await loadLocationLibrary()
    } else {
      ElMessage.error(data.error || '添加失败')
    }
  } catch (e) {
    ElMessage.error('添加失败')
  }
}

onMounted(async () => {
  const saved = localStorage.getItem('household_profile')
  if (saved) {
    try {
      const profile = JSON.parse(saved)
      profileId.value = profile.id
      creatorKey.value = profile.creator_key
      profileName.value = profile.name
      await loadLocationLibrary()
      await loadItems()
    } catch (e) {
      console.error('加载档案失败', e)
    }
  }

  initScene()
  const shared = await loadSharedLayout()
  if (!shared) {
    const loaded = await loadLayoutFromServer()
    if (!loaded) {
      loadLayout()
    }
  }
  highlightFromQuery()
})

onBeforeUnmount(() => {
  if (animationId) cancelAnimationFrame(animationId)
  if (renderer && renderer.domElement) {
    renderer.domElement.removeEventListener('pointerdown', handlePointerDown)
  }
  window.removeEventListener('resize', handleResize)
  if (renderer) renderer.dispose()
  scene = null
  camera = null
  renderer = null
  controls = null
  raycaster = null
  floorMesh = null
  templateGroup = null
  markerGroup = null
})
</script>

<style scoped>
.space3d-page {
  padding: 20px;
}

.space3d-layout {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 16px;
  min-height: 600px;
}

.space3d-sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sidebar-header {
  background: #fff;
  padding: 12px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.sidebar-actions {
  display: flex;
  gap: 8px;
}

.sidebar-card {
  background: #fff;
  padding: 12px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.summary-item {
  font-size: 13px;
  color: #606266;
  margin-bottom: 6px;
}

.card-title {
  font-weight: bold;
  margin-bottom: 10px;
}

.location-form {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.location-actions,
.template-actions,
.marker-actions {
  margin-top: 10px;
  display: flex;
  gap: 8px;
}

.sidebar-tip {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 8px;
  color: #606266;
  font-size: 12px;
}

.sidebar-tip ul {
  padding-left: 16px;
  margin: 8px 0 0;
}

.space3d-canvas {
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  display: flex;
  flex-direction: column;
}

.canvas-header {
  display: flex;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid #ebeef5;
}

.canvas-area {
  flex: 1;
  min-height: 560px;
}

.space3d-empty {
  background: #fff;
  border-radius: 8px;
  padding: 40px;
}

@media (max-width: 1024px) {
  .space3d-layout {
    grid-template-columns: 1fr;
  }

  .canvas-area {
    min-height: 420px;
  }
}
</style>
