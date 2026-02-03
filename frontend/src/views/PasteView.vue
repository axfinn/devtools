<template>
  <div class="tool-container">
    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
      <span>加载中...</span>
    </div>

    <div v-else-if="needPassword" class="password-section">
      <el-card class="password-card">
        <template #header>
          <div class="card-header">
            <el-icon><Lock /></el-icon>
            <span>需要密码访问</span>
          </div>
        </template>
        <el-form @submit.prevent="submitPassword">
          <el-form-item label="访问密码">
            <el-input
              v-model="password"
              type="password"
              placeholder="请输入访问密码"
              show-password
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submitPassword" :loading="verifying">
              验证
            </el-button>
          </el-form-item>
        </el-form>
        <div v-if="passwordError" class="error-text">{{ passwordError }}</div>
      </el-card>
    </div>

    <div v-else-if="error" class="error-section">
      <el-result icon="error" :title="error">
        <template #extra>
          <el-button type="primary" @click="$router.push('/paste')">
            创建新分享
          </el-button>
        </template>
      </el-result>
    </div>

    <div v-else-if="paste" class="paste-content">
      <div class="paste-header">
        <div class="paste-title">
          <h2>{{ paste.title || '无标题' }}</h2>
          <el-tag size="small">{{ paste.language }}</el-tag>
        </div>
        <div class="paste-meta">
          <span>创建于 {{ formatDate(paste.created_at) }}</span>
          <span>·</span>
          <span>访问 {{ paste.views }}/{{ paste.max_views }}</span>
          <span>·</span>
          <span>过期于 {{ formatDate(paste.expires_at) }}</span>
        </div>
      </div>

      <div class="editor-panel" v-if="paste.content">
        <div class="panel-header">
          <span>内容</span>
          <div class="actions">
            <el-button size="small" @click="copyContent">
              <el-icon><CopyDocument /></el-icon>
              复制
            </el-button>
            <el-button size="small" @click="downloadContent">
              <el-icon><Download /></el-icon>
              下载
            </el-button>
          </div>
        </div>
        <pre class="code-content" v-html="highlightedContent"></pre>
      </div>

      <!-- 文件展示 -->
      <div class="file-gallery" v-if="filesList.length > 0">
        <div class="gallery-header">
          <span class="gallery-title">
            <el-icon><Picture /></el-icon>
            附件文件 ({{ filesList.length }})
          </span>
        </div>
        <div class="gallery-grid">
          <div
            class="gallery-item"
            v-for="(file, index) in filesList"
            :key="index"
          >
            <!-- 图片预览 -->
            <div v-if="file.type === 'image'" class="file-preview-img" @click="openPreview(file)">
              <img :src="API_BASE + file.url" alt="图片" />
              <div class="gallery-overlay">
                <el-icon :size="24"><ZoomIn /></el-icon>
              </div>
            </div>
            <!-- 视频预览 -->
            <div v-else-if="file.type === 'video'" class="file-preview-video">
              <video :src="API_BASE + file.url" controls style="width: 100%; max-height: 300px;"></video>
            </div>
            <!-- 音频预览 -->
            <div v-else-if="file.type === 'audio'" class="file-preview-audio">
              <el-icon :size="48"><Headset /></el-icon>
              <audio :src="API_BASE + file.url" controls style="width: 100%; margin-top: 10px;"></audio>
            </div>
            <!-- PDF预览 -->
            <div v-else-if="file.type === 'document' && file.url.endsWith('.pdf')" class="file-preview-doc">
              <el-icon :size="48"><Document /></el-icon>
              <span class="doc-label">PDF文档</span>
              <el-button size="small" type="primary" @click="viewPDF(file)">
                <el-icon><View /></el-icon>
                在线查看
              </el-button>
            </div>
            <!-- 其他文件 -->
            <div v-else class="file-preview-other">
              <el-icon :size="48">
                <Document v-if="file.type === 'document'" />
                <Folder v-else-if="file.type === 'archive'" />
                <Files v-else />
              </el-icon>
              <span class="file-type-label">{{ getFileTypeLabel(file) }}</span>
            </div>
            <div class="file-info">
              <span class="file-name">{{ file.original_name || file.filename }}</span>
              <span class="file-size">{{ (file.size / 1024 / 1024).toFixed(2) }} MB</span>
              <el-button size="small" @click="downloadFile(file)">
                <el-icon><Download /></el-icon>
                下载
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- PDF预览弹窗 -->
      <el-dialog v-model="showPDFPreview" title="PDF预览" width="90%" :close-on-click-modal="false" fullscreen>
        <iframe v-if="pdfUrl" :src="pdfUrl" style="width: 100%; height: 80vh; border: none;"></iframe>
      </el-dialog>

      <!-- 图片预览弹窗 -->
      <div class="image-preview" v-if="previewImage" @click.self="closePreview">
        <div class="preview-container">
          <img :src="API_BASE + previewImage.url" alt="预览" />
          <div class="preview-actions">
            <el-button circle @click="prevImage" :disabled="previewIndex === 0">
              <el-icon><Back /></el-icon>
            </el-button>
            <span class="preview-index">{{ previewIndex + 1 }} / {{ imageFiles.length }}</span>
            <el-button circle @click="nextImage" :disabled="previewIndex === imageFiles.length - 1">
              <el-icon style="transform: rotate(180deg)"><Back /></el-icon>
            </el-button>
            <el-button circle @click="downloadFile(previewImage)">
              <el-icon><Download /></el-icon>
            </el-button>
            <el-button circle type="danger" @click="closePreview">
              ✕
            </el-button>
          </div>
        </div>
      </div>

      <div class="back-section">
        <el-button @click="$router.push('/paste')">
          <el-icon><Back /></el-icon>
          创建新分享
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading, Lock, Back, Picture, Download, ZoomIn, Document, Folder, Files, Headset, View } from '@element-plus/icons-vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import { API_BASE } from '../api'

const route = useRoute()
const loading = ref(true)
const paste = ref(null)
const error = ref('')
const needPassword = ref(false)
const password = ref('')
const passwordError = ref('')
const verifying = ref(false)
const previewImage = ref(null)
const previewIndex = ref(0)
const showPDFPreview = ref(false)
const pdfUrl = ref('')

const fetchPaste = async (pwd = '') => {
  const id = route.params.id
  if (!id) {
    error.value = '无效的分享 ID'
    loading.value = false
    return
  }

  try {
    const url = pwd
      ? `${API_BASE}/api/paste/${id}?password=${encodeURIComponent(pwd)}`
      : `${API_BASE}/api/paste/${id}`

    const response = await fetch(url)
    const data = await response.json()

    if (response.status === 401) {
      if (data.has_password) {
        needPassword.value = true
        passwordError.value = pwd ? '密码错误' : ''
      } else {
        error.value = data.error || '访问被拒绝'
      }
    } else if (!response.ok) {
      error.value = data.error || '加载失败'
    } else {
      paste.value = data
      needPassword.value = false
    }
  } catch (e) {
    error.value = '网络错误，请稍后重试'
  } finally {
    loading.value = false
    verifying.value = false
  }
}

const submitPassword = async () => {
  if (!password.value) {
    passwordError.value = '请输入密码'
    return
  }
  verifying.value = true
  passwordError.value = ''
  await fetchPaste(password.value)
}

const highlightedContent = computed(() => {
  if (!paste.value) return ''
  const lang = paste.value.language
  const content = paste.value.content

  if (hljs.getLanguage(lang)) {
    try {
      return hljs.highlight(content, { language: lang }).value
    } catch (e) {
      return escapeHtml(content)
    }
  }
  return escapeHtml(content)
})

const escapeHtml = (text) => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const formatDate = (dateStr) => {
  return new Date(dateStr).toLocaleString('zh-CN')
}

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(paste.value.content)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const downloadContent = () => {
  const blob = new Blob([paste.value.content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${paste.value.title || paste.value.id}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

// 解析文件列表
const filesList = computed(() => {
  if (!paste.value || !paste.value.files) return []
  // 后端已经返回数组，直接使用
  if (Array.isArray(paste.value.files)) {
    return paste.value.files
  }
  // 兼容旧数据（字符串格式）
  try {
    return JSON.parse(paste.value.files)
  } catch (e) {
    return []
  }
})

// 只获取图片类型的文件用于预览
const imageFiles = computed(() => {
  return filesList.value.filter(f => f.type === 'image')
})

// 图片预览
const openPreview = (file) => {
  const images = imageFiles.value
  const index = images.findIndex(f => f.filename === file.filename)
  if (index !== -1) {
    previewIndex.value = index
    previewImage.value = images[index]
  }
}

const closePreview = () => {
  previewImage.value = null
}

const prevImage = () => {
  if (previewIndex.value > 0) {
    previewIndex.value--
    previewImage.value = imageFiles.value[previewIndex.value]
  }
}

const nextImage = () => {
  if (previewIndex.value < imageFiles.value.length - 1) {
    previewIndex.value++
    previewImage.value = imageFiles.value[previewIndex.value]
  }
}

// 下载文件
const downloadFile = async (file) => {
  try {
    const response = await fetch(API_BASE + file.url)
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = file.original_name || file.filename
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('下载成功')
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

// 获取文件类型标签
const getFileTypeLabel = (file) => {
  const filename = file.original_name || file.filename
  const ext = filename.split('.').pop().toUpperCase()

  if (file.type === 'document') {
    return `${ext} 文档`
  } else if (file.type === 'archive') {
    return `${ext} 压缩包`
  } else {
    return `${ext} 文件`
  }
}

// 在线查看PDF
const viewPDF = (file) => {
  pdfUrl.value = API_BASE + file.url
  showPDFPreview.value = true
}

onMounted(() => {
  fetchPaste()
})
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-height: 400px;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 15px;
  min-height: 300px;
  color: var(--text-secondary);
}


.password-section {
  display: flex;
  justify-content: center;
  padding-top: 50px;
}

.password-card {
  width: 400px;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
}


.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-primary);
}


.error-text {
  color: #f56c6c;
  margin-top: 10px;
}

.error-section {
  display: flex;
  justify-content: center;
  padding-top: 50px;
}

.paste-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.paste-header {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  padding: 20px;
  border-radius: var(--radius-md);
}


.paste-title {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 10px;
}

.paste-title h2 {
  margin: 0;
  color: var(--text-primary);
}


.paste-meta {
  display: flex;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 14px;
}


.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
}


.panel-header {
  padding: 10px 15px;
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #e0e0e0;
}


.actions {
  display: flex;
  gap: 10px;
}

.code-content {
  margin: 0;
  padding: 20px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-x: auto;
  min-height: 200px;
  max-height: 600px;
  overflow-y: auto;
}


.back-section {
  display: flex;
  justify-content: center;
}

/* 文件画廊 */
.file-gallery {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  padding: 20px;
}


.gallery-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.gallery-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 16px;
}


.gallery-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 15px;
}

.gallery-item {
  border-radius: var(--radius-md);
  overflow: hidden;
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-base);
  display: flex;
  flex-direction: column;
}

.file-preview-img {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  cursor: pointer;
  background-color: #000;
}

.file-preview-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.file-preview-img:hover img {
  transform: scale(1.05);
}

.file-preview-video {
  width: 100%;
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-preview-audio {
  width: 100%;
  padding: 20px;
  background-color: #1a1a1a;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #67c23a;
}

.file-preview-doc {
  width: 100%;
  padding: 30px 20px;
  background-color: #1a1a1a;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #e74c3c;
}

.doc-label {
  color: var(--text-secondary);
  font-size: 14px;
}

.file-preview-other {
  width: 100%;
  padding: 30px 20px;
  background-color: #1a1a1a;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #9b59b6;
}

.file-type-label {
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: bold;
}

.file-info {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background-color: var(--bg-secondary);
}

.file-name {
  color: var(--text-primary);
  font-size: 14px;
  word-break: break-all;
}

.file-size {
  color: var(--text-secondary);
  font-size: 12px;
}

.gallery-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  opacity: 0;
  transition: opacity 0.2s;
}

.file-preview-img:hover .gallery-overlay {
  opacity: 1;
}

/* 图片预览弹窗 */
.image-preview {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.9);
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.preview-container {
  max-width: 90vw;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.preview-container img {
  max-width: 100%;
  max-height: calc(90vh - 80px);
  object-fit: contain;
  border-radius: var(--radius-md);
}

.preview-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}

.preview-index {
  color: var(--text-primary);
  font-size: 14px;
  min-width: 60px;
  text-align: center;
}

@media (max-width: 768px) {
  .gallery-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .preview-actions {
    flex-wrap: wrap;
    justify-content: center;
  }
}
</style>
