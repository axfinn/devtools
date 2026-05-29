<template>
  <div
    class="json-node-row"
    :class="{ 'search-highlight': isThisPathMatched }"
    :style="{ paddingLeft: (node.depth * 20) + 'px' }"
  >
    <div class="json-node-line">
      <!-- Expand/collapse toggle -->
      <span
        v-if="node.type === 'object' || node.type === 'array'"
        class="node-toggle"
        :class="{ 'is-expanded': !node.collapsed }"
        @click.stop="$emit('toggle', node.path)"
      >
        <el-icon :size="12"><ArrowRight /></el-icon>
      </span>
      <span v-else class="node-toggle node-toggle--spacer"></span>

      <!-- Key -->
      <span v-if="node.key !== null && node.key !== undefined" class="node-key">
        <span class="node-key-text">{{ node.key }}</span>
        <span class="node-colon">: </span>
      </span>

      <!-- Object/Array (collapsed) -->
      <template v-if="(node.type === 'object' || node.type === 'array') && node.collapsed">
        <span class="node-bracket">{{ node.type === 'object' ? '{' : '[' }}</span>
        <span class="node-summary">{{ getSummary(node) }}</span>
        <span class="node-bracket">{{ node.type === 'object' ? '}' : ']' }}</span>
      </template>

      <!-- Object/Array (expanded) -->
      <template v-else-if="(node.type === 'object' || node.type === 'array') && !node.collapsed">
        <span class="node-bracket">{{ node.type === 'object' ? '{' : '[' }}</span>
        <span class="node-type-tag">{{ node.type }} {{ node.childrenCount }}</span>
      </template>

      <!-- Primitive values -->
      <template v-else>
        <span
          class="node-value"
          :class="'node-value--' + node.type"
          @click.stop="startEdit"
        >
          <span v-if="node.type === 'string'" class="node-string">"{{ displayValue }}"</span>
          <span v-else-if="node.type === 'number'" class="node-number">{{ displayValue }}</span>
          <span v-else-if="node.type === 'boolean'" class="node-boolean">{{ displayValue }}</span>
          <span v-else-if="node.type === 'null'" class="node-null">null</span>
        </span>

        <!-- Edit popover -->
        <el-popover
          v-model:visible="editVisible"
          placement="right"
          trigger="manual"
          :width="320"
          :show-arrow="true"
          popper-class="json-edit-popover"
        >
          <div class="edit-form">
            <div class="edit-form-header">
              <span class="edit-path">{{ '$.' + node.path.join('.') }}</span>
              <el-tag size="small">{{ node.type }}</el-tag>
            </div>

            <!-- string -->
            <el-input
              v-if="node.type === 'string'"
              v-model="editValue"
              type="textarea"
              :rows="3"
              placeholder="输入字符串值"
            />

            <!-- number -->
            <el-input
              v-else-if="node.type === 'number'"
              v-model="editValue"
              type="text"
              placeholder="输入数字"
            />

            <!-- boolean -->
            <div v-else-if="node.type === 'boolean'" class="bool-edit">
              <el-button :type="editValue === 'true' ? 'primary' : 'default'" size="small" @click="editValue = 'true'">true</el-button>
              <el-button :type="editValue === 'false' ? 'primary' : 'default'" size="small" @click="editValue = 'false'">false</el-button>
            </div>

            <!-- null -->
            <div v-else-if="node.type === 'null'" class="null-edit">
              <span class="null-label">null</span>
              <el-button size="small" @click="convertNull('string')">转为字符串</el-button>
              <el-button size="small" @click="convertNull('number')">转为数字</el-button>
              <el-button size="small" @click="convertNull('boolean')">转为布尔</el-button>
            </div>

            <div class="edit-actions">
              <el-button size="small" @click="editVisible = false">取消</el-button>
              <el-button size="small" type="primary" @click="saveEdit">保存</el-button>
            </div>
          </div>
        </el-popover>
      </template>

      <!-- Node actions (hover) -->
      <span class="node-actions">
        <el-button text size="small" @click.stop="copyValue" title="复制值">
          <el-icon :size="12"><CopyDocument /></el-icon>
        </el-button>
        <el-button text size="small" @click.stop="copyPath" title="复制路径">
          <el-icon :size="12"><Link /></el-icon>
        </el-button>
        <template v-if="node.type === 'object' || node.type === 'array'">
          <el-button text size="small" @click.stop="startEditNode" title="编辑">
            <el-icon :size="12"><Edit /></el-icon>
          </el-button>
          <el-button text size="small" @click.stop="$emit('addChild', node.path)" title="添加">
            <el-icon :size="12"><Plus /></el-icon>
          </el-button>
          <el-button text size="small" @click.stop="$emit('deleteNode', node.path)" title="删除" v-if="node.depth > 0">
            <el-icon :size="12"><Delete /></el-icon>
          </el-button>
        </template>
        <template v-else>
          <el-button text size="small" @click.stop="$emit('deleteNode', node.path)" title="删除" v-if="node.depth > 0">
            <el-icon :size="12"><Delete /></el-icon>
          </el-button>
        </template>
      </span>

      <!-- Object/Array edit popover -->
      <el-popover
        v-if="node.type === 'object' || node.type === 'array'"
        v-model:visible="nodeEditVisible"
        placement="right"
        trigger="manual"
        :width="420"
        :show-arrow="true"
        popper-class="json-edit-popover"
      >
        <div class="edit-form">
          <div class="edit-form-header">
            <span class="edit-path">{{ '$.' + node.path.join('.') }}</span>
            <el-tag size="small">{{ node.type }}</el-tag>
          </div>
          <el-input
            v-model="nodeEditJson"
            type="textarea"
            :rows="10"
            placeholder="编辑 JSON"
            class="node-edit-textarea"
          />
          <div v-if="nodeEditError" class="edit-error">{{ nodeEditError }}</div>
          <div class="edit-actions">
            <el-button size="small" @click="nodeEditVisible = false">取消</el-button>
            <el-button size="small" type="primary" @click="saveNodeEdit">保存</el-button>
          </div>
        </div>
      </el-popover>
    </div>

    <!-- Children (recursive) -->
    <template v-if="!node.collapsed && node.children && node.children.length > 0">
      <JsonNodeRow
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :has-truncation="hasTruncation"
        :truncation-node-paths="truncationNodePaths"
        :search-matched-paths="searchMatchedPaths"
        @toggle="(p) => $emit('toggle', p)"
        @edit="(n) => $emit('edit', n)"
        @save-edit="(id, val) => $emit('saveEdit', id, val)"
        @cancel-edit="(id) => $emit('cancelEdit', id)"
        @load-more="(p) => $emit('loadMore', p)"
        @delete-node="(p) => $emit('deleteNode', p)"
        @add-child="(p) => $emit('addChild', p)"
        @edit-node="(p, json) => $emit('editNode', p, json)"
      />
    </template>

    <!-- Closing bracket -->
    <div
      v-if="!node.collapsed && (node.type === 'object' || node.type === 'array')"
      class="json-node-row"
      :style="{ paddingLeft: (node.depth * 20) + 'px' }"
    >
      <div class="json-node-line">
        <span class="node-toggle node-toggle--spacer"></span>
        <span class="node-bracket">{{ node.type === 'object' ? '}' : ']' }}</span>
      </div>
    </div>

    <!-- Show more button -->
    <div
      v-if="!node.collapsed && hasTruncation && truncationNodePaths && truncationNodePaths.has(nodePathStr)"
      class="json-node-row show-more-row"
      :style="{ paddingLeft: ((node.depth + 1) * 20) + 'px' }"
    >
      <el-button size="small" text type="primary" @click.stop="$emit('loadMore', node.path)">
        显示更多 (还有 {{ getRemainingCount }} 项)...
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ArrowRight, CopyDocument, Link, Edit, Plus, Delete } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  node: { type: Object, required: true },
  searchMatchedPaths: { type: Set, default: () => new Set() },
  hasTruncation: { type: Boolean, default: false },
  truncationNodePaths: { type: Set, default: () => new Set() },
  truncationMap: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['toggle', 'edit', 'saveEdit', 'cancelEdit', 'loadMore', 'deleteNode', 'addChild', 'editNode'])

const editVisible = ref(false)
const editValue = ref('')
const nodeEditVisible = ref(false)
const nodeEditJson = ref('')
const nodeEditError = ref('')

const nodePathStr = computed(() => {
  return props.node.path.length === 0 ? '$' : '$.' + props.node.path.join('.')
})

const isThisPathMatched = computed(() => {
  if (!props.searchMatchedPaths || props.searchMatchedPaths.size === 0) return false
  return props.searchMatchedPaths.has(nodePathStr.value)
})

const displayValue = computed(() => {
  const v = props.node.value
  if (props.node.type === 'string') return String(v)
  if (props.node.type === 'number') return String(v)
  if (props.node.type === 'boolean') return String(v)
  if (props.node.type === 'null') return 'null'
  return JSON.stringify(v)
})

const getRemainingCount = computed(() => {
  const map = props.truncationMap || {}
  const info = map[nodePathStr.value]
  if (!info) return 0
  return info.total - info.shown
})

function getSummary(node) {
  if (node.type === 'object') {
    const keys = node.children.slice(0, 3).map(c => String(c.key)).join(', ')
    const more = node.childrenCount > 3 ? `, ...` : ''
    return `${keys}${more}`
  }
  if (node.type === 'array') {
    if (node.childrenCount === 0) return ''
    const preview = node.children.slice(0, 2).map(c => summaryValue(c)).join(', ')
    const more = node.childrenCount > 2 ? `, ...` : ''
    return `${preview}${more}`
  }
  return ''
}

function summaryValue(node) {
  if (node.type === 'string') {
    const s = String(node.value)
    return s.length > 20 ? `"${s.slice(0, 20)}..."` : `"${s}"`
  }
  if (node.type === 'number' || node.type === 'boolean') return String(node.value)
  if (node.type === 'null') return 'null'
  return ''
}

function startEdit() {
  editValue.value = props.node.type === 'string'
    ? String(props.node.value)
    : String(props.node.value)
  editVisible.value = true
}

function saveEdit() {
  let val = editValue.value
  if (props.node.type === 'number') {
    const num = Number(val)
    if (isNaN(num)) return
    val = num
  }
  if (props.node.type === 'boolean') {
    val = val === 'true' || val === true
  }
  emit('saveEdit', props.node.id, val)
  editVisible.value = false
}

function convertNull(toType) {
  if (toType === 'string') editValue.value = ''
  else if (toType === 'number') editValue.value = '0'
  else if (toType === 'boolean') editValue.value = 'false'
}

function reconstructNodeValue(node) {
  if (node.type === 'object') {
    const obj = {}
    for (const child of node.children) {
      obj[child.key] = reconstructNodeValue(child)
    }
    return obj
  }
  if (node.type === 'array') {
    return node.children.map(c => reconstructNodeValue(c))
  }
  return node.value
}

function copyValue() {
  const val = reconstructNodeValue(props.node)
  const text = typeof val === 'string' ? val : JSON.stringify(val, null, 2)
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制值')
}

function copyPath() {
  const path = props.node.path.length === 0 ? '$' : '$.' + props.node.path.join('.')
  navigator.clipboard.writeText(path)
  ElMessage.success('已复制路径')
}

function startEditNode() {
  const val = reconstructNodeValue(props.node)
  nodeEditJson.value = JSON.stringify(val, null, 2)
  nodeEditError.value = ''
  nodeEditVisible.value = true
}

function saveNodeEdit() {
  try {
    JSON.parse(nodeEditJson.value)
  } catch (e) {
    nodeEditError.value = 'JSON 格式错误: ' + e.message
    return
  }
  emit('editNode', props.node.path, nodeEditJson.value)
  nodeEditVisible.value = false
}
</script>

<style scoped>
.json-node-row {
  white-space: nowrap;
  min-width: max-content;
}

.json-node-line {
  display: flex;
  align-items: baseline;
  padding: 1px 8px;
  border-radius: 2px;
  transition: background-color 0.1s;
}

.json-node-line:hover {
  background-color: var(--bg-hover, rgba(0,0,0,0.04));
}

.search-highlight > .json-node-line {
  background-color: rgba(255, 196, 0, 0.15);
  outline: 1px solid rgba(255, 196, 0, 0.4);
}

.node-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  margin-right: 4px;
  cursor: pointer;
  flex-shrink: 0;
  color: var(--text-tertiary, #999);
  transition: transform 0.15s ease;
  user-select: none;
}

.node-toggle.is-expanded {
  transform: rotate(90deg);
}

.node-toggle--spacer {
  visibility: hidden;
}

.node-key {
  color: #0451a5;
  margin-right: 0;
  user-select: none;
}

.node-colon {
  color: var(--text-secondary, #666);
}

.node-bracket {
  color: var(--text-secondary, #666);
  font-weight: 400;
}

.node-type-tag {
  display: inline-block;
  font-size: 10px;
  color: var(--text-tertiary, #999);
  font-style: italic;
  margin-left: 4px;
  user-select: none;
}

.node-summary {
  color: var(--text-tertiary, #999);
  font-style: italic;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 360px;
}

.node-value {
  cursor: pointer;
  transition: opacity 0.1s;
}

.node-value:hover {
  opacity: 0.75;
  text-decoration: underline;
  text-decoration-style: dotted;
}

.node-string { color: #a31515; }
.node-number { color: #098658; }
.node-boolean { color: #0000ff; }
.node-null { color: #808080; font-style: italic; }

.show-more-row {
  cursor: pointer;
}

.node-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-left: 8px;
  opacity: 0;
  transition: opacity 0.15s;
}

.json-node-line:hover > .node-actions {
  opacity: 1;
}

.node-edit-textarea :deep(.el-textarea__inner) {
  font-family: var(--font-family-mono, 'Consolas', monospace);
  font-size: 12px;
}

.edit-error {
  color: #f56c6c;
  font-size: 12px;
}

/* Edit popover */
.edit-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.edit-form-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-family-mono);
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-all;
}

.edit-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}

.bool-edit, .null-edit {
  display: flex;
  gap: 8px;
  align-items: center;
}

.null-label {
  color: #808080;
  font-style: italic;
  font-family: var(--font-family-mono);
}

/* Dark mode */
.dark .node-key { color: #9cdcfe; }
.dark .node-string { color: #ce9178; }
.dark .node-number { color: #b5cea8; }
.dark .node-boolean { color: #569cd6; }

@media (max-width: 768px) {
  .node-toggle {
    width: 22px;
    height: 22px;
  }

  .json-node-line {
    padding: 3px 6px;
  }
}
</style>
