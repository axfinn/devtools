<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>JSON 工具</h2>
      <div class="actions">
        <el-button-group>
          <el-button type="primary" @click="formatJson">格式化</el-button>
          <el-button @click="compressJson">压缩</el-button>
          <el-button @click="validateJson">校验</el-button>
        </el-button-group>
        <el-button-group style="margin-left: 10px">
          <el-button @click="toGoStruct">转 Go Struct</el-button>
          <el-button @click="toTypeScript">转 TypeScript</el-button>
        </el-button-group>
        <el-button @click="copyOutput" style="margin-left: 10px">
          <el-icon><CopyDocument /></el-icon>
          复制
        </el-button>
      </div>
    </div>

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">输入 JSON</div>
        <textarea
          ref="leftEditor"
          v-model="inputJson"
          class="code-editor"
          placeholder="请输入 JSON..."
          spellcheck="false"
          @scroll="onScroll('left')"
        ></textarea>
      </div>
      <div class="editor-panel">
        <div class="panel-header">输出结果</div>
        <textarea
          ref="rightEditor"
          v-model="outputJson"
          class="code-editor"
          readonly
          spellcheck="false"
          @scroll="onScroll('right')"
        ></textarea>
      </div>
    </div>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon :closable="false" />
    </div>

    <!-- JSON Path 查询 -->
    <div class="jsonpath-section">
      <h3>JSON Path 查询</h3>
      <div class="jsonpath-input">
        <el-input
          v-model="jsonPath"
          placeholder="输入 JSON Path，如 $.data.items[0].name"
          @keyup.enter="queryJsonPath"
        >
          <template #append>
            <el-button @click="queryJsonPath">查询</el-button>
          </template>
        </el-input>
      </div>
      <div v-if="jsonPathResult" class="jsonpath-result">
        <pre>{{ jsonPathResult }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

const inputJson = ref('')
const outputJson = ref('')
const errorMsg = ref('')
const jsonPath = ref('')
const jsonPathResult = ref('')

const leftEditor = ref(null)
const rightEditor = ref(null)
let isScrolling = false

const onScroll = (source) => {
  if (isScrolling) return
  isScrolling = true

  const sourceEl = source === 'left' ? leftEditor.value : rightEditor.value
  const targetEl = source === 'left' ? rightEditor.value : leftEditor.value

  if (sourceEl && targetEl) {
    targetEl.scrollTop = sourceEl.scrollTop
    targetEl.scrollLeft = sourceEl.scrollLeft
  }

  requestAnimationFrame(() => {
    isScrolling = false
  })
}

const formatJson = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    outputJson.value = JSON.stringify(parsed, null, 2)
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = 'JSON 解析错误: ' + e.message
  }
}

const compressJson = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    outputJson.value = JSON.stringify(parsed)
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = 'JSON 解析错误: ' + e.message
  }
}

const validateJson = () => {
  try {
    JSON.parse(inputJson.value)
    errorMsg.value = ''
    ElMessage.success('JSON 格式正确')
  } catch (e) {
    errorMsg.value = 'JSON 解析错误: ' + e.message
  }
}

const toGoStruct = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    outputJson.value = jsonToGoStruct(parsed, 'Root')
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = 'JSON 解析错误: ' + e.message
  }
}

const jsonToGoStruct = (obj, name) => {
  if (typeof obj !== 'object' || obj === null) {
    return getGoType(obj)
  }

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
  if (typeof value === 'object') {
    return '*' + name
  }
  return getGoType(value)
}

const toPascalCase = (str) => {
  return str.replace(/(^|_)(\w)/g, (_, __, c) => c.toUpperCase())
}

const toTypeScript = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    outputJson.value = jsonToTypeScript(parsed, 'Root')
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = 'JSON 解析错误: ' + e.message
  }
}

const jsonToTypeScript = (obj, name) => {
  if (typeof obj !== 'object' || obj === null) {
    return getTsType(obj)
  }

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
  if (typeof value === 'object') {
    return name
  }
  return getTsType(value)
}

const queryJsonPath = () => {
  try {
    const parsed = JSON.parse(inputJson.value)
    const result = evaluateJsonPath(parsed, jsonPath.value)
    jsonPathResult.value = JSON.stringify(result, null, 2)
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = 'JSON Path 查询错误: ' + e.message
  }
}

const evaluateJsonPath = (obj, path) => {
  if (!path.startsWith('$')) {
    throw new Error('JSON Path 必须以 $ 开头')
  }

  const parts = path.slice(1).split(/\.|\[|\]/).filter(p => p)
  let current = obj

  for (const part of parts) {
    if (current === undefined || current === null) {
      return undefined
    }
    if (/^\d+$/.test(part)) {
      current = current[parseInt(part)]
    } else {
      current = current[part]
    }
  }

  return current
}

const copyOutput = async () => {
  try {
    await navigator.clipboard.writeText(outputJson.value)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.editor-container {
  flex: 1;
}

.jsonpath-section {
  margin-top: 20px;
}

.jsonpath-section h3 {
  margin-bottom: 10px;
  color: var(--text-primary);
}

.jsonpath-input {
  margin-bottom: 10px;
}

.jsonpath-result {
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-base);
  padding: 15px;
  border-radius: var(--radius-md);
}

.jsonpath-result pre {
  margin: 0;
  color: var(--text-primary);
  font-family: var(--font-family-mono);
  white-space: pre-wrap;
}
</style>
