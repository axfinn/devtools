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
          <div class="panel-title-left">
            <span>{{ paste.content_type === 'markdown' ? 'Markdown 文档' : '代码内容' }}</span>
            <el-tag v-if="paste.language" size="small" type="info">{{ paste.language }}</el-tag>
          </div>
          <div class="actions">
            <el-button v-if="paste.content_type === 'code'" size="small" @click="toggleLineNumbers">
              <el-icon><List /></el-icon>
              {{ showLineNumbers ? '隐藏行号' : '显示行号' }}
            </el-button>
            <el-button v-if="paste.content_type === 'code'" size="small" @click="toggleSearch">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
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

        <!-- 搜索框 -->
        <div v-if="showSearchBox && paste.content_type === 'code'" class="search-box">
          <el-input
            v-model="searchText"
            placeholder="搜索代码... (Enter 搜索, Esc 关闭)"
            size="small"
            @keyup.enter="doSearch"
            @keyup.escape="closeSearch"
            clearable
          >
            <template #prepend>
              <el-button @click="doSearch"><Search /></el-button>
            </template>
            <template #append>
              <span>{{ searchResults.length }} / {{ totalMatches }}</span>
            </template>
          </el-input>
        </div>

        <!-- 代码分析信息 -->
        <div v-if="paste.content_type === 'code' && codeStats" class="code-stats">
          <el-tag size="small">行数: {{ codeStats.lines }}</el-tag>
          <el-tag size="small" type="success">代码: {{ codeStats.code_lines }}</el-tag>
          <el-tag size="small" type="info">注释: {{ codeStats.comment_lines }}</el-tag>
          <el-tag size="small" type="warning">函数: {{ codeStats.functions }}</el-tag>
        </div>

        <!-- Markdown 渲染 -->
        <div v-if="paste.content_type === 'markdown'" class="markdown-content" v-html="highlightedContent"></div>
        <!-- 代码高亮显示 -->
        <div v-else class="code-wrapper">
          <!-- 带行号显示 -->
          <div v-if="showLineNumbers" class="code-with-line-numbers">
            <div class="line-numbers">
              <span v-for="n in lineCount" :key="n" :class="{ 'highlight-line': isHighlightedLine(n) }">{{ n }}</span>
            </div>
            <pre class="code-content" v-html="highlightedContent"></pre>
          </div>
          <!-- 不带行号 -->
          <pre v-else class="code-content" v-html="highlightedContent"></pre>
        </div>
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
            <div v-else-if="file.type === 'video'" class="file-preview-video" @click="openVideoPreview(file)">
              <video :src="API_BASE + file.url" controls style="width: 100%; max-height: 300px;"></video>
              <div class="gallery-overlay">
                <el-icon :size="24"><VideoPlay /></el-icon>
              </div>
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
            <!-- 代码文件预览 -->
            <div v-else-if="isCodeFile(file)" class="file-preview-code" @click="previewCodeFile(file)">
              <el-icon :size="48"><Document /></el-icon>
              <span class="doc-label">{{ getCodeFileLanguage(file) }} 代码</span>
              <el-button size="small" type="primary">
                <el-icon><View /></el-icon>
                预览
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

      <!-- 代码文件预览弹窗 -->
      <el-dialog
        v-model="showCodePreview"
        :title="(currentCodeFile?.original_name || currentCodeFile?.filename || '代码预览') + ' - ' + codeFileLanguage"
        width="80%"
        :close-on-click-modal="true"
        destroy-on-close
      >
        <div class="code-preview-dialog">
          <div class="code-preview-header">
            <el-tag size="small">{{ codeFileLanguage }}</el-tag>
            <el-button size="small" @click="copyCodeContent">
              <el-icon><CopyDocument /></el-icon>
              复制
            </el-button>
            <el-button size="small" @click="downloadCodeFile">
              <el-icon><Download /></el-icon>
              下载
            </el-button>
          </div>
          <pre class="code-preview-content"><code v-html="highlightedCodeContent"></code></pre>
        </div>
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

      <!-- 视频预览弹窗 -->
      <el-dialog
        v-model="showVideoPreview"
        :title="currentVideoFile?.original_name || currentVideoFile?.filename || '视频预览'"
        width="80%"
        :close-on-click-modal="true"
        destroy-on-close
        @close="closeVideoPreview"
      >
        <div class="video-fullscreen-preview" v-if="currentVideoFile">
          <video
            :src="API_BASE + currentVideoFile.url"
            controls
            autoplay
            class="preview-video-full"
          ></video>
        </div>
      </el-dialog>

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
import { Loading, Lock, Back, Picture, Download, ZoomIn, Document, Folder, Files, Headset, View, VideoPlay, Search, List } from '@element-plus/icons-vue'
import hljs from 'highlight.js'
import MarkdownIt from 'markdown-it'
import footnote from 'markdown-it-footnote'
import mark from 'markdown-it-mark'
import taskLists from 'markdown-it-task-lists'
import 'highlight.js/styles/github-dark.css'
import { API_BASE } from '../api'

// 初始化 markdown-it
const md = new MarkdownIt({
  html: false,        // 禁用 HTML 标签输入（安全）
  linkify: true,     // 自动转换链接
  typographer: true, // 优化排版
  highlight: function (str, lang) {
    // 使用 highlight.js 进行代码高亮
    if (lang && hljs.getLanguage(lang)) {
      try {
        return '<pre class="hljs"><code>' +
               hljs.highlight(str, { language: lang, ignoreIllegals: true }).value +
               '</code></pre>'
      } catch (__) {}
    }
    return '<pre class="hljs"><code>' + md.utils.escapeHtml(str) + '</code></pre>'
  }
})
  .use(footnote)
  .use(mark)
  .use(taskLists, { enabled: true, label: true })

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
const showVideoPreview = ref(false)
const currentVideoFile = ref(null)

// 代码预览增强功能
const showLineNumbers = ref(true)
const showSearchBox = ref(false)
const searchText = ref('')
const searchResults = ref([])
const currentSearchIndex = ref(0)
const codeStats = ref(null)

// 代码文件预览
const showCodePreview = ref(false)
const codeFileContent = ref('')
const codeFileLanguage = ref('')
const currentCodeFile = ref(null)

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
  const contentType = paste.value.content_type || 'text'
  const lang = paste.value.language
  const content = paste.value.content

  // 如果是 Markdown 类型，渲染 Markdown
  if (contentType === 'markdown') {
    return md.render(content)
  }

  // 否则进行代码高亮
  if (lang && hljs.getLanguage(lang)) {
    try {
      return hljs.highlight(content, { language: lang, ignoreIllegals: true }).value
    } catch (e) {
      return escapeHtml(content)
    }
  }
  return escapeHtml(content)
})

// 获取带行号的内容
const contentWithLineNumbers = computed(() => {
  if (!paste.value) return []
  const content = paste.value.content || ''
  return content.split('\n')
})

const escapeHtml = (text) => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const formatDate = (dateStr) => {
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 行数统计
const lineCount = computed(() => {
  if (!paste.value) return 0
  const content = paste.value.content || ''
  return content.split('\n').length
})

// 切换行号显示
const toggleLineNumbers = () => {
  showLineNumbers.value = !showLineNumbers.value
}

// 切换搜索框
const toggleSearch = () => {
  showSearchBox.value = !showSearchBox.value
  if (!showSearchBox.value) {
    searchText.value = ''
    searchResults.value = []
  }
}

// 关闭搜索
const closeSearch = () => {
  showSearchBox.value = false
  searchText.value = ''
  searchResults.value = []
}

// 执行搜索
const doSearch = () => {
  if (!searchText.value || !paste.value) {
    searchResults.value = []
    return
  }

  const content = paste.value.content
  const regex = new RegExp(escapeRegExp(searchText.value), 'gi')
  const results = []
  let match
  while ((match = regex.exec(content)) !== null) {
    results.push(match.index)
  }
  searchResults.value = results
  currentSearchIndex.value = 0
  ElMessage.success(`找到 ${results.length} 个匹配`)
}

// 计算总匹配数
const totalMatches = computed(() => searchResults.value.length)

// 检查是否为高亮行
const isHighlightedLine = (lineNum) => {
  if (searchResults.value.length === 0 || !paste.value) return false

  // 计算这一行的起始位置
  const lines = paste.value.content.split('\n')
  let pos = 0
  for (let i = 0; i < lineNum - 1 && i < lines.length; i++) {
    pos += lines[i].length + 1
  }

  // 检查这一行是否包含搜索结果
  const lineContent = lines[lineNum - 1] || ''
  return lineContent.toLowerCase().includes(searchText.value.toLowerCase())
}

// 转义正则表达式特殊字符
const escapeRegExp = (string) => {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// 获取代码统计信息
const fetchCodeStats = async () => {
  if (!paste.value || paste.value.content_type !== 'code') return

  try {
    const response = await fetch(`${API_BASE}/api/paste/analyze`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        content: paste.value.content,
        language: paste.value.language || ''
      })
    })

    if (response.ok) {
      const data = await response.json()
      codeStats.value = data
    }
  } catch (e) {
    // 静默失败，不影响主功能
    console.log('获取代码统计失败', e)
  }
}

// 检查是否为代码文件
const isCodeFile = (file) => {
  const filename = (file.original_name || file.filename || '').toLowerCase()
  const codeExtensions = ['.js', '.ts', '.jsx', '.tsx', '.json', '.html', '.css', '.xml',
    '.py', '.go', '.java', '.c', '.cpp', '.h', '.hpp', '.cs', '.rb', '.php',
    '.swift', '.kt', '.scala', '.rs', '.sh', '.bash', '.sql', '.yaml', '.yml',
    '.md', '.txt', '.log', '.vue', '.jsx', '.tsx']
  return codeExtensions.some(ext => filename.endsWith(ext))
}

// 获取代码文件语言
const getCodeFileLanguage = (file) => {
  const filename = (file.original_name || file.filename || '').toLowerCase()
  const ext = filename.split('.').pop()
  const langMap = {
    'js': 'JavaScript', 'jsx': 'JavaScript', 'ts': 'TypeScript', 'tsx': 'TypeScript',
    'py': 'Python', 'go': 'Go', 'java': 'Java', 'c': 'C', 'cpp': 'C++', 'h': 'C',
    'hpp': 'C++', 'cs': 'C#', 'rb': 'Ruby', 'php': 'PHP', 'swift': 'Swift',
    'kt': 'Kotlin', 'scala': 'Scala', 'rs': 'Rust', 'sh': 'Shell', 'bash': 'Shell',
    'sql': 'SQL', 'html': 'HTML', 'css': 'CSS', 'xml': 'XML', 'json': 'JSON',
    'yaml': 'YAML', 'yml': 'YAML', 'md': 'Markdown', 'vue': 'Vue', 'txt': 'Text'
  }
  return langMap[ext] || ext.toUpperCase()
}

// 预览代码文件
const previewCodeFile = async (file) => {
  currentCodeFile.value = file

  try {
    // 直接获取文件内容
    const response = await fetch(API_BASE + file.url)
    const text = await response.text()

    // 限制预览内容大小 (最大 100KB)
    if (text.length > 100 * 1024) {
      ElMessage.warning('文件过大，仅显示前 100KB')
      codeFileContent.value = text.substring(0, 100 * 1024)
    } else {
      codeFileContent.value = text
    }

    codeFileLanguage.value = getCodeFileLanguage(file)
    showCodePreview.value = true
  } catch (e) {
    ElMessage.error('无法加载文件内容')
  }
}

// 关闭代码预览
const closeCodePreview = () => {
  showCodePreview.value = false
  codeFileContent.value = ''
  currentCodeFile.value = null
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

// 代码文件预览的高亮内容
const highlightedCodeContent = computed(() => {
  if (!codeFileContent.value) return ''

  const lang = codeFileLanguage.value.toLowerCase()
  if (lang && hljs.getLanguage(lang)) {
    try {
      return hljs.highlight(codeFileContent.value, { language: lang, ignoreIllegals: true }).value
    } catch (e) {
      return escapeHtml(codeFileContent.value)
    }
  }
  return escapeHtml(codeFileContent.value)
})

// 复制代码内容
const copyCodeContent = async () => {
  try {
    await navigator.clipboard.writeText(codeFileContent.value)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

// 下载代码文件
const downloadCodeFile = () => {
  const blob = new Blob([codeFileContent.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = currentCodeFile.value?.original_name || currentCodeFile.value?.filename || 'code.txt'
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

// 视频预览
const openVideoPreview = (file) => {
  currentVideoFile.value = file
  showVideoPreview.value = true
}

const closeVideoPreview = () => {
  showVideoPreview.value = false
  currentVideoFile.value = null
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
  fetchPaste().then(() => {
    // 获取代码统计信息
    if (paste.value && paste.value.content_type === 'code') {
      fetchCodeStats()
    }
  })
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
  position: relative;
  width: 100%;
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
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

  .paste-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .paste-meta {
    flex-direction: column;
    gap: 5px;
  }

  .password-card {
    width: 90%;
    max-width: 400px;
  }

  .panel-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .actions {
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 480px) {
  .gallery-grid {
    grid-template-columns: 1fr;
  }

  .file-info {
    padding: 8px;
  }

  .file-name {
    font-size: 12px;
  }

  .preview-container {
    max-width: 95vw;
  }

  .preview-actions .el-button {
    padding: 8px;
  }

  .paste-header {
    padding: 15px;
  }

  .code-content {
    padding: 15px;
    font-size: 12px;
  }
}

/* 视频全屏预览 */
.video-fullscreen-preview {
  display: flex;
  justify-content: center;
  background: #000;
  border-radius: 8px;
}

.preview-video-full {
  width: 100%;
  max-height: 70vh;
  background: #000;
}

/* 代码预览弹窗 */
.code-preview-dialog {
  display: flex;
  flex-direction: column;
  height: 70vh;
}

.code-preview-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 15px;
  border-bottom: 1px solid var(--border-base);
  margin-bottom: 15px;
}

.code-preview-content {
  flex: 1;
  overflow: auto;
  background-color: #1e1e1e;
  color: #d4d4d4;
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.6;
  padding: 15px;
  border-radius: 4px;
  margin: 0;
}

.code-preview-content code {
  font-family: var(--font-family-mono);
  background: transparent;
}

/* 代码高亮样式 */
.code-wrapper {
  position: relative;
  overflow-x: auto;
}

.panel-title-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-box {
  padding: 10px 15px;
  background-color: var(--bg-secondary);
  border-bottom: 1px solid var(--border-base);
}

.code-stats {
  display: flex;
  gap: 10px;
  padding: 10px 15px;
  background-color: var(--bg-secondary);
  border-bottom: 1px solid var(--border-base);
  flex-wrap: wrap;
}

.code-with-line-numbers {
  display: flex;
  max-height: 600px;
  overflow-y: auto;
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 20px 10px;
  background-color: #2d2d2d;
  color: #858585;
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.6;
  text-align: right;
  user-select: none;
  min-width: 40px;
  border-right: 1px solid #404040;
}

.line-numbers span {
  display: block;
}

.line-numbers .highlight-line {
  background-color: rgba(255, 235, 59, 0.3);
  color: #fff;
}

.code-content {
  margin: 0;
  padding: 20px;
  background-color: #1e1e1e;
  color: #d4d4d4;
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

.code-content pre.hljs {
  background: #1e1e1e;
  padding: 0;
  margin: 0;
}

.code-content code {
  font-family: var(--font-family-mono);
  background: transparent;
  padding: 0;
}

/* Markdown 样式 */
.markdown-content {
  padding: 20px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  line-height: 1.7;
  min-height: 200px;
  max-height: 600px;
  overflow-y: auto;
}

.markdown-content h1,
.markdown-content h2,
.markdown-content h3,
.markdown-content h4,
.markdown-content h5,
.markdown-content h6 {
  margin-top: 24px;
  margin-bottom: 16px;
  font-weight: 600;
  line-height: 1.25;
  color: var(--text-primary);
}

.markdown-content h1 {
  font-size: 2em;
  padding-bottom: 0.3em;
  border-bottom: 1px solid var(--border-base);
}

.markdown-content h2 {
  font-size: 1.5em;
  padding-bottom: 0.3em;
  border-bottom: 1px solid var(--border-base);
}

.markdown-content h3 {
  font-size: 1.25em;
}

.markdown-content p {
  margin-bottom: 16px;
}

.markdown-content a {
  color: #409eff;
  text-decoration: none;
}

.markdown-content a:hover {
  text-decoration: underline;
}

.markdown-content code {
  padding: 0.2em 0.4em;
  margin: 0;
  font-size: 85%;
  background-color: rgba(27, 31, 35, 0.05);
  border-radius: 3px;
  font-family: var(--font-family-mono);
}

.markdown-content pre {
  padding: 16px;
  overflow: auto;
  font-size: 85%;
  line-height: 1.45;
  background-color: #1e1e1e;
  border-radius: 6px;
  margin-bottom: 16px;
}

.markdown-content pre code {
  padding: 0;
  margin: 0;
  background-color: transparent;
  border: 0;
}

.markdown-content pre.hljs {
  background-color: #1e1e1e;
}

.markdown-content blockquote {
  padding: 0 1em;
  color: #6a737d;
  border-left: 0.25em solid #dfe2e5;
  margin: 0 0 16px 0;
}

.markdown-content ul,
.markdown-content ol {
  padding-left: 2em;
  margin-bottom: 16px;
}

.markdown-content li + li {
  margin-top: 0.25em;
}

.markdown-content table {
  border-collapse: collapse;
  width: 100%;
  margin-bottom: 16px;
}

.markdown-content table th,
.markdown-content table td {
  padding: 6px 13px;
  border: 1px solid #dfe2e5;
}

.markdown-content table th {
  font-weight: 600;
  background-color: var(--bg-secondary);
}

.markdown-content table tr:nth-child(2n) {
  background-color: var(--bg-secondary);
}

.markdown-content img {
  max-width: 100%;
  box-sizing: content-box;
}

.markdown-content hr {
  height: 0.25em;
  padding: 0;
  margin: 24px 0;
  background-color: #e1e4e8;
  border: 0;
}

/* 任务列表样式 */
.markdown-content .task-list-item {
  list-style-type: none;
}

.markdown-content .task-list-item input {
  margin-right: 0.5em;
}

/* 脚注样式 */
.markdown-content .footnote-ref {
  font-size: 0.75em;
  vertical-align: super;
}

.markdown-content .footnotes {
  margin-top: 32px;
  padding-top: 16px;
  border-top: 1px solid var(--border-base);
  font-size: 0.875em;
  color: var(--text-secondary);
}

/* 标记样式 */
.markdown-content mark {
  background-color: #fff3cd;
  padding: 0.2em 0.4em;
  border-radius: 2px;
}
</style>
