<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Mermaid 图表编辑器</h2>
      <div class="actions">
        <el-dropdown @command="insertTemplate" trigger="click">
          <el-button>
            <el-icon><Files /></el-icon>
            插入模板
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="flowchart">流程图 (Flowchart)</el-dropdown-item>
              <el-dropdown-item command="sequence">时序图 (Sequence)</el-dropdown-item>
              <el-dropdown-item command="classDiagram">类图 (Class)</el-dropdown-item>
              <el-dropdown-item command="stateDiagram">状态图 (State)</el-dropdown-item>
              <el-dropdown-item command="erDiagram">ER 图</el-dropdown-item>
              <el-dropdown-item command="gantt">甘特图 (Gantt)</el-dropdown-item>
              <el-dropdown-item command="pie">饼图 (Pie)</el-dropdown-item>
              <el-dropdown-item command="mindmap">思维导图 (Mindmap)</el-dropdown-item>
              <el-dropdown-item command="timeline">时间线 (Timeline)</el-dropdown-item>
              <el-dropdown-item command="gitGraph">Git 图 (Git Graph)</el-dropdown-item>
              <el-dropdown-item command="quadrant">象限图 (Quadrant)</el-dropdown-item>
              <el-dropdown-item command="sankey">桑基图 (Sankey)</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-select v-model="theme" placeholder="主题" style="width: 120px;" @change="rerender">
          <el-option label="默认" value="default" />
          <el-option label="森林" value="forest" />
          <el-option label="暗黑" value="dark" />
          <el-option label="中性" value="neutral" />
          <el-option label="基础" value="base" />
        </el-select>
        <el-button type="success" @click="exportPng">
          <el-icon><Picture /></el-icon>
          导出 PNG
        </el-button>
        <el-button type="primary" @click="exportSvg">
          <el-icon><Download /></el-icon>
          导出 SVG
        </el-button>
        <el-button @click="copyCode">
          <el-icon><CopyDocument /></el-icon>
          复制代码
        </el-button>
      </div>
    </div>

    <div class="feature-hints">
      <el-tag type="success" size="small">实时预览</el-tag>
      <el-tag type="info" size="small">多种图表类型</el-tag>
      <el-tag type="warning" size="small">PNG/SVG 导出</el-tag>
      <el-tag size="small">主题切换</el-tag>
    </div>

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">
          <span>Mermaid 代码</span>
          <el-tag v-if="error" type="danger" size="small">语法错误</el-tag>
          <el-tag v-else type="success" size="small">语法正确</el-tag>
        </div>
        <textarea
          ref="editorRef"
          v-model="code"
          class="code-editor"
          placeholder="输入 Mermaid 代码..."
          spellcheck="false"
          @input="debouncedRender"
          @scroll="onScroll('editor')"
        ></textarea>
      </div>
      <div class="editor-panel preview-panel">
        <div class="panel-header">
          <span>预览 <small class="zoom-hint">(滚轮缩放 / 拖拽平移)</small></span>
          <div class="preview-controls">
            <el-button size="small" @click="zoomOut" :disabled="zoomLevel <= 0.25">
              <el-icon><Minus /></el-icon>
            </el-button>
            <span class="zoom-display">{{ Math.round(zoomLevel * 100) }}%</span>
            <el-button size="small" @click="zoomIn" :disabled="zoomLevel >= 10">
              <el-icon><Plus /></el-icon>
            </el-button>
            <el-button size="small" @click="resetZoom" title="重置缩放和位置">
              <el-icon><Refresh /></el-icon>
            </el-button>
            <el-button size="small" @click="toggleFullscreen">
              <el-icon><FullScreen /></el-icon>
            </el-button>
          </div>
        </div>
        <div
          class="preview-content"
          ref="previewRef"
          @wheel.prevent="onWheel"
          @mousedown="onMouseDown"
          @mousemove="onMouseMove"
          @mouseup="onMouseUp"
          @mouseleave="onMouseUp"
          :class="{ 'is-dragging': isDragging }"
        >
          <div v-if="error" class="error-message">
            <el-icon><WarningFilled /></el-icon>
            <div>
              <div class="error-title">图表渲染错误</div>
              <pre class="error-detail">{{ error }}</pre>
            </div>
          </div>
          <div
            v-else
            ref="diagramRef"
            class="diagram-container"
            :style="{
              transform: `translate(${panX}px, ${panY}px) scale(${zoomLevel})`,
              transformOrigin: 'center center'
            }"
            v-html="svgContent"
          ></div>
        </div>
      </div>
    </div>

    <!-- 全屏预览模态框 -->
    <teleport to="body">
      <div v-if="isFullscreen" class="fullscreen-overlay" @click.self="toggleFullscreen">
        <div class="fullscreen-container">
          <div class="fullscreen-header">
            <span>全屏预览 <small class="zoom-hint">(滚轮缩放 / 拖拽平移)</small></span>
            <div class="fullscreen-controls">
              <el-button size="small" @click="fullscreenZoomOut" :disabled="fullscreenZoom <= 0.25">
                <el-icon><Minus /></el-icon>
              </el-button>
              <span class="zoom-display">{{ Math.round(fullscreenZoom * 100) }}%</span>
              <el-button size="small" @click="fullscreenZoomIn" :disabled="fullscreenZoom >= 10">
                <el-icon><Plus /></el-icon>
              </el-button>
              <el-button size="small" @click="resetFullscreenZoom" title="重置缩放和位置">
                <el-icon><Refresh /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="toggleFullscreen">
                <el-icon><Close /></el-icon>
                关闭
              </el-button>
            </div>
          </div>
          <div
            class="fullscreen-content"
            @wheel.prevent="onFullscreenWheel"
            @mousedown="onFullscreenMouseDown"
            @mousemove="onFullscreenMouseMove"
            @mouseup="onFullscreenMouseUp"
            @mouseleave="onFullscreenMouseUp"
            :class="{ 'is-dragging': isFullscreenDragging }"
          >
            <div
              class="fullscreen-diagram"
              :style="{
                transform: `translate(${fullscreenPanX}px, ${fullscreenPanY}px) scale(${fullscreenZoom})`,
                transformOrigin: 'center center'
              }"
              v-html="svgContent"
            ></div>
          </div>
        </div>
      </div>
    </teleport>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import mermaid from 'mermaid'
import { ElMessage } from 'element-plus'

const code = ref('')
const svgContent = ref('')
const error = ref('')
const theme = ref('default')
const zoomLevel = ref(1)
const previewRef = ref(null)
const diagramRef = ref(null)
const editorRef = ref(null)
const isFullscreen = ref(false)
const fullscreenZoom = ref(1)
let isScrolling = false

// 平移状态
const panX = ref(0)
const panY = ref(0)
const isDragging = ref(false)
let dragStartX = 0
let dragStartY = 0
let startPanX = 0
let startPanY = 0

// 全屏模式的平移状态
const fullscreenPanX = ref(0)
const fullscreenPanY = ref(0)
const isFullscreenDragging = ref(false)
let fullscreenDragStartX = 0
let fullscreenDragStartY = 0
let fullscreenStartPanX = 0
let fullscreenStartPanY = 0

const onScroll = (source) => {
  if (isScrolling) return
  isScrolling = true

  const sourceEl = source === 'editor' ? editorRef.value : previewRef.value
  const targetEl = source === 'editor' ? previewRef.value : editorRef.value

  if (sourceEl && targetEl) {
    const sourceScrollRatio = sourceEl.scrollTop / (sourceEl.scrollHeight - sourceEl.clientHeight || 1)
    targetEl.scrollTop = sourceScrollRatio * (targetEl.scrollHeight - targetEl.clientHeight)
  }

  requestAnimationFrame(() => {
    isScrolling = false
  })
}

// 切换全屏
const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  if (isFullscreen.value) {
    fullscreenZoom.value = 1
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
}

// 防抖计时器
let debounceTimer = null

// 初始化 Mermaid
const initMermaid = () => {
  mermaid.initialize({
    startOnLoad: false,
    theme: theme.value,
    securityLevel: 'loose',
    flowchart: {
      useMaxWidth: true,
      htmlLabels: true,
      curve: 'basis'
    },
    sequence: {
      useMaxWidth: true,
      diagramMarginX: 50,
      diagramMarginY: 10,
      actorMargin: 50,
      width: 150,
      height: 65,
      boxMargin: 10,
      boxTextMargin: 5,
      noteMargin: 10,
      messageMargin: 35,
      mirrorActors: true
    },
    gantt: {
      useMaxWidth: true,
      barHeight: 20,
      barGap: 4,
      topPadding: 50,
      leftPadding: 75
    },
    pie: {
      useMaxWidth: true
    },
    er: {
      useMaxWidth: true
    }
  })
}

// 渲染图表
const render = async () => {
  if (!code.value.trim()) {
    svgContent.value = ''
    error.value = ''
    return
  }

  try {
    // 重新初始化以应用主题
    initMermaid()

    const id = `mermaid-${Date.now()}`
    const { svg } = await mermaid.render(id, code.value)
    svgContent.value = svg
    error.value = ''
  } catch (e) {
    error.value = e.message || '未知错误'
    svgContent.value = ''
  }
}

// 防抖渲染
const debouncedRender = () => {
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
  debounceTimer = setTimeout(render, 300)
}

// 重新渲染（主题改变时）
const rerender = () => {
  render()
}

// 设置缩放
const setZoom = (level) => {
  zoomLevel.value = Math.min(10, Math.max(0.25, level))
}

// 缩放控制
const zoomIn = () => {
  setZoom(zoomLevel.value * 1.25)
}

const zoomOut = () => {
  setZoom(zoomLevel.value / 1.25)
}

const resetZoom = () => {
  zoomLevel.value = 1
  panX.value = 0
  panY.value = 0
}

// 鼠标滚轮缩放
const onWheel = (e) => {
  const delta = e.deltaY > 0 ? 0.9 : 1.1
  const newZoom = Math.min(10, Math.max(0.25, zoomLevel.value * delta))
  zoomLevel.value = newZoom
}

// 拖拽平移
const onMouseDown = (e) => {
  if (e.button !== 0) return // 只响应左键
  isDragging.value = true
  dragStartX = e.clientX
  dragStartY = e.clientY
  startPanX = panX.value
  startPanY = panY.value
}

const onMouseMove = (e) => {
  if (!isDragging.value) return
  const dx = e.clientX - dragStartX
  const dy = e.clientY - dragStartY
  panX.value = startPanX + dx
  panY.value = startPanY + dy
}

const onMouseUp = () => {
  isDragging.value = false
}

// 全屏模式缩放控制
const fullscreenZoomIn = () => {
  fullscreenZoom.value = Math.min(10, fullscreenZoom.value * 1.25)
}

const fullscreenZoomOut = () => {
  fullscreenZoom.value = Math.max(0.25, fullscreenZoom.value / 1.25)
}

const resetFullscreenZoom = () => {
  fullscreenZoom.value = 1
  fullscreenPanX.value = 0
  fullscreenPanY.value = 0
}

// 全屏模式滚轮缩放
const onFullscreenWheel = (e) => {
  const delta = e.deltaY > 0 ? 0.9 : 1.1
  fullscreenZoom.value = Math.min(10, Math.max(0.25, fullscreenZoom.value * delta))
}

// 全屏模式拖拽平移
const onFullscreenMouseDown = (e) => {
  if (e.button !== 0) return
  isFullscreenDragging.value = true
  fullscreenDragStartX = e.clientX
  fullscreenDragStartY = e.clientY
  fullscreenStartPanX = fullscreenPanX.value
  fullscreenStartPanY = fullscreenPanY.value
}

const onFullscreenMouseMove = (e) => {
  if (!isFullscreenDragging.value) return
  const dx = e.clientX - fullscreenDragStartX
  const dy = e.clientY - fullscreenDragStartY
  fullscreenPanX.value = fullscreenStartPanX + dx
  fullscreenPanY.value = fullscreenStartPanY + dy
}

const onFullscreenMouseUp = () => {
  isFullscreenDragging.value = false
}

// 图表模板
const templates = {
  flowchart: `flowchart TD
    A[开始] --> B{是否满足条件?}
    B -->|是| C[执行操作 A]
    B -->|否| D[执行操作 B]
    C --> E[处理结果]
    D --> E
    E --> F[结束]`,

  sequence: `sequenceDiagram
    participant 用户
    participant 前端
    participant 后端
    participant 数据库

    用户->>前端: 发起请求
    前端->>后端: API 调用
    后端->>数据库: 查询数据
    数据库-->>后端: 返回结果
    后端-->>前端: 返回响应
    前端-->>用户: 显示结果`,

  classDiagram: `classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
    }
    class Dog {
        +String breed
        +bark()
        +fetch()
    }
    class Cat {
        +String color
        +meow()
        +scratch()
    }
    Animal <|-- Dog
    Animal <|-- Cat`,

  stateDiagram: `stateDiagram-v2
    [*] --> 待处理
    待处理 --> 处理中: 开始处理
    处理中 --> 已完成: 处理成功
    处理中 --> 失败: 处理失败
    失败 --> 处理中: 重试
    已完成 --> [*]`,

  erDiagram: `erDiagram
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--|{ LINE-ITEM : contains
    CUSTOMER {
        string name
        string email
        int customerId
    }
    ORDER {
        int orderId
        date orderDate
        string status
    }
    LINE-ITEM {
        int quantity
        float price
        string productName
    }`,

  gantt: `gantt
    title 项目开发计划
    dateFormat  YYYY-MM-DD
    section 设计阶段
    需求分析           :a1, 2024-01-01, 7d
    系统设计           :a2, after a1, 5d
    UI 设计            :a3, after a1, 7d
    section 开发阶段
    前端开发           :b1, after a2, 14d
    后端开发           :b2, after a2, 14d
    API 联调           :b3, after b1, 5d
    section 测试阶段
    功能测试           :c1, after b3, 7d
    性能测试           :c2, after c1, 3d
    上线部署           :milestone, after c2, 0d`,

  pie: `pie showData
    title 技术栈使用比例
    "Vue.js" : 35
    "React" : 30
    "Angular" : 15
    "Svelte" : 10
    "其他" : 10`,

  mindmap: `mindmap
  root((前端技术))
    框架
      Vue
        Vue 2
        Vue 3
      React
        Hooks
        Redux
      Angular
    构建工具
      Webpack
      Vite
      Rollup
    样式
      CSS
      Sass
      TailwindCSS
    测试
      Jest
      Vitest
      Cypress`,

  timeline: `timeline
    title 互联网发展史
    section 早期
      1969 : ARPANET 诞生
      1971 : 第一封电子邮件
    section WWW 时代
      1989 : WWW 概念提出
      1993 : Mosaic 浏览器
      1995 : JavaScript 诞生
    section 现代
      2004 : Web 2.0 时代
      2008 : HTML5 草案
      2015 : ES6 发布`,

  gitGraph: `gitGraph
    commit id: "初始化项目"
    commit id: "添加基础功能"
    branch develop
    checkout develop
    commit id: "开发新功能"
    commit id: "功能完善"
    checkout main
    merge develop id: "合并开发分支"
    commit id: "发布 v1.0"`,

  quadrant: `quadrantChart
    title 技术评估矩阵
    x-axis 低复杂度 --> 高复杂度
    y-axis 低价值 --> 高价值
    quadrant-1 优先实施
    quadrant-2 需要规划
    quadrant-3 可以考虑
    quadrant-4 暂时搁置
    功能A: [0.3, 0.8]
    功能B: [0.7, 0.9]
    功能C: [0.2, 0.3]
    功能D: [0.8, 0.4]`,

  sankey: `sankey-beta
    访问量,注册,500
    访问量,跳出,300
    注册,活跃用户,350
    注册,流失,150
    活跃用户,付费,200
    活跃用户,免费,150`
}

// 插入模板
const insertTemplate = (type) => {
  code.value = templates[type] || ''
  render()
}

// 内联 SVG 样式，确保导出效果与预览一致
const inlineSvgStyles = (svgElement) => {
  const clonedSvg = svgElement.cloneNode(true)

  // 收集原始 SVG 中的所有样式规则
  const styleSheets = document.styleSheets
  const cssRules = []

  // 从页面样式表中提取相关规则
  for (let i = 0; i < styleSheets.length; i++) {
    try {
      const rules = styleSheets[i].cssRules || styleSheets[i].rules
      if (rules) {
        for (let j = 0; j < rules.length; j++) {
          cssRules.push(rules[j])
        }
      }
    } catch (e) {
      // 跨域样式表无法访问，忽略
    }
  }

  // 获取原始 SVG 内的 style 标签
  const originalStyles = svgElement.querySelectorAll('style')
  const styleTexts = []
  originalStyles.forEach(style => {
    styleTexts.push(style.textContent)
  })

  // 将 style 标签复制到克隆的 SVG 中
  const existingStyles = clonedSvg.querySelectorAll('style')
  existingStyles.forEach(style => style.remove())

  if (styleTexts.length > 0) {
    const newStyle = document.createElementNS('http://www.w3.org/2000/svg', 'style')
    newStyle.textContent = styleTexts.join('\n')
    clonedSvg.insertBefore(newStyle, clonedSvg.firstChild)
  }

  // 需要内联的样式属性列表
  const styleProps = [
    'fill', 'stroke', 'stroke-width', 'stroke-dasharray', 'stroke-linecap', 'stroke-linejoin',
    'font-family', 'font-size', 'font-weight', 'font-style',
    'text-anchor', 'dominant-baseline', 'alignment-baseline',
    'opacity', 'fill-opacity', 'stroke-opacity',
    'transform', 'visibility', 'display',
    'marker-start', 'marker-mid', 'marker-end'
  ]

  // 遍历原始 SVG 和克隆 SVG 中的所有元素
  const originalElements = svgElement.querySelectorAll('*')
  const clonedElements = clonedSvg.querySelectorAll('*')

  originalElements.forEach((origEl, index) => {
    const clonedEl = clonedElements[index]
    if (!clonedEl) return

    // 获取原始元素的计算样式
    const computedStyle = window.getComputedStyle(origEl)

    // 内联关键样式
    styleProps.forEach(prop => {
      const value = computedStyle.getPropertyValue(prop)
      if (value && value !== '' && value !== 'none' && value !== 'normal') {
        // 转换 CSS 属性名为驼峰式（用于 style 对象）
        const camelProp = prop.replace(/-([a-z])/g, (g) => g[1].toUpperCase())
        clonedEl.style[camelProp] = value
      }
    })

    // 特殊处理颜色相关属性
    const fill = computedStyle.getPropertyValue('fill')
    const stroke = computedStyle.getPropertyValue('stroke')

    if (fill && fill !== 'none') {
      clonedEl.setAttribute('fill', fill)
    }
    if (stroke && stroke !== 'none') {
      clonedEl.setAttribute('stroke', stroke)
      const strokeWidth = computedStyle.getPropertyValue('stroke-width')
      if (strokeWidth) {
        clonedEl.setAttribute('stroke-width', strokeWidth)
      }
    }
  })

  // 处理 foreignObject 元素（HTML 内容无法直接转为 canvas）
  const foreignObjects = clonedSvg.querySelectorAll('foreignObject')
  foreignObjects.forEach(fo => {
    // 获取 foreignObject 的位置和尺寸
    const x = parseFloat(fo.getAttribute('x')) || 0
    const y = parseFloat(fo.getAttribute('y')) || 0
    const width = parseFloat(fo.getAttribute('width')) || 100
    const height = parseFloat(fo.getAttribute('height')) || 20

    // 获取内部文本内容
    const htmlContent = fo.querySelector('div, span, p')
    const text = htmlContent?.textContent?.trim() || fo.textContent?.trim() || ''

    if (text) {
      // 创建 SVG text 元素替代 foreignObject
      const g = document.createElementNS('http://www.w3.org/2000/svg', 'g')

      // 添加背景矩形
      const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect')
      rect.setAttribute('x', x)
      rect.setAttribute('y', y)
      rect.setAttribute('width', width)
      rect.setAttribute('height', height)
      rect.setAttribute('fill', '#ffffff')
      rect.setAttribute('stroke', '#333333')
      rect.setAttribute('stroke-width', '1')
      rect.setAttribute('rx', '4')
      g.appendChild(rect)

      // 添加文本
      const textEl = document.createElementNS('http://www.w3.org/2000/svg', 'text')
      textEl.setAttribute('x', x + width / 2)
      textEl.setAttribute('y', y + height / 2)
      textEl.setAttribute('text-anchor', 'middle')
      textEl.setAttribute('dominant-baseline', 'middle')
      textEl.setAttribute('fill', '#333333')
      textEl.setAttribute('font-family', 'Arial, sans-serif')
      textEl.setAttribute('font-size', '14')
      textEl.textContent = text
      g.appendChild(textEl)

      fo.parentNode?.replaceChild(g, fo)
    } else {
      fo.remove()
    }
  })

  return clonedSvg
}

// 导出 PNG
const exportPng = async () => {
  if (!svgContent.value) {
    ElMessage.warning('没有可导出的图表')
    return
  }

  try {
    const svgElement = diagramRef.value?.querySelector('svg')
    if (!svgElement) {
      ElMessage.error('无法获取 SVG 元素')
      return
    }

    // 克隆 SVG（深拷贝）
    const clonedSvg = svgElement.cloneNode(true)

    // 获取 SVG 实际渲染尺寸
    const svgRect = svgElement.getBoundingClientRect()
    let width = svgRect.width
    let height = svgRect.height

    // 尝试从 viewBox 获取尺寸
    const viewBox = svgElement.getAttribute('viewBox')
    if (viewBox) {
      const parts = viewBox.split(/\s+/)
      if (parts.length === 4) {
        const vbWidth = parseFloat(parts[2])
        const vbHeight = parseFloat(parts[3])
        if (vbWidth > width) width = vbWidth
        if (vbHeight > height) height = vbHeight
      }
    }

    // 尝试从 SVG 属性获取尺寸
    const svgWidth = svgElement.getAttribute('width')
    const svgHeight = svgElement.getAttribute('height')
    if (svgWidth && !svgWidth.includes('%')) {
      const w = parseFloat(svgWidth)
      if (w > width) width = w
    }
    if (svgHeight && !svgHeight.includes('%')) {
      const h = parseFloat(svgHeight)
      if (h > height) height = h
    }

    // 添加一些边距
    const padding = 40
    width = Math.max(width, 200) + padding
    height = Math.max(height, 100) + padding

    // 设置克隆 SVG 的尺寸和命名空间
    clonedSvg.setAttribute('width', width)
    clonedSvg.setAttribute('height', height)
    clonedSvg.setAttribute('xmlns', 'http://www.w3.org/2000/svg')
    clonedSvg.setAttribute('xmlns:xlink', 'http://www.w3.org/1999/xlink')

    // 确保有正确的 viewBox
    if (!clonedSvg.getAttribute('viewBox')) {
      clonedSvg.setAttribute('viewBox', `0 0 ${width} ${height}`)
    }

    // 添加白色背景
    const bgRect = document.createElementNS('http://www.w3.org/2000/svg', 'rect')
    bgRect.setAttribute('x', '0')
    bgRect.setAttribute('y', '0')
    bgRect.setAttribute('width', width)
    bgRect.setAttribute('height', height)
    bgRect.setAttribute('fill', '#ffffff')

    // 找到第一个非 style/defs 的元素，在它前面插入背景
    const firstContent = clonedSvg.querySelector(':scope > :not(style):not(defs)')
    if (firstContent) {
      clonedSvg.insertBefore(bgRect, firstContent)
    } else {
      clonedSvg.appendChild(bgRect)
    }

    // 内联所有计算后的样式
    const allElements = clonedSvg.querySelectorAll('*')
    const originalElements = svgElement.querySelectorAll('*')

    allElements.forEach((el, index) => {
      const origEl = originalElements[index]
      if (!origEl) return

      const computedStyle = window.getComputedStyle(origEl)

      // 关键样式属性 - 特别注意文本相关的
      const importantStyles = [
        'fill', 'stroke', 'stroke-width', 'stroke-dasharray',
        'font-family', 'font-size', 'font-weight', 'font-style',
        'text-anchor', 'dominant-baseline', 'alignment-baseline',
        'opacity', 'fill-opacity', 'stroke-opacity',
        'transform', 'display', 'visibility',
        'letter-spacing', 'word-spacing', 'line-height'
      ]

      importantStyles.forEach(prop => {
        const value = computedStyle.getPropertyValue(prop)
        if (value && value !== 'none' && value !== 'normal' && value !== '') {
          // 直接设置属性而不是 style，对于 SVG 更可靠
          if (['fill', 'stroke', 'stroke-width', 'opacity', 'text-anchor', 'dominant-baseline'].includes(prop)) {
            el.setAttribute(prop, value)
          }
          el.style[prop] = value
        }
      })

      // 特殊处理 text 元素
      if (el.tagName === 'text' || el.tagName === 'tspan') {
        // 确保 x, y 坐标被保留
        const x = origEl.getAttribute('x')
        const y = origEl.getAttribute('y')
        const dx = origEl.getAttribute('dx')
        const dy = origEl.getAttribute('dy')
        if (x) el.setAttribute('x', x)
        if (y) el.setAttribute('y', y)
        if (dx) el.setAttribute('dx', dx)
        if (dy) el.setAttribute('dy', dy)
      }
    })

    // 序列化 SVG
    const serializer = new XMLSerializer()
    let svgString = serializer.serializeToString(clonedSvg)

    // 创建 Blob URL（更可靠）
    const svgBlob = new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' })
    const url = URL.createObjectURL(svgBlob)

    // 创建 Image 并等待加载
    const img = new Image()
    img.crossOrigin = 'anonymous'

    img.onload = () => {
      try {
        // 创建 Canvas - 使用高分辨率
        const canvas = document.createElement('canvas')
        const scale = 2 // 高清导出
        canvas.width = width * scale
        canvas.height = height * scale

        const ctx = canvas.getContext('2d')

        // 启用图像平滑
        ctx.imageSmoothingEnabled = true
        ctx.imageSmoothingQuality = 'high'

        // 缩放并绘制白色背景
        ctx.scale(scale, scale)
        ctx.fillStyle = '#ffffff'
        ctx.fillRect(0, 0, width, height)

        // 绘制图像
        ctx.drawImage(img, 0, 0, width, height)

        // 清理 URL
        URL.revokeObjectURL(url)

        // 导出 PNG
        canvas.toBlob((blob) => {
          if (blob) {
            const downloadUrl = URL.createObjectURL(blob)
            const a = document.createElement('a')
            a.href = downloadUrl
            a.download = `mermaid-diagram-${Date.now()}.png`
            a.click()
            URL.revokeObjectURL(downloadUrl)
            ElMessage.success('PNG 图片已下载')
          } else {
            ElMessage.error('PNG 导出失败')
          }
        }, 'image/png', 1.0)
      } catch (err) {
        console.error('Canvas 处理失败:', err)
        URL.revokeObjectURL(url)
        ElMessage.error('PNG 导出失败: ' + err.message)
      }
    }

    img.onerror = (err) => {
      console.error('图片加载失败:', err)
      URL.revokeObjectURL(url)
      ElMessage.error('PNG 导出失败')
    }

    // 加载图片
    img.src = url
  } catch (e) {
    console.error('导出 PNG 失败:', e)
    ElMessage.error('PNG 导出失败: ' + e.message)
  }
}

// 导出 SVG
const exportSvg = () => {
  if (!svgContent.value) {
    ElMessage.warning('没有可导出的图表')
    return
  }

  try {
    const svgElement = diagramRef.value?.querySelector('svg')
    if (!svgElement) {
      ElMessage.error('无法获取 SVG 元素')
      return
    }

    // 克隆并添加背景
    const clonedSvg = svgElement.cloneNode(true)

    // 添加白色背景
    const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect')
    rect.setAttribute('width', '100%')
    rect.setAttribute('height', '100%')
    rect.setAttribute('fill', '#ffffff')
    clonedSvg.insertBefore(rect, clonedSvg.firstChild)

    const serializer = new XMLSerializer()
    let svgString = serializer.serializeToString(clonedSvg)

    // 添加 XML 声明
    svgString = '<?xml version="1.0" encoding="UTF-8"?>\n' + svgString

    const blob = new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `mermaid-diagram-${Date.now()}.svg`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('SVG 文件已下载')
  } catch (e) {
    console.error('导出 SVG 失败:', e)
    ElMessage.error('SVG 导出失败: ' + e.message)
  }
}

// 复制代码
const copyCode = async () => {
  if (!code.value) {
    ElMessage.warning('没有可复制的代码')
    return
  }

  try {
    await navigator.clipboard.writeText(code.value)
    ElMessage.success('代码已复制到剪贴板')
  } catch (e) {
    // 备用方案
    const textarea = document.createElement('textarea')
    textarea.value = code.value
    textarea.style.position = 'fixed'
    textarea.style.left = '-9999px'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    ElMessage.success('代码已复制到剪贴板')
  }
}

// 初始化
onMounted(() => {
  initMermaid()
  // 设置默认示例
  code.value = templates.flowchart
  render()
})

// 监听主题变化
watch(theme, () => {
  rerender()
})
</script>

<style scoped>
.tool-container {
  min-height: 500px;
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
  color: var(--text-primary);
}


.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
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
  line-height: 1.6;
  outline: none;
  tab-size: 4;
}


.code-editor::placeholder {
  color: var(--text-tertiary);
}


.preview-panel {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
}


.preview-content {
  flex: 1;
  padding: 20px;
  overflow: auto;
  background-color: var(--bg-secondary);
  display: flex;
  justify-content: center;
  align-items: flex-start;
}


.diagram-container {
  background-color: var(--bg-primary);
  padding: 20px;
  border-radius: var(--radius-md);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  min-width: 200px;
}

.diagram-container :deep(svg) {
  max-width: 100%;
  height: auto;
}

.error-message {
  display: flex;
  gap: 12px;
  padding: 20px;
  background-color: #fef0f0;
  border: 1px solid #fbc4c4;
  border-radius: var(--radius-md);
  color: #c53929;
  max-width: 100%;
}

.error-message .el-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.error-title {
  font-weight: 600;
  margin-bottom: 8px;
}

.error-detail {
  margin: 0;
  font-size: 12px;
  font-family: var(--font-family-mono);
  white-space: pre-wrap;
  word-break: break-word;
  background-color: rgba(0, 0, 0, 0.05);
  padding: 8px;
  border-radius: var(--radius-sm);
  max-height: 200px;
  overflow-y: auto;
}

/* 响应式设计 */
@media (max-width: 900px) {
  .editor-container {
    grid-template-columns: 1fr;
    grid-template-rows: 1fr 1fr;
  }

  .actions {
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 600px) {
  .tool-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .actions {
    flex-wrap: wrap;
  }

  .panel-header {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
}

.preview-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

.zoom-display {
  min-width: 50px;
  text-align: center;
  font-size: 12px;
  color: #606266;
}

.zoom-hint {
  font-weight: normal;
  color: var(--text-tertiary);
  font-size: 12px;
}

.preview-content.is-dragging,
.fullscreen-content.is-dragging {
  cursor: grabbing !important;
}

.preview-content:not(.is-dragging),
.fullscreen-content:not(.is-dragging) {
  cursor: grab;
}

/* 全屏预览样式 */
.fullscreen-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.85);
  z-index: 9999;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
}

.fullscreen-container {
  width: 100%;
  height: 100%;
  background-color: var(--bg-primary);
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.fullscreen-header {
  padding: 12px 20px;
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;
  border-bottom: 1px solid #e0e0e0;
}


.fullscreen-header span {
  font-size: 16px;
  font-weight: 500;
}

.fullscreen-controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.fullscreen-content {
  flex: 1;
  overflow: auto;
  padding: 40px;
  background-color: #f0f0f0;
  display: flex;
  justify-content: center;
  align-items: flex-start;
}

.fullscreen-diagram {
  background-color: var(--bg-primary);
  padding: 40px;
  border-radius: var(--radius-md);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.fullscreen-diagram :deep(svg) {
  max-width: 100%;
  height: auto;
}
</style>
