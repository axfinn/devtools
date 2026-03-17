<template>
  <div class="pdf-viewer-container">
    <!-- 工具栏 -->
    <div class="pdf-toolbar">
      <div class="toolbar-left">
        <el-button-group>
          <el-button :disabled="currentPage <= 1" @click="prevPage" size="small">
            <el-icon><ArrowLeft /></el-icon>
            上一页
          </el-button>
          <el-button :disabled="currentPage >= totalPages" @click="nextPage" size="small">
            下一页
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </el-button-group>
        <span class="page-info">
          {{ currentPage }} / {{ totalPages || '-' }}
        </span>
      </div>

      <div class="toolbar-center">
        <el-button-group>
          <el-button :disabled="scale <= 0.5" @click="zoomOut" size="small">
            <el-icon><ZoomOut /></el-icon>
          </el-button>
          <el-button disabled size="small" class="scale-display">
            {{ Math.round(scale * 100) }}%
          </el-button>
          <el-button :disabled="scale >= 3" @click="zoomIn" size="small">
            <el-icon><ZoomIn /></el-icon>
          </el-button>
        </el-button-group>
      </div>

      <div class="toolbar-right">
        <el-button @click="fitWidth" size="small" title="适合宽度">
          <el-icon><FullScreen /></el-icon>
          适合宽度
        </el-button>
        <el-button @click="downloadPdf" size="small" title="下载">
          <el-icon><Download /></el-icon>
          下载
        </el-button>
      </div>
    </div>

    <!-- PDF 渲染区域 -->
    <div class="pdf-content" ref="pdfContainer">
      <div v-if="loading" class="pdf-loading">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <span>加载中...</span>
      </div>

      <div v-else-if="error" class="pdf-error">
        <el-result icon="error" title="加载失败">
          <template #sub-title>
            <p>{{ error }}</p>
          </template>
          <template #extra>
            <el-button type="primary" @click="retryLoad">重试</el-button>
            <el-button @click="downloadPdf">下载文件</el-button>
          </template>
        </el-result>
      </div>

      <vue-pdf-embed
        v-else
        :source="pdfSource"
        :page="currentPage"
        :scale="scale"
        :fit-parent="fitParent"
        @loaded="onPdfLoaded"
        @error="onPdfError"
        class="pdf-embed"
      />
    </div>

    <!-- 底部页码跳转 -->
    <div class="pdf-footer" v-if="totalPages > 0">
      <el-input-number
        v-model="pageInput"
        :min="1"
        :max="totalPages"
        size="small"
        @keyup.enter="goToPage"
      />
      <el-button size="small" @click="goToPage">跳转</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading, ArrowLeft, ArrowRight, ZoomIn, ZoomOut, FullScreen, Download } from '@element-plus/icons-vue'
import VuePdfEmbed from 'vue-pdf-embed'

const props = defineProps({
  url: {
    type: String,
    required: true
  },
  filename: {
    type: String,
    default: 'document.pdf'
  }
})

const emit = defineEmits(['download'])

// 状态
const pdfSource = ref('')
const currentPage = ref(1)
const totalPages = ref(0)
const scale = ref(1.0)
const fitParent = ref(false)
const loading = ref(true)
const error = ref('')
const pageInput = ref(1)
const pdfContainer = ref(null)

// 监听 URL 变化
watch(() => props.url, (newUrl) => {
  if (newUrl) {
    loadPdf(newUrl)
  }
}, { immediate: true })

// 加载 PDF
const loadPdf = (url) => {
  loading.value = true
  error.value = ''
  currentPage.value = 1
  totalPages.value = 0
  scale.value = 1.0
  pdfSource.value = url
}

// PDF 加载完成
const onPdfLoaded = (pdf) => {
  loading.value = false
  totalPages.value = pdf.numPages
  pageInput.value = 1
}

// PDF 加载错误
const onPdfError = (err) => {
  loading.value = false
  error.value = 'PDF 加载失败，请尝试下载后查看'
  console.error('PDF load error:', err)
}

// 重试加载
const retryLoad = () => {
  loadPdf(props.url)
}

// 上一页
const prevPage = () => {
  if (currentPage.value > 1) {
    currentPage.value--
    pageInput.value = currentPage.value
  }
}

// 下一页
const nextPage = () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
    pageInput.value = currentPage.value
  }
}

// 放大
const zoomIn = () => {
  if (scale.value < 3) {
    scale.value = Math.min(3, scale.value + 0.25)
    fitParent.value = false
  }
}

// 缩小
const zoomOut = () => {
  if (scale.value > 0.5) {
    scale.value = Math.max(0.5, scale.value - 0.25)
    fitParent.value = false
  }
}

// 适合宽度
const fitWidth = () => {
  fitParent.value = true
  scale.value = 1.0
}

// 跳转页码
const goToPage = () => {
  if (pageInput.value >= 1 && pageInput.value <= totalPages.value) {
    currentPage.value = pageInput.value
  }
}

// 下载 PDF
const downloadPdf = () => {
  emit('download', props.url)
}

// 监听键盘事件
onMounted(() => {
  const handleKeydown = (e) => {
    if (e.key === 'ArrowLeft') {
      prevPage()
    } else if (e.key === 'ArrowRight') {
      nextPage()
    } else if (e.key === '+' || e.key === '=') {
      zoomIn()
    } else if (e.key === '-') {
      zoomOut()
    }
  }

  document.addEventListener('keydown', handleKeydown)

  return () => {
    document.removeEventListener('keydown', handleKeydown)
  }
})
</script>

<style scoped>
.pdf-viewer-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--bg-secondary, #f5f5f5);
}

.pdf-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 15px;
  background-color: var(--bg-primary, #fff);
  border-bottom: 1px solid var(--border-base, #e0e0e0);
  gap: 10px;
  flex-wrap: wrap;
}

.toolbar-left,
.toolbar-center,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.page-info {
  font-size: 14px;
  color: var(--text-secondary, #666);
  min-width: 60px;
}

.scale-display {
  min-width: 60px;
}

.pdf-content {
  flex: 1;
  overflow: auto;
  display: flex;
  justify-content: center;
  padding: 20px;
  background-color: #525659;
}

.pdf-embed {
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
}

.pdf-loading,
.pdf-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: #fff;
}

.pdf-loading span {
  margin-top: 10px;
}

.pdf-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 10px;
  background-color: var(--bg-primary, #fff);
  border-top: 1px solid var(--border-base, #e0e0e0);
}

@media (max-width: 768px) {
  .pdf-toolbar {
    flex-direction: column;
    gap: 10px;
  }

  .toolbar-left,
  .toolbar-center,
  .toolbar-right {
    width: 100%;
    justify-content: center;
  }

  .pdf-content {
    padding: 10px;
  }
}
</style>
