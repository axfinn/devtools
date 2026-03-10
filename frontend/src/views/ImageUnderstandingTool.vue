<template>
  <div class="tool-container image-understanding-page">
    <div class="tool-header">
      <h2>图像理解</h2>
      <p>调用 MiniMax MCP 图像理解能力，支持上传图片并返回描述结果。</p>
    </div>

    <div class="tool-grid">
      <el-card class="panel-card">
        <template #header>
          <div class="panel-header">
            <span>输入</span>
            <el-button text @click="loadTools" :loading="loadingTools">刷新工具</el-button>
          </div>
        </template>

        <el-form label-position="top">
          <el-form-item label="选择工具">
            <el-select v-model="selectedTool" placeholder="自动选择" class="w-full" :loading="loadingTools">
              <el-option
                v-for="tool in tools"
                :key="tool.name"
                :label="tool.name"
                :value="tool.name"
              >
                <div class="tool-option">
                  <span>{{ tool.name }}</span>
                  <span class="tool-desc">{{ tool.description || '未填写描述' }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <el-form-item label="图片">
            <el-upload
              class="image-uploader"
              :auto-upload="false"
              :show-file-list="false"
              :on-change="handleFileChange"
              accept="image/*"
            >
              <div class="upload-area">
                <el-icon :size="46" color="#409eff"><Upload /></el-icon>
                <p>点击或拖拽上传图片</p>
              </div>
            </el-upload>
            <div v-if="imagePreview" class="image-preview">
              <img :src="imagePreview" alt="preview" />
              <el-button size="small" text type="danger" @click="clearImage">移除图片</el-button>
            </div>
          </el-form-item>

          <el-form-item label="提示词（可选）">
            <el-input v-model="prompt" type="textarea" :rows="3" placeholder="例如：描述画面里的主要物体和场景" />
          </el-form-item>

          <el-form-item label="自定义参数 JSON（可选）">
            <el-input
              v-model="argsText"
              type="textarea"
              :rows="4"
              placeholder='{"detail": "high"}'
            />
          </el-form-item>

          <el-form-item>
            <div class="action-row">
              <el-button type="primary" :loading="submitting" :disabled="!imageBase64" @click="submit">
                开始识别（同步）
              </el-button>
              <el-button type="success" :loading="sseLoading" :disabled="!imageBase64" @click="submitSse">
                开始识别（SSE）
              </el-button>
              <el-button @click="resetAll">清空</el-button>
            </div>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="panel-card result-card">
        <template #header>
          <div class="panel-header">
            <span>结果</span>
            <el-button text :disabled="!resultText" @click="copyResult">复制文本</el-button>
          </div>
        </template>

        <el-tabs v-if="resultText" v-model="resultTab" class="result-tabs">
          <el-tab-pane label="渲染预览" name="render">
            <div class="markdown-body" v-html="renderedHtml"></div>
          </el-tab-pane>
          <el-tab-pane label="原文" name="raw">
            <el-input v-model="resultText" type="textarea" :rows="10" readonly />
          </el-tab-pane>
        </el-tabs>
        <div v-else class="empty-result">
          暂无结果，请上传图片并点击开始识别。
        </div>

        <div v-if="rawResult" class="raw-block">
          <h4>原始响应</h4>
          <pre class="raw-json">{{ rawResult }}</pre>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Upload } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'

const tools = ref([])
const loadingTools = ref(false)
const selectedTool = ref('')
const prompt = ref('')
const imageBase64 = ref('')
const imagePreview = ref('')
const imageFile = ref(null)
const argsText = ref('')
const submitting = ref(false)
const sseLoading = ref(false)
let eventSource = null
const resultText = ref('')
const rawResult = ref('')
const resultTab = ref('render')
const isListening = ref(false)

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true
})

const renderedHtml = computed(() => md.render(resultText.value || ''))

const parseJsonSafe = async (resp) => {
  const text = await resp.text()
  if (!text) return {}
  try {
    return JSON.parse(text)
  } catch (err) {
    throw new Error(`JSON 解析失败: ${text.slice(0, 200)}`)
  }
}

const loadTools = async () => {
  loadingTools.value = true
  try {
    const resp = await fetch('/api/image-understanding/tools')
    const data = await parseJsonSafe(resp)
    if (!resp.ok) {
      throw new Error(data.error || '加载工具失败')
    }
    tools.value = data.tools || []
    if (!selectedTool.value && tools.value.length) {
      selectedTool.value = pickDefaultTool(tools.value)
    }
  } catch (err) {
    ElMessage.error(err.message || '加载工具失败')
  } finally {
    loadingTools.value = false
  }
}

const pickDefaultTool = (toolList) => {
  const keywords = ['image', 'vision', 'understanding', 'describe', 'caption']
  for (const keyword of keywords) {
    const found = toolList.find((tool) => tool.name?.toLowerCase().includes(keyword))
    if (found) return found.name
  }
  return toolList[0]?.name || ''
}

const handleFileChange = (file) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    imageBase64.value = e.target.result
    imagePreview.value = e.target.result
    imageFile.value = file.raw
  }
  reader.readAsDataURL(file.raw)
}

const handlePaste = (e) => {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      if (file) {
        const reader = new FileReader()
        reader.onload = (event) => {
          imageBase64.value = event.target.result
          imagePreview.value = event.target.result
          imageFile.value = file
          ElMessage.success('已从剪贴板读取图片')
        }
        reader.readAsDataURL(file)
      }
      break
    }
  }
}

const setupPasteListener = (enable) => {
  if (enable && !isListening.value) {
    window.addEventListener('paste', handlePaste)
    isListening.value = true
  } else if (!enable && isListening.value) {
    window.removeEventListener('paste', handlePaste)
    isListening.value = false
  }
}

const clearImage = () => {
  imageBase64.value = ''
  imagePreview.value = ''
  imageFile.value = null
}

const resetAll = () => {
  clearImage()
  prompt.value = ''
  argsText.value = ''
  resultText.value = ''
  rawResult.value = ''
}

const submit = async () => {
  if (!imageBase64.value) {
    ElMessage.warning('请先上传图片')
    return
  }
  let args = undefined
  if (argsText.value.trim()) {
    try {
      args = JSON.parse(argsText.value)
    } catch (err) {
      ElMessage.error('自定义参数 JSON 解析失败')
      return
    }
  }
  submitting.value = true
  resultText.value = ''
  rawResult.value = ''
  try {
    let resp
    if (imageFile.value) {
      const formData = new FormData()
      formData.append('file', imageFile.value)
      if (prompt.value) formData.append('prompt', prompt.value)
      if (selectedTool.value) formData.append('tool', selectedTool.value)
      if (args) formData.append('args', JSON.stringify(args))
      resp = await fetch('/api/image-understanding/describe-file', {
        method: 'POST',
        body: formData
      })
    } else {
      resp = await fetch('/api/image-understanding/describe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          image: imageBase64.value,
          prompt: prompt.value,
          tool: selectedTool.value || undefined,
          args
        })
      })
    }
    const data = await parseJsonSafe(resp)
    if (!resp.ok) {
      throw new Error(data.error || '识别失败')
    }
    resultText.value = data.text || ''
    rawResult.value = JSON.stringify(data.result || {}, null, 2)
    if (!resultText.value) {
      ElMessage.warning('已返回结果，但未解析到文本，请查看原始响应')
    }
  } catch (err) {
    ElMessage.error(err.message || '识别失败')
  } finally {
    submitting.value = false
  }
}

const submitSse = async () => {
  if (!imageBase64.value) {
    ElMessage.warning('请先上传图片')
    return
  }
  let args = undefined
  if (argsText.value.trim()) {
    try {
      args = JSON.parse(argsText.value)
    } catch (err) {
      ElMessage.error('自定义参数 JSON 解析失败')
      return
    }
  }

  // 关闭之前的连接
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }

  sseLoading.value = true
  resultText.value = ''
  rawResult.value = ''

  try {
    // 1. 创建任务
    let resp
    if (imageFile.value) {
      const formData = new FormData()
      formData.append('file', imageFile.value)
      if (prompt.value) formData.append('prompt', prompt.value)
      if (selectedTool.value) formData.append('tool', selectedTool.value)
      if (args) formData.append('args', JSON.stringify(args))
      resp = await fetch('/api/image-understanding/sse/create-file', {
        method: 'POST',
        body: formData
      })
    } else {
      resp = await fetch('/api/image-understanding/sse/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          image: imageBase64.value,
          prompt: prompt.value,
          tool: selectedTool.value || undefined,
          args
        })
      })
    }
    const data = await parseJsonSafe(resp)
    if (!resp.ok) {
      throw new Error(data.error || '创建任务失败')
    }

    const taskId = data.task_id
    ElMessage.info('任务已创建，ID: ' + taskId + '，正在处理...')

    // 2. 建立 SSE 连接
    eventSource = new EventSource('/api/image-understanding/sse/stream/' + taskId)

    eventSource.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.status === 'completed') {
          resultText.value = msg.text || ''
          rawResult.value = JSON.stringify(msg.result || {}, null, 2)
          ElMessage.success('识别完成')
          sseLoading.value = false
          if (eventSource) {
            eventSource.close()
            eventSource = null
          }
        } else if (msg.status === 'failed') {
          ElMessage.error(msg.error || '识别失败')
          sseLoading.value = false
          if (eventSource) {
            eventSource.close()
            eventSource = null
          }
        }
      } catch (err) {
        console.error('SSE 消息解析失败:', err)
      }
    }

    eventSource.onerror = () => {
      ElMessage.error('SSE 连接失败，请稍后重试')
      sseLoading.value = false
      if (eventSource) {
        eventSource.close()
        eventSource = null
      }
    }

  } catch (err) {
    ElMessage.error(err.message || 'SSE 模式启动失败')
    sseLoading.value = false
  }
}

const copyResult = async () => {
  if (!resultText.value) return
  try {
    await navigator.clipboard.writeText(resultText.value)
    ElMessage.success('已复制')
  } catch (err) {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  loadTools()
  setupPasteListener(true)
})

onUnmounted(() => {
  setupPasteListener(false)
})
</script>

<style scoped>
.image-understanding-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tool-header h2 {
  margin: 0 0 6px;
  font-size: 24px;
}

.tool-header p {
  margin: 0;
  color: #6b7280;
}

.tool-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
}

.panel-card {
  border-radius: 12px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tool-option {
  display: flex;
  flex-direction: column;
}

.tool-desc {
  font-size: 12px;
  color: #909399;
}

.image-uploader :deep(.el-upload) {
  width: 100%;
}

.upload-area {
  border: 1px dashed #dcdfe6;
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  color: #606266;
  background: #fafafa;
}

.image-preview {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.image-preview img {
  max-width: 100%;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.action-row {
  display: flex;
  gap: 12px;
}

.result-card {
  min-height: 320px;
}

.empty-result {
  color: #909399;
  text-align: center;
  padding: 36px 12px;
}

.result-tabs {
  margin-top: 4px;
}

.markdown-body {
  font-size: 14px;
  line-height: 1.7;
  color: #1f2937;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 16px 0 8px;
  font-weight: 600;
}

.markdown-body :deep(p) {
  margin: 8px 0;
}

.markdown-body :deep(pre) {
  padding: 12px;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  overflow-x: auto;
}

.markdown-body :deep(code) {
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 6px;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
}

.raw-block {
  margin-top: 16px;
}

.raw-json {
  margin: 8px 0 0;
  padding: 12px;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 10px;
  font-size: 12px;
  overflow-x: auto;
}
</style>
