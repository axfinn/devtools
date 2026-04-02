<template>
  <div class="mindmap-tool">
    <!-- 顶部工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <span class="tool-title">🧠 思维导图</span>
        <el-input v-model="title" placeholder="无标题" class="title-input" size="small" />
      </div>
      <div class="toolbar-center">
        <el-button-group>
          <el-button size="small" @click="activeTab = 'visual'" :type="activeTab === 'visual' ? 'primary' : ''">可视化</el-button>
          <el-button size="small" @click="activeTab = 'code'" :type="activeTab === 'code' ? 'primary' : ''">代码</el-button>
        </el-button-group>
      </div>
      <div class="toolbar-right">
        <el-tooltip content="我的导图" placement="bottom">
          <el-button size="small" @click="openManage"><el-icon><FolderOpened /></el-icon> 我的</el-button>
        </el-tooltip>
        <el-tooltip content="AI 生成" placement="bottom">
          <el-button size="small" type="primary" @click="showAIDialog = true">✨ AI 生成</el-button>
        </el-tooltip>
        <el-tooltip content="新建" placement="bottom">
          <el-button size="small" @click="newMap"><el-icon><DocumentAdd /></el-icon></el-button>
        </el-tooltip>
        <el-tooltip content="导入 JSON" placement="bottom">
          <el-button size="small" @click="triggerImport"><el-icon><Upload /></el-icon></el-button>
        </el-tooltip>
        <el-dropdown @command="handleExport">
          <el-button size="small">导出<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="svg">导出 SVG</el-dropdown-item>
              <el-dropdown-item command="png">导出 PNG</el-dropdown-item>
              <el-dropdown-item command="json">导出 JSON</el-dropdown-item>
              <el-dropdown-item command="md">导出 Markdown</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-tooltip content="云端保存" placement="bottom">
          <el-button size="small" type="success" @click="saveToCloud" :loading="saving">
            <el-icon><Upload /></el-icon> 保存
          </el-button>
        </el-tooltip>
        <el-tooltip content="分享" placement="bottom">
          <el-button size="small" @click="shareMap" :disabled="!cloudId">
            <el-icon><Share /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="缩小" placement="bottom">
          <el-button size="small" @click="zoom(-0.1)"><el-icon><Minus /></el-icon></el-button>
        </el-tooltip>
        <span class="zoom-label">{{ Math.round(zoomLevel * 100) }}%</span>
        <el-tooltip content="放大" placement="bottom">
          <el-button size="small" @click="zoom(0.1)"><el-icon><Plus /></el-icon></el-button>
        </el-tooltip>
        <el-tooltip content="适应屏幕" placement="bottom">
          <el-button size="small" @click="fitView"><el-icon><FullScreen /></el-icon></el-button>
        </el-tooltip>
      </div>
    </div>

    <!-- 主体区域 -->
    <div class="main-area">
      <!-- 可视化编辑器 -->
      <div v-show="activeTab === 'visual'" class="visual-area">
        <!-- 左侧节点属性面板 -->
        <div class="side-panel" v-if="selectedNode">
          <div class="panel-title">节点属性</div>
          <el-form size="small" label-width="50px">
            <el-form-item label="文本">
              <el-input v-model="selectedNode.text" @input="updateNodeText" />
            </el-form-item>
            <el-form-item label="颜色">
              <el-color-picker v-model="selectedNode.color" @change="updateNodeColor" size="small" />
            </el-form-item>
            <el-form-item label="形状">
              <el-select v-model="selectedNode.shape" @change="updateNodeShape" size="small">
                <el-option label="圆角矩形" value="rounded" />
                <el-option label="矩形" value="rect" />
                <el-option label="圆形" value="circle" />
                <el-option label="云形" value="cloud" />
                <el-option label="六边形" value="hexagon" />
              </el-select>
            </el-form-item>
          </el-form>
          <div class="panel-actions">
            <el-button size="small" type="primary" @click="addChild">+ 子节点</el-button>
            <el-button size="small" type="primary" @click="addSibling">+ 兄弟节点</el-button>
            <el-button size="small" type="danger" @click="deleteNode" :disabled="!selectedNode.parent">删除</el-button>
          </div>
        </div>

        <!-- 画布 -->
        <div
          class="canvas-wrap"
          ref="canvasWrap"
          @wheel.prevent="onWheel"
          @mousedown="onCanvasMouseDown"
          @mousemove="onCanvasMouseMove"
          @mouseup="onCanvasMouseUp"
          @mouseleave="onCanvasMouseUp"
          :class="{ dragging: isDragging }"
        >
          <svg
            ref="svgRef"
            class="mindmap-svg"
            :width="svgWidth"
            :height="svgHeight"
            :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
          >
            <g :transform="`translate(${panX}, ${panY}) scale(${zoomLevel})`">
              <!-- 连线 -->
              <g class="links-layer">
                <path
                  v-for="link in links"
                  :key="link.id"
                  :d="link.path"
                  :stroke="link.color"
                  stroke-width="2"
                  fill="none"
                  stroke-linecap="round"
                />
              </g>
              <!-- 节点 -->
              <g class="nodes-layer">
                <g
                  v-for="node in flatNodes"
                  :key="node.id"
                  :transform="`translate(${node.x}, ${node.y})`"
                  class="node-group"
                  :class="{ selected: selectedNode?.id === node.id }"
                  @click.stop="selectNode(node)"
                  @dblclick.stop="startEdit(node)"
                  style="cursor: pointer"
                >
                  <!-- 节点形状 -->
                  <rect
                    v-if="!node.shape || node.shape === 'rect' || node.shape === 'rounded'"
                    :x="-node.w/2" :y="-node.h/2"
                    :width="node.w" :height="node.h"
                    :rx="node.shape === 'rounded' || !node.shape ? 8 : 0"
                    :fill="node.color || defaultColor(node.depth)"
                    :stroke="selectedNode?.id === node.id ? '#409eff' : 'transparent'"
                    stroke-width="2"
                    class="node-rect"
                  />
                  <ellipse
                    v-else-if="node.shape === 'circle'"
                    :rx="node.w/2" :ry="node.h/2"
                    :fill="node.color || defaultColor(node.depth)"
                    :stroke="selectedNode?.id === node.id ? '#409eff' : 'transparent'"
                    stroke-width="2"
                  />
                  <polygon
                    v-else-if="node.shape === 'hexagon'"
                    :points="hexPoints(node.w, node.h)"
                    :fill="node.color || defaultColor(node.depth)"
                    :stroke="selectedNode?.id === node.id ? '#409eff' : 'transparent'"
                    stroke-width="2"
                  />
                  <!-- 文字 -->
                  <text
                    text-anchor="middle"
                    dominant-baseline="middle"
                    :font-size="node.depth === 0 ? 16 : 13"
                    :font-weight="node.depth === 0 ? 'bold' : 'normal'"
                    fill="#fff"
                    style="pointer-events: none; user-select: none"
                  >{{ node.text }}</text>
                  <!-- 折叠按钮 -->
                  <g
                    v-if="node.children && node.children.length"
                    :transform="`translate(${node.w/2 - 2}, 0)`"
                    @click.stop="toggleCollapse(node)"
                    style="cursor: pointer"
                  >
                    <circle r="8" fill="#fff" stroke="#ccc" stroke-width="1" />
                    <text text-anchor="middle" dominant-baseline="middle" font-size="12" fill="#666">
                      {{ node.collapsed ? '+' : '−' }}
                    </text>
                  </g>
                </g>
              </g>
              <!-- 内联编辑 -->
              <foreignObject
                v-if="editingNode"
                :x="editingNode.x - editingNode.w/2"
                :y="editingNode.y - editingNode.h/2"
                :width="editingNode.w"
                :height="editingNode.h"
              >
                <input
                  xmlns="http://www.w3.org/1999/xhtml"
                  ref="inlineInput"
                  v-model="editingText"
                  class="inline-edit"
                  @keydown.enter="commitEdit"
                  @keydown.escape="cancelEdit"
                  @blur="commitEdit"
                />
              </foreignObject>
            </g>
          </svg>
          <!-- 空状态提示 -->
          <div v-if="!flatNodes.length" class="empty-hint">
            点击下方按钮或使用 AI 生成思维导图
            <br />
            <el-button type="primary" @click="initDefault" style="margin-top: 12px">创建示例导图</el-button>
          </div>
        </div>
      </div>

      <!-- 代码编辑器 -->
      <div v-show="activeTab === 'code'" class="code-area">
        <div class="code-toolbar">
          <span class="code-hint">Mermaid mindmap 语法</span>
          <el-button size="small" @click="renderFromCode">渲染</el-button>
          <el-button size="small" @click="syncCodeFromVisual">从可视化同步</el-button>
        </div>
        <textarea
          v-model="mermaidCode"
          class="code-editor"
          spellcheck="false"
          placeholder="mindmap&#10;  root((中心主题))&#10;    分支1&#10;      子节点1&#10;      子节点2&#10;    分支2"
        ></textarea>
        <div class="code-preview">
          <div v-if="codeError" class="code-error">{{ codeError }}</div>
          <div v-else ref="codePreviewRef" class="code-preview-svg" v-html="codeSvg"></div>
        </div>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="statusbar">
      <span>节点数: {{ flatNodes.length }}</span>
      <span v-if="cloudId">云端 ID: {{ cloudId }}</span>
      <span v-if="lastSaved">上次保存: {{ lastSaved }}</span>
      <span v-if="shareUrl" class="share-url">
        分享链接: <a :href="shareUrl" target="_blank">{{ shareUrl }}</a>
        <el-button size="small" text @click="copyShareUrl">复制</el-button>
      </span>
    </div>

    <!-- AI 生成对话框 -->
    <el-dialog v-model="showAIDialog" title="✨ AI 生成思维导图" width="520px">
      <el-form label-width="70px" size="small">
        <el-form-item label="主题">
          <el-input v-model="aiPrompt" placeholder="例如：项目管理、学习计划、产品设计..." />
        </el-form-item>
        <el-form-item label="层级">
          <el-slider v-model="aiDepth" :min="2" :max="5" :step="1" show-stops />
        </el-form-item>
        <el-form-item label="风格">
          <el-radio-group v-model="aiStyle">
            <el-radio value="detailed">详细</el-radio>
            <el-radio value="simple">简洁</el-radio>
            <el-radio value="creative">创意</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div v-if="aiResult" class="ai-result-preview">
        <div class="ai-result-label">生成结果预览：</div>
        <pre class="ai-result-code">{{ aiResult }}</pre>
      </div>
      <template #footer>
        <el-button @click="showAIDialog = false">取消</el-button>
        <el-button type="primary" @click="generateAI" :loading="aiLoading">生成</el-button>
        <el-button v-if="aiResult" type="success" @click="applyAIResult">应用</el-button>
      </template>
    </el-dialog>

    <!-- 云端保存对话框 -->
    <el-dialog v-model="showSaveDialog" title="云端保存" width="420px">
      <el-form label-width="80px" size="small">
        <el-form-item label="过期时间">
          <el-select v-model="saveExpireDays" style="width: 100%">
            <el-option label="7 天" :value="7" />
            <el-option label="30 天" :value="30" />
            <el-option label="90 天" :value="90" />
            <el-option label="365 天" :value="365" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSaveDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmSave" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- 分享对话框 -->
    <el-dialog v-model="showShareDialog" title="分享思维导图" width="480px">
      <div class="share-info">
        <el-form label-width="80px" size="small">
          <el-form-item label="分享链接">
            <el-input :model-value="shareUrl" readonly>
              <template #append>
                <el-button @click="copyShareUrl">复制</el-button>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="短链接" v-if="shortUrl">
            <el-input :model-value="shortUrl" readonly>
              <template #append>
                <el-button @click="copyText(shortUrl)">复制</el-button>
              </template>
            </el-input>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="showShareDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 我的导图管理 -->
    <el-dialog v-model="showManageDialog" title="我的导图" width="700px">
      <div class="manage-toolbar">
        <el-button size="small" type="primary" @click="loadMyMaps">刷新</el-button>
        <el-button size="small" @click="showAdminLogin = true" v-if="!adminAuthed">超管模式</el-button>
        <el-tag v-if="adminAuthed" type="success" size="small">超管模式</el-tag>
        <el-button size="small" type="danger" @click="adminAuthed = false; adminPwd = ''" v-if="adminAuthed">退出超管</el-button>
      </div>

      <!-- 超管登录 -->
      <el-dialog v-model="showAdminLogin" title="超管登录" width="320px" append-to-body>
        <el-input v-model="adminPwd" type="password" placeholder="管理员密码" show-password />
        <template #footer>
          <el-button @click="showAdminLogin = false">取消</el-button>
          <el-button type="primary" @click="loginAdmin">确认</el-button>
        </template>
      </el-dialog>

      <el-table :data="manageMaps" style="width:100%; margin-top:12px" size="small" v-loading="manageLoading">
        <el-table-column prop="title" label="标题" min-width="160">
          <template #default="{ row }">
            <span>{{ row.title || '无标题' }}</span>
            <el-tag v-if="row.source === 'local'" size="small" style="margin-left:6px">本地</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="views" label="访问" width="60" />
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button size="small" @click="loadMap(row)">打开</el-button>
            <el-button size="small" type="danger" @click="deleteMap(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="showManageDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 隐藏的文件输入 -->
    <input ref="fileInput" type="file" accept=".json" style="display:none" @change="importJSON" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import mermaid from 'mermaid'
import { API_BASE } from '../api'

// ===== 状态 =====
const title = ref('新建思维导图')
const activeTab = ref('visual')
const svgRef = ref(null)
const canvasWrap = ref(null)
const inlineInput = ref(null)
const fileInput = ref(null)
const codePreviewRef = ref(null)

// 画布变换
const zoomLevel = ref(1)
const panX = ref(0)
const panY = ref(0)
const isDragging = ref(false)
let dragStart = null
let panStart = null

// 画布尺寸
const svgWidth = ref(1200)
const svgHeight = ref(800)

// 节点树
const rootNode = ref(null)
const selectedNode = ref(null)
const editingNode = ref(null)
const editingText = ref('')

// 云端/分享
// 管理面板
const showManageDialog = ref(false)
const showAdminLogin = ref(false)
const adminAuthed = ref(false)
const adminPwd = ref('')
const manageMaps = ref([])
const manageLoading = ref(false)

const cloudId = ref('')
const creatorKey = ref('')
const shareUrl = ref('')
const shortUrl = ref('')
const lastSaved = ref('')
const saving = ref(false)
const showSaveDialog = ref(false)
const showShareDialog = ref(false)
const saveExpireDays = ref(30)

// AI
const showAIDialog = ref(false)
const aiPrompt = ref('')
const aiDepth = ref(3)
const aiStyle = ref('detailed')
const aiLoading = ref(false)
const aiResult = ref('')

// 代码模式
const mermaidCode = ref('')
const codeSvg = ref('')
const codeError = ref('')

// ===== 颜色方案 =====
const COLORS = ['#5b8dee', '#f5a623', '#7ed321', '#d0021b', '#9b59b6', '#1abc9c', '#e67e22', '#3498db']
const defaultColor = (depth) => {
  if (depth === 0) return '#5b8dee'
  return COLORS[depth % COLORS.length]
}

// ===== 节点 ID =====
let nodeIdCounter = 1
const genId = () => `n${nodeIdCounter++}`

// ===== 布局计算 =====
const NODE_H = 36
const NODE_PADDING_X = 20
const LEVEL_GAP = 120
const SIBLING_GAP = 16

function measureText(text, depth) {
  const charW = depth === 0 ? 10 : 8.5
  return Math.max(80, text.length * charW + NODE_PADDING_X * 2)
}

function layoutTree(node, depth = 0) {
  node.depth = depth
  node.w = measureText(node.text, depth)
  node.h = depth === 0 ? 44 : NODE_H

  if (!node.children || node.children.length === 0 || node.collapsed) {
    node.subtreeH = node.h
    return
  }

  let totalH = 0
  for (const child of node.children) {
    layoutTree(child, depth + 1)
    totalH += child.subtreeH + SIBLING_GAP
  }
  totalH -= SIBLING_GAP
  node.subtreeH = Math.max(node.h, totalH)
}

function positionTree(node, x, y) {
  node.x = x
  node.y = y

  if (!node.children || node.children.length === 0 || node.collapsed) return

  const childX = x + node.w / 2 + LEVEL_GAP
  let startY = y - node.subtreeH / 2

  for (const child of node.children) {
    const childY = startY + child.subtreeH / 2
    positionTree(child, childX + child.w / 2, childY)
    startY += child.subtreeH + SIBLING_GAP
  }
}

function collectNodes(node, result = []) {
  result.push(node)
  if (node.children && !node.collapsed) {
    for (const c of node.children) collectNodes(c, result)
  }
  return result
}

function collectLinks(node, result = []) {
  if (!node.children || node.collapsed) return result
  for (const child of node.children) {
    const x1 = node.x + node.w / 2
    const y1 = node.y
    const x2 = child.x - child.w / 2
    const y2 = child.y
    const mx = (x1 + x2) / 2
    result.push({
      id: `${node.id}-${child.id}`,
      path: `M ${x1} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${x2} ${y2}`,
      color: child.color || defaultColor(child.depth)
    })
    collectLinks(child, result)
  }
  return result
}

const flatNodes = computed(() => {
  if (!rootNode.value) return []
  layoutTree(rootNode.value)
  const cx = svgWidth.value / 2 / zoomLevel.value
  const cy = svgHeight.value / 2 / zoomLevel.value
  positionTree(rootNode.value, cx, cy)
  return collectNodes(rootNode.value)
})

const links = computed(() => {
  if (!rootNode.value) return []
  return collectLinks(rootNode.value)
})

// ===== 节点操作 =====
function selectNode(node) {
  selectedNode.value = node
}

function startEdit(node) {
  editingNode.value = node
  editingText.value = node.text
  nextTick(() => inlineInput.value?.focus())
}

function commitEdit() {
  if (editingNode.value && editingText.value.trim()) {
    editingNode.value.text = editingText.value.trim()
    editingNode.value.w = measureText(editingNode.value.text, editingNode.value.depth)
  }
  editingNode.value = null
}

function cancelEdit() {
  editingNode.value = null
}

function updateNodeText() {
  if (selectedNode.value) {
    selectedNode.value.w = measureText(selectedNode.value.text, selectedNode.value.depth)
  }
}

function updateNodeColor() {}
function updateNodeShape() {}

function addChild() {
  if (!selectedNode.value) return
  if (!selectedNode.value.children) selectedNode.value.children = []
  const child = { id: genId(), text: '新节点', children: [], parent: selectedNode.value }
  selectedNode.value.children.push(child)
  selectedNode.value = child
}

function addSibling() {
  if (!selectedNode.value || !selectedNode.value.parent) return
  const parent = selectedNode.value.parent
  const idx = parent.children.indexOf(selectedNode.value)
  const sibling = { id: genId(), text: '新节点', children: [], parent }
  parent.children.splice(idx + 1, 0, sibling)
  selectedNode.value = sibling
}

function deleteNode() {
  if (!selectedNode.value || !selectedNode.value.parent) return
  const parent = selectedNode.value.parent
  const idx = parent.children.indexOf(selectedNode.value)
  parent.children.splice(idx, 1)
  selectedNode.value = parent
}

function toggleCollapse(node) {
  node.collapsed = !node.collapsed
}

function hexPoints(w, h) {
  const hw = w / 2, hh = h / 2, q = hw * 0.3
  return `${-hw},0 ${-q},${-hh} ${q},${-hh} ${hw},0 ${q},${hh} ${-q},${hh}`
}

// ===== 画布交互 =====
function zoom(delta) {
  zoomLevel.value = Math.max(0.2, Math.min(3, zoomLevel.value + delta))
}

function fitView() {
  zoomLevel.value = 1
  panX.value = 0
  panY.value = 0
}

function onWheel(e) {
  const delta = e.deltaY > 0 ? -0.08 : 0.08
  zoom(delta)
}

function onCanvasMouseDown(e) {
  if (e.target === svgRef.value || e.target.classList.contains('canvas-wrap')) {
    isDragging.value = true
    dragStart = { x: e.clientX, y: e.clientY }
    panStart = { x: panX.value, y: panY.value }
    selectedNode.value = null
  }
}

function onCanvasMouseMove(e) {
  if (!isDragging.value || !dragStart) return
  panX.value = panStart.x + (e.clientX - dragStart.x)
  panY.value = panStart.y + (e.clientY - dragStart.y)
}

function onCanvasMouseUp() {
  isDragging.value = false
  dragStart = null
}

// ===== 初始化 =====
function initDefault() {
  nodeIdCounter = 1
  const root = {
    id: genId(), text: '中心主题', depth: 0, children: [], collapsed: false,
    color: '#5b8dee', shape: 'rounded'
  }
  const topics = ['分支一', '分支二', '分支三']
  topics.forEach((t, i) => {
    const branch = { id: genId(), text: t, children: [], parent: root, color: COLORS[i + 1], shape: 'rounded' }
    branch.children = [
      { id: genId(), text: '子节点 A', children: [], parent: branch, shape: 'rounded' },
      { id: genId(), text: '子节点 B', children: [], parent: branch, shape: 'rounded' }
    ]
    root.children.push(branch)
  })
  rootNode.value = root
  title.value = '示例思维导图'
}

function newMap() {
  rootNode.value = null
  selectedNode.value = null
  cloudId.value = ''
  creatorKey.value = ''
  shareUrl.value = ''
  shortUrl.value = ''
  lastSaved.value = ''
  title.value = '新建思维导图'
  mermaidCode.value = ''
  codeSvg.value = ''
}

// ===== 序列化 =====
function serializeNode(node) {
  const { id, text, color, shape, collapsed, children } = node
  return { id, text, color, shape, collapsed, children: (children || []).map(serializeNode) }
}

function deserializeNode(data, parent = null) {
  const node = { ...data, parent, children: [] }
  if (data.children) {
    node.children = data.children.map(c => deserializeNode(c, node))
  }
  return node
}

function toJSON() {
  return JSON.stringify({ title: title.value, root: serializeNode(rootNode.value) }, null, 2)
}

function fromJSON(json) {
  const data = JSON.parse(json)
  title.value = data.title || '思维导图'
  rootNode.value = deserializeNode(data.root)
}

// ===== Mermaid 代码互转 =====
function nodeToMermaid(node, indent = '  ') {
  const shape = node.shape || 'rounded'
  let label = node.text
  let line = ''
  if (node.depth === 0) {
    line = `${indent}root((${label}))`
  } else if (shape === 'circle') {
    line = `${indent}((${label}))`
  } else if (shape === 'hexagon') {
    line = `${indent}{{${label}}}`
  } else if (shape === 'cloud') {
    line = `${indent})${label}(`
  } else {
    line = `${indent}${label}`
  }
  let result = line + '\n'
  if (node.children && !node.collapsed) {
    for (const c of node.children) {
      result += nodeToMermaid(c, indent + '  ')
    }
  }
  return result
}

function syncCodeFromVisual() {
  if (!rootNode.value) return
  mermaidCode.value = 'mindmap\n' + nodeToMermaid(rootNode.value)
  renderFromCode()
}

async function renderFromCode() {
  if (!mermaidCode.value.trim()) return
  try {
    mermaid.initialize({ startOnLoad: false, theme: 'default', securityLevel: 'loose' })
    const id = `mm-${Date.now()}`
    const { svg } = await mermaid.render(id, mermaidCode.value)
    codeSvg.value = svg
    codeError.value = ''
  } catch (e) {
    codeError.value = e.message
    codeSvg.value = ''
  }
}

// ===== 导入/导出 =====
function triggerImport() {
  fileInput.value?.click()
}

function importJSON(e) {
  const file = e.target.files[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      fromJSON(ev.target.result)
      ElMessage.success('导入成功')
    } catch {
      ElMessage.error('JSON 格式错误')
    }
  }
  reader.readAsText(file)
  e.target.value = ''
}

function handleExport(type) {
  if (!rootNode.value) return ElMessage.warning('请先创建思维导图')
  if (type === 'json') {
    const blob = new Blob([toJSON()], { type: 'application/json' })
    downloadBlob(blob, `${title.value}.json`)
  } else if (type === 'md') {
    const md = nodeToMarkdown(rootNode.value)
    const blob = new Blob([md], { type: 'text/markdown' })
    downloadBlob(blob, `${title.value}.md`)
  } else if (type === 'svg') {
    const svgEl = svgRef.value
    if (!svgEl) return
    const blob = new Blob([svgEl.outerHTML], { type: 'image/svg+xml' })
    downloadBlob(blob, `${title.value}.svg`)
  } else if (type === 'png') {
    exportPNG()
  }
}

function nodeToMarkdown(node, depth = 0) {
  const prefix = depth === 0 ? '# ' : '#'.repeat(Math.min(depth + 1, 6)) + ' '
  let md = prefix + node.text + '\n'
  if (node.children) {
    for (const c of node.children) md += nodeToMarkdown(c, depth + 1)
  }
  return md
}

function downloadBlob(blob, name) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
}

function exportPNG() {
  const svgEl = svgRef.value
  if (!svgEl) return
  const svgData = new XMLSerializer().serializeToString(svgEl)
  const canvas = document.createElement('canvas')
  canvas.width = svgEl.clientWidth * 2
  canvas.height = svgEl.clientHeight * 2
  const ctx = canvas.getContext('2d')
  const img = new Image()
  img.onload = () => {
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
    canvas.toBlob(blob => downloadBlob(blob, `${title.value}.png`))
  }
  img.src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svgData)))
}

// ===== 云端保存 =====
async function saveToCloud() {
  if (!rootNode.value) return ElMessage.warning('请先创建思维导图')
  showSaveDialog.value = true
}

async function confirmSave() {
  saving.value = true
  try {
    const content = JSON.stringify({ title: title.value, root: serializeNode(rootNode.value), mermaidCode: mermaidCode.value })
    const body = {
      content,
      title: title.value,
      max_views: 10,
      expires_in: saveExpireDays.value
    }

    let res, data
    if (cloudId.value && creatorKey.value) {
      // 更新
      res = await fetch(`${API_BASE}/api/mdshare/${cloudId.value}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'edit', content, title: title.value, creator_key: creatorKey.value })
      })
      data = await res.json()
      if (!res.ok) throw new Error(data.error || '更新失败')
      ElMessage.success('已更新云端存档')
    } else {
      // 新建
      res = await fetch(`${API_BASE}/api/mdshare`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })
      data = await res.json()
      if (!res.ok) throw new Error(data.error || '保存失败')
      cloudId.value = data.id
      creatorKey.value = data.creator_key
      const base = `${location.origin}/mindmap?id=${data.id}&key=${data.access_key}`
      shareUrl.value = base
      if (data.short_code) {
        shortUrl.value = `${location.origin}/s/${data.short_code}`
      }
      // 持久化到 localStorage
      saveLocalRecord()
      ElMessage.success('已保存到云端')
    }
    lastSaved.value = new Date().toLocaleTimeString()
    showSaveDialog.value = false
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

function saveLocalRecord() {
  const records = JSON.parse(localStorage.getItem('mindmap_records') || '[]')
  const existing = records.findIndex(r => r.id === cloudId.value)
  const record = { id: cloudId.value, creatorKey: creatorKey.value, title: title.value, savedAt: Date.now() }
  if (existing >= 0) records[existing] = record
  else records.unshift(record)
  localStorage.setItem('mindmap_records', JSON.stringify(records.slice(0, 20)))
}

// ===== 分享 =====
function shareMap() {
  if (!cloudId.value) return
  showShareDialog.value = true
}

function copyShareUrl() {
  navigator.clipboard.writeText(shareUrl.value)
  ElMessage.success('已复制')
}

function copyText(text) {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制')
}

// ===== AI 生成 =====
async function generateAI() {
  if (!aiPrompt.value.trim()) return ElMessage.warning('请输入主题')
  aiLoading.value = true
  aiResult.value = ''
  try {
    const styleDesc = { detailed: '详细丰富', simple: '简洁清晰', creative: '创意发散' }[aiStyle.value]
    const systemPrompt = `你是一个思维导图专家。用户给你一个主题，你需要生成 Mermaid mindmap 格式的思维导图代码。
要求：
1. 只输出 Mermaid mindmap 代码，不要任何解释
2. 层级深度约 ${aiDepth.value} 层
3. 风格：${styleDesc}
4. 每个节点文字简洁（不超过15字）
5. 格式示例：
mindmap
  root((主题))
    分支1
      子节点
    分支2`

    const res = await fetch(`${API_BASE}/api/internal/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'MiniMax-M2.5',
        messages: [
          { role: 'system', content: systemPrompt },
          { role: 'user', content: `请为"${aiPrompt.value}"生成思维导图` }
        ],
        max_tokens: 2048
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'AI 请求失败')
    const text = data.choices?.[0]?.message?.content || data.content?.[0]?.text || ''
    // 提取 mindmap 代码块
    const match = text.match(/```(?:mermaid)?\s*(mindmap[\s\S]*?)```/i) || text.match(/(mindmap[\s\S]+)/)
    aiResult.value = match ? match[1].trim() : text.trim()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    aiLoading.value = false
  }
}

function applyAIResult() {
  if (!aiResult.value) return
  mermaidCode.value = aiResult.value
  // 解析 mermaid mindmap 为节点树
  try {
    parseMermaidToTree(aiResult.value)
    activeTab.value = 'visual'
    ElMessage.success('已应用 AI 生成结果')
  } catch {
    // 如果解析失败，切换到代码模式
    activeTab.value = 'code'
    renderFromCode()
    ElMessage.info('已切换到代码模式查看')
  }
  showAIDialog.value = false
}

function parseMermaidToTree(code) {
  const lines = code.split('\n').filter(l => l.trim() && !l.trim().startsWith('mindmap'))
  nodeIdCounter = 1

  function getDepth(line) {
    let i = 0
    while (i < line.length && (line[i] === ' ' || line[i] === '\t')) i++
    return Math.floor(i / 2)
  }

  function parseText(raw) {
    return raw.trim()
      .replace(/^root\(\((.+)\)\)$/, '$1')
      .replace(/^\(\((.+)\)\)$/, '$1')
      .replace(/^\[(.+)\]$/, '$1')
      .replace(/^\((.+)\)$/, '$1')
      .replace(/^\{\{(.+)\}\}$/, '$1')
      .replace(/^\)(.+)\($/, '$1')
      .trim()
  }

  const stack = []
  let root = null

  for (const line of lines) {
    if (!line.trim()) continue
    const depth = getDepth(line)
    const text = parseText(line)
    if (!text) continue

    const node = { id: genId(), text, children: [], depth, shape: 'rounded' }

    if (depth === 0 || stack.length === 0) {
      root = node
      stack.length = 0
      stack.push(node)
    } else {
      while (stack.length > 1 && stack[stack.length - 1].depth >= depth) {
        stack.pop()
      }
      const parent = stack[stack.length - 1]
      node.parent = parent
      parent.children.push(node)
      stack.push(node)
    }
  }

  if (root) {
    rootNode.value = root
    title.value = root.text || '思维导图'
  }
}

// ===== 从 URL 加载 =====
async function loadFromURL() {
  const params = new URLSearchParams(location.search)
  const id = params.get('id')
  const key = params.get('key')
  if (!id || !key) return

  try {
    const res = await fetch(`${API_BASE}/api/mdshare/${id}?key=${key}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)
    const parsed = JSON.parse(data.content)
    title.value = parsed.title || '思维导图'
    rootNode.value = deserializeNode(parsed.root)
    if (parsed.mermaidCode) mermaidCode.value = parsed.mermaidCode
    cloudId.value = id
    shareUrl.value = `${location.origin}/mindmap?id=${id}&key=${key}`
    ElMessage.success('已加载云端思维导图')
  } catch (e) {
    ElMessage.error('加载失败: ' + e.message)
  }
}

// ===== 键盘快捷键 =====
function onKeyDown(e) {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return
  if (e.key === 'Tab' && selectedNode.value) {
    e.preventDefault()
    addChild()
  } else if (e.key === 'Enter' && selectedNode.value) {
    e.preventDefault()
    addSibling()
  } else if ((e.key === 'Delete' || e.key === 'Backspace') && selectedNode.value?.parent) {
    e.preventDefault()
    deleteNode()
  } else if (e.key === 'F2' && selectedNode.value) {
    startEdit(selectedNode.value)
  }
}

onMounted(() => {
  mermaid.initialize({ startOnLoad: false, theme: 'default', securityLevel: 'loose' })
  loadFromURL()
  document.addEventListener('keydown', onKeyDown)
  // 响应式画布尺寸
  const ro = new ResizeObserver(() => {
    if (canvasWrap.value) {
      svgWidth.value = canvasWrap.value.clientWidth
      svgHeight.value = canvasWrap.value.clientHeight
    }
  })
  if (canvasWrap.value) ro.observe(canvasWrap.value)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeyDown)
})

// ===== 管理面板 =====
const ADMIN_PWD_KEY = 'mindmap_admin_pwd'

function openManage() {
  showManageDialog.value = true
  loadMyMaps()
}

function loginAdmin() {
  if (!adminPwd.value.trim()) return
  adminAuthed.value = true
  showAdminLogin.value = false
  loadMyMaps()
}

async function loadMyMaps() {
  manageLoading.value = true
  try {
    if (adminAuthed.value) {
      // 超管：拉全部
      const res = await fetch(`${API_BASE}/api/mdshare/admin/list?admin_password=${encodeURIComponent(adminPwd.value)}`)
      const data = await res.json()
      if (!res.ok) {
        ElMessage.error(data.error || '获取失败')
        adminAuthed.value = false
        return
      }
      manageMaps.value = (data.list || []).map(m => ({ ...m, source: 'admin' }))
    } else {
      // 普通用户：读 localStorage
      const records = JSON.parse(localStorage.getItem('mindmap_records') || '[]')
      manageMaps.value = records.map(r => ({ ...r, source: 'local' }))
    }
  } finally {
    manageLoading.value = false
  }
}

async function loadMap(row) {
  // 用 creator_key 获取内容
  const key = row.creator_key || row.creatorKey
  if (!key && !adminAuthed.value) {
    ElMessage.warning('无法获取该导图内容')
    return
  }
  try {
    let res, data
    if (adminAuthed.value) {
      res = await fetch(`${API_BASE}/api/mdshare/admin/${row.id}?admin_password=${encodeURIComponent(adminPwd.value)}`)
    } else {
      res = await fetch(`${API_BASE}/api/mdshare/${row.id}/creator?creator_key=${encodeURIComponent(key)}`)
    }
    data = await res.json()
    if (!res.ok) { ElMessage.error(data.error || '加载失败'); return }

    // 解析内容
    try {
      const parsed = JSON.parse(data.content)
      if (parsed.root) {
        title.value = parsed.title || data.title || '无标题'
        rootNode.value = restoreParentRefs(parsed.root)
        cloudId.value = row.id
        creatorKey.value = key || ''
        shortUrl.value = row.short_code ? `${location.origin}/s/${row.short_code}` : ''
        showManageDialog.value = false
        ElMessage.success('已加载')
        return
      }
    } catch (_) {}
    // fallback: mermaid code
    mermaidCode.value = data.content
    activeTab.value = 'code'
    showManageDialog.value = false
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function deleteMap(row) {
  const key = row.creator_key || row.creatorKey
  try {
    let res
    if (adminAuthed.value) {
      res = await fetch(`${API_BASE}/api/mdshare/admin/${row.id}?admin_password=${encodeURIComponent(adminPwd.value)}`, { method: 'DELETE' })
    } else if (key) {
      res = await fetch(`${API_BASE}/api/mdshare/${row.id}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ creator_key: key })
      })
    } else {
      ElMessage.warning('无删除权限')
      return
    }
    if (!res.ok) { const d = await res.json(); ElMessage.error(d.error || '删除失败'); return }
    // 从 localStorage 移除
    const records = JSON.parse(localStorage.getItem('mindmap_records') || '[]')
    localStorage.setItem('mindmap_records', JSON.stringify(records.filter(r => r.id !== row.id)))
    ElMessage.success('已删除')
    loadMyMaps()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

function formatTime(t) {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function restoreParentRefs(node, parent = null) {
  node.parent = parent
  if (node.children) node.children.forEach(c => restoreParentRefs(c, node))
  return node
}
</script>

<style scoped>
.mindmap-tool {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary);
  overflow: hidden;
}

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--card-bg);
  border-bottom: 1px solid var(--border-base);
  flex-shrink: 0;
  flex-wrap: wrap;
}
.toolbar-left { display: flex; align-items: center; gap: 8px; flex: 1; min-width: 200px; }
.toolbar-center { display: flex; align-items: center; }
.toolbar-right { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; }
.tool-title { font-weight: 600; font-size: 15px; white-space: nowrap; }
.title-input { width: 180px; }
.zoom-label { font-size: 12px; color: var(--text-secondary); min-width: 38px; text-align: center; }

/* 主体 */
.main-area {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* 可视化区域 */
.visual-area {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.side-panel {
  width: 200px;
  flex-shrink: 0;
  background: var(--card-bg);
  border-right: 1px solid var(--border-base);
  padding: 12px;
  overflow-y: auto;
}
.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 10px;
}
.panel-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 10px;
}

.canvas-wrap {
  flex: 1;
  overflow: hidden;
  position: relative;
  background: var(--bg-secondary);
  cursor: default;
}
.canvas-wrap.dragging { cursor: grabbing; }

.mindmap-svg {
  display: block;
  width: 100%;
  height: 100%;
}

.node-group { transition: opacity 0.15s; }
.node-group:hover .node-rect { filter: brightness(1.1); }
.node-group.selected .node-rect { filter: brightness(1.05); }

.inline-edit {
  width: 100%;
  height: 100%;
  border: 2px solid #409eff;
  border-radius: 6px;
  background: #fff;
  color: #333;
  font-size: 13px;
  text-align: center;
  outline: none;
  padding: 0 6px;
}

.empty-hint {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  color: var(--text-tertiary);
  font-size: 14px;
  pointer-events: none;
}
.empty-hint .el-button { pointer-events: all; }

/* 代码区域 */
.code-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.code-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--card-bg);
  border-bottom: 1px solid var(--border-base);
  flex-shrink: 0;
}
.code-hint { font-size: 12px; color: var(--text-tertiary); flex: 1; }
.code-editor {
  flex: 1;
  resize: none;
  border: none;
  outline: none;
  padding: 12px;
  font-family: monospace;
  font-size: 13px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  min-height: 200px;
  max-height: 40%;
}
.code-preview {
  flex: 1;
  overflow: auto;
  padding: 12px;
  background: var(--card-bg);
  border-top: 1px solid var(--border-base);
}
.code-preview-svg :deep(svg) { max-width: 100%; }
.code-error { color: #f56c6c; font-size: 13px; }

/* 状态栏 */
.statusbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 4px 12px;
  background: var(--card-bg);
  border-top: 1px solid var(--border-base);
  font-size: 12px;
  color: var(--text-tertiary);
  flex-shrink: 0;
}
.share-url a { color: var(--color-primary); }

/* AI 结果预览 */
.ai-result-preview {
  margin-top: 12px;
  border: 1px solid var(--border-base);
  border-radius: 6px;
  overflow: hidden;
}
.ai-result-label {
  padding: 6px 10px;
  background: var(--bg-tertiary);
  font-size: 12px;
  color: var(--text-secondary);
}
.ai-result-code {
  margin: 0;
  padding: 10px;
  font-size: 12px;
  font-family: monospace;
  max-height: 200px;
  overflow-y: auto;
  background: var(--bg-secondary);
  color: var(--text-primary);
}

/* 分享信息 */
.share-info { padding: 4px 0; }

/* 管理面板 */
.manage-toolbar { display: flex; align-items: center; gap: 8px; }

@media (max-width: 768px) {
  .toolbar { padding: 6px 8px; }
  .title-input { width: 120px; }
  .side-panel { display: none; }
}
</style>
