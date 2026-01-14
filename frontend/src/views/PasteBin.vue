<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>共享粘贴板</h2>
      <div class="info-text">
        多设备快速同步 - 创建后自动复制链接
      </div>
    </div>

    <!-- 快捷模式：简洁界面 -->
    <div class="quick-section" v-if="!showAdvanced">
      <div
        class="quick-editor"
        :class="{ 'is-dragging': isDragging }"
        @dragenter="onDragEnter"
        @dragleave="onDragLeave"
        @dragover="onDragOver"
        @drop="onDrop"
      >
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="粘贴或输入内容/图片，点击分享即可获得链接..."
          spellcheck="false"
          @paste="onPaste"
        ></textarea>
        <div v-if="isDragging" class="drop-overlay">
          <el-icon :size="48"><Upload /></el-icon>
          <span>拖放图片到此处</span>
        </div>
      </div>

      <!-- 图片上传区域 -->
      <div class="image-section">
        <div class="image-header">
          <span class="image-title">
            <el-icon><Picture /></el-icon>
            图片分享 ({{ images.length }}/{{ MAX_IMAGES }})
          </span>
          <span class="size-info" v-if="images.length > 0">
            总大小: {{ totalSizeMB }} MB / 30 MB
          </span>
        </div>

        <div class="image-grid" v-if="images.length > 0">
          <div class="image-item" v-for="(img, index) in images" :key="index">
            <img :src="img.preview" alt="预览" />
            <div class="image-overlay">
              <el-button type="danger" circle size="small" @click="removeImage(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="image-add" v-if="canAddMore" @click="selectFiles">
            <el-icon :size="24"><Plus /></el-icon>
            <span>添加图片</span>
          </div>
        </div>

        <div class="image-upload" v-else @click="selectFiles">
          <el-icon :size="32"><Picture /></el-icon>
          <span>点击上传图片或直接粘贴/拖拽</span>
          <span class="upload-hint">最多 15 张，总大小不超过 30MB</span>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          style="display: none"
          @change="onFileChange"
        />
      </div>

      <div class="quick-actions">
        <el-button type="primary" size="large" @click="quickCreate" :loading="creating" :disabled="!content.trim() && images.length === 0">
          <el-icon><Share /></el-icon>
          一键分享
        </el-button>
        <el-button size="small" text @click="showAdvanced = true">
          高级选项
        </el-button>
      </div>
    </div>

    <!-- 高级模式 -->
    <div class="create-section" v-else>
      <div
        class="editor-panel"
        :class="{ 'is-dragging': isDragging }"
        @dragenter="onDragEnter"
        @dragleave="onDragLeave"
        @dragover="onDragOver"
        @drop="onDrop"
      >
        <div class="panel-header">
          <el-input
            v-model="title"
            placeholder="标题（可选）"
            style="width: 200px"
            size="small"
          />
          <el-select v-model="language" placeholder="语言" style="width: 120px" size="small">
            <el-option label="纯文本" value="text" />
            <el-option label="JSON" value="json" />
            <el-option label="JavaScript" value="javascript" />
            <el-option label="TypeScript" value="typescript" />
            <el-option label="Python" value="python" />
            <el-option label="Go" value="go" />
            <el-option label="Java" value="java" />
            <el-option label="C/C++" value="cpp" />
            <el-option label="HTML" value="html" />
            <el-option label="CSS" value="css" />
            <el-option label="SQL" value="sql" />
            <el-option label="Markdown" value="markdown" />
            <el-option label="Shell" value="bash" />
            <el-option label="YAML" value="yaml" />
            <el-option label="XML" value="xml" />
          </el-select>
          <el-button size="small" text @click="showAdvanced = false">简洁模式</el-button>
        </div>
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="在此输入要分享的内容..."
          spellcheck="false"
          @paste="onPaste"
        ></textarea>
        <div v-if="isDragging" class="drop-overlay">
          <el-icon :size="48"><Upload /></el-icon>
          <span>拖放图片到此处</span>
        </div>
      </div>

      <!-- 高级模式图片上传 -->
      <div class="image-section">
        <div class="image-header">
          <span class="image-title">
            <el-icon><Picture /></el-icon>
            图片分享 ({{ images.length }}/{{ MAX_IMAGES }})
          </span>
          <span class="size-info" v-if="images.length > 0">
            总大小: {{ totalSizeMB }} MB / 30 MB
          </span>
        </div>

        <div class="image-grid" v-if="images.length > 0">
          <div class="image-item" v-for="(img, index) in images" :key="index">
            <img :src="img.preview" alt="预览" />
            <div class="image-overlay">
              <el-button type="danger" circle size="small" @click="removeImage(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="image-add" v-if="canAddMore" @click="selectFiles">
            <el-icon :size="24"><Plus /></el-icon>
            <span>添加图片</span>
          </div>
        </div>

        <div class="image-upload" v-else @click="selectFiles">
          <el-icon :size="32"><Picture /></el-icon>
          <span>点击上传图片或直接粘贴/拖拽</span>
          <span class="upload-hint">最多 15 张，总大小不超过 30MB</span>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          style="display: none"
          @change="onFileChange"
        />
      </div>

      <div class="options-row">
        <div class="option-item">
          <span class="option-label">过期时间</span>
          <el-select v-model="expiresIn" style="width: 120px">
            <el-option label="1 小时" :value="1" />
            <el-option label="6 小时" :value="6" />
            <el-option label="24 小时" :value="24" />
            <el-option label="3 天" :value="72" />
            <el-option label="7 天" :value="168" />
          </el-select>
        </div>
        <div class="option-item">
          <span class="option-label">最大访问次数</span>
          <el-input-number v-model="maxViews" :min="1" :max="1000" />
        </div>
        <div class="option-item">
          <span class="option-label">访问密码</span>
          <el-input
            v-model="password"
            type="password"
            placeholder="可选"
            style="width: 150px"
            show-password
          />
        </div>
        <el-button type="primary" size="large" @click="createPaste" :loading="creating" :disabled="!content.trim() && images.length === 0">
          创建分享
        </el-button>
      </div>
    </div>

    <!-- 分享结果：突出显示 -->
    <div v-if="showResult" class="result-section">
      <div class="result-card">
        <div class="result-header">
          <el-icon class="success-icon"><CircleCheck /></el-icon>
          <span>分享创建成功！链接已复制</span>
        </div>

        <div class="share-url-box">
          <div class="url-display">{{ shareUrl }}</div>
          <div class="url-actions">
            <el-button type="primary" @click="copyUrl">
              <el-icon><CopyDocument /></el-icon>
              复制链接
            </el-button>
            <el-button @click="openShare">
              <el-icon><Link /></el-icon>
              打开
            </el-button>
          </div>
        </div>

        <div class="qr-section">
          <div class="qr-title">扫码访问</div>
          <canvas ref="qrCanvas" class="qr-code"></canvas>
        </div>

        <div class="result-info">
          <span>ID: {{ createdId }}</span>
          <span>过期: {{ createdExpires }}</span>
          <span v-if="password">密码: {{ password }}</span>
        </div>

        <el-button class="new-share-btn" @click="resetForm" type="success" plain>
          <el-icon><Plus /></el-icon>
          创建新分享
        </el-button>
      </div>
    </div>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon :closable="false" />
    </div>

    <div class="tips-section" v-if="!showResult">
      <h4>使用提示</h4>
      <ul>
        <li>直接粘贴内容，点击分享即可</li>
        <li>链接自动复制到剪贴板</li>
        <li>手机可扫描二维码访问</li>
        <li>默认 24 小时后过期</li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Share, CircleCheck, CopyDocument, Link, Plus, Picture, Delete, Upload } from '@element-plus/icons-vue'
import QRCode from 'qrcode'

const API_BASE = import.meta.env.VITE_API_BASE || ''

const content = ref('')
const title = ref('')
const language = ref('text')
const expiresIn = ref(24)
const maxViews = ref(100)
const password = ref('')
const creating = ref(false)
const showResult = ref(false)
const showAdvanced = ref(false)
const createdId = ref('')
const createdExpires = ref('')
const shareUrl = ref('')
const errorMsg = ref('')
const qrCanvas = ref(null)
const images = ref([]) // { file: File, preview: string, base64: string }
const fileInput = ref(null)
const isDragging = ref(false)

const MAX_IMAGES = 15
const MAX_TOTAL_SIZE = 30 * 1024 * 1024 // 30MB

// 计算当前总大小
const totalSize = computed(() => {
  let size = content.value.length
  for (const img of images.value) {
    size += img.base64.length
  }
  return size
})

const totalSizeMB = computed(() => (totalSize.value / 1024 / 1024).toFixed(2))

const canAddMore = computed(() => images.value.length < MAX_IMAGES && totalSize.value < MAX_TOTAL_SIZE)

// 快捷创建
const quickCreate = async () => {
  await createPaste()
}

// 粘贴时自动检测内容类型或图片
const onPaste = async (e) => {
  const items = e.clipboardData?.items
  if (items) {
    for (const item of items) {
      if (item.type.startsWith('image/')) {
        e.preventDefault()
        const file = item.getAsFile()
        if (file) {
          await addImage(file)
        }
        return
      }
    }
  }

  const text = e.clipboardData?.getData('text') || ''
  // 简单检测 JSON
  if (text.trim().startsWith('{') || text.trim().startsWith('[')) {
    try {
      JSON.parse(text)
      language.value = 'json'
    } catch {}
  }
}

// 添加图片
const addImage = async (file) => {
  if (!file.type.startsWith('image/')) {
    ElMessage.warning('只支持图片文件')
    return
  }

  if (images.value.length >= MAX_IMAGES) {
    ElMessage.warning(`最多只能上传 ${MAX_IMAGES} 张图片`)
    return
  }

  // 读取为 base64
  const base64 = await fileToBase64(file)

  // 检查总大小
  const newSize = totalSize.value + base64.length
  if (newSize > MAX_TOTAL_SIZE) {
    ElMessage.warning('总大小超过 30MB 限制')
    return
  }

  images.value.push({
    file,
    preview: URL.createObjectURL(file),
    base64
  })
}

// 文件转 base64
const fileToBase64 = (file) => {
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.readAsDataURL(file)
  })
}

// 删除图片
const removeImage = (index) => {
  const img = images.value[index]
  if (img.preview) {
    URL.revokeObjectURL(img.preview)
  }
  images.value.splice(index, 1)
}

// 选择文件
const selectFiles = () => {
  fileInput.value?.click()
}

// 文件选择变化
const onFileChange = async (e) => {
  const files = e.target.files
  for (const file of files) {
    await addImage(file)
  }
  e.target.value = ''
}

// 拖拽处理
const onDragEnter = (e) => {
  e.preventDefault()
  isDragging.value = true
}

const onDragLeave = (e) => {
  e.preventDefault()
  isDragging.value = false
}

const onDragOver = (e) => {
  e.preventDefault()
}

const onDrop = async (e) => {
  e.preventDefault()
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (files) {
    for (const file of files) {
      await addImage(file)
    }
  }
}

const createPaste = async () => {
  if (!content.value.trim() && images.value.length === 0) {
    errorMsg.value = '请输入内容或上传图片'
    return
  }

  creating.value = true
  errorMsg.value = ''

  try {
    const response = await fetch(`${API_BASE}/api/paste`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: content.value,
        title: title.value,
        language: language.value,
        expires_in: expiresIn.value,
        max_views: maxViews.value,
        password: password.value,
        images: images.value.map(img => img.base64)
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }

    createdId.value = data.id
    createdExpires.value = new Date(data.expires_at).toLocaleString('zh-CN')
    shareUrl.value = `${window.location.origin}/paste/${data.id}`
    showResult.value = true

    // 自动复制链接
    try {
      await navigator.clipboard.writeText(shareUrl.value)
      ElMessage.success('链接已自动复制到剪贴板')
    } catch {
      ElMessage.success('分享创建成功')
    }

    // 生成二维码
    await nextTick()
    if (qrCanvas.value) {
      QRCode.toCanvas(qrCanvas.value, shareUrl.value, {
        width: 150,
        margin: 2,
        color: {
          dark: '#333',
          light: '#fff'
        }
      })
    }
  } catch (e) {
    errorMsg.value = e.message
  } finally {
    creating.value = false
  }
}

const copyUrl = async () => {
  try {
    await navigator.clipboard.writeText(shareUrl.value)
    ElMessage.success('链接已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const openShare = () => {
  window.open(shareUrl.value, '_blank')
}

const resetForm = () => {
  content.value = ''
  title.value = ''
  password.value = ''
  showResult.value = false
  createdId.value = ''
  // 清理图片
  for (const img of images.value) {
    if (img.preview) {
      URL.revokeObjectURL(img.preview)
    }
  }
  images.value = []
}
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.info-text {
  color: #67c23a;
  font-size: 14px;
}

/* 快捷模式 */
.quick-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.quick-editor {
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  min-height: 300px;
  display: flex;
}

.quick-editor .code-editor {
  flex: 1;
  padding: 20px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 15px;
  line-height: 1.6;
  outline: none;
}

.quick-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}

.quick-actions .el-button--large {
  padding: 15px 40px;
  font-size: 16px;
}

/* 高级模式 */
.create-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  min-height: 300px;
}

.panel-header {
  padding: 10px 15px;
  background-color: #2d2d2d;
  display: flex;
  gap: 10px;
  align-items: center;
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
  line-height: 1.5;
  outline: none;
}

.options-row {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
  background-color: #1e1e1e;
  padding: 15px;
  border-radius: 8px;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.option-label {
  color: #a0a0a0;
  font-size: 14px;
}

/* 结果展示 */
.result-section {
  display: flex;
  justify-content: center;
}

.result-card {
  background: linear-gradient(135deg, #1e3a2f 0%, #1e1e1e 100%);
  border: 2px solid #67c23a;
  border-radius: 16px;
  padding: 30px;
  max-width: 500px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.result-header {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #67c23a;
  font-size: 18px;
  font-weight: 500;
}

.success-icon {
  font-size: 28px;
}

.share-url-box {
  width: 100%;
  background: #252525;
  border-radius: 8px;
  padding: 15px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.url-display {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  color: #67c23a;
  word-break: break-all;
  padding: 10px;
  background: #1a1a1a;
  border-radius: 4px;
  text-align: center;
}

.url-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.qr-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.qr-title {
  color: #a0a0a0;
  font-size: 14px;
}

.qr-code {
  border-radius: 8px;
  background: #fff;
  padding: 10px;
}

.result-info {
  display: flex;
  gap: 20px;
  color: #808080;
  font-size: 13px;
  flex-wrap: wrap;
  justify-content: center;
}

.new-share-btn {
  margin-top: 10px;
}

.tips-section {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.tips-section h4 {
  margin: 0 0 10px 0;
  color: #e0e0e0;
}

.tips-section ul {
  margin: 0;
  padding-left: 20px;
  color: #a0a0a0;
  line-height: 1.8;
}

.error-msg {
  margin-top: 10px;
}

/* 图片上传区域 */
.image-section {
  background-color: #1e1e1e;
  border-radius: 8px;
  padding: 15px;
}

.image-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.image-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #e0e0e0;
  font-size: 14px;
}

.size-info {
  color: #67c23a;
  font-size: 13px;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 12px;
}

.image-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: 8px;
  overflow: hidden;
  background-color: #2d2d2d;
}

.image-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
}

.image-item:hover .image-overlay {
  opacity: 1;
}

.image-add {
  aspect-ratio: 1;
  border: 2px dashed #404040;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #808080;
  cursor: pointer;
  transition: all 0.2s;
}

.image-add:hover {
  border-color: #409eff;
  color: #409eff;
}

.image-add span {
  font-size: 12px;
}

.image-upload {
  border: 2px dashed #404040;
  border-radius: 8px;
  padding: 30px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #808080;
  cursor: pointer;
  transition: all 0.2s;
}

.image-upload:hover {
  border-color: #409eff;
  color: #409eff;
}

.upload-hint {
  font-size: 12px;
  color: #606060;
}

/* 拖拽状态 */
.quick-editor,
.editor-panel {
  position: relative;
}

.is-dragging {
  border: 2px dashed #409eff !important;
}

.drop-overlay {
  position: absolute;
  inset: 0;
  background: rgba(30, 30, 30, 0.95);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 15px;
  color: #409eff;
  font-size: 18px;
  z-index: 10;
  border-radius: 8px;
}

@media (max-width: 480px) {
  .image-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .image-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
