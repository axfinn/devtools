<template>
  <div class="share-view-container">
    <!-- Loading -->
    <div v-if="loading" class="status-container">
      <el-icon class="loading-icon"><Loading /></el-icon>
      <span>加载中...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="status-container error">
      <el-icon class="error-icon"><CircleClose /></el-icon>
      <h3>{{ error }}</h3>
      <el-button type="primary" @click="$router.push('/markdown')">返回编辑器</el-button>
    </div>

    <!-- Content -->
    <template v-else>
      <div class="share-header">
        <div class="title-section">
          <h2>{{ shareData.title || 'Markdown 分享' }}</h2>
          <div class="meta-info">
            <el-tag :type="remainingType" size="small">
              剩余 {{ shareData.remaining_views }} 次查看
            </el-tag>
            <span class="create-time">{{ formatDate(shareData.created_at) }}</span>
          </div>
        </div>
        <div class="actions">
          <el-button @click="copyContent" size="small">
            <el-icon><CopyDocument /></el-icon>
            复制
          </el-button>
        </div>
      </div>

      <div class="content-container">
        <div ref="previewRef" class="preview-content markdown-body" v-html="renderedHtml"></div>
      </div>

      <div class="footer-notice">
        <el-alert type="warning" :closable="false" show-icon>
          <template #title>
            此分享链接还可查看 {{ shareData.remaining_views }} 次，请及时保存内容
          </template>
        </el-alert>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import mermaid from 'mermaid'
import texmath from 'markdown-it-texmath'
import katex from 'katex'
import 'katex/dist/katex.min.css'
import markdownItTaskLists from 'markdown-it-task-lists'
import markdownItFootnote from 'markdown-it-footnote'
import markdownItMark from 'markdown-it-mark'
import markdownItSub from 'markdown-it-sub'
import markdownItSup from 'markdown-it-sup'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

// Initialize Mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  securityLevel: 'loose',
  flowchart: { useMaxWidth: true, htmlLabels: true }
})

// Initialize Markdown-it with extensions
const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  breaks: true,
  highlight: function (str, lang) {
    if (lang === 'mermaid') {
      return `<div class="mermaid">${str}</div>`
    }
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

// Add extensions
md.use(texmath, {
  engine: katex,
  delimiters: ['dollars', 'brackets'],
  katexOptions: { macros: { "\\RR": "\\mathbb{R}" }, throwOnError: false }
})
md.use(markdownItTaskLists, { enabled: true })
md.use(markdownItFootnote)
md.use(markdownItMark)
md.use(markdownItSub)
md.use(markdownItSup)

const previewRef = ref(null)
const loading = ref(true)
const error = ref('')
const shareData = ref({})

const renderedHtml = computed(() => {
  if (!shareData.value.content) return ''
  return md.render(shareData.value.content)
})

const remainingType = computed(() => {
  const remaining = shareData.value.remaining_views || 0
  if (remaining <= 1) return 'danger'
  if (remaining <= 3) return 'warning'
  return 'success'
})

const renderMermaid = async () => {
  await nextTick()
  if (previewRef.value) {
    const elements = previewRef.value.querySelectorAll('.mermaid')
    elements.forEach(async (element, index) => {
      if (!element.getAttribute('data-processed')) {
        try {
          const graphDefinition = element.textContent
          const { svg } = await mermaid.render(`mermaid-${Date.now()}-${index}`, graphDefinition)
          element.innerHTML = svg
          element.setAttribute('data-processed', 'true')
        } catch (e) {
          element.innerHTML = `<div class="mermaid-error">图表渲染错误: ${e.message}</div>`
        }
      }
    })
  }
}

watch(renderedHtml, () => renderMermaid())

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN')
}

const loadShare = async () => {
  const id = route.params.id
  const key = route.query.key

  if (!key) {
    error.value = '缺少访问密钥'
    loading.value = false
    return
  }

  try {
    const res = await fetch(`/api/mdshare/${id}?key=${key}`)
    const data = await res.json()

    if (!res.ok) {
      error.value = data.error || '加载失败'
      loading.value = false
      return
    }

    shareData.value = data
    loading.value = false
    await nextTick()
    renderMermaid()
  } catch (e) {
    error.value = '网络错误'
    loading.value = false
  }
}

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(shareData.value.content)
    ElMessage.success('内容已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  loadShare()
})
</script>

<style scoped>
.share-view-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  padding: 20px;
  gap: 15px;
  background-color: #121212;
}

.status-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 20px;
}

.status-container.error h3 {
  color: #f56c6c;
}

.loading-icon {
  font-size: 48px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.error-icon {
  font-size: 64px;
  color: #f56c6c;
}

.share-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-shrink: 0;
}

.title-section h2 {
  margin: 0 0 8px 0;
  color: #e0e0e0;
}

.meta-info {
  display: flex;
  align-items: center;
  gap: 15px;
}

.create-time {
  font-size: 12px;
  color: #909399;
}

.content-container {
  flex: 1;
  background-color: #fff;
  border-radius: 8px;
  overflow: hidden;
  min-height: 0;
}

.preview-content {
  height: 100%;
  padding: 25px;
  overflow-y: auto;
  color: #333;
  line-height: 1.8;
}

.footer-notice {
  flex-shrink: 0;
}

/* Markdown styles */
.preview-content :deep(pre) {
  background-color: #1e1e1e;
  padding: 16px;
  border-radius: 6px;
  overflow-x: auto;
}

.preview-content :deep(code) {
  font-family: Consolas, Monaco, monospace;
}

.preview-content :deep(:not(pre) > code) {
  background-color: rgba(175, 184, 193, 0.2);
  padding: 0.2em 0.4em;
  border-radius: 3px;
  color: #e83e8c;
}

.preview-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 16px 0;
}

.preview-content :deep(th),
.preview-content :deep(td) {
  border: 1px solid #ddd;
  padding: 8px 12px;
  text-align: left;
}

.preview-content :deep(th) {
  background-color: #f6f8fa;
}

.preview-content :deep(blockquote) {
  border-left: 4px solid #0969da;
  margin: 16px 0;
  padding: 0 16px;
  color: #57606a;
  background-color: #f6f8fa;
}

.preview-content :deep(img) {
  max-width: 100%;
  border-radius: 4px;
}

.preview-content :deep(h1),
.preview-content :deep(h2) {
  border-bottom: 1px solid #eee;
  padding-bottom: 0.3em;
}

.preview-content :deep(hr) {
  border: 0;
  height: 1px;
  background: #d0d7de;
  margin: 24px 0;
}

.preview-content :deep(.mermaid) {
  display: flex;
  justify-content: center;
  margin: 20px 0;
  background: #f8f9fa;
  padding: 20px;
  border-radius: 8px;
}

.preview-content :deep(.mermaid svg) {
  max-width: 100%;
  height: auto;
}

.preview-content :deep(.mermaid-error) {
  color: #d32f2f;
  padding: 10px;
  background: #ffebee;
  border-radius: 4px;
}

.preview-content :deep(.katex-display) {
  margin: 16px 0;
  overflow-x: auto;
}

.preview-content :deep(.katex) {
  font-size: 1.1em;
}

/* Task list styles */
.preview-content :deep(.task-list-item) {
  list-style: none;
}

.preview-content :deep(.task-list-item input) {
  margin-right: 8px;
}

/* Mark/highlight */
.preview-content :deep(mark) {
  background-color: #fff3cd;
  padding: 0.1em 0.2em;
  border-radius: 2px;
}

/* Footnotes */
.preview-content :deep(.footnotes) {
  border-top: 1px solid #eee;
  margin-top: 30px;
  padding-top: 20px;
  font-size: 0.9em;
  color: #666;
}

@media (max-width: 768px) {
  .share-view-container {
    padding: 10px;
  }

  .share-header {
    flex-direction: column;
    gap: 10px;
  }

  .content-container {
    min-height: 300px;
  }
}
</style>
