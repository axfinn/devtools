<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Markdown 编辑器</h2>
      <div class="actions">
        <el-button type="primary" @click="exportHtml">
          <el-icon><Download /></el-icon>
          导出 HTML
        </el-button>
        <el-button @click="exportPdf">
          <el-icon><Printer /></el-icon>
          打印/导出 PDF
        </el-button>
        <el-button @click="copyHtml">
          <el-icon><CopyDocument /></el-icon>
          复制 HTML
        </el-button>
      </div>
    </div>

    <div class="feature-hints">
      <el-tag type="success" size="small">Mermaid 图表</el-tag>
      <el-tag type="info" size="small">KaTeX 数学公式</el-tag>
      <el-tag type="warning" size="small">代码高亮</el-tag>
      <el-tag size="small">表格/链接/图片</el-tag>
    </div>

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">Markdown 输入</div>
        <textarea
          v-model="markdownText"
          class="code-editor"
          placeholder="输入 Markdown 内容..."
          spellcheck="false"
        ></textarea>
      </div>
      <div class="editor-panel preview-panel">
        <div class="panel-header">预览</div>
        <div ref="previewRef" class="preview-content markdown-body" v-html="renderedHtml"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import mermaid from 'mermaid'
import texmath from 'markdown-it-texmath'
import katex from 'katex'
import 'katex/dist/katex.min.css'
import { ElMessage } from 'element-plus'

// 初始化 Mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  securityLevel: 'loose',
  flowchart: {
    useMaxWidth: true,
    htmlLabels: true
  }
})

const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  highlight: function (str, lang) {
    // Mermaid 代码块特殊处理
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

// 添加 KaTeX 数学公式支持
md.use(texmath, {
  engine: katex,
  delimiters: ['dollars', 'brackets'],
  katexOptions: {
    macros: { "\\RR": "\\mathbb{R}" },
    throwOnError: false
  }
})

const previewRef = ref(null)

const markdownText = ref(`# 欢迎使用增强版 Markdown 编辑器

## 功能特点

- ✅ 实时预览
- ✅ 代码高亮
- ✅ **Mermaid 图表支持**
- ✅ **KaTeX 数学公式**
- ✅ 导出 HTML/PDF

---

## 🎨 Mermaid 图表示例

### 流程图

\`\`\`mermaid
flowchart TD
    A[开始] --> B{判断条件}
    B -->|是| C[执行操作A]
    B -->|否| D[执行操作B]
    C --> E[结束]
    D --> E
\`\`\`

### 时序图

\`\`\`mermaid
sequenceDiagram
    participant 用户
    participant 前端
    participant 后端
    participant 数据库

    用户->>前端: 发起请求
    前端->>后端: API调用
    后端->>数据库: 查询数据
    数据库-->>后端: 返回结果
    后端-->>前端: 返回响应
    前端-->>用户: 显示结果
\`\`\`

### 甘特图

\`\`\`mermaid
gantt
    title 项目进度
    dateFormat  YYYY-MM-DD
    section 设计阶段
    需求分析           :a1, 2024-01-01, 7d
    UI设计            :a2, after a1, 5d
    section 开发阶段
    前端开发           :b1, after a2, 10d
    后端开发           :b2, after a2, 12d
    section 测试阶段
    功能测试           :c1, after b2, 5d
    上线部署           :c2, after c1, 2d
\`\`\`

### 饼图

\`\`\`mermaid
pie title 技术栈分布
    "Vue" : 35
    "Go" : 30
    "TypeScript" : 20
    "其他" : 15
\`\`\`

### 类图

\`\`\`mermaid
classDiagram
    class User {
        +String name
        +String email
        +login()
        +logout()
    }
    class Order {
        +Int id
        +Date createTime
        +submit()
    }
    User "1" --> "*" Order : creates
\`\`\`

---

## 📐 数学公式示例

### 行内公式

这是一个行内公式：$E = mc^2$，以及 $\\sum_{i=1}^{n} x_i$

### 块级公式

$$
\\int_{-\\infty}^{\\infty} e^{-x^2} dx = \\sqrt{\\pi}
$$

$$
\\begin{aligned}
f(x) &= x^2 + 2x + 1 \\\\
     &= (x + 1)^2
\\end{aligned}
$$

### 矩阵

$$
\\begin{pmatrix}
a & b \\\\
c & d
\\end{pmatrix}
\\times
\\begin{pmatrix}
e \\\\
f
\\end{pmatrix}
=
\\begin{pmatrix}
ae + bf \\\\
ce + df
\\end{pmatrix}
$$

---

## 代码示例

\`\`\`javascript
async function fetchData(url) {
  const response = await fetch(url);
  const data = await response.json();
  return data;
}
\`\`\`

\`\`\`go
func main() {
    http.HandleFunc("/", handler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
\`\`\`

---

## 表格支持

| 功能 | 状态 | 说明 |
|------|:----:|------|
| Mermaid 流程图 | ✅ | 支持多种图表类型 |
| KaTeX 公式 | ✅ | 支持行内和块级公式 |
| 代码高亮 | ✅ | 支持 100+ 语言 |
| HTML 导出 | ✅ | 包含完整样式 |

---

> 💡 提示：导出的 HTML 文件包含所有图表和公式的渲染结果！
`)

const renderedHtml = computed(() => {
  return md.render(markdownText.value)
})

// 渲染 Mermaid 图表
const renderMermaid = async () => {
  await nextTick()
  if (previewRef.value) {
    const mermaidElements = previewRef.value.querySelectorAll('.mermaid')
    mermaidElements.forEach(async (element, index) => {
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

watch(renderedHtml, () => {
  renderMermaid()
})

onMounted(() => {
  renderMermaid()
})

const getFullHtml = async () => {
  // 等待 Mermaid 渲染完成
  await renderMermaid()
  await new Promise(resolve => setTimeout(resolve, 500))

  // 获取渲染后的 HTML（包含 SVG 图表）
  const renderedContent = previewRef.value ? previewRef.value.innerHTML : renderedHtml.value

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Markdown Export</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
      line-height: 1.6;
      max-width: 900px;
      margin: 0 auto;
      padding: 40px 20px;
      color: #333;
      background: #fff;
    }
    h1, h2, h3, h4, h5, h6 {
      margin-top: 24px;
      margin-bottom: 16px;
      font-weight: 600;
      line-height: 1.25;
    }
    h1 { font-size: 2em; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
    h2 { font-size: 1.5em; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
    h3 { font-size: 1.25em; }
    pre {
      background-color: #1e1e1e;
      padding: 16px;
      border-radius: 6px;
      overflow-x: auto;
      color: #d4d4d4;
    }
    code {
      font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
      font-size: 85%;
    }
    :not(pre) > code {
      background-color: rgba(175, 184, 193, 0.2);
      padding: 0.2em 0.4em;
      border-radius: 3px;
    }
    table {
      border-collapse: collapse;
      width: 100%;
      margin: 16px 0;
    }
    th, td {
      border: 1px solid #ddd;
      padding: 8px 12px;
      text-align: left;
    }
    th { background-color: #f6f8fa; font-weight: 600; }
    blockquote {
      border-left: 4px solid #0969da;
      margin: 16px 0;
      padding: 0 16px;
      color: #57606a;
      background-color: #f6f8fa;
    }
    img { max-width: 100%; }
    hr {
      border: 0;
      height: 1px;
      background: #d0d7de;
      margin: 24px 0;
    }
    .mermaid {
      display: flex;
      justify-content: center;
      margin: 20px 0;
      background: #f8f9fa;
      padding: 20px;
      border-radius: 8px;
    }
    .mermaid svg {
      max-width: 100%;
      height: auto;
    }
    .katex-display {
      margin: 16px 0;
      overflow-x: auto;
      overflow-y: hidden;
    }
    .katex {
      font-size: 1.1em;
    }
    ul, ol {
      padding-left: 2em;
      margin: 16px 0;
    }
    li { margin: 4px 0; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .hljs { background: #1e1e1e; }
    @media print {
      body { max-width: none; padding: 20px; }
      pre { white-space: pre-wrap; word-wrap: break-word; }
      .mermaid { break-inside: avoid; }
    }
  </style>
</head>
<body>
${renderedContent}
</body>
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
  ElMessage.success('HTML 文件已下载（包含图表和公式）')
}

const exportPdf = async () => {
  const html = await getFullHtml()
  const printWindow = window.open('', '_blank')
  printWindow.document.write(html)
  printWindow.document.close()
  printWindow.focus()
  setTimeout(() => {
    printWindow.print()
  }, 1000) // 等待图表渲染
}

const copyHtml = async () => {
  try {
    await renderMermaid()
    await new Promise(resolve => setTimeout(resolve, 300))
    const content = previewRef.value ? previewRef.value.innerHTML : renderedHtml.value

    // 尝试使用 Clipboard API
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(content)
      ElMessage.success('HTML 已复制到剪贴板')
    } else {
      // 备用方案：使用 execCommand
      const textarea = document.createElement('textarea')
      textarea.value = content
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      ElMessage.success('HTML 已复制到剪贴板')
    }
  } catch (e) {
    console.error('复制失败:', e)
    // 最后尝试备用方案
    try {
      const content = previewRef.value ? previewRef.value.innerHTML : renderedHtml.value
      const textarea = document.createElement('textarea')
      textarea.value = content
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      ElMessage.success('HTML 已复制到剪贴板')
    } catch (fallbackError) {
      ElMessage.error('复制失败，请手动复制')
    }
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

.feature-hints {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.editor-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
  flex: 1;
  min-height: 500px;
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

.preview-panel {
  background-color: #fff;
}

.preview-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  color: #333;
}

.preview-content :deep(pre) {
  background-color: #1e1e1e;
  padding: 16px;
  border-radius: 6px;
  overflow-x: auto;
}

.preview-content :deep(code) {
  font-family: Consolas, Monaco, monospace;
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
  overflow-y: hidden;
}

.preview-content :deep(.katex) {
  font-size: 1.1em;
}
</style>
