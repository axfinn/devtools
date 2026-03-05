<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>背景图库</h2>
      <div class="header-actions">
        <el-button type="warning" :icon="Refresh" @click="cacheImages" :loading="caching">缓存图片</el-button>
        <el-button type="primary" :icon="Refresh" @click="loadImages">刷新</el-button>
        <el-button :icon="Document" @click="showApiDocs = true">API 文档</el-button>
      </div>
    </div>

    <div class="stats-bar">
      <el-tag type="success">已缓存: {{ stats.cached }}/{{ stats.max }}</el-tag>
      <el-tag v-if="hasExternal" type="info">外部图: {{ externalCount }}</el-tag>
    </div>

    <div class="filter-bar">
      <el-radio-group v-model="filterSource" @change="filterImages">
        <el-radio-button value="all">全部</el-radio-button>
        <el-radio-button value="cached">缓存图</el-radio-button>
        <el-radio-button v-if="hasExternal" value="external">外部图</el-radio-button>
      </el-radio-group>
      <span class="image-count">共 {{ filteredImages.length }} 张图片</span>
    </div>

    <div v-loading="loading" class="image-grid">
      <div
        v-for="img in paginatedImages"
        :key="img.id"
        class="image-card"
        @click="openPreview(img)"
      >
        <div class="image-wrapper">
          <img :src="img.cached_url || img.url" :alt="img.filename" loading="lazy" @error="handleImageError($event, img)" />
          <div class="image-overlay">
            <el-icon><ZoomIn /></el-icon>
          </div>
        </div>
        <div class="image-info">
          <span class="source-tag" :class="img.source">{{ getSourceLabel(img.source) }}</span>
          <span class="image-id">{{ img.id }}</span>
        </div>
      </div>
    </div>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[12, 24, 48, 100]"
        :total="filteredImages.length"
        layout="total, sizes, prev, pager, next"
        background
      />
    </div>

    <!-- 预览对话框 -->
    <el-dialog
      v-model="previewVisible"
      :title="previewImage?.filename || previewImage?.id"
      width="90%"
      destroy-on-close
    >
      <div class="preview-container">
        <img v-if="previewImage" :src="previewImage.cached_url || previewImage.url" :alt="previewImage.filename" />
      </div>
      <template #footer>
        <div class="preview-actions">
          <el-button :icon="CopyDocument" @click="copyUrl">复制 URL</el-button>
          <el-button type="primary" :icon="Download" @click="downloadImage">下载</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- API 文档对话框 -->
    <el-dialog v-model="showApiDocs" title="背景图 API 文档" width="600px">
      <div class="api-docs">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="获取图片列表">
            <code>GET /api/bg</code>
          </el-descriptions-item>
          <el-descriptions-item label="缓存图片">
            <code>POST /api/bg/cache</code>
          </el-descriptions-item>
          <el-descriptions-item label="随机替换">
            <code>POST /api/bg/replace?count=10</code>
          </el-descriptions-item>
          <el-descriptions-item label="随机图片">
            <code>GET /api/bg/random</code>
          </el-descriptions-item>
          <el-descriptions-item label="获取缓存图片">
            <code>GET /api/bg/cached/:filename</code>
          </el-descriptions-item>
        </el-descriptions>
        <div class="api-note">
          <p><strong>说明：</strong></p>
          <ul>
            <li>缓存图优先返回本地缓存的 URL</li>
            <li>外部图返回 picsum.photos 随机图</li>
            <li>本地存储路径：./backend/data/backgrounds/</li>
          </ul>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, ZoomIn, CopyDocument, Download, Document } from '@element-plus/icons-vue'

const loading = ref(false)
const caching = ref(false)
const images = ref([])
const stats = ref({ cached: 0, max: 1000 })
const filterSource = ref('all')
const previewVisible = ref(false)
const currentPage = ref(1)
const pageSize = ref(24)
const previewImage = ref(null)
const showApiDocs = ref(false)

const filteredImages = computed(() => {
  if (filterSource.value === 'all') return images.value
  return images.value.filter(img => img.source === filterSource.value)
})

const hasExternal = computed(() => images.value.some(img => img.source === 'external'))
const externalCount = computed(() => images.value.filter(img => img.source === 'external').length)

const paginatedImages = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredImages.value.slice(start, end)
})

const getSourceLabel = (source) => {
  const labels = { cached: '缓存', external: '外部' }
  return labels[source] || source
}

const loadImages = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/bg')
    const data = await res.json()
    images.value = data.images || []
    stats.value = { cached: data.cached || 0, max: data.max_cache || 1000 }
  } catch (err) {
    ElMessage.error('加载图片失败')
  } finally {
    loading.value = false
  }
}

const cacheImages = async () => {
  caching.value = true
  try {
    const res = await fetch('/api/bg/cache', { method: 'POST' })
    const data = await res.json()
    ElMessage.success(data.message || '缓存完成')
    loadImages()
  } catch (err) {
    ElMessage.error('缓存失败')
  } finally {
    caching.value = false
  }
}

const filterImages = () => {}

const openPreview = (img) => {
  previewImage.value = img
  previewVisible.value = true
}

const handleImageError = (event, img) => {
  if (img.cached_url && img.url) {
    event.target.src = img.url
  }
}

const copyUrl = async () => {
  if (!previewImage.value) return
  try {
    const relativeUrl = previewImage.value.cached_url || previewImage.value.url
    const url = relativeUrl.startsWith('http') ? relativeUrl : window.location.origin + relativeUrl
    await navigator.clipboard.writeText(url)
    ElMessage.success('URL 已复制')
  } catch (err) {
    ElMessage.error('复制失败')
  }
}

const downloadImage = async () => {
  if (!previewImage.value) return
  try {
    const url = previewImage.value.cached_url || previewImage.value.url
    const response = await fetch(url)
    const blob = await response.blob()
    const blobUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = previewImage.value.filename || `bg-${previewImage.value.id}.jpg`
    link.click()
    URL.revokeObjectURL(blobUrl)
    ElMessage.success('下载成功')
  } catch (err) {
    ElMessage.error('下载失败')
  }
}

onMounted(() => {
  loadImages()
})
</script>

<style scoped>
.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.stats-bar {
  margin-bottom: 15px;
  display: flex;
  gap: 10px;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 20px;
}

.image-count {
  color: var(--text-secondary);
  font-size: 14px;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}

.image-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.image-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.image-wrapper {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
}

.image-wrapper img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.image-card:hover .image-wrapper img {
  transform: scale(1.05);
}

.image-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.3s;
}

.image-card:hover .image-overlay {
  opacity: 1;
}

.image-overlay .el-icon {
  font-size: 32px;
  color: white;
}

.image-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
}

.source-tag {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
}

.source-tag.cached {
  background: var(--color-success);
  color: white;
}

.source-tag.external {
  background: #909399;
  color: white;
}

.image-id {
  font-size: 12px;
  color: var(--text-secondary);
}

.preview-container {
  display: flex;
  justify-content: center;
  background: #000;
}

.preview-container img {
  max-width: 100%;
  max-height: 70vh;
  object-fit: contain;
}

.preview-actions {
  display: flex;
  justify-content: center;
  gap: 10px;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  padding: 20px 0;
}

.api-docs {
  font-size: 14px;
}

.api-docs code {
  background: var(--card-bg);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: var(--font-family-mono);
}

.api-note {
  margin-top: 20px;
  padding: 15px;
  background: var(--card-bg);
  border-radius: var(--radius-md);
}

.api-note ul {
  margin: 10px 0 0 0;
  padding-left: 20px;
}
</style>
