<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>共享粘贴板</h2>
      <div class="info-text">
        支持文本、图片、视频分享 - 自动压缩优化
      </div>
    </div>

    <!-- 简洁模式 -->
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
          placeholder="粘贴或输入内容,支持拖拽图片/视频..."
          spellcheck="false"
        ></textarea>
        <div v-if="isDragging" class="drop-overlay">
          <el-icon :size="48"><Upload /></el-icon>
          <span>拖放文件到此处</span>
        </div>
      </div>

      <!-- 文件上传区域 -->
      <div class="file-section">
        <div class="file-header">
          <span class="file-title">
            <el-icon><Folder /></el-icon>
            文件 ({{ files.length }}/{{ MAX_FILES }})
          </span>
          <span class="size-info" v-if="files.length > 0">
            总大小: {{ (totalSize / 1024 / 1024).toFixed(2) }} MB
          </span>
        </div>

        <div class="file-grid" v-if="files.length > 0">
          <div class="file-item" v-for="(file, index) in files" :key="index">
            <div class="file-preview">
              <img v-if="file.type === 'image'" :src="file.preview" alt="预览" />
              <video v-else-if="file.type === 'video'" :src="file.preview" controls></video>
            </div>
            <div class="file-info">
              <span class="file-name">{{ file.name }}</span>
              <span class="file-size">{{ (file.size / 1024 / 1024).toFixed(2) }} MB</span>
              <el-tag v-if="file.compressed" type="success" size="small">已压缩</el-tag>
              <el-tag v-if="file.compressing" type="warning" size="small">压缩中...</el-tag>
            </div>
            <div class="file-actions">
              <el-button v-if="!file.compressed && canCompress(file)" size="small" @click="compressFile(index)" :loading="file.compressing">
                压缩
              </el-button>
              <el-button type="danger" size="small" @click="removeFile(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-add" v-if="canAddMore" @click="selectFiles">
            <el-icon :size="24"><Plus /></el-icon>
            <span>添加文件</span>
          </div>
        </div>

        <div class="file-upload" v-else @click="selectFiles">
          <el-icon :size="32"><Upload /></el-icon>
          <span>点击上传文件或直接拖拽</span>
          <span class="upload-hint">支持图片和视频 (最大 200MB)</span>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*,video/*"
          multiple
          style="display: none"
          @change="onFileChange"
        />
      </div>

      <div class="quick-actions">
        <el-button type="primary" size="large" @click="quickCreate" :loading="creating" :disabled="!content.trim() && files.length === 0">
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
            <el-option label="Python" value="python" />
            <el-option label="Go" value="go" />
            <el-option label="Markdown" value="markdown" />
          </el-select>
          <el-button size="small" text @click="showAdvanced = false">简洁模式</el-button>
        </div>
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="在此输入要分享的内容..."
          spellcheck="false"
        ></textarea>
        <div v-if="isDragging" class="drop-overlay">
          <el-icon :size="48"><Upload /></el-icon>
          <span>拖放文件到此处</span>
        </div>
      </div>

      <!-- 高级模式文件区域 (同上) -->
      <div class="file-section">
        <div class="file-header">
          <span class="file-title">
            <el-icon><Folder /></el-icon>
            文件 ({{ files.length }}/{{ MAX_FILES }})
          </span>
          <span class="size-info" v-if="files.length > 0">
            总大小: {{ (totalSize / 1024 / 1024).toFixed(2) }} MB
          </span>
        </div>

        <div class="file-grid" v-if="files.length > 0">
          <div class="file-item" v-for="(file, index) in files" :key="index">
            <div class="file-preview">
              <img v-if="file.type === 'image'" :src="file.preview" alt="预览" />
              <video v-else-if="file.type === 'video'" :src="file.preview" controls></video>
            </div>
            <div class="file-info">
              <span class="file-name">{{ file.name }}</span>
              <span class="file-size">{{ (file.size / 1024 / 1024).toFixed(2) }} MB</span>
              <el-tag v-if="file.compressed" type="success" size="small">已压缩</el-tag>
              <el-tag v-if="file.compressing" type="warning" size="small">压缩中...</el-tag>
            </div>
            <div class="file-actions">
              <el-button v-if="!file.compressed && canCompress(file)" size="small" @click="compressFile(index)" :loading="file.compressing">
                压缩
              </el-button>
              <el-button type="danger" size="small" @click="removeFile(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-add" v-if="canAddMore" @click="selectFiles">
            <el-icon :size="24"><Plus /></el-icon>
            <span>添加文件</span>
          </div>
        </div>

        <div class="file-upload" v-else @click="selectFiles">
          <el-icon :size="32"><Upload /></el-icon>
          <span>点击上传文件或直接拖拽</span>
          <span class="upload-hint">支持图片和视频 (最大 200MB)</span>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*,video/*"
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
          <el-input-number v-model="maxViews" :min="1" :max="hasVideo ? 10 : 1000" />
          <span v-if="hasVideo" class="hint-text">(视频默认限制10次)</span>
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
        <div class="option-item" v-if="hasVideo">
          <span class="option-label">管理员密码</span>
          <el-input
            v-model="adminPassword"
            type="password"
            placeholder="可设置更多次数"
            style="width: 150px"
            show-password
          />
        </div>
        <el-button type="primary" size="large" @click="createPaste" :loading="creating" :disabled="!content.trim() && files.length === 0">
          创建分享
        </el-button>
      </div>
    </div>

    <!-- 分享结果 -->
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
          <span>最大访问: {{ createdMaxViews }} 次</span>
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
        <li>支持文本、图片、视频分享</li>
        <li>大文件自动压缩优化</li>
        <li>视频默认最多10次访问（防止滥用）</li>
        <li>管理员可设置更多访问次数或永久访问</li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Share, CircleCheck, CopyDocument, Link, Plus, Folder, Delete, Upload } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import { API_BASE } from '../api'
import { FFmpeg } from '@ffmpeg/ffmpeg'
import { fetchFile, toBlobURL } from '@ffmpeg/util'

const content = ref('')
const title = ref('')
const language = ref('text')
const expiresIn = ref(24)
const maxViews = ref(0)
const password = ref('')
const adminPassword = ref('')
const creating = ref(false)
const showResult = ref(false)
const showAdvanced = ref(false)
const createdId = ref('')
const createdExpires = ref('')
const createdMaxViews = ref(0)
const shareUrl = ref('')
const errorMsg = ref('')
const qrCanvas = ref(null)
const files = ref([]) // [{ file: File, preview: string, type: 'image'|'video', name: string, size: number, compressed: boolean, compressing: boolean, uploadedId: string }]
const fileInput = ref(null)
const isDragging = ref(false)

const MAX_FILES = 10
const MAX_FILE_SIZE = 200 * 1024 * 1024 // 200MB

// FFmpeg 实例 (懒加载)
let ffmpegInstance = null
let ffmpegLoaded = false

const totalSize = computed(() => {
  return files.value.reduce((sum, file) => sum + file.size, 0)
})

const canAddMore = computed(() => files.value.length < MAX_FILES)

const hasVideo = computed(() => files.value.some(f => f.type === 'video'))

// 快捷创建
const quickCreate = async () => {
  await createPaste()
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
  const droppedFiles = e.dataTransfer?.files
  if (droppedFiles) {
    for (const file of droppedFiles) {
      await addFile(file)
    }
  }
}

// 选择文件
const selectFiles = () => {
  fileInput.value?.click()
}

// 文件选择变化
const onFileChange = async (e) => {
  const selectedFiles = e.target.files
  for (const file of selectedFiles) {
    await addFile(file)
  }
  e.target.value = ''
}

// 添加文件
const addFile = async (file) => {
  if (files.value.length >= MAX_FILES) {
    ElMessage.warning(`最多只能上传 ${MAX_FILES} 个文件`)
    return
  }

  if (file.size > MAX_FILE_SIZE) {
    ElMessage.warning(`文件 ${file.name} 超过 200MB 限制`)
    return
  }

  // 检测文件类型
  let fileType = 'unknown'
  if (file.type.startsWith('image/')) {
    fileType = 'image'
  } else if (file.type.startsWith('video/')) {
    fileType = 'video'
  } else {
    ElMessage.warning('只支持图片和视频文件')
    return
  }

  // 创建预览
  const preview = URL.createObjectURL(file)

  files.value.push({
    file,
    preview,
    type: fileType,
    name: file.name,
    size: file.size,
    compressed: false,
    compressing: false,
    uploadedId: null
  })
}

// 删除文件
const removeFile = (index) => {
  const file = files.value[index]
  if (file.preview) {
    URL.revokeObjectURL(file.preview)
  }
  files.value.splice(index, 1)
}

// 是否可以压缩
const canCompress = (file) => {
  // 图片大于 1MB 或视频大于 10MB 可以压缩
  if (file.type === 'image' && file.size > 1024 * 1024) {
    return true
  }
  if (file.type === 'video' && file.size > 10 * 1024 * 1024) {
    return true
  }
  return false
}

// 压缩文件
const compressFile = async (index) => {
  const fileObj = files.value[index]
  if (fileObj.compressing) return

  fileObj.compressing = true

  try {
    if (fileObj.type === 'image') {
      await compressImage(index)
    } else if (fileObj.type === 'video') {
      await compressVideo(index)
    }
  } catch (err) {
    ElMessage.error(`压缩失败: ${err.message}`)
    fileObj.compressing = false
  }
}

// 压缩图片 (Canvas)
const compressImage = async (index) => {
  const fileObj = files.value[index]

  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      const canvas = document.createElement('canvas')
      let width = img.width
      let height = img.height

      // 限制最大尺寸为 1920x1080
      const maxWidth = 1920
      const maxHeight = 1080

      if (width > maxWidth || height > maxHeight) {
        const ratio = Math.min(maxWidth / width, maxHeight / height)
        width *= ratio
        height *= ratio
      }

      canvas.width = width
      canvas.height = height

      const ctx = canvas.getContext('2d')
      ctx.drawImage(img, 0, 0, width, height)

      canvas.toBlob(
        (blob) => {
          if (!blob) {
            reject(new Error('压缩失败'))
            return
          }

          const compressedFile = new File([blob], fileObj.name, { type: 'image/jpeg' })

          // 更新文件对象
          URL.revokeObjectURL(fileObj.preview)
          fileObj.file = compressedFile
          fileObj.preview = URL.createObjectURL(compressedFile)
          fileObj.size = compressedFile.size
          fileObj.compressed = true
          fileObj.compressing = false

          ElMessage.success(`图片已压缩: ${(fileObj.size / 1024 / 1024).toFixed(2)} MB`)
          resolve()
        },
        'image/jpeg',
        0.8 // 质量 80%
      )
    }

    img.onerror = () => {
      reject(new Error('图片加载失败'))
    }

    img.src = fileObj.preview
  })
}

// 初始化 FFmpeg
const initFFmpeg = async () => {
  if (ffmpegLoaded) return ffmpegInstance

  try {
    const ffmpeg = new FFmpeg()

    // 加载 FFmpeg 核心
    const baseURL = 'https://unpkg.com/@ffmpeg/core@0.12.6/dist/umd'
    await ffmpeg.load({
      coreURL: await toBlobURL(`${baseURL}/ffmpeg-core.js`, 'text/javascript'),
      wasmURL: await toBlobURL(`${baseURL}/ffmpeg-core.wasm`, 'application/wasm'),
    })

    ffmpegInstance = ffmpeg
    ffmpegLoaded = true
    return ffmpeg
  } catch (err) {
    console.error('FFmpeg 加载失败:', err)
    throw new Error('FFmpeg 初始化失败')
  }
}

// 压缩视频 (FFmpeg.wasm)
const compressVideo = async (index) => {
  const fileObj = files.value[index]

  try {
    ElMessage.info('正在初始化视频压缩工具...')
    const ffmpeg = await initFFmpeg()

    ElMessage.info('正在压缩视频，请稍候...')

    // 读取文件
    await ffmpeg.writeFile('input.mp4', await fetchFile(fileObj.file))

    // 压缩视频: 降低分辨率和比特率
    await ffmpeg.exec([
      '-i', 'input.mp4',
      '-vf', 'scale=-2:720', // 720p
      '-b:v', '1M', // 1 Mbps
      '-c:v', 'libx264',
      '-preset', 'fast',
      '-c:a', 'aac',
      '-b:a', '128k',
      'output.mp4'
    ])

    // 读取输出
    const data = await ffmpeg.readFile('output.mp4')
    const compressedBlob = new Blob([data.buffer], { type: 'video/mp4' })
    const compressedFile = new File([compressedBlob], fileObj.name.replace(/\.[^.]+$/, '.mp4'), { type: 'video/mp4' })

    // 清理 FFmpeg 文件
    await ffmpeg.deleteFile('input.mp4')
    await ffmpeg.deleteFile('output.mp4')

    // 更新文件对象
    URL.revokeObjectURL(fileObj.preview)
    fileObj.file = compressedFile
    fileObj.preview = URL.createObjectURL(compressedFile)
    fileObj.size = compressedFile.size
    fileObj.compressed = true
    fileObj.compressing = false

    ElMessage.success(`视频已压缩: ${(fileObj.size / 1024 / 1024).toFixed(2)} MB`)
  } catch (err) {
    console.error('视频压缩失败:', err)
    fileObj.compressing = false
    throw err
  }
}

// 上传文件到服务器
const uploadFile = async (fileObj) => {
  const formData = new FormData()
  formData.append('file', fileObj.file)

  try {
    const response = await fetch(`${API_BASE}/api/paste/upload`, {
      method: 'POST',
      body: formData
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '上传失败')
    }

    return data.id
  } catch (err) {
    throw new Error(`文件 ${fileObj.name} 上传失败: ${err.message}`)
  }
}

// 创建分享
const createPaste = async () => {
  if (!content.value.trim() && files.value.length === 0) {
    errorMsg.value = '请输入内容或上传文件'
    return
  }

  creating.value = true
  errorMsg.value = ''

  try {
    // 1. 上传所有文件
    const fileIDs = []
    for (const fileObj of files.value) {
      if (!fileObj.uploadedId) {
        ElMessage.info(`正在上传 ${fileObj.name}...`)
        const id = await uploadFile(fileObj)
        fileObj.uploadedId = id
        fileIDs.push(id)
      } else {
        fileIDs.push(fileObj.uploadedId)
      }
    }

    // 2. 创建粘贴板
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
        file_ids: fileIDs,
        admin_password: adminPassword.value
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }

    createdId.value = data.id
    createdExpires.value = new Date(data.expires_at).toLocaleString('zh-CN')
    createdMaxViews.value = data.max_views
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
  adminPassword.value = ''
  showResult.value = false
  createdId.value = ''

  // 清理文件
  for (const fileObj of files.value) {
    if (fileObj.preview) {
      URL.revokeObjectURL(fileObj.preview)
    }
  }
  files.value = []
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
  color: var(--text-primary);
}

.info-text {
  color: #67c23a;
  font-size: 14px;
}

/* 简洁模式 */
.quick-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.quick-editor {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  min-height: 200px;
  display: flex;
  position: relative;
}

.quick-editor .code-editor {
  flex: 1;
  padding: 20px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  border: none;
  resize: none;
  font-family: var(--font-family-mono);
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
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  min-height: 200px;
  position: relative;
}

.panel-header {
  padding: 10px 15px;
  background-color: var(--bg-secondary);
  display: flex;
  gap: 10px;
  align-items: center;
  border-bottom: 1px solid var(--border-base);
}

.code-editor {
  flex: 1;
  width: 100%;
  padding: 15px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  border: none;
  resize: none;
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.5;
  outline: none;
}

.options-row {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  padding: 15px;
  border-radius: var(--radius-md);
}

.option-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.option-label {
  color: var(--text-secondary);
  font-size: 14px;
}

.hint-text {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* 文件上传区域 */
.file-section {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  padding: 15px;
}

.file-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.file-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 14px;
}

.size-info {
  color: #67c23a;
  font-size: 13px;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.file-item {
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  background-color: var(--bg-secondary);
  display: flex;
  flex-direction: column;
}

.file-preview {
  width: 100%;
  height: 150px;
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-preview img,
.file-preview video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.file-info {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.file-name {
  font-size: 13px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  font-size: 12px;
  color: var(--text-tertiary);
}

.file-actions {
  padding: 10px;
  display: flex;
  gap: 5px;
  border-top: 1px solid var(--border-base);
}

.file-add {
  border: 2px dashed #d0d0d0;
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s;
  min-height: 150px;
}

.file-add:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.file-upload {
  border: 2px dashed #d0d0d0;
  border-radius: var(--radius-md);
  padding: 30px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s;
}

.file-upload:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.upload-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* 拖拽状态 */
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
  color: var(--color-primary);
  font-size: 18px;
  z-index: 10;
  border-radius: var(--radius-md);
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
  border-radius: var(--radius-md);
  padding: 15px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.url-display {
  font-family: var(--font-family-mono);
  font-size: 14px;
  color: #67c23a;
  word-break: break-all;
  padding: 10px;
  background: #1a1a1a;
  border-radius: var(--radius-sm);
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
  color: var(--text-secondary);
  font-size: 14px;
}

.qr-code {
  border-radius: var(--radius-md);
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
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-base);
  padding: 20px;
  border-radius: var(--radius-md);
}

.tips-section h4 {
  margin: 0 0 10px 0;
  color: var(--text-primary);
}

.tips-section ul {
  margin: 0;
  padding-left: 20px;
  color: var(--text-secondary);
  line-height: 1.8;
}

.error-msg {
  margin-top: 10px;
}
</style>
