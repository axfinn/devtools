<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>URL 编解码工具</h2>
    </div>

    <el-tabs v-model="activeTab" class="tool-tabs">
      <el-tab-pane label="URL 编解码" name="encode">
        <div class="tab-content">
          <div class="editor-panel">
            <div class="panel-header">原始文本</div>
            <textarea
              v-model="textInput"
              class="code-editor"
              placeholder="输入要编码的文本..."
              spellcheck="false"
            ></textarea>
          </div>
          <div class="button-group">
            <el-button type="primary" @click="encodeUrl">
              编码 →
            </el-button>
            <el-button type="success" @click="decodeUrl">
              ← 解码
            </el-button>
            <el-button @click="encodeComponent">
              组件编码
            </el-button>
          </div>
          <div class="editor-panel">
            <div class="panel-header">
              编码结果
              <el-button size="small" @click="copyResult">复制</el-button>
            </div>
            <textarea
              v-model="encodedOutput"
              class="code-editor"
              placeholder="URL 编码结果..."
              spellcheck="false"
            ></textarea>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="URL 解析" name="parse">
        <div class="parse-content">
          <div class="input-section">
            <el-input
              v-model="urlInput"
              placeholder="输入完整 URL，如 https://example.com/path?param=value#hash"
              size="large"
              @input="parseUrl"
            >
              <template #prepend>URL</template>
              <template #append>
                <el-button @click="parseUrl">解析</el-button>
              </template>
            </el-input>
          </div>

          <div v-if="parsedUrl" class="parsed-result">
            <el-descriptions :column="1" border>
              <el-descriptions-item label="协议 (Protocol)">
                {{ parsedUrl.protocol }}
              </el-descriptions-item>
              <el-descriptions-item label="主机 (Host)">
                {{ parsedUrl.host }}
              </el-descriptions-item>
              <el-descriptions-item label="主机名 (Hostname)">
                {{ parsedUrl.hostname }}
              </el-descriptions-item>
              <el-descriptions-item label="端口 (Port)">
                {{ parsedUrl.port || '(默认)' }}
              </el-descriptions-item>
              <el-descriptions-item label="路径 (Pathname)">
                {{ parsedUrl.pathname }}
              </el-descriptions-item>
              <el-descriptions-item label="查询字符串 (Search)">
                {{ parsedUrl.search || '(无)' }}
              </el-descriptions-item>
              <el-descriptions-item label="哈希 (Hash)">
                {{ parsedUrl.hash || '(无)' }}
              </el-descriptions-item>
            </el-descriptions>

            <div v-if="queryParams.length > 0" class="query-params">
              <h4>查询参数</h4>
              <el-table :data="queryParams" border stripe>
                <el-table-column prop="key" label="参数名" />
                <el-table-column prop="value" label="值" />
                <el-table-column prop="decoded" label="解码后的值" />
              </el-table>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="参数构建" name="build">
        <div class="build-content">
          <div class="base-url-input">
            <el-input
              v-model="baseUrl"
              placeholder="输入基础 URL，如 https://example.com/api"
            >
              <template #prepend>基础 URL</template>
            </el-input>
          </div>

          <div class="params-builder">
            <h4>添加参数</h4>
            <div v-for="(param, index) in buildParams" :key="index" class="param-row">
              <el-input v-model="param.key" placeholder="参数名" style="width: 200px" />
              <span>=</span>
              <el-input v-model="param.value" placeholder="参数值" style="flex: 1" />
              <el-button type="danger" :icon="Delete" circle @click="removeParam(index)" />
            </div>
            <el-button type="primary" @click="addParam">添加参数</el-button>
          </div>

          <div class="built-url">
            <div class="panel-header">
              生成的 URL
              <el-button size="small" @click="copyBuiltUrl">复制</el-button>
            </div>
            <div class="url-preview">{{ builtUrl }}</div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon :closable="false" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Delete } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const activeTab = ref('encode')
const textInput = ref('')
const encodedOutput = ref('')
const urlInput = ref('')
const parsedUrl = ref(null)
const queryParams = ref([])
const baseUrl = ref('https://example.com/api')
const buildParams = ref([{ key: '', value: '' }])
const errorMsg = ref('')

const encodeUrl = () => {
  try {
    encodedOutput.value = encodeURI(textInput.value)
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = '编码错误: ' + e.message
  }
}

const decodeUrl = () => {
  try {
    textInput.value = decodeURI(encodedOutput.value)
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = '解码错误: ' + e.message
  }
}

const encodeComponent = () => {
  try {
    encodedOutput.value = encodeURIComponent(textInput.value)
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = '编码错误: ' + e.message
  }
}

const copyResult = async () => {
  try {
    await navigator.clipboard.writeText(encodedOutput.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const parseUrl = () => {
  if (!urlInput.value) {
    parsedUrl.value = null
    queryParams.value = []
    return
  }

  try {
    const url = new URL(urlInput.value)
    parsedUrl.value = {
      protocol: url.protocol,
      host: url.host,
      hostname: url.hostname,
      port: url.port,
      pathname: url.pathname,
      search: url.search,
      hash: url.hash
    }

    queryParams.value = []
    url.searchParams.forEach((value, key) => {
      queryParams.value.push({
        key,
        value,
        decoded: decodeURIComponent(value)
      })
    })
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = 'URL 解析错误: ' + e.message
    parsedUrl.value = null
    queryParams.value = []
  }
}

const addParam = () => {
  buildParams.value.push({ key: '', value: '' })
}

const removeParam = (index) => {
  buildParams.value.splice(index, 1)
}

const builtUrl = computed(() => {
  if (!baseUrl.value) return ''
  const params = buildParams.value
    .filter(p => p.key)
    .map(p => `${encodeURIComponent(p.key)}=${encodeURIComponent(p.value)}`)
    .join('&')
  return params ? `${baseUrl.value}?${params}` : baseUrl.value
})

const copyBuiltUrl = async () => {
  try {
    await navigator.clipboard.writeText(builtUrl.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.tool-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.tool-header h2 {
  margin: 0;
  color: #333;
}

:global(.dark) .tool-header h2 {
  color: #e0e0e0;
}

.tool-tabs {
  flex: 1;
}

.tab-content {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 15px;
  min-height: 400px;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: #ffffff;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
}

:global(.dark) .editor-panel {
  background-color: #1e1e1e;
  border-color: #333;
}

.panel-header {
  padding: 10px 15px;
  background-color: #f5f5f5;
  color: #333;
  font-size: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #e0e0e0;
}

:global(.dark) .panel-header {
  background-color: #2d2d2d;
  color: #e0e0e0;
  border-bottom-color: #404040;
}

.code-editor {
  flex: 1;
  width: 100%;
  padding: 15px;
  background-color: #ffffff;
  color: #333;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.5;
  outline: none;
}

:global(.dark) .code-editor {
  background-color: #1e1e1e;
  color: #d4d4d4;
}

.button-group {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
}

.parse-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.parsed-result {
  background-color: #ffffff;
  border: 1px solid #e0e0e0;
  padding: 20px;
  border-radius: 8px;
}

:global(.dark) .parsed-result {
  background-color: #1e1e1e;
  border-color: #333;
}

.query-params {
  margin-top: 20px;
}

.query-params h4 {
  margin-bottom: 10px;
  color: #333;
}

:global(.dark) .query-params h4 {
  color: #e0e0e0;
}

.build-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.params-builder {
  background-color: #ffffff;
  border: 1px solid #e0e0e0;
  padding: 20px;
  border-radius: 8px;
}

:global(.dark) .params-builder {
  background-color: #1e1e1e;
  border-color: #333;
}

.params-builder h4 {
  margin-bottom: 15px;
  color: #333;
}

:global(.dark) .params-builder h4 {
  color: #e0e0e0;
}

.param-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.param-row span {
  color: #666;
}

:global(.dark) .param-row span {
  color: #a0a0a0;
}

.built-url {
  background-color: #ffffff;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
}

:global(.dark) .built-url {
  background-color: #1e1e1e;
  border-color: #333;
}

.url-preview {
  padding: 15px;
  font-family: 'Consolas', 'Monaco', monospace;
  color: #4caf50;
  word-break: break-all;
}

.error-msg {
  margin-top: 10px;
}
</style>
