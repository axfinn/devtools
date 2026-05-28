<template>
  <div class="json-tree-viewer">
    <!-- Toolbar -->
    <div class="json-tree-toolbar">
      <div class="tree-stats">
        <span class="stat-label">{{ totalNodes }} 个节点</span>
        <span v-if="truncatedCount > 0" class="stat-truncated">已折叠 {{ truncatedCount }} 项</span>
      </div>
      <div class="tree-actions">
        <el-button size="small" @click="expandAll">展开全部</el-button>
        <el-button size="small" @click="collapseAll">折叠全部</el-button>
        <el-button-group size="small">
          <el-button :type="expandDepth === 1 ? 'primary' : ''" @click="expandToDepth(1)">1级</el-button>
          <el-button :type="expandDepth === 2 ? 'primary' : ''" @click="expandToDepth(2)">2级</el-button>
          <el-button :type="expandDepth === 3 ? 'primary' : ''" @click="expandToDepth(3)">3级</el-button>
        </el-button-group>
      </div>
    </div>

    <!-- Tree content -->
    <div v-if="parsedRoot" class="json-tree-content" ref="treeScrollRef">
      <JsonNodeRow
        :node="parsedRoot"
        :search-matched-paths="matchedPathSet"
        :has-truncation="truncatedCount > 0"
        :truncation-node-paths="truncationPathsSet"
        :truncation-map="truncationMap"
        @toggle="handleToggle"
        @save-edit="handleSaveEdit"
        @load-more="handleLoadMore"
      />
    </div>

    <!-- Parse error -->
    <div v-else class="json-parse-error">
      <el-alert
        title="JSON 解析失败"
        type="error"
        :description="parseError"
        show-icon
        :closable="false"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import JsonNodeRow from './JsonNodeRow.vue'

const props = defineProps({
  jsonString: { type: String, default: '' },
  searchText: { type: String, default: '' },
  highlightPaths: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:jsonString'])

const DEFAULT_EXPAND_DEPTH = 3
const MAX_CHILDREN_PER_PAGE = 50

let nextId = 1

const parsedRoot = ref(null)
const parseError = ref('')
const expandDepth = ref(DEFAULT_EXPAND_DEPTH)
const truncationMap = ref({})
const treeScrollRef = ref(null)

// ---- Path helpers ----

function jsonPath(pathArr) {
  return pathArr.length === 0 ? '$' : '$.' + pathArr.join('.')
}

function pathsEqual(a, b) {
  if (a.length !== b.length) return false
  return a.every((seg, i) => String(seg) === String(b[i]))
}

// ---- Parse JSON to tree ----

function parseJsonToTree(jsonStr, maxDepth = DEFAULT_EXPAND_DEPTH) {
  nextId = 1
  truncationMap.value = {}
  parseError.value = ''

  let parsed
  try {
    parsed = JSON.parse(jsonStr)
  } catch (e) {
    parseError.value = e.message
    return null
  }

  return buildNode(null, parsed, 0, [], maxDepth)
}

function getJsonType(val) {
  if (val === null) return 'null'
  if (Array.isArray(val)) return 'array'
  return typeof val
}

function buildNode(key, value, depth, path, maxDepth) {
  const type = getJsonType(value)
  const id = nextId++
  const collapsed = depth >= maxDepth

  let children = []
  let childrenCount = 0

  if (type === 'object') {
    const entries = Object.entries(value)
    childrenCount = entries.length
    const show = entries.slice(0, MAX_CHILDREN_PER_PAGE)
    if (entries.length > MAX_CHILDREN_PER_PAGE) {
      truncationMap.value[jsonPath(path)] = { total: entries.length, shown: MAX_CHILDREN_PER_PAGE }
    }
    children = show.map(([k, v]) => buildNode(k, v, depth + 1, [...path, String(k)], maxDepth))
  } else if (type === 'array') {
    childrenCount = value.length
    const show = value.slice(0, MAX_CHILDREN_PER_PAGE)
    if (value.length > MAX_CHILDREN_PER_PAGE) {
      truncationMap.value[jsonPath(path)] = { total: value.length, shown: MAX_CHILDREN_PER_PAGE }
    }
    children = show.map((item, i) => buildNode(i, item, depth + 1, [...path, String(i)], maxDepth))
  }

  return { id, key, value, type, depth, path, collapsed, children, childrenCount }
}

// ---- Reconstruct JSON from tree ----

function reconstructJson(node) {
  if (node.type === 'object') {
    const obj = {}
    for (const child of node.children) {
      obj[child.key] = reconstructJson(child)
    }
    return obj
  }
  if (node.type === 'array') {
    return node.children.map(c => reconstructJson(c))
  }
  return node.value
}

// ---- Tree operations ----

function findNode(root, targetPath) {
  if (pathsEqual(root.path, targetPath)) return root
  for (const child of root.children) {
    const found = findNode(child, targetPath)
    if (found) return found
  }
  return null
}

function walkTree(node, fn) {
  fn(node)
  for (const child of node.children) {
    walkTree(child, fn)
  }
}

function findMaxId(node) {
  let max = node.id
  for (const child of node.children) {
    max = Math.max(max, findMaxId(child))
  }
  return max
}

function refreshRoot() {
  parsedRoot.value = { ...parsedRoot.value }
}

// ---- Event handlers ----

function handleToggle(path) {
  const node = findNode(parsedRoot.value, path)
  if (!node) return
  node.collapsed = !node.collapsed
  refreshRoot()
}

function handleSaveEdit(nodeId, newValue) {
  // Find the node by ID
  let targetNode = null
  walkTree(parsedRoot.value, (n) => {
    if (n.id === nodeId) targetNode = n
  })
  if (!targetNode) return

  const origType = targetNode.type

  if (origType === 'number') {
    const num = Number(newValue)
    if (isNaN(num)) {
      ElMessage.warning('请输入有效数字')
      return
    }
    targetNode.value = num
  } else if (origType === 'boolean') {
    targetNode.value = newValue === true || newValue === 'true'
  } else if (origType === 'null') {
    if (newValue === 'string') targetNode.value = ''
    else if (newValue === 'number') targetNode.value = 0
    else targetNode.value = newValue
  } else {
    targetNode.value = newValue
  }

  targetNode.type = getJsonType(targetNode.value)

  if (targetNode.type !== 'object' && targetNode.type !== 'array') {
    targetNode.children = []
    targetNode.childrenCount = 0
  }

  // Reconstruct and emit
  try {
    const json = JSON.stringify(reconstructJson(parsedRoot.value), null, 2)
    emit('update:jsonString', json)
    refreshRoot()
    ElMessage.success('已更新')
  } catch (e) {
    ElMessage.error('更新失败: ' + e.message)
  }
}

function handleLoadMore(path) {
  if (!props.jsonString) return

  let parsed
  try { parsed = JSON.parse(props.jsonString) } catch { return }

  // Navigate to target in raw data
  let target = parsed
  for (const seg of path) {
    if (target === undefined || target === null) return
    if (Array.isArray(target)) {
      target = target[Number(seg)]
    } else if (typeof target === 'object') {
      target = target[seg]
    } else {
      return
    }
  }

  if (target === undefined || target === null) return

  const node = findNode(parsedRoot.value, path)
  if (!node) return

  // Rebuild all children with no page limit
  nextId = findMaxId(parsedRoot.value) + 1
  const collapsed = node.depth >= expandDepth.value

  if (Array.isArray(target)) {
    node.children = target.map((item, i) =>
      buildNode(i, item, node.depth + 1, [...path, String(i)], expandDepth.value)
    )
    node.childrenCount = target.length
    node.collapsed = collapsed
  } else {
    const entries = Object.entries(target)
    node.children = entries.map(([k, v]) =>
      buildNode(k, v, node.depth + 1, [...path, String(k)], expandDepth.value)
    )
    node.childrenCount = entries.length
    node.collapsed = collapsed
  }

  delete truncationMap.value[jsonPath(path)]
  refreshRoot()
}

// ---- Expand / Collapse ----

function expandAll() {
  if (!parsedRoot.value) return
  walkTree(parsedRoot.value, n => {
    if (n.type === 'object' || n.type === 'array') n.collapsed = false
  })
  expandDepth.value = 99
  refreshRoot()
}

function collapseAll() {
  if (!parsedRoot.value) return
  walkTree(parsedRoot.value, n => {
    if (n.depth > 0 && (n.type === 'object' || n.type === 'array')) n.collapsed = true
  })
  expandDepth.value = 0
  refreshRoot()
}

function expandToDepth(depth) {
  if (!parsedRoot.value) return
  expandDepth.value = depth
  walkTree(parsedRoot.value, n => {
    if (n.type === 'object' || n.type === 'array') {
      n.collapsed = n.depth >= depth
    }
  })
  refreshRoot()
}

// ---- Search ----

const matchedPathSet = computed(() => {
  const paths = props.highlightPaths || []
  return new Set(paths.map(p => Array.isArray(p) ? jsonPath(p) : p))
})

watch(() => props.highlightPaths, (paths) => {
  if (!parsedRoot.value || !paths || paths.length === 0) return
  for (const p of paths) {
    for (let i = 0; i < p.length; i++) {
      const ancestor = findNode(parsedRoot.value, p.slice(0, i))
      if (ancestor && ancestor.collapsed) {
        ancestor.collapsed = false
      }
    }
  }
  refreshRoot()
}, { deep: true })

// ---- Stats ----

const totalNodes = computed(() => {
  if (!parsedRoot.value) return 0
  let count = 0
  walkTree(parsedRoot.value, () => { count++ })
  return count
})

const truncatedCount = computed(() => {
  let count = 0
  for (const key of Object.keys(truncationMap.value)) {
    count += truncationMap.value[key].total - truncationMap.value[key].shown
  }
  return count
})

const truncationPathsSet = computed(() => {
  return new Set(Object.keys(truncationMap.value))
})

// ---- Watch prop ----

watch(() => props.jsonString, (newVal) => {
  if (newVal) {
    parsedRoot.value = parseJsonToTree(newVal, expandDepth.value)
  } else {
    parsedRoot.value = null
  }
}, { immediate: true })
</script>

<style scoped>
.json-tree-viewer {
  background: var(--bg-primary, #fff);
  border-radius: var(--radius-md, 8px);
  overflow: hidden;
}

.json-tree-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: var(--bg-secondary, #f5f7fa);
  border-bottom: 1px solid var(--border-base, #e4e7ed);
  flex-wrap: wrap;
  gap: 8px;
  position: sticky;
  top: 0;
  z-index: 5;
}

.tree-stats {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--text-secondary, #666);
}

.stat-truncated {
  color: var(--color-warning, #e6a23c);
}

.tree-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.json-tree-content {
  padding: 8px 0;
  font-family: var(--font-family-mono, 'Consolas', 'Monaco', monospace);
  font-size: 13px;
  line-height: 1.9;
  max-height: 70vh;
  overflow: auto;
}

.json-parse-error {
  padding: 16px;
}

@media (max-width: 768px) {
  .json-tree-content {
    font-size: 12px;
    max-height: 55vh;
  }

  .json-tree-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
