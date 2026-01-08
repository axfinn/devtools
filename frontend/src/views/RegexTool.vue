<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>正则表达式测试工具</h2>
    </div>

    <div class="regex-input-section">
      <div class="regex-row">
        <span class="regex-delimiter">/</span>
        <el-input
          v-model="pattern"
          placeholder="输入正则表达式"
          class="regex-input"
          @input="testRegex"
        />
        <span class="regex-delimiter">/</span>
        <el-input
          v-model="flags"
          placeholder="gim"
          class="flags-input"
          style="width: 80px"
          @input="testRegex"
        />
      </div>
      <div class="flag-toggles">
        <el-checkbox v-model="flagG" @change="updateFlags">g (全局匹配)</el-checkbox>
        <el-checkbox v-model="flagI" @change="updateFlags">i (忽略大小写)</el-checkbox>
        <el-checkbox v-model="flagM" @change="updateFlags">m (多行模式)</el-checkbox>
        <el-checkbox v-model="flagS" @change="updateFlags">s (单行模式)</el-checkbox>
      </div>
    </div>

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">测试文本</div>
        <textarea
          v-model="testText"
          class="code-editor"
          placeholder="输入要测试的文本..."
          spellcheck="false"
          @input="testRegex"
        ></textarea>
      </div>
      <div class="editor-panel result-panel">
        <div class="panel-header">
          匹配结果
          <span class="match-count">{{ matches.length }} 个匹配</span>
        </div>
        <div class="highlighted-text" v-html="highlightedText"></div>
      </div>
    </div>

    <div v-if="matches.length > 0" class="matches-detail">
      <h4>匹配详情</h4>
      <el-table :data="matches" border stripe max-height="300">
        <el-table-column prop="index" label="#" width="60" />
        <el-table-column prop="match" label="匹配内容" />
        <el-table-column prop="position" label="位置" width="100" />
        <el-table-column label="捕获组">
          <template #default="{ row }">
            <span v-if="row.groups.length === 0">-</span>
            <div v-else>
              <el-tag
                v-for="(g, i) in row.groups"
                :key="i"
                size="small"
                style="margin: 2px"
              >
                ${{ i + 1 }}: {{ g }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div v-if="regexError" class="error-msg">
      <el-alert :title="regexError" type="error" show-icon :closable="false" />
    </div>

    <div class="common-patterns">
      <h4>常用正则表达式</h4>
      <div class="pattern-grid">
        <div
          v-for="item in commonPatterns"
          :key="item.name"
          class="pattern-card"
          @click="usePattern(item)"
        >
          <div class="pattern-name">{{ item.name }}</div>
          <div class="pattern-regex">{{ item.pattern }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const pattern = ref('')
const flags = ref('g')
const testText = ref('Hello World!\nTest email: test@example.com\nPhone: 138-1234-5678\nURL: https://www.example.com/path?query=value')
const matches = ref([])
const regexError = ref('')
const flagG = ref(true)
const flagI = ref(false)
const flagM = ref(false)
const flagS = ref(false)

const commonPatterns = [
  { name: '邮箱', pattern: '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}' },
  { name: '手机号', pattern: '1[3-9]\\d{9}' },
  { name: 'URL', pattern: 'https?://[\\w.-]+(?:/[\\w./?%&=-]*)?' },
  { name: 'IP地址', pattern: '\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}' },
  { name: '中文', pattern: '[\\u4e00-\\u9fa5]+' },
  { name: '日期 YYYY-MM-DD', pattern: '\\d{4}-\\d{2}-\\d{2}' },
  { name: '时间 HH:MM:SS', pattern: '\\d{2}:\\d{2}:\\d{2}' },
  { name: '整数', pattern: '-?\\d+' },
  { name: '浮点数', pattern: '-?\\d+\\.\\d+' },
  { name: 'HTML标签', pattern: '<[^>]+>' },
  { name: '空白行', pattern: '^\\s*$' },
  { name: '首尾空白', pattern: '^\\s+|\\s+$' }
]

const updateFlags = () => {
  let f = ''
  if (flagG.value) f += 'g'
  if (flagI.value) f += 'i'
  if (flagM.value) f += 'm'
  if (flagS.value) f += 's'
  flags.value = f
  testRegex()
}

watch(flags, (newVal) => {
  flagG.value = newVal.includes('g')
  flagI.value = newVal.includes('i')
  flagM.value = newVal.includes('m')
  flagS.value = newVal.includes('s')
})

const testRegex = () => {
  matches.value = []
  regexError.value = ''

  if (!pattern.value || !testText.value) return

  try {
    const regex = new RegExp(pattern.value, flags.value)
    let match
    let index = 0

    if (flags.value.includes('g')) {
      while ((match = regex.exec(testText.value)) !== null) {
        matches.value.push({
          index: ++index,
          match: match[0],
          position: match.index,
          groups: match.slice(1)
        })
        // 防止无限循环
        if (match[0].length === 0) {
          regex.lastIndex++
        }
      }
    } else {
      match = regex.exec(testText.value)
      if (match) {
        matches.value.push({
          index: 1,
          match: match[0],
          position: match.index,
          groups: match.slice(1)
        })
      }
    }
  } catch (e) {
    regexError.value = '正则表达式错误: ' + e.message
  }
}

const highlightedText = computed(() => {
  if (!pattern.value || !testText.value || regexError.value) {
    return escapeHtml(testText.value)
  }

  try {
    const regex = new RegExp(pattern.value, flags.value.includes('g') ? flags.value : flags.value + 'g')
    return escapeHtml(testText.value).replace(
      new RegExp(escapeHtml(pattern.value), flags.value.includes('g') ? flags.value : flags.value + 'g'),
      '<mark class="highlight">$&</mark>'
    )
  } catch (e) {
    return escapeHtml(testText.value)
  }
})

const escapeHtml = (text) => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const usePattern = (item) => {
  pattern.value = item.pattern
  testRegex()
}
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.regex-input-section {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.regex-row {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-bottom: 15px;
}

.regex-delimiter {
  color: #f44336;
  font-size: 24px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.regex-input {
  flex: 1;
}

.regex-input :deep(.el-input__inner) {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 16px;
}

.flag-toggles {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.editor-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
  min-height: 300px;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
}

.panel-header {
  padding: 10px 15px;
  background-color: #2d2d2d;
  color: #a0a0a0;
  font-size: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.match-count {
  color: #4caf50;
}

.code-editor {
  flex: 1;
  width: 100%;
  padding: 15px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.6;
  outline: none;
}

.highlighted-text {
  flex: 1;
  padding: 15px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: #d4d4d4;
  overflow-y: auto;
}

.highlighted-text :deep(.highlight) {
  background-color: rgba(255, 235, 59, 0.4);
  color: #ffeb3b;
  padding: 2px 4px;
  border-radius: 3px;
}

.matches-detail {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.matches-detail h4 {
  margin: 0 0 15px 0;
  color: #e0e0e0;
}

.common-patterns {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.common-patterns h4 {
  margin: 0 0 15px 0;
  color: #e0e0e0;
}

.pattern-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
}

.pattern-card {
  background-color: #2d2d2d;
  padding: 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.pattern-card:hover {
  background-color: #3d3d3d;
  transform: translateY(-2px);
}

.pattern-name {
  color: #e0e0e0;
  font-weight: 500;
  margin-bottom: 5px;
}

.pattern-regex {
  color: #4caf50;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  word-break: break-all;
}

.error-msg {
  margin-top: 10px;
}
</style>
