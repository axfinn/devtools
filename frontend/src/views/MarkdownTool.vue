<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Markdown 编辑器</h2>
      <div class="actions">
        <el-button type="success" @click="showShareDialog = true">
          <el-icon><Share /></el-icon>
          分享
        </el-button>
        <el-button v-if="hasCreatedShares" type="warning" @click="showMyShares = true">
          <el-icon><Folder /></el-icon>
          我的分享
        </el-button>
        <el-button v-if="isAdmin" type="danger" @click="openAdminPanel">
          <el-icon><Setting /></el-icon>
          管理
        </el-button>
        <el-button v-else @click="showAdminLogin = true">
          <el-icon><Setting /></el-icon>
          管理
        </el-button>
        <el-button type="primary" @click="exportHtml">
          <el-icon><Download /></el-icon>
          导出 HTML
        </el-button>
        <el-button @click="exportPdf">
          <el-icon><Printer /></el-icon>
          打印/PDF
        </el-button>
        <el-button @click="copyHtml">
          <el-icon><CopyDocument /></el-icon>
          复制 HTML
        </el-button>
      </div>
    </div>

    <div class="feature-hints">
      <el-tag type="success" size="small">Mermaid 图表</el-tag>
      <el-tag type="info" size="small">KaTeX 公式</el-tag>
      <el-tag type="warning" size="small">代码高亮</el-tag>
      <el-tag size="small">任务列表</el-tag>
      <el-tag type="danger" size="small">粘贴/拖拽图片</el-tag>
      <el-tag type="info" size="small">脚注/高亮/上下标</el-tag>
    </div>

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">
          <span>Markdown 输入</span>
          <span class="char-count">{{ markdownText.length }} 字符</span>
        </div>
        <textarea
          ref="editorRef"
          v-model="markdownText"
          class="code-editor"
          placeholder="输入 Markdown 内容...&#10;&#10;支持粘贴图片 (Ctrl+V)&#10;支持拖拽图片"
          spellcheck="false"
          @scroll="onScroll('editor')"
          @paste="handlePaste"
          @drop="handleDrop"
          @dragover.prevent
        ></textarea>
      </div>
      <div class="editor-panel preview-panel">
        <div class="panel-header">预览</div>
        <div ref="previewRef" class="preview-content markdown-body" v-html="renderedHtml" @scroll="onScroll('preview')"></div>
      </div>
    </div>

    <!-- Share Dialog -->
    <el-dialog v-model="showShareDialog" title="分享 Markdown" width="450px" :close-on-click-modal="false">
      <el-form label-width="100px" class="share-form">
        <el-form-item label="标题">
          <el-input v-model="shareForm.title" placeholder="可选，方便识别" maxlength="50" show-word-limit />
        </el-form-item>
        <el-form-item label="查看次数">
          <el-select v-model="shareForm.maxViews" style="width: 100%" popper-class="share-select-dropdown">
            <el-option v-for="opt in viewsOptions" :key="opt.value" :value="opt.value" :label="opt.label">
              <div class="select-option">
                <span class="option-label">{{ opt.label }}</span>
                <el-tag v-if="opt.recommended" type="success" size="small" effect="plain">推荐</el-tag>
                <span v-else class="option-desc">{{ opt.desc }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="有效期">
          <el-select v-model="shareForm.expiresDays" style="width: 100%" popper-class="share-select-dropdown">
            <el-option v-for="opt in expiresOptions" :key="opt.value" :value="opt.value" :label="opt.label">
              <div class="select-option">
                <span class="option-label">{{ opt.label }}</span>
                <el-tag v-if="opt.recommended" type="success" size="small" effect="plain">推荐</el-tag>
                <span v-else class="option-desc">{{ opt.desc }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <div class="content-preview">
        <div class="preview-label">内容预览 ({{ markdownText.length }} 字符)</div>
        <div class="preview-text">{{ markdownText.substring(0, 200) }}{{ markdownText.length > 200 ? '...' : '' }}</div>
      </div>
      <template #footer>
        <el-button @click="showShareDialog = false">取消</el-button>
        <el-button type="primary" @click="createShare" :loading="shareLoading">创建分享</el-button>
      </template>
    </el-dialog>

    <!-- Share Result Dialog -->
    <el-dialog v-model="showShareResult" title="分享创建成功" width="500px" :close-on-click-modal="false">
      <el-result icon="success" title="分享已创建">
        <template #sub-title>
          <p>可查看 {{ shareResult.maxViews }} 次，{{ shareResult.expiresDays }} 天后过期</p>
        </template>
      </el-result>

      <el-form label-width="80px" class="result-form">
        <el-form-item label="短链接" v-if="shareResult.shortUrl">
          <el-input :model-value="shareResult.shortUrl" readonly>
            <template #append>
              <el-button @click="copyToClipboard(shareResult.shortUrl, '短链接')">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="完整链接">
          <el-input :model-value="shareResult.fullUrl" readonly>
            <template #append>
              <el-button @click="copyToClipboard(shareResult.fullUrl, '完整链接')">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>

      <el-alert type="info" :closable="false" style="margin-top: 15px;">
        <template #title>管理密钥已自动保存</template>
        <div style="font-size: 12px;">用此浏览器可管理分享（续期、再分享、编辑）</div>
      </el-alert>

      <template #footer>
        <el-button @click="showShareResult = false">关闭</el-button>
        <el-button type="primary" @click="copyAndClose">复制链接并关闭</el-button>
      </template>
    </el-dialog>

    <!-- My Shares Dialog -->
    <el-dialog v-model="showMyShares" title="我的分享" width="700px">
      <el-table :data="mySharesList" v-loading="loadingShares" empty-text="暂无分享" max-height="400">
        <el-table-column prop="title" label="标题" min-width="120">
          <template #default="{ row }">
            {{ row.title || '(无标题)' }}
          </template>
        </el-table-column>
        <el-table-column label="剩余/总次数" width="110">
          <template #default="{ row }">
            <el-tag :type="row.remaining_views <= 1 ? 'danger' : row.remaining_views <= 3 ? 'warning' : 'success'" size="small">
              {{ row.remaining_views }}/{{ row.max_views }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="160">
          <template #default="{ row }">
            {{ formatDate(row.expires_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="loadShareToEditor(row)">编辑</el-button>
            <el-button size="small" type="warning" @click="reshareShare(row)">再分享</el-button>
            <el-button size="small" type="danger" @click="deleteShare(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- Reshare Dialog -->
    <el-dialog v-model="showReshareDialog" title="再次分享" width="400px">
      <el-form label-width="100px" class="share-form">
        <el-form-item label="新查看次数">
          <el-select v-model="reshareForm.maxViews" style="width: 100%" popper-class="share-select-dropdown">
            <el-option v-for="opt in viewsOptions" :key="opt.value" :value="opt.value" :label="opt.label">
              <div class="select-option">
                <span class="option-label">{{ opt.label }}</span>
                <el-tag v-if="opt.recommended" type="success" size="small" effect="plain">推荐</el-tag>
                <span v-else class="option-desc">{{ opt.desc }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <el-alert type="warning" :closable="false">
        <template #title>生成新链接后，旧链接将立即失效</template>
      </el-alert>
      <template #footer>
        <el-button @click="showReshareDialog = false">取消</el-button>
        <el-button type="primary" @click="doReshare" :loading="reshareLoading">生成新链接</el-button>
      </template>
    </el-dialog>

    <!-- Admin Login Dialog -->
    <el-dialog v-model="showAdminLogin" title="管理员登录" width="350px" :close-on-click-modal="false">
      <el-form @submit.prevent="verifyAdminPassword">
        <el-form-item>
          <el-input
            v-model="adminPassword"
            type="password"
            placeholder="请输入管理员密码"
            show-password
            @keyup.enter="verifyAdminPassword"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdminLogin = false">取消</el-button>
        <el-button type="primary" @click="verifyAdminPassword" :loading="adminLoading">确认</el-button>
      </template>
    </el-dialog>

    <!-- Admin Panel Dialog -->
    <el-dialog v-model="showAdminPanel" title="分享管理" width="900px" :close-on-click-modal="false">
      <div class="admin-toolbar">
        <el-input
          v-model="adminSearchKeyword"
          placeholder="搜索标题或 ID"
          clearable
          style="width: 250px"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button @click="loadAllShares" :loading="loadingAllShares">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <span class="share-count">共 {{ filteredAllShares.length }} 条</span>
      </div>

      <el-table
        :data="paginatedShares"
        v-loading="loadingAllShares"
        empty-text="暂无分享"
        max-height="400"
        stripe
      >
        <el-table-column prop="id" label="ID" width="100">
          <template #default="{ row }">
            <el-tooltip :content="row.id" placement="top">
              <span class="id-cell">{{ row.id }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="150">
          <template #default="{ row }">
            {{ row.title || '(无标题)' }}
          </template>
        </el-table-column>
        <el-table-column label="次数" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.remaining_views <= 1 ? 'danger' : row.remaining_views <= 3 ? 'warning' : 'success'"
              size="small"
            >
              {{ row.remaining_views }}/{{ row.max_views }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="160">
          <template #default="{ row }">
            <span :class="{ 'text-danger': isExpiringSoon(row.expires_at) }">
              {{ formatDate(row.expires_at) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="previewShareAdmin(row)">
              <el-icon><View /></el-icon>
              查看
            </el-button>
            <el-button size="small" @click="copyShareLink(row)">
              <el-icon><Link /></el-icon>
              链接
            </el-button>
            <el-button size="small" type="danger" @click="deleteShareAdmin(row)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="admin-pagination">
        <el-pagination
          v-model:current-page="adminCurrentPage"
          :page-size="adminPageSize"
          :total="filteredAllShares.length"
          layout="prev, pager, next"
          :hide-on-single-page="true"
        />
      </div>

      <template #footer>
        <el-button @click="logoutAdmin" type="warning">退出登录</el-button>
        <el-button @click="showAdminPanel = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- Admin Preview Dialog -->
    <el-dialog v-model="showAdminPreview" title="内容预览" width="800px">
      <div class="admin-preview-header">
        <span class="preview-title">{{ adminPreviewShare?.title || '(无标题)' }}</span>
        <el-tag size="small">{{ adminPreviewShare?.remaining_views }}/{{ adminPreviewShare?.max_views }} 次</el-tag>
      </div>
      <div class="admin-preview-content markdown-body" v-html="adminPreviewHtml"></div>
      <template #footer>
        <el-button @click="showAdminPreview = false">关闭</el-button>
        <el-button type="primary" @click="loadAdminShareToEditor">加载到编辑器</el-button>
      </template>
    </el-dialog>

    <!-- Image uploading indicator -->
    <div v-if="uploadingImage" class="upload-overlay">
      <el-icon class="upload-icon"><Loading /></el-icon>
      <span>图片上传中...</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useRouter } from 'vue-router'
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
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()

// Initialize Mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  securityLevel: 'loose',
  flowchart: { useMaxWidth: true, htmlLabels: true }
})

// Initialize Markdown-it with all extensions
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

// Add all extensions
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
const editorRef = ref(null)
let isScrolling = false

const onScroll = (source) => {
  if (isScrolling) return
  isScrolling = true

  const sourceEl = source === 'editor' ? editorRef.value : previewRef.value
  const targetEl = source === 'editor' ? previewRef.value : editorRef.value

  if (sourceEl && targetEl) {
    const sourceScrollRatio = sourceEl.scrollTop / (sourceEl.scrollHeight - sourceEl.clientHeight || 1)
    targetEl.scrollTop = sourceScrollRatio * (targetEl.scrollHeight - targetEl.clientHeight)
  }

  requestAnimationFrame(() => { isScrolling = false })
}

const markdownText = ref(`# Markdown 编辑器

## 功能特点

- [x] 实时预览
- [x] 代码高亮
- [x] **Mermaid 图表**
- [x] **KaTeX 数学公式**
- [x] ==高亮文本==
- [x] H~2~O 下标
- [x] X^2^ 上标
- [x] 脚注支持[^1]
- [ ] 待办事项

[^1]: 这是一个脚注示例

---

## Mermaid 图表

\`\`\`mermaid
flowchart LR
    A[编写] --> B{预览}
    B -->|满意| C[分享]
    B -->|修改| A
    C --> D[查看]
\`\`\`

---

## 数学公式

行内公式: $E = mc^2$

块级公式:
$$
\\int_{-\\infty}^{\\infty} e^{-x^2} dx = \\sqrt{\\pi}
$$

---

## 代码示例

\`\`\`javascript
const share = async (content) => {
  const res = await fetch('/api/mdshare', {
    method: 'POST',
    body: JSON.stringify({ content })
  })
  return res.json()
}
\`\`\`

---

## 图片支持

可以直接 **粘贴** 或 **拖拽** 图片到编辑器中！

图片会自动上传到服务器。

---

> 提示: 点击右上角"分享"按钮可生成短链接分享给他人
`)

const renderedHtml = computed(() => md.render(markdownText.value))

// Render Mermaid
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
onMounted(() => renderMermaid())

// Image upload
const uploadingImage = ref(false)

const handlePaste = async (e) => {
  const items = e.clipboardData?.items
  if (!items) return

  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      await uploadAndInsertImage(file)
      break
    }
  }
}

const handleDrop = async (e) => {
  e.preventDefault()
  const files = e.dataTransfer?.files
  if (!files) return

  for (const file of files) {
    if (file.type.startsWith('image/')) {
      await uploadAndInsertImage(file)
    }
  }
}

const uploadAndInsertImage = async (file) => {
  // Check size (max 5MB)
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.warning('图片大小不能超过 5MB')
    return
  }

  uploadingImage.value = true

  try {
    const formData = new FormData()
    formData.append('file', file)

    const res = await fetch('/api/chat/upload', {
      method: 'POST',
      body: formData
    })

    const data = await res.json()

    if (!res.ok) {
      ElMessage.error(data.error || '上传失败')
      uploadingImage.value = false
      return
    }

    // Insert markdown image syntax
    const imageUrl = data.url
    const markdown = `\n![image](${imageUrl})\n`

    // Insert at cursor position
    const textarea = editorRef.value
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const text = markdownText.value
    markdownText.value = text.substring(0, start) + markdown + text.substring(end)

    // Move cursor after the inserted image
    await nextTick()
    textarea.selectionStart = textarea.selectionEnd = start + markdown.length
    textarea.focus()

    ElMessage.success('图片已上传')
  } catch (err) {
    ElMessage.error('上传失败: ' + err.message)
  }

  uploadingImage.value = false
}

// Export functions
const getFullHtml = async () => {
  await renderMermaid()
  await new Promise(resolve => setTimeout(resolve, 500))
  const renderedContent = previewRef.value ? previewRef.value.innerHTML : renderedHtml.value

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Markdown Export</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif; line-height: 1.6; max-width: 900px; margin: 0 auto; padding: 40px 20px; color: var(--text-primary); background: #fff; }
    h1, h2, h3, h4, h5, h6 { margin-top: 24px; margin-bottom: 16px; font-weight: 600; line-height: 1.25; }
    h1 { font-size: 2em; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
    h2 { font-size: 1.5em; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
    pre { background-color: var(--bg-primary); padding: 16px; border-radius: var(--radius-sm); overflow-x: auto; color: var(--text-primary); }
    code { font-family: 'SFMono-Regular', Consolas, monospace; font-size: 85%; }
    :not(pre) > code { background-color: rgba(175, 184, 193, 0.2); padding: 0.2em 0.4em; border-radius: 3px; }
    table { border-collapse: collapse; width: 100%; margin: 16px 0; }
    th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
    th { background-color: #f6f8fa; }
    blockquote { border-left: 4px solid #0969da; margin: 16px 0; padding: 0 16px; color: #57606a; background-color: #f6f8fa; }
    img { max-width: 100%; }
    hr { border: 0; height: 1px; background: #d0d7de; margin: 24px 0; }
    .mermaid { display: flex; justify-content: center; margin: 20px 0; background: #f8f9fa; padding: 20px; border-radius: var(--radius-md); }
    mark { background-color: var(--bg-primary);3cd; padding: 0.1em 0.2em; }
    .task-list-item { list-style: none; }
    .task-list-item input { margin-right: 8px; }
    @media print { body { max-width: none; padding: 20px; } }
  </style>
</head>
<body>${renderedContent}</body>
</html>`
}

const exportHtml = async () => {
  const html = await getFullHtml()
  const blob = new Blob([html], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'markdown-export.html'
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('HTML 已下载')
}

const exportPdf = async () => {
  const html = await getFullHtml()
  const printWindow = window.open('', '_blank')
  printWindow.document.write(html)
  printWindow.document.close()
  printWindow.focus()
  setTimeout(() => printWindow.print(), 1000)
}

const copyHtml = async () => {
  try {
    await renderMermaid()
    await new Promise(resolve => setTimeout(resolve, 300))
    const content = previewRef.value ? previewRef.value.innerHTML : renderedHtml.value
    await navigator.clipboard.writeText(content)
    ElMessage.success('HTML 已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

// Share functionality
const showShareDialog = ref(false)
const showShareResult = ref(false)
const shareLoading = ref(false)
const shareForm = ref({
  title: '',
  maxViews: 5,
  expiresDays: 30
})
const shareResult = ref({
  shortUrl: '',
  fullUrl: '',
  maxViews: 5,
  expiresDays: 30
})

// Select options with descriptions
const viewsOptions = [
  { value: 2, label: '2 次', desc: '私密分享' },
  { value: 3, label: '3 次', desc: '小范围' },
  { value: 5, label: '5 次', recommended: true },
  { value: 7, label: '7 次', desc: '团队内' },
  { value: 10, label: '10 次', desc: '最大限制' }
]

const expiresOptions = [
  { value: 7, label: '7 天', desc: '临时分享' },
  { value: 30, label: '30 天', recommended: true },
  { value: 90, label: '90 天', desc: '长期保存' },
  { value: 365, label: '1 年', desc: '归档用途' }
]

const createShare = async () => {
  if (!markdownText.value.trim()) {
    ElMessage.warning('请输入内容后再分享')
    return
  }

  if (markdownText.value.length > 2 * 1024 * 1024) {
    ElMessage.warning('内容超过 2MB 限制')
    return
  }

  shareLoading.value = true
  try {
    const res = await fetch('/api/mdshare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: markdownText.value,
        title: shareForm.value.title,
        max_views: shareForm.value.maxViews,
        expires_in: shareForm.value.expiresDays
      })
    })
    const data = await res.json()

    if (!res.ok) {
      ElMessage.error(data.error || '创建失败')
      shareLoading.value = false
      return
    }

    // Save creator key
    saveCreatorKey(data.id, data.creator_key, {
      title: shareForm.value.title,
      max_views: shareForm.value.maxViews,
      expires_at: data.expires_at
    })

    // Show result
    const baseUrl = window.location.origin
    shareResult.value = {
      shortUrl: data.share_url ? baseUrl + data.share_url : '',
      fullUrl: `${baseUrl}/md/${data.id}?key=${data.access_key}`,
      maxViews: shareForm.value.maxViews,
      expiresDays: shareForm.value.expiresDays
    }

    showShareDialog.value = false
    showShareResult.value = true
    shareLoading.value = false

    // Reset form
    shareForm.value.title = ''
  } catch {
    ElMessage.error('网络错误')
    shareLoading.value = false
  }
}

const copyAndClose = () => {
  const url = shareResult.value.shortUrl || shareResult.value.fullUrl
  copyToClipboard(url, '链接')
  showShareResult.value = false
}

// My Shares
const showMyShares = ref(false)
const mySharesList = ref([])
const loadingShares = ref(false)

const hasCreatedShares = computed(() => {
  const keys = getCreatorKeys()
  return Object.keys(keys).length > 0
})

const getCreatorKeys = () => {
  try {
    return JSON.parse(localStorage.getItem('mdshare_creator_keys') || '{}')
  } catch {
    return {}
  }
}

const saveCreatorKey = (id, key, meta = {}) => {
  const keys = getCreatorKeys()
  keys[id] = { key, ...meta, created_at: new Date().toISOString() }
  localStorage.setItem('mdshare_creator_keys', JSON.stringify(keys))
}

const removeCreatorKey = (id) => {
  const keys = getCreatorKeys()
  delete keys[id]
  localStorage.setItem('mdshare_creator_keys', JSON.stringify(keys))
}

watch(showMyShares, async (val) => {
  if (val) {
    await loadMyShares()
  }
})

const loadMyShares = async () => {
  loadingShares.value = true
  const keys = getCreatorKeys()
  const shares = []

  for (const [id, data] of Object.entries(keys)) {
    try {
      const res = await fetch(`/api/mdshare/${id}/creator?creator_key=${data.key}`)
      if (res.ok) {
        const share = await res.json()
        shares.push({ ...share, creator_key: data.key })
      } else if (res.status === 404 || res.status === 410) {
        // Share expired or deleted, remove from local storage
        removeCreatorKey(id)
      }
    } catch {
      // Network error, skip
    }
  }

  mySharesList.value = shares
  loadingShares.value = false
}

const loadShareToEditor = async (share) => {
  markdownText.value = share.content
  showMyShares.value = false
  ElMessage.success('已加载到编辑器')
}

const deleteShare = async (share) => {
  try {
    await ElMessageBox.confirm('确定要删除此分享吗？删除后无法恢复。', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  try {
    const res = await fetch(`/api/mdshare/${share.id}?creator_key=${share.creator_key}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      removeCreatorKey(share.id)
      mySharesList.value = mySharesList.value.filter(s => s.id !== share.id)
      ElMessage.success('删除成功')
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '删除失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

// Reshare
const showReshareDialog = ref(false)
const reshareLoading = ref(false)
const reshareForm = ref({ maxViews: 5 })
const currentReshareShare = ref(null)

const reshareShare = (share) => {
  currentReshareShare.value = share
  reshareForm.value.maxViews = share.max_views
  showReshareDialog.value = true
}

const doReshare = async () => {
  if (!currentReshareShare.value) return

  reshareLoading.value = true
  try {
    const res = await fetch(`/api/mdshare/${currentReshareShare.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'reshare',
        max_views: reshareForm.value.maxViews,
        creator_key: currentReshareShare.value.creator_key
      })
    })
    const data = await res.json()

    if (res.ok) {
      const baseUrl = window.location.origin
      const newUrl = data.share_url ? baseUrl + data.share_url : `${baseUrl}/md/${currentReshareShare.value.id}?key=${data.access_key}`

      showReshareDialog.value = false
      await loadMyShares()

      ElMessageBox.alert(`新链接: ${newUrl}`, '再分享成功', {
        confirmButtonText: '复制链接',
        callback: () => {
          navigator.clipboard.writeText(newUrl)
          ElMessage.success('链接已复制')
        }
      })
    } else {
      ElMessage.error(data.error || '操作失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
  reshareLoading.value = false
}

// Utils
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

const copyToClipboard = async (text, name) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${name}已复制`)
  } catch {
    ElMessage.error('复制失败')
  }
}

// Admin functionality
const isAdmin = ref(false)
const adminPassword = ref('')
const showAdminLogin = ref(false)
const showAdminPanel = ref(false)
const adminLoading = ref(false)
const loadingAllShares = ref(false)
const allShares = ref([])
const adminSearchKeyword = ref('')
const adminCurrentPage = ref(1)
const adminPageSize = 10

// Check if already logged in
onMounted(() => {
  const savedPwd = sessionStorage.getItem('mdshare_admin_pwd')
  if (savedPwd) {
    isAdmin.value = true
  }
})

const filteredAllShares = computed(() => {
  if (!adminSearchKeyword.value) return allShares.value
  const keyword = adminSearchKeyword.value.toLowerCase()
  return allShares.value.filter(s =>
    s.id.toLowerCase().includes(keyword) ||
    (s.title && s.title.toLowerCase().includes(keyword))
  )
})

const paginatedShares = computed(() => {
  const start = (adminCurrentPage.value - 1) * adminPageSize
  return filteredAllShares.value.slice(start, start + adminPageSize)
})

const verifyAdminPassword = async () => {
  if (!adminPassword.value.trim()) {
    ElMessage.warning('请输入密码')
    return
  }

  adminLoading.value = true
  try {
    const res = await fetch(`/api/mdshare/admin/list?admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (res.ok) {
      sessionStorage.setItem('mdshare_admin_pwd', adminPassword.value)
      isAdmin.value = true
      showAdminLogin.value = false
      const data = await res.json()
      allShares.value = data.list || []
      showAdminPanel.value = true
      ElMessage.success('登录成功')
    } else {
      ElMessage.error('密码错误')
    }
  } catch {
    ElMessage.error('网络错误')
  }
  adminLoading.value = false
}

const openAdminPanel = async () => {
  showAdminPanel.value = true
  await loadAllShares()
}

const loadAllShares = async () => {
  const pwd = sessionStorage.getItem('mdshare_admin_pwd')
  if (!pwd) {
    isAdmin.value = false
    showAdminPanel.value = false
    showAdminLogin.value = true
    return
  }

  loadingAllShares.value = true
  try {
    const res = await fetch(`/api/mdshare/admin/list?admin_password=${encodeURIComponent(pwd)}`)
    if (res.ok) {
      const data = await res.json()
      allShares.value = data.list || []
    } else if (res.status === 401 || res.status === 403) {
      sessionStorage.removeItem('mdshare_admin_pwd')
      isAdmin.value = false
      showAdminPanel.value = false
      ElMessage.error('密码已失效，请重新登录')
      showAdminLogin.value = true
    }
  } catch {
    ElMessage.error('加载失败')
  }
  loadingAllShares.value = false
}

const logoutAdmin = () => {
  sessionStorage.removeItem('mdshare_admin_pwd')
  isAdmin.value = false
  showAdminPanel.value = false
  adminPassword.value = ''
  allShares.value = []
  ElMessage.success('已退出管理')
}

// Admin preview
const showAdminPreview = ref(false)
const adminPreviewShare = ref(null)
const adminPreviewHtml = ref('')

const previewShareAdmin = async (share) => {
  const pwd = sessionStorage.getItem('mdshare_admin_pwd')
  try {
    const res = await fetch(`/api/mdshare/admin/${share.id}?admin_password=${encodeURIComponent(pwd)}`)
    if (res.ok) {
      const data = await res.json()
      adminPreviewShare.value = data
      adminPreviewHtml.value = md.render(data.content || '')
      showAdminPreview.value = true
    } else {
      ElMessage.error('加载失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

const loadAdminShareToEditor = () => {
  if (adminPreviewShare.value?.content) {
    markdownText.value = adminPreviewShare.value.content
    showAdminPreview.value = false
    showAdminPanel.value = false
    ElMessage.success('已加载到编辑器')
  }
}

const deleteShareAdmin = async (share) => {
  try {
    await ElMessageBox.confirm(`确定要删除 "${share.title || share.id}" 吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  const pwd = sessionStorage.getItem('mdshare_admin_pwd')
  try {
    const res = await fetch(`/api/mdshare/admin/${share.id}?admin_password=${encodeURIComponent(pwd)}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      allShares.value = allShares.value.filter(s => s.id !== share.id)
      ElMessage.success('删除成功')
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '删除失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

const copyShareLink = (share) => {
  const baseUrl = window.location.origin
  // Use short URL if available
  const url = share.short_code
    ? `${baseUrl}/s/${share.short_code}`
    : `${baseUrl}/md/${share.id}?key=${share.access_key}`
  copyToClipboard(url, '链接')
}

const isExpiringSoon = (dateStr) => {
  if (!dateStr) return false
  const expires = new Date(dateStr)
  const now = new Date()
  const daysLeft = (expires - now) / (1000 * 60 * 60 * 24)
  return daysLeft <= 3
}
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 15px;
  height: 100%;
  overflow: hidden;
  position: relative;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  flex-shrink: 0;
}

.tool-header h2 {
  margin: 0;
  color: var(--text-primary);
}


.feature-hints {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.editor-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  height: 100%;
}


.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 14px;
  flex-shrink: 0;
  border-bottom: 1px solid #e0e0e0;
}


.char-count {
  font-size: 12px;
  color: var(--text-secondary);
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
  overflow-y: auto;
  min-height: 0;
}


.preview-panel {
  background-color: var(--bg-primary);
}

.preview-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  color: var(--text-primary);
  min-height: 0;
  line-height: 1.8;
}

/* Upload overlay */
.upload-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #fff;
  font-size: 16px;
  z-index: 100;
}

.upload-icon {
  font-size: 32px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Markdown styles */
.preview-content :deep(pre) {
  background-color: var(--bg-primary);
  padding: 16px;
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-md);
}

.preview-content :deep(.mermaid svg) {
  max-width: 100%;
  height: auto;
}

.preview-content :deep(.mermaid-error) {
  color: #d32f2f;
  padding: 10px;
  background: #ffebee;
  border-radius: var(--radius-sm);
}

.preview-content :deep(.katex-display) {
  margin: 16px 0;
  overflow-x: auto;
}

.preview-content :deep(.katex) {
  font-size: 1.1em;
}

/* Task list */
.preview-content :deep(.task-list-item) {
  list-style: none;
}

.preview-content :deep(.task-list-item input) {
  margin-right: 8px;
}

/* Mark/highlight */
.preview-content :deep(mark) {
  background-color: var(--bg-primary);3cd;
  padding: 0.1em 0.2em;
  border-radius: 2px;
}

/* Footnotes */
.preview-content :deep(.footnotes) {
  border-top: 1px solid #eee;
  margin-top: 30px;
  padding-top: 20px;
  font-size: 0.9em;
  color: var(--text-secondary);
}

/* Dialog styles */
.content-preview {
  margin-top: 20px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: var(--radius-sm);
}

.preview-label {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-bottom: 8px;
}

.preview-text {
  font-size: 13px;
  color: #606266;
  white-space: pre-wrap;
  word-break: break-all;
}

.result-form {
  margin-top: 20px;
}

/* Admin panel styles */
.admin-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.share-count {
  color: var(--text-tertiary);
  font-size: 13px;
  margin-left: auto;
}

.id-cell {
  font-family: var(--font-family-mono);
  font-size: 12px;
  color: var(--color-primary);
  cursor: pointer;
}

.text-danger {
  color: #f56c6c;
}

.admin-pagination {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

.admin-preview-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #ebeef5;
}

.preview-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

.admin-preview-content {
  max-height: 400px;
  overflow-y: auto;
  padding: 16px;
  background: #fafafa;
  border-radius: var(--radius-sm);
}

/* Share form select styles */
.share-form :deep(.el-select) {
  width: 100%;
}

.select-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.option-label {
  font-weight: 500;
}

.option-desc {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* Mobile */
@media (max-width: 768px) {
  .tool-container {
    height: auto;
    overflow: visible;
  }

  .editor-container {
    grid-template-columns: 1fr;
    flex: none;
    overflow: visible;
  }

  .editor-panel {
    height: 400px;
  }
}
</style>
