<template>
  <div class="tool-container" :class="{ 'is-fullscreen': isFullscreen }">
    <div class="tool-header">
      <h2>JSON 工具</h2>
      <div class="actions">
        <el-button-group>
          <el-button @click="triggerLoadFile">
            <el-icon><FolderOpened /></el-icon>
            加载文件
          </el-button>
          <el-button @click="loadSample">
            <el-icon><Document /></el-icon>
            示例
          </el-button>
          <el-button @click="clearAll">
            <el-icon><Delete /></el-icon>
            清除
          </el-button>
        </el-button-group>
        <el-button-group style="margin-left: 10px">
          <el-button type="primary" @click="formatJson">格式化</el-button>
          <el-button @click="compressJson">压缩</el-button>
          <el-button @click="validateJson">校验</el-button>
        </el-button-group>
        <el-button-group style="margin-left: 10px">
          <el-button @click="transformTo('yaml')">转 YAML</el-button>
          <el-button @click="transformTo('toml')">转 TOML</el-button>
          <el-button @click="transformTo('csv')" :disabled="!canArrayTransform">转 CSV</el-button>
          <el-button @click="transformTo('md')" :disabled="!canArrayTransform">转 MD</el-button>
          <el-button @click="openSchemaDialog">Schema 校验</el-button>
        </el-button-group>
        <el-button @click="copyOutput" style="margin-left: 10px">
          <el-icon><CopyDocument /></el-icon>
          复制
        </el-button>
        <el-button class="fullscreen-btn" @click="toggleFullscreen" style="margin-left: 10px">
          <el-icon><FullScreen v-if="!isFullscreen" /><Close v-else /></el-icon>
          {{ isFullscreen ? '退出' : '放大' }}
        </el-button>
      </div>
    </div>

    <input
      type="file"
      ref="fileInputRef"
      style="display: none"
      accept=".json,.jsonc,.txt,application/json"
      @change="onFilePick"
    />

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">
          <span>输入 JSON</span>
          <span class="parse-status" :class="parseStatusClass" :title="parseStatusTooltip">
            <span class="status-dot" :class="parseStatusClass"></span>
            <span v-if="parseStatus.valid === true">格式正确 · {{ parsedStats.nodes }} 节点 · {{ parsedStats.bytes }} 字节</span>
            <span v-else-if="parseStatus.valid === false">第 {{ parseStatus.line }} 行 第 {{ parseStatus.column }} 列 · {{ parseStatus.error }}</span>
            <span v-else>等待输入</span>
          </span>
        </div>
        <textarea
          ref="leftEditor"
          v-model="inputJson"
          class="code-editor"
          spellcheck="false"
          placeholder="拖入 .json 文件 / 粘贴 / 直接编辑 (Ctrl+Enter 强制格式化)"
          @scroll="onScroll('left')"
          @paste="handlePaste"
          @drop="handleDrop"
          @dragover.prevent
          @keydown.meta.enter.prevent="forceFormat"
          @keydown.ctrl.enter.prevent="forceFormat"
        ></textarea>
      </div>

      <div class="editor-panel output-panel">
        <el-tabs v-model="activeRightTab" class="json-tabs">
          <!-- 树视图 -->
          <el-tab-pane label="树视图" name="tree">
            <div class="tab-toolbar">
              <el-input
                v-model="treeSearch"
                placeholder="搜索键名或值…"
                clearable
                size="small"
                style="max-width: 280px"
              />
              <span class="hint">点节点右侧按钮可复制路径 / 值 / 编辑 / 删除</span>
            </div>
            <div class="tab-body">
              <JsonTreeViewer
                v-if="parseStatus.valid === true"
                :json-string="outputJson"
                :search-text="treeSearch"
                @update:json-string="onTreeEdit"
              />
              <div v-else-if="parseStatus.valid === false" class="empty-state err">
                输入有语法错误,先把左侧改正再看树
              </div>
              <div v-else class="empty-state">输入合法的 JSON 后,这里会自动出现折叠树视图</div>
            </div>
          </el-tab-pane>

          <!-- 源码 -->
          <el-tab-pane label="源码" name="source">
            <div class="tab-toolbar">
              <el-button-group size="small">
                <el-button @click="toGoStruct">→ Go Struct</el-button>
                <el-button @click="toTypeScript">→ TypeScript</el-button>
              </el-button-group>
              <el-button-group size="small" style="margin-left: 8px">
                <el-button @click="formatJson">格式化</el-button>
                <el-button @click="compressJson">压缩</el-button>
              </el-button-group>
              <el-button size="small" style="margin-left: 8px" @click="copyOutput">
                <el-icon><CopyDocument /></el-icon>
                复制
              </el-button>
              <span v-if="transformNotice" class="transform-notice">{{ transformNotice }}</span>
            </div>
            <textarea
              ref="rightEditor"
              v-model="outputJson"
              class="code-editor"
              readonly
              spellcheck="false"
              @scroll="onScroll('right')"
            ></textarea>
          </el-tab-pane>

          <!-- JSON Path -->
          <el-tab-pane label="JSON Path" name="path">
            <div class="tab-toolbar column">
              <div class="path-row">
                <el-input
                  v-model="jsonPath"
                  placeholder="$.store.book[*].title 或 $..[?(@.price<10)]"
                  clearable
                  @keyup.enter="runJsonPath"
                >
                  <template #prepend>
                    <span class="prepend-label">$.</span>
                  </template>
                  <template #append>
                    <el-button @click="runJsonPath">查询</el-button>
                  </template>
                </el-input>
              </div>
              <div class="path-chips">
                <span class="chip-label">常用:</span>
                <el-tag
                  v-for="c in jsonPathChips"
                  :key="c.path"
                  class="chip"
                  @click="jsonPath = c.path"
                  effect="plain"
                >{{ c.label }}</el-tag>
              </div>
            </div>
            <div class="tab-body">
              <div v-if="jsonPathError" class="empty-state err">{{ jsonPathError }}</div>
              <div v-else-if="jsonPathResult" class="path-result">
                <div class="path-result-meta">命中 {{ jsonPathResultCount }} 个</div>
                <pre>{{ jsonPathResult }}</pre>
              </div>
              <div v-else class="empty-state">输入 JSONPath 后回车查询</div>
            </div>
          </el-tab-pane>

          <!-- Diff -->
          <el-tab-pane label="Diff" name="diff">
            <div class="diff-pane">
              <div class="diff-toolbar">
                <el-button-group size="small">
                  <el-button @click="formatDiffBoth">格式化两侧</el-button>
                  <el-button @click="swapDiffSides" :disabled="!diffInputA && !diffInputB">交换 A/B</el-button>
                  <el-button @click="copyDiffResult" :disabled="diffVisibleRows.length === 0">复制 diff</el-button>
                  <el-button @click="clearDiffSides">清除</el-button>
                </el-button-group>
                <el-checkbox v-model="diffOnlyChanges" size="small">只显示差异</el-checkbox>
                <el-checkbox v-model="diffWordHighlight" size="small" :disabled="diffVisibleRows.length === 0">词级高亮</el-checkbox>
                <span class="diff-toolbar-spacer"></span>
                <span class="diff-legend">
                  <span class="legend-add">+ {{ diffStats.added }}</span>
                  <span class="legend-rem">- {{ diffStats.removed }}</span>
                  <span class="legend-eq">= {{ diffStats.equal }}</span>
                </span>
              </div>
              <div class="diff-inputs">
                <div class="diff-side">
                  <div class="diff-side-header">
                    <span>A (旧)</span>
                    <el-button size="small" @click="triggerDiffLoad('a')">加载文件</el-button>
                    <input
                      type="file"
                      ref="diffFileA"
                      style="display:none"
                      accept=".json,.txt,.yaml,.yml,.toml"
                      @change="(e) => onDiffFilePick(e, 'a')"
                    />
                  </div>
                  <textarea
                    v-model="diffInputA"
                    class="code-editor"
                    spellcheck="false"
                    @scroll="onDiffScroll('a', $event)"
                    ref="diffTextareaA"
                  />
                </div>
                <div class="diff-side">
                  <div class="diff-side-header">
                    <span>B (新)</span>
                    <el-button size="small" @click="triggerDiffLoad('b')">加载文件</el-button>
                    <input
                      type="file"
                      ref="diffFileB"
                      style="display:none"
                      accept=".json,.txt,.yaml,.yml,.toml"
                      @change="(e) => onDiffFilePick(e, 'b')"
                    />
                  </div>
                  <textarea
                    v-model="diffInputB"
                    class="code-editor"
                    spellcheck="false"
                    @scroll="onDiffScroll('b', $event)"
                    ref="diffTextareaB"
                  />
                </div>
              </div>
              <div class="diff-result">
                <div v-if="diffVisibleRows.length === 0" class="empty-state">两侧都填入内容后,会自动显示差异</div>
                <div v-else>
                  <div class="diff-body">
                    <div
                      v-for="(row, i) in diffVisibleRows"
                      :key="i"
                      class="diff-line"
                      :class="rowClass(row)"
                    >
                      <span class="diff-gutter diff-gutter-a">{{ row.aLine ?? '' }}</span>
                      <span class="diff-gutter diff-gutter-b">{{ row.bLine ?? '' }}</span>
                      <span class="diff-prefix">{{ rowPrefix(row) }}</span>
                      <pre class="diff-text" v-html="renderDiffRowText(row)"></pre>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon :closable="false" />
    </div>

    <!-- Schema 校验弹窗 -->
    <el-dialog
      v-model="schemaDialogVisible"
      title="JSON Schema 校验"
      width="780px"
      :close-on-click-modal="false"
    >
      <div class="schema-dialog">
        <div class="schema-section">
          <div class="schema-label">Schema (draft-07 / 2019-09 / 2020-12):</div>
          <textarea v-model="schemaInput" class="code-editor schema-input" spellcheck="false" placeholder="粘贴 JSON Schema,例如 { &quot;type&quot;:&quot;object&quot;, &quot;required&quot;:[&quot;id&quot;], &quot;properties&quot;:{ &quot;id&quot;:{&quot;type&quot;:&quot;number&quot;} } }" />
        </div>
        <div class="schema-result">
          <div v-if="schemaError" class="empty-state err">Schema 解析失败: {{ schemaError }}</div>
          <div v-else-if="!schemaRunTriggered" class="empty-state">点下方按钮运行校验</div>
          <div v-else-if="schemaErrors.length === 0" class="empty-state ok">✓ 数据符合 Schema</div>
          <div v-else>
            <div class="schema-result-label">校验错误 ({{ schemaErrors.length }} 条):</div>
            <ul class="schema-errors">
              <li v-for="(err, idx) in schemaErrors" :key="idx">
                <code>{{ err.instancePath || '/' }}</code>
                <span>{{ err.message }}</span>
                <span v-if="err.params" class="schema-params">{{ formatAjvParams(err.params) }}</span>
              </li>
            </ul>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="schemaDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="runSchemaValidate" :disabled="!hasValidData">运行校验</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { FullScreen, Close, FolderOpened, Document, Delete, CopyDocument } from '@element-plus/icons-vue'
import JsonTreeViewer from '../../components/JsonTreeViewer.vue'
import { JSONPath } from 'jsonpath-plus'
import yaml from 'js-yaml'
import Papa from 'papaparse'
import Ajv2020 from 'ajv/dist/2020.js'
import addFormats from 'ajv-formats'
import { diffLines, diffWords } from 'diff'

// ===== 基础状态 =====
const isFullscreen = ref(false)
const activeRightTab = ref('tree')

const inputJson = ref('')
const outputJson = ref('')
const errorMsg = ref('')
const treeSearch = ref('')

const leftEditor = ref(null)
const rightEditor = ref(null)
let isScrolling = false

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

const onKeydown = (e) => {
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
  }
}
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

// ===== 实时解析 =====
const parseStatus = computed(() => {
  const raw = inputJson.value
  if (!raw || !raw.trim()) return { valid: null, error: null, line: null, column: null }
  try {
    JSON.parse(raw)
    return { valid: true, error: null, line: null, column: null }
  } catch (e) {
    const msg = String(e.message || '')
    const m = msg.match(/position (\d+)/)
    let line = null
    let column = null
    if (m) {
      const pos = parseInt(m[1], 10)
      const upto = raw.slice(0, pos)
      const lines = upto.split('\n')
      line = lines.length
      column = pos - (upto.length - lines[lines.length - 1].length - 1)
    } else {
      const lm = msg.match(/line (\d+) column (\d+)/)
      if (lm) {
        line = parseInt(lm[1], 10)
        column = parseInt(lm[2], 10)
      }
    }
    return { valid: false, error: msg, line, column }
  }
})

const parseStatusClass = computed(() => {
  if (parseStatus.value.valid === true) return 'status--ok'
  if (parseStatus.value.valid === false) return 'status--err'
  return 'status--idle'
})

const parseStatusTooltip = computed(() => {
  const p = parseStatus.value
  if (p.valid === true) return 'JSON 格式正确'
  if (p.valid === false) return p.error || ''
  return ''
})

const parsedStats = computed(() => {
  if (parseStatus.value.valid !== true) return { nodes: 0, bytes: 0 }
  try {
    const v = JSON.parse(inputJson.value)
    let nodes = 0
    const walk = (n) => {
      if (n === null || typeof n !== 'object') return
      nodes++
      if (Array.isArray(n)) n.forEach(walk)
      else Object.values(n).forEach(walk)
    }
    walk(v)
    return { nodes, bytes: new Blob([inputJson.value]).size }
  } catch { return { nodes: 0, bytes: 0 } }
})

const canArrayTransform = computed(() => {
  if (parseStatus.value.valid !== true) return false
  try {
    const v = JSON.parse(inputJson.value)
    return Array.isArray(v) && v.every(x => x && typeof x === 'object' && !Array.isArray(x))
  } catch { return false }
})

// ===== 自动把已解析的 JSON 灌进 outputJson(只格式化不限"压缩") =====
// 用户改 inputJson → 若 valid 且不是已格式化版本,debounced 后自动格式化灌到 outputJson;
// 这样树视图/源码 tab/转换/Schema 全都基于最新格式化后的值,而不必各自按按钮。
let parseTimer = null
watch(inputJson, () => {
  if (parseTimer) clearTimeout(parseTimer)
  parseTimer = setTimeout(() => {
    if (parseStatus.value.valid === true) {
      outputJson.value = JSON.stringify(JSON.parse(inputJson.value), null, 2)
      errorMsg.value = ''
    } else if (parseStatus.value.valid === false && inputJson.value.trim()) {
      outputJson.value = ''
    }
  }, 300)
})

// ===== 格式化 / 压缩 / 校验(老逻辑保留) =====
const formatJson = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    const formatted = JSON.stringify(parsed, null, 2)
    outputJson.value = formatted
    inputJson.value = formatted
    errorMsg.value = ''
    activeRightTab.value = 'source'
    ElMessage.success('已格式化')
  } catch (e) { errorMsg.value = 'JSON 解析错误: ' + e.message }
}
const compressJson = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    const minified = JSON.stringify(parsed)
    outputJson.value = minified
    inputJson.value = minified
    errorMsg.value = ''
    activeRightTab.value = 'source'
    ElMessage.success('已压缩')
  } catch (e) { errorMsg.value = 'JSON 解析错误: ' + e.message }
}
const validateJson = () => {
  try {
    JSON.parse(inputJson.value)
    errorMsg.value = ''
    ElMessage.success('JSON 格式正确')
  } catch (e) { errorMsg.value = 'JSON 解析错误: ' + e.message }
}
const forceFormat = () => formatJson()

// ===== 加载文件 / 拖文件 / 粘贴 =====
const fileInputRef = ref(null)
const triggerLoadFile = () => fileInputRef.value?.click()
const onFilePick = async (e) => {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  await loadFileIntoInput(file, inputJson, 'left')
}
const loadFileIntoInput = async (file, target, label) => {
  try {
    if (file.size > 5 * 1024 * 1024) {
      ElMessage.warning('文件超过 5MB,无法加载')
      return
    }
    const text = await file.text()
    try {
      JSON.parse(text)
    } catch {
      ElMessage.warning(`${label}: 已加载,但不是合法 JSON(已写入,可继续编辑)`)
      target.value = text
      return
    }
    target.value = text
    ElMessage.success(`已加载 ${file.name} (${(file.size / 1024).toFixed(1)} KB)`)
    activeRightTab.value = 'tree'
  } catch (err) {
    ElMessage.error('读取文件失败: ' + err.message)
  }
}
const handleDrop = async (e) => {
  const files = e.dataTransfer?.files
  if (!files || !files.length) return
  await loadFileIntoInput(files[0], inputJson, '左侧')
}
const handlePaste = (e) => {
  const text = e.clipboardData?.getData('text/plain')
  if (!text) return
  if (inputJson.value.trim()) return  // 当前已有内容,不做自动格式化
  setTimeout(() => {
    try {
      const parsed = JSON.parse(inputJson.value || text)
      inputJson.value = JSON.stringify(parsed, null, 2)
    } catch {}
  }, 0)
}

// ===== 示例 / 清除 =====
const SAMPLE = `{
  "store": {
    "name": "示例书城",
    "books": [
      { "id": 1, "title": "北疆自驾笔记", "author": "林清远", "price": 58.0, "tags": ["travel", "china"] },
      { "id": 2, "title": "Catcher in the Rye", "author": "J.D. Salinger", "price": 12.5, "tags": ["fiction", "classic"] },
      { "id": 3, "title": "Deep Work", "author": "Cal Newport", "price": 14.99, "tags": ["productivity"] }
    ]
  },
  "updatedAt": "2026-07-21T10:00:00Z"
}`
const loadSample = () => { inputJson.value = SAMPLE }
const clearAll = () => {
  inputJson.value = ''
  outputJson.value = ''
  errorMsg.value = ''
  treeSearch.value = ''
  jsonPath.value = ''
  jsonPathResult.value = ''
  jsonPathError.value = ''
}

// ===== 复制 output =====
const copyOutput = async () => {
  if (!outputJson.value) { ElMessage.warning('输出为空') ; return }
  try {
    await navigator.clipboard.writeText(outputJson.value)
    ElMessage.success('已复制到剪贴板')
  } catch { ElMessage.error('复制失败') }
}

// ===== 滚动同步 =====
const onScroll = (source) => {
  if (activeRightTab.value !== 'source') return
  if (isScrolling) return
  isScrolling = true
  const sourceEl = source === 'left' ? leftEditor.value : rightEditor.value
  const targetEl = source === 'left' ? rightEditor.value : leftEditor.value
  if (sourceEl && targetEl) {
    targetEl.scrollTop = sourceEl.scrollTop
    targetEl.scrollLeft = sourceEl.scrollLeft
  }
  requestAnimationFrame(() => { isScrolling = false })
}

// ===== 树视图编辑回到 input =====
const onTreeEdit = (newJson) => {
  inputJson.value = newJson
  outputJson.value = newJson
}

// ===== 转换 (YAML/TOML/CSV/MD) =====
const transformNotice = ref('')

// 简易 JSON→TOML 转换:覆盖常见 JSON 形态(标量/对象/对象数组),
// 对 mixed-type 数组输出 conservative 形式(可能不是最优 TOML 但语义正确)
const jsonToToml = (value, keyPath = '') => {
  if (value === null) return `${keyPath} = ""\n`  // TOML 没有 null,统一空串
  if (typeof value === 'string') return `${keyPath} = ${JSON.stringify(value)}\n`
  if (typeof value === 'number' || typeof value === 'boolean') return `${keyPath} = ${value}\n`
  if (Array.isArray(value)) {
    const allScalar = value.every(v => v === null || ['string', 'number', 'boolean'].includes(typeof v))
    if (allScalar) {
      return `${keyPath} = ${JSON.stringify(value.map(v => v === null ? '' : v))}\n`
    }
    if (value.every(v => v && typeof v === 'object' && !Array.isArray(v))) {
      const prefix = keyPath ? `[${keyPath}]\n` : ''
      let out = ''
      value.forEach(item => {
        out += prefix + Object.entries(item).map(([k, v]) => jsonToToml(v, k)).join('')
      })
      return out
    }
    return `${keyPath} = ${JSON.stringify(value)}\n`  // mixed — 退化为内联
  }
  if (typeof value === 'object') {
    let out = ''
    const scalars = []
    const subTables = []
    for (const [k, v] of Object.entries(value)) {
      if (v && typeof v === 'object' && !Array.isArray(v)) {
        subTables.push([k, v])
      } else if (Array.isArray(v) && v.length && v.every(it => it && typeof it === 'object' && !Array.isArray(it))) {
        subTables.push([k, v])
      } else {
        scalars.push([k, v])
      }
    }
    for (const [k, v] of scalars) out += jsonToToml(v, k)
    for (const [k, v] of subTables) {
      const header = keyPath ? `${keyPath}.${k}` : k
      if (Array.isArray(v)) {
        v.forEach(item => { out += `\n[${header}]\n` ; for (const [sk, sv] of Object.entries(item)) out += jsonToToml(sv, sk) })
      } else {
        out += `\n[${header}]\n`
        out += jsonToToml(v, '')
      }
    }
    return out
  }
  return ''
}

const transformTo = async (kind) => {
  if (parseStatus.value.valid !== true) {
    ElMessage.warning('输入不是合法 JSON,无法转换')
    return
  }
  // 直接 parse inputJson:它刚被 parseStatus 验证为合法 JSON。
  // 不能用 outputJson.value — 它在 transform 成功后会被覆盖成 TOML/YAML/CSV/MD,
  // 再次点击 transform 时 JSON.parse 会炸("Unexpected token in TOML text")。
  const v = JSON.parse(inputJson.value)
  let result = ''
  try {
    if (kind === 'yaml') {
      result = yaml.dump(v, { indent: 2, lineWidth: -1, noRefs: true })
      transformNotice.value = '由 YAML 输出'
    } else if (kind === 'toml') {
      result = jsonToToml(v, '')
      if (!result.trim()) throw new Error('无法转 TOML(JSON 形态不被支持)')
      transformNotice.value = '由 TOML 输出'
    } else if (kind === 'csv') {
      if (!Array.isArray(v)) throw new Error('CSV 转换要求顶层数组')
      if (!v.every(x => x && typeof x === 'object' && !Array.isArray(x))) {
        throw new Error('CSV 转换要求数组元素都是对象')
      }
      result = Papa.unparse(v)
      transformNotice.value = '由 CSV 输出'
    } else if (kind === 'md') {
      if (!Array.isArray(v)) throw new Error('MD 表格要求顶层数组')
      if (!v.every(x => x && typeof x === 'object' && !Array.isArray(x))) {
        throw new Error('MD 表格要求数组元素都是对象')
      }
      const first = v[0] || {}
      const headers = Object.keys(first)
      const escape = (s) => String(s ?? '').replace(/\|/g, '\\|').replace(/\n/g, ' ')
      const lines = []
      lines.push('| ' + headers.join(' | ') + ' |')
      lines.push('|' + headers.map(() => ' --- ').join('|') + '|')
      for (const row of v) lines.push('| ' + headers.map(h => escape(row[h])).join(' | ') + ' |')
      result = lines.join('\n')
      transformNotice.value = '由 Markdown 表格输出'
    }
    outputJson.value = result
    activeRightTab.value = 'source'
    setTimeout(() => { transformNotice.value = '' }, 4000)
    ElMessage.success(`已转 ${kind.toUpperCase()}`)
    await nextTick()
  } catch (e) {
    ElMessage.error(`${kind.toUpperCase()} 转换失败: ${e.message}`)
  }
}

// ===== JSONPath(jsonpath-plus) =====
const jsonPath = ref('')
const jsonPathResult = ref('')
const jsonPathError = ref('')
const jsonPathChips = [
  { label: '所有键: $..*', path: '$..*' },
  { label: '顶层: $.*', path: '$.*' },
  { label: '所有 title: $..title', path: '$..title' },
  { label: '第一本书: $.store.books[0]', path: '$.store.books[0]' },
  { label: '便宜的书: $..[?(@.price<20)]', path: '$.store..[?(@.price<20)]' }
]
const jsonPathResultCount = computed(() => {
  if (!jsonPathResult.value) return 0
  try {
    const v = JSON.parse(jsonPathResult.value)
    return Array.isArray(v) ? v.length : 1
  } catch { return 0 }
})
const runJsonPath = () => {
  jsonPathError.value = ''
  jsonPathResult.value = ''
  if (!jsonPath.value.trim()) { jsonPathError.value = '请输入 JSONPath 表达式' ; return }
  if (parseStatus.value.valid !== true) { jsonPathError.value = '输入不是合法 JSON,无法查询' ; return }
  try {
    const data = JSON.parse(outputJson.value || inputJson.value)
    const result = JSONPath({ path: jsonPath.value, json: data })
    jsonPathResult.value = JSON.stringify(result, null, 2)
    if (Array.isArray(result) && result.length === 0) {
      jsonPathError.value = '没有命中任何节点'
    }
  } catch (e) {
    jsonPathError.value = e.message || String(e)
  }
}

// ===== Diff =====
// 数据流:
//   diffInputA / diffInputB  →  diffTryParse(每侧)
//   → diffRawParts(diffLines 出的 parts) → diffRows(摊平为行,A/B 行号,词级 segments)
//   → diffVisibleRows(可选过滤 equal 行,带前后 1 行 context)
const diffInputA = ref('')
const diffInputB = ref('')
const diffFileA = ref(null)
const diffFileB = ref(null)
const diffTextareaA = ref(null)
const diffTextareaB = ref(null)
const diffOnlyChanges = ref(false)
const diffWordHighlight = ref(true)
const diffParseErrorA = ref('')
const diffParseErrorB = ref('')

const triggerDiffLoad = (which) => (which === 'a' ? diffFileA.value : diffFileB.value)?.click()

const onDiffFilePick = async (e, which) => {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  try {
    const text = await file.text()
    if (which === 'a') diffInputA.value = text
    else diffInputB.value = text
    ElMessage.success(`已加载 ${file.name} 到 ${which.toUpperCase()}`)
  } catch (err) {
    ElMessage.error('读取失败: ' + err.message)
  }
}

// 两侧独立 try parse:能 parse 就 pretty,不能 parse 就当文本对比(允许任意格式 diff)
const diffTryParse = (raw) => {
  if (!raw || !raw.trim()) return { text: '', error: '' }
  try {
    return { text: JSON.stringify(JSON.parse(raw), null, 2) + '\n', error: '' }
  } catch (e) {
    return { text: raw.endsWith('\n') ? raw : raw + '\n', error: e.message }
  }
}

const formatDiffBoth = () => {
  // 一键格式化两侧:能 parse 的 pretty,不能 parse 的保持原样
  const a = diffTryParse(diffInputA.value)
  const b = diffTryParse(diffInputB.value)
  diffInputA.value = a.text
  diffInputB.value = b.text
  ElMessage.success('已格式化两侧(无法 parse 的保持原样)')
}

const swapDiffSides = () => {
  const tmp = diffInputA.value
  diffInputA.value = diffInputB.value
  diffInputB.value = tmp
  ElMessage.success('已交换 A/B')
}

const clearDiffSides = () => {
  diffInputA.value = ''
  diffInputB.value = ''
}

// ===== 同步滚动 =====
// 用一个 flag 防止 A 滚动触发 B 滚动,反过来又触发 A 的循环。
let syncing = false
const onDiffScroll = (which, e) => {
  if (syncing) return
  const src = e.target
  const dst = which === 'a' ? diffTextareaB.value : diffTextareaA.value
  if (!dst) return
  syncing = true
  dst.scrollTop = src.scrollTop
  dst.scrollLeft = src.scrollLeft
  // 用 rAF 解锁,比 setTimeout 0 更稳:Vue 模板更新在 rAF 里 commit
  requestAnimationFrame(() => { syncing = false })
}

// ===== 原始 parts (diffLines) =====
const diffRawParts = computed(() => {
  const aRes = diffTryParse(diffInputA.value)
  const bRes = diffTryParse(diffInputB.value)
  diffParseErrorA.value = aRes.error
  diffParseErrorB.value = bRes.error
  if (!aRes.text && !bRes.text) return []
  return diffLines(aRes.text, bRes.text)
})

// 把 part.value 切成单行(diffLines 的 value 末尾带 \n,空 part 也会有空行)
const splitPartLines = (part) => {
  const v = part.value
  if (!v) return []
  const lines = v.split('\n')
  // 末尾的 \n 会产生一个空串,去掉;但纯空 part(value === '\n')应该得到 [''] 不行,实际是 ['', ''] → 去掉两个空 → 0 行
  // diff.js 习惯:part.value 已经把 trailing \n 算进去,所以 'a\nb\n' 切出来是 ['a','b',''],我们去尾空
  if (lines.length > 0 && lines[lines.length - 1] === '') lines.pop()
  return lines
}

// 找连续 removed+added 的 part 对,在 part 内部做词级 diff
// 返回 [{part, lineWordSegments: Map<lineIdx, segments>}, ...]
const buildPartWordMap = (parts) => {
  const result = new Map() // partIdx -> Map(lineIdx -> segments)
  for (let i = 0; i + 1 < parts.length; i++) {
    if (parts[i].removed && parts[i + 1].added) {
      const aLines = splitPartLines(parts[i])
      const bLines = splitPartLines(parts[i + 1])
      const aMap = new Map()
      const bMap = new Map()
      const len = Math.max(aLines.length, bLines.length)
      for (let k = 0; k < len; k++) {
        const a = aLines[k] ?? null
        const b = bLines[k] ?? null
        if (a !== null && b !== null) {
          aMap.set(k, diffWords(a, b))
          bMap.set(k, diffWords(a, b))
        }
      }
      result.set(i, aMap)
      result.set(i + 1, bMap)
      i++ // 跳过下一个 added
    }
  }
  return result
}

// 摊平为行:每行带 A/B 行号 + 内容 + (可选) 词级 segments
const diffRows = computed(() => {
  const parts = diffRawParts.value
  if (parts.length === 0) return []
  const wordMap = buildPartWordMap(parts)
  const rows = []
  let aLine = 1
  let bLine = 1

  for (let i = 0; i < parts.length; i++) {
    const p = parts[i]
    const lines = splitPartLines(p)
    const isPaired = wordMap.has(i)
    lines.forEach((text, lineIdx) => {
      if (p.added) {
        rows.push({ kind: 'add', bLine, text, words: isPaired ? wordMap.get(i).get(lineIdx) : null })
        bLine++
      } else if (p.removed) {
        rows.push({ kind: 'rem', aLine, text, words: isPaired ? wordMap.get(i).get(lineIdx) : null })
        aLine++
      } else {
        rows.push({ kind: 'eq', aLine, bLine, text })
        aLine++
        bLine++
      }
    })
  }
  return rows
})

// "只显示差异" + 前后 1 行 context
const diffVisibleRows = computed(() => {
  const rows = diffRows.value
  if (!diffOnlyChanges.value) return rows
  if (rows.length === 0) return rows
  const hasChange = rows.some(r => r.kind !== 'eq')
  if (!hasChange) return rows
  const keep = new Array(rows.length).fill(false)
  for (let i = 0; i < rows.length; i++) {
    if (rows[i].kind !== 'eq') {
      keep[i] = true
      if (i > 0) keep[i - 1] = true
      if (i + 1 < rows.length) keep[i + 1] = true
    }
  }
  return rows.filter((_, i) => keep[i])
})

const diffStats = computed(() => {
  let added = 0, removed = 0, equal = 0
  for (const r of diffRows.value) {
    if (r.kind === 'add') added++
    else if (r.kind === 'rem') removed++
    else equal++
  }
  return { added, removed, equal }
})

const rowClass = (r) => {
  if (r.kind === 'add') return 'diff-line--add'
  if (r.kind === 'rem') return 'diff-line--rem'
  return 'diff-line--eq'
}

const rowPrefix = (r) => r.kind === 'add' ? '+' : r.kind === 'rem' ? '-' : ' '

const escapeHtml = (s) => {
  const div = document.createElement('div')
  div.textContent = s
  return div.innerHTML
}

// 词级高亮:把 removed/added 段标底色,equal 段不变。
// 关掉开关就只返回 escape 后的原文。
const renderDiffRowText = (r) => {
  const text = escapeHtml(r.text || '')
  if (!diffWordHighlight.value || !r.words) return text
  // words 是 diffWords 的输出(只在有 paired 行的 add/rem 上有)
  // 把 segments 拼回去,removed/added 加 span
  let out = ''
  for (const seg of r.words) {
    const html = escapeHtml(seg.value)
    if (r.kind === 'rem') {
      if (seg.removed) out += `<span class="word-rem">${html}</span>`
      else if (!seg.added) out += html
      // 另一半(added)是 B 独有的,这里不显示
    } else if (r.kind === 'add') {
      if (seg.added) out += `<span class="word-add">${html}</span>`
      else if (!seg.removed) out += html
    } else {
      out += html
    }
  }
  return out
}

// 复制 unified diff 文本(unified diff 格式,贴到任何 diff 工具里都能识别)
const copyDiffResult = async () => {
  if (diffRows.value.length === 0) {
    ElMessage.warning('没有 diff 可复制')
    return
  }
  const out = []
  out.push('--- A')
  out.push('+++ B')
  out.push(`@@ -1,${diffStats.value.removed + diffStats.value.equal} +1,${diffStats.value.added + diffStats.value.equal} @@`)
  for (const r of diffRows.value) {
    const prefix = r.kind === 'add' ? '+' : r.kind === 'rem' ? '-' : ' '
    out.push(prefix + (r.text || ''))
  }
  const text = out.join('\n')
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制 unified diff')
  } catch {
    ElMessage.error('复制失败')
  }
}

// ===== Schema 校验(ajv 2020-12) =====
const schemaDialogVisible = ref(false)
const schemaInput = ref('')
const schemaError = ref('')
const schemaErrors = ref([])
const schemaRunTriggered = ref(false)
const hasValidData = computed(() => parseStatus.value.valid === true)
const openSchemaDialog = () => {
  schemaDialogVisible.value = true
  schemaRunTriggered.value = false
  schemaErrors.value = []
  schemaError.value = ''
  if (!schemaInput.value) {
    schemaInput.value = JSON.stringify({
      type: 'object',
      required: ['store'],
      properties: {
        store: {
          type: 'object',
          required: ['books'],
          properties: {
            books: {
              type: 'array',
              items: {
                type: 'object',
                required: ['id', 'title', 'price'],
                properties: {
                  id: { type: 'integer' },
                  title: { type: 'string' },
                  price: { type: 'number' }
                }
              }
            }
          }
        }
      }
    }, null, 2)
  }
}
const formatAjvParams = (params) => {
  if (!params) return ''
  const entries = Object.entries(params).map(([k, v]) => `${k}=${JSON.stringify(v)}`)
  return '(' + entries.join(', ') + ')'
}
const runSchemaValidate = () => {
  schemaErrors.value = []
  schemaError.value = ''
  schemaRunTriggered.value = true
  let schema
  try {
    schema = JSON.parse(schemaInput.value)
  } catch (e) {
    schemaError.value = e.message
    return
  }
  try {
    const data = JSON.parse(outputJson.value || inputJson.value)
    const ajv = new Ajv2020({ allErrors: true, strict: false })
    addFormats(ajv)
    const validate = ajv.compile(schema)
    const valid = validate(data)
    if (valid) { schemaErrors.value = [] ; ElMessage.success('数据符合 Schema') ; return }
    schemaErrors.value = (validate.errors || []).map(e => ({
      instancePath: e.instancePath || '/',
      message: e.message,
      params: e.params
    }))
  } catch (e) {
    schemaError.value = e.message
  }
}

// ===== 老逻辑:Go Struct / TypeScript =====
const jsonToGoStruct = (obj, name) => {
  if (typeof obj !== 'object' || obj === null) return getGoType(obj)
  if (Array.isArray(obj)) {
    if (obj.length === 0) return '[]interface{}'
    return '[]' + jsonToGoStruct(obj[0], name + 'Item')
  }
  let result = `type ${name} struct {\n`
  for (const [key, value] of Object.entries(obj)) {
    const fieldName = toPascalCase(key)
    const fieldType = getGoFieldType(value, fieldName)
    result += `\t${fieldName} ${fieldType} \`json:"${key}"\`\n`
  }
  result += '}'
  return result
}
const getGoType = (value) => {
  if (value === null) return 'interface{}'
  switch (typeof value) {
    case 'string': return 'string'
    case 'number': return Number.isInteger(value) ? 'int64' : 'float64'
    case 'boolean': return 'bool'
    default: return 'interface{}'
  }
}
const getGoFieldType = (value, name) => {
  if (value === null) return 'interface{}'
  if (Array.isArray(value)) {
    if (value.length === 0) return '[]interface{}'
    return '[]' + getGoFieldType(value[0], name + 'Item')
  }
  if (typeof value === 'object') return '*' + name
  return getGoType(value)
}
const toPascalCase = (str) => str.replace(/(^|_)(\w)/g, (_, __, c) => c.toUpperCase())

const toGoStruct = () => {
  if (parseStatus.value.valid !== true) { ElMessage.warning('输入不是合法 JSON') ; return }
  const parsed = JSON.parse(outputJson.value || inputJson.value)
  outputJson.value = jsonToGoStruct(parsed, 'Root')
  activeRightTab.value = 'source'
  transformNotice.value = '由 Go Struct 输出'
  setTimeout(() => { transformNotice.value = '' }, 4000)
  ElMessage.success('已转 Go Struct')
}

const jsonToTypeScript = (obj, name) => {
  if (typeof obj !== 'object' || obj === null) return getTsType(obj)
  if (Array.isArray(obj)) {
    if (obj.length === 0) return 'any[]'
    return jsonToTypeScript(obj[0], name + 'Item') + '[]'
  }
  let result = `interface ${name} {\n`
  for (const [key, value] of Object.entries(obj)) {
    const fieldType = getTsFieldType(value, toPascalCase(key))
    result += `  ${key}: ${fieldType};\n`
  }
  result += '}'
  return result
}
const getTsType = (value) => {
  if (value === null) return 'null'
  switch (typeof value) {
    case 'string': return 'string'
    case 'number': return 'number'
    case 'boolean': return 'boolean'
    default: return 'any'
  }
}
const getTsFieldType = (value, name) => {
  if (value === null) return 'null'
  if (Array.isArray(value)) {
    if (value.length === 0) return 'any[]'
    return getTsFieldType(value[0], name + 'Item') + '[]'
  }
  if (typeof value === 'object') return name
  return getTsType(value)
}
const toTypeScript = () => {
  if (parseStatus.value.valid !== true) { ElMessage.warning('输入不是合法 JSON') ; return }
  const parsed = JSON.parse(outputJson.value || inputJson.value)
  outputJson.value = jsonToTypeScript(parsed, 'Root')
  activeRightTab.value = 'source'
  transformNotice.value = '由 TypeScript 输出'
  setTimeout(() => { transformNotice.value = '' }, 4000)
  ElMessage.success('已转 TypeScript')
}
</script>

<style scoped>
.parse-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.status--ok .status-dot { background: #67c23a; }
.status--ok { color: #67c23a; }
.status--err .status-dot { background: #f56c6c; }
.status--err { color: #f56c6c; }
.status--idle .status-dot { background: #909399; }

.output-panel {
  padding: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.json-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.json-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.json-tabs :deep(.el-tab-pane) {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.tab-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-base);
  background: var(--bg-secondary);
}
.tab-toolbar.column {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.tab-toolbar .hint {
  margin-left: auto;
  color: var(--text-secondary);
  font-size: 12px;
}
.tab-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0;
}
.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
}
.empty-state.err { color: #f56c6c; }
.empty-state.ok { color: #67c23a; font-size: 16px; }

.transform-notice {
  margin-left: 8px;
  padding: 2px 8px;
  background: #ecf5ff;
  color: #409eff;
  border-radius: 4px;
  font-size: 12px;
}

/* Path tab */
.path-row {
  display: flex;
  align-items: center;
}
.prepend-label {
  font-family: monospace;
  padding: 0 8px;
  color: var(--text-secondary);
  font-size: 13px;
}
.path-chips {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.chip-label { color: var(--text-secondary); font-size: 12px; }
.chip { cursor: pointer; }
.path-result-meta {
  padding: 8px 16px;
  color: var(--text-secondary);
  font-size: 12px;
  border-bottom: 1px solid var(--border-base);
  background: var(--bg-secondary);
}
.path-result pre {
  margin: 0;
  padding: 16px;
  font-family: var(--font-family-mono);
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-primary);
}

/* Diff tab */
.diff-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.diff-toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-base);
  flex-wrap: wrap;
}
.diff-toolbar-spacer { flex: 1; }
.diff-toolbar .diff-legend {
  background: transparent;
  border: 0;
  padding: 0;
  font-size: 12px;
}
.diff-inputs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-base);
  background: var(--bg-secondary);
  max-height: 40vh;
}
.diff-side {
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.diff-side-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}
.diff-side textarea {
  min-height: 100px;
  max-height: 220px;
  flex: 1;
}
.diff-result {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.diff-legend {
  display: flex;
  gap: 14px;
  padding: 6px 12px;
  background: var(--bg-secondary);
  font-size: 12px;
  border-bottom: 1px solid var(--border-base);
}
.legend-add { color: #67c23a; }
.legend-rem { color: #f56c6c; }
.legend-eq { color: var(--text-secondary); }
.diff-body {
  font-family: var(--font-family-mono);
  font-size: 13px;
  /* 关键:让 pre 里的内容不会被 nowrap 截断;行内允许折行,行号与 prefix 仍对齐 */
  white-space: pre;
}
.diff-line {
  display: flex;
  align-items: stretch;
  border-left: 3px solid transparent;
  min-height: 18px;
  line-height: 1.5;
}
.diff-gutter {
  display: inline-block;
  width: 38px;
  flex-shrink: 0;
  text-align: right;
  padding: 0 6px;
  color: #999;
  background: rgba(0, 0, 0, 0.04);
  user-select: none;
  font-variant-numeric: tabular-nums;
}
.diff-prefix {
  display: inline-block;
  width: 16px;
  flex-shrink: 0;
  text-align: center;
  color: var(--text-secondary);
  user-select: none;
}
.diff-text {
  flex: 1;
  margin: 0;
  padding: 0 8px;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-wrap: anywhere;
}
.diff-line--add { background: #e8f5e9; border-left-color: #67c23a; }
.diff-line--add .diff-gutter { background: rgba(103, 194, 58, 0.12); }
.diff-line--rem { background: #ffebee; border-left-color: #f56c6c; }
.diff-line--rem .diff-gutter { background: rgba(245, 108, 108, 0.12); }
.diff-line--eq { color: var(--text-secondary); }
.diff-line--eq .diff-gutter { color: #bbb; }

/* 词级高亮:在 add 行标 inserted 词,rem 行标 deleted 词 */
.word-add {
  background: rgba(103, 194, 58, 0.35);
  border-radius: 2px;
  padding: 0 1px;
}
.word-rem {
  background: rgba(245, 108, 108, 0.35);
  border-radius: 2px;
  padding: 0 1px;
  text-decoration: line-through;
  text-decoration-color: rgba(245, 108, 108, 0.6);
}

/* Schema dialog */
.schema-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 70vh;
}
.schema-section { display: flex; flex-direction: column; }
.schema-label { font-size: 13px; color: var(--text-secondary); margin-bottom: 6px; }
.schema-input { height: 200px; }
.schema-result {
  border: 1px solid var(--border-base);
  border-radius: 4px;
  padding: 12px;
  max-height: 280px;
  overflow: auto;
  background: var(--bg-secondary);
}
.schema-result-label { font-size: 13px; color: var(--text-secondary); margin-bottom: 8px; }
.schema-errors { list-style: none; padding: 0; margin: 0; }
.schema-errors li {
  padding: 6px 0;
  border-bottom: 1px dashed var(--border-base);
  font-size: 13px;
}
.schema-errors code {
  background: #ffeaa7;
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--font-family-mono);
  margin-right: 6px;
}
.schema-params { color: var(--text-secondary); font-size: 12px; margin-left: 6px; }
</style>
