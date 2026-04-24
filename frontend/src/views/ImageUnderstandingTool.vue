<template>
  <div class="tool-container image-understanding-page">
    <div class="tool-header">
      <h2>图像理解</h2>
      <p>支持 MiniMax MCP 和 Qwen 大模型两种图像理解方式。</p>
    </div>

    <el-tabs v-model="activeMode" class="mode-tabs">
      <!-- MiniMax MCP Tab -->
      <el-tab-pane label="MiniMax MCP" name="minimax">
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
                <el-button text :disabled="!resultText" :loading="sharingMcp" @click="shareMcpResult">保存分享</el-button>
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
      </el-tab-pane>

      <!-- Qwen 大模型理解 Tab -->
      <el-tab-pane name="qwen">
        <template #label>
          <span>大模型理解</span>
          <el-tag size="small" type="success" class="ml-1">Qwen</el-tag>
        </template>
        <div class="tool-grid">
          <el-card class="panel-card">
            <template #header>
              <div class="panel-header">
                <span>输入</span>
              </div>
            </template>

            <el-form label-position="top">
              <el-form-item label="模型">
                <el-select v-model="qwenModel" class="w-full">
                  <el-option value="qwen3.5-plus" label="qwen3.5-plus（推荐，深度思考+视觉）" />
                  <el-option value="kimi-k2.5" label="kimi-k2.5（Kimi 视觉理解）" />
                </el-select>
              </el-form-item>

              <el-form-item :label="`图片（已添加 ${qwenImages.length}/10 张）`">
                <!-- URL 输入行 -->
                <div class="img-url-row">
                  <el-input
                    v-model="qwenUrlInput"
                    placeholder="粘贴图片链接 https://..."
                    clearable
                    @keyup.enter="addQwenUrl"
                  />
                  <el-button @click="addQwenUrl" :disabled="qwenImages.length >= 10">添加 URL</el-button>
                </div>

                <!-- 上传区域 -->
                <el-upload
                  class="image-uploader mt-2"
                  :auto-upload="false"
                  :show-file-list="false"
                  :on-change="handleQwenFileChange"
                  accept="image/*"
                  :disabled="qwenImages.length >= 10"
                  multiple
                >
                  <div class="upload-area" :class="{ 'upload-disabled': qwenImages.length >= 10 }">
                    <el-icon :size="36" :color="qwenImages.length >= 10 ? '#c0c4cc' : '#67c23a'"><Upload /></el-icon>
                    <p>{{ qwenImages.length >= 10 ? '已达上限' : '点击/拖拽上传或粘贴图片（可多选）' }}</p>
                  </div>
                </el-upload>

                <!-- 图片列表预览 -->
                <div v-if="qwenImages.length" class="qwen-img-list">
                  <div v-for="(img, idx) in qwenImages" :key="idx" class="qwen-img-item">
                    <img
                      v-if="img.preview"
                      :src="img.preview"
                      class="qwen-thumb"
                      @error="img.previewError = true"
                    />
                    <div v-else class="qwen-thumb qwen-thumb-url">
                      <el-icon><Picture /></el-icon>
                    </div>
                    <div class="qwen-img-info">
                      <span class="qwen-img-label">{{ img.label }}</span>
                      <el-tag size="small" :type="img.type === 'url' ? 'warning' : 'success'">
                        {{ img.type === 'url' ? 'URL' : '文件' }}
                      </el-tag>
                    </div>
                    <el-button size="small" text type="danger" @click="removeQwenImage(idx)">删除</el-button>
                  </div>
                </div>
              </el-form-item>

              <el-form-item label="提示词">
                <el-input
                  v-model="qwenPrompt"
                  type="textarea"
                  :rows="3"
                  placeholder="例如：请详细描述图片内容，识别其中的文字和关键信息。"
                />
              </el-form-item>

              <el-form-item>
                <div class="action-row">
                  <el-button type="primary" :loading="qwenSubmitting" :disabled="!qwenImages.length" @click="submitQwen">
                    开始理解
                  </el-button>
                  <el-button @click="resetQwen">清空</el-button>
                  <el-button text @click="showLogsDialog = true">查看流水</el-button>
                </div>
              </el-form-item>
            </el-form>
          </el-card>

          <el-card class="panel-card result-card">
            <template #header>
              <div class="panel-header">
                <span>结果</span>
                <div class="panel-actions">
                  <el-tag v-if="qwenUsedModel" size="small" type="info">{{ qwenUsedModel }}</el-tag>
                  <el-button text :disabled="!qwenResultText" @click="copyQwenResult">复制文本</el-button>
                  <el-button text :disabled="!qwenResultText" :loading="sharingQwen" @click="shareQwenResult">保存分享</el-button>
                </div>
              </div>
            </template>

            <el-tabs v-if="qwenResultText" v-model="qwenResultTab" class="result-tabs">
              <el-tab-pane label="渲染预览" name="render">
                <div class="markdown-body" v-html="qwenRenderedHtml"></div>
              </el-tab-pane>
              <el-tab-pane label="原文" name="raw">
                <el-input v-model="qwenResultText" type="textarea" :rows="10" readonly />
              </el-tab-pane>
            </el-tabs>
            <div v-else class="empty-result">
              暂无结果，请上传图片并点击开始理解。
            </div>
          </el-card>
        </div>
      </el-tab-pane>

      <!-- 请求流水对话框 -->
      <el-dialog v-model="showLogsDialog" title="Qwen 视觉请求流水" width="800px" top="5vh">
        <div class="logs-toolbar">
          <el-input
            v-model="logsAdminPassword"
            type="password"
            show-password
            placeholder="输入管理员密码（image_understanding.admin_password）"
            style="max-width: 320px"
            @keyup.enter="loadLogs"
          />
          <el-button type="primary" :loading="logsLoading" @click="loadLogs">查询</el-button>
          <span class="logs-total" v-if="logsTotal > 0">共 {{ logsTotal }} 条</span>
        </div>

        <el-table :data="logs" v-loading="logsLoading" size="small" class="logs-table" max-height="480">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="model" label="模型" width="130" />
          <el-table-column prop="client_ip" label="IP" width="120" />
          <el-table-column label="状态" width="70">
            <template #default="{ row }">
              <el-tag :type="row.success ? 'success' : 'danger'" size="small">
                {{ row.success ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="latency_ms" label="延迟(ms)" width="90" />
          <el-table-column label="请求摘要" min-width="180">
            <template #default="{ row }">
              <span class="log-body">{{ row.request_body }}</span>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="150">
            <template #default="{ row }">
              {{ new Date(row.created_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
        </el-table>

        <div class="logs-pagination" v-if="logsTotal > logsPageSize">
          <el-pagination
            v-model:current-page="logsPage"
            :page-size="logsPageSize"
            :total="logsTotal"
            layout="prev, pager, next"
            @current-change="loadLogs"
          />
        </div>
      </el-dialog>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Upload, Picture } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'

const props = defineProps({
  superAdminPassword: { type: String, default: '' },
  apiKey: { type: String, default: '' }
})

const emit = defineEmits(['share'])

// ---- 模式切换 ----
const activeMode = ref('minimax')

// ---- MiniMax MCP ----
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
const sharingMcp = ref(false)
const sharingQwen = ref(false)

// ---- Qwen 大模型理解 ----
const qwenModel = ref('qwen3.5-plus')
const qwenPrompt = ref('')
// qwenImages: { type: 'url'|'file', data: string (base64 or url), preview: string, label: string }
const qwenImages = ref([])
const qwenUrlInput = ref('')
const qwenSubmitting = ref(false)
const qwenResultText = ref('')
const qwenResultTab = ref('render')
const qwenUsedModel = ref('')

// ---- 流水日志 ----
const showLogsDialog = ref(false)
const logsAdminPassword = ref('')
const logsLoading = ref(false)
const logs = ref([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsPageSize = 50

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true
})

const renderedHtml = computed(() => md.render(resultText.value || ''))
const qwenRenderedHtml = computed(() => md.render(qwenResultText.value || ''))

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

// ---- Qwen 相关方法 ----
const isValidUrl = (str) => {
  try {
    const u = new URL(str)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

const addQwenUrl = () => {
  const url = qwenUrlInput.value.trim()
  if (!url) return
  if (!isValidUrl(url)) {
    ElMessage.warning('请输入有效的图片 URL（http/https）')
    return
  }
  if (qwenImages.value.length >= 10) {
    ElMessage.warning('最多添加 10 张图片')
    return
  }
  qwenImages.value.push({ type: 'url', data: url, preview: url, label: url.slice(0, 50), previewError: false })
  qwenUrlInput.value = ''
}

const handleQwenFileChange = (file) => {
  if (qwenImages.value.length >= 10) {
    ElMessage.warning('最多添加 10 张图片')
    return
  }
  const reader = new FileReader()
  reader.onload = (e) => {
    qwenImages.value.push({
      type: 'file',
      data: e.target.result,
      preview: e.target.result,
      label: file.raw.name,
      previewError: false
    })
  }
  reader.readAsDataURL(file.raw)
}

const removeQwenImage = (idx) => {
  qwenImages.value.splice(idx, 1)
}

const resetQwen = () => {
  qwenImages.value = []
  qwenUrlInput.value = ''
  qwenPrompt.value = ''
  qwenResultText.value = ''
  qwenUsedModel.value = ''
}

const submitQwen = async () => {
  if (!qwenImages.value.length) {
    ElMessage.warning('请先添加至少一张图片')
    return
  }
  qwenSubmitting.value = true
  qwenResultText.value = ''
  qwenUsedModel.value = ''
  try {
    const images = qwenImages.value.map((img) => img.data)
    const resp = await fetch('/api/image-understanding/qwen-vision', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        images,
        prompt: qwenPrompt.value || undefined,
        model: qwenModel.value
      })
    })
    const data = await parseJsonSafe(resp)
    if (!resp.ok) {
      throw new Error(data.error || '识别失败')
    }
    qwenResultText.value = data.text || ''
    qwenUsedModel.value = data.model || qwenModel.value
    qwenResultTab.value = 'render'
    if (!qwenResultText.value) {
      ElMessage.warning('已返回结果，但未解析到文本')
    }
  } catch (err) {
    ElMessage.error(err.message || '识别失败')
  } finally {
    qwenSubmitting.value = false
  }
}

const copyQwenResult = async () => {
  if (!qwenResultText.value) return
  try {
    await navigator.clipboard.writeText(qwenResultText.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

async function shareMcpResult() {
  if (!resultText.value) return
  sharingMcp.value = true
  try {
    const draft = {
      sourceLabel: '图像理解',
      title: `图像理解 · ${(prompt.value || '').trim().slice(0, 36)}`,
      summary: `MiniMax MCP 图像理解 · 工具 ${selectedTool.value || 'auto'}`,
      resultType: 'image_understanding',
      model: selectedTool.value || 'MiniMax-MCP',
      payload: { text: resultText.value, prompt: prompt.value, raw: rawResult.value },
      assets: []
    }
    emit('share', draft)
  } catch (err) {
    ElMessage.error(err.message || '保存分享失败')
  } finally {
    sharingMcp.value = false
  }
}

async function shareQwenResult() {
  if (!qwenResultText.value) return
  sharingQwen.value = true
  try {
    const draft = {
      sourceLabel: '图像理解',
      title: `图像理解 · ${(qwenPrompt.value || '').trim().slice(0, 36)}`,
      summary: `Qwen 模型理解 · ${qwenUsedModel.value || qwenModel.value}`,
      resultType: 'image_understanding',
      model: qwenUsedModel.value || qwenModel.value,
      payload: { text: qwenResultText.value, prompt: qwenPrompt.value, images_count: qwenImages.value.length },
      assets: []
    }
    emit('share', draft)
  } catch (err) {
    ElMessage.error(err.message || '保存分享失败')
  } finally {
    sharingQwen.value = false
  }
}

const loadLogs = async () => {
  if (!logsAdminPassword.value.trim()) {
    ElMessage.warning('请输入管理员密码')
    return
  }
  logsLoading.value = true
  try {
    const offset = (logsPage.value - 1) * logsPageSize
    const url = `/api/image-understanding/qwen-vision/logs?limit=${logsPageSize}&offset=${offset}`
    const resp = await fetch(url, {
      headers: { 'X-Image-Admin-Password': logsAdminPassword.value }
    })
    const data = await parseJsonSafe(resp)
    if (!resp.ok) {
      throw new Error(data.error || '获取流水失败')
    }
    logs.value = data.logs || []
    logsTotal.value = data.total || 0
  } catch (err) {
    ElMessage.error(err.message || '获取流水失败')
  } finally {
    logsLoading.value = false
  }
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
          if (activeMode.value === 'qwen') {
            if (qwenImages.value.length >= 10) {
              ElMessage.warning('最多添加 10 张图片')
              return
            }
            qwenImages.value.push({
              type: 'file',
              data: event.target.result,
              preview: event.target.result,
              label: file.name || '粘贴图片',
              previewError: false
            })
          } else {
            imageBase64.value = event.target.result
            imagePreview.value = event.target.result
            imageFile.value = file
          }
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

    // 处理自定义事件 completed
    eventSource.addEventListener('completed', (event) => {
      try {
        const msg = JSON.parse(event.data)
        resultText.value = msg.text || ''
        rawResult.value = JSON.stringify(msg.result || {}, null, 2)
        ElMessage.success('识别完成')
        sseLoading.value = false
        if (eventSource) {
          eventSource.close()
          eventSource = null
        }
      } catch (err) {
        console.error('SSE completed 解析失败:', err)
      }
    })

    // 处理自定义事件 error
    eventSource.addEventListener('error', (event) => {
      try {
        const msg = JSON.parse(event.data)
        ElMessage.error(msg.error || '识别失败')
        sseLoading.value = false
        if (eventSource) {
          eventSource.close()
          eventSource = null
        }
      } catch (err) {
        ElMessage.error('SSE 连接失败，请稍后重试')
        sseLoading.value = false
        if (eventSource) {
          eventSource.close()
          eventSource = null
        }
      }
    })

    // 处理默认消息事件 (status, ping)
    eventSource.onmessage = (event) => {
      try {
        JSON.parse(event.data)
      } catch (err) {
        console.error('SSE 消息解析失败:', err)
      }
    }

    eventSource.onerror = () => {
      // 只有当没有自定义事件处理时才报错
      if (sseLoading.value) {
        ElMessage.error('SSE 连接失败，请稍后重试')
        sseLoading.value = false
      }
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

.panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mode-tabs {
  margin-top: 0;
}

.mode-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}

.ml-1 {
  margin-left: 4px;
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

.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.logs-total {
  color: #909399;
  font-size: 13px;
}

.logs-table {
  width: 100%;
}

.log-body {
  font-size: 12px;
  color: #606266;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
  display: block;
}

.logs-pagination {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

.img-url-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.img-url-row .el-input {
  flex: 1;
}

.mt-2 {
  margin-top: 8px;
}

.upload-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.qwen-img-list {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.qwen-img-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  background: #f5f7fa;
  border-radius: 8px;
}

.qwen-thumb {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 6px;
  border: 1px solid #ebeef5;
  flex-shrink: 0;
}

.qwen-thumb-url {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #e8f4ff;
  color: #409eff;
  font-size: 20px;
}

.qwen-img-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.qwen-img-label {
  font-size: 12px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.url-error {
  font-size: 12px;
  color: #f56c6c;
  margin-top: 4px;
}
</style>
