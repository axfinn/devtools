<template>
  <div class="share-page">
    <div v-if="loading" class="state-box">
      <el-icon class="spin"><Loading /></el-icon>
      <span>照片墙加载中...</span>
    </div>

    <div v-else-if="error" class="state-box">
      <el-result icon="error" :title="error">
        <template #extra>
          <el-button type="primary" @click="$router.push('/photowall')">返回照片墙</el-button>
        </template>
      </el-result>
    </div>

    <template v-else>
      <div class="share-hero">
        <div>
          <div class="share-kicker">分享照片墙</div>
          <h1>{{ data.title || '照片墙' }}</h1>
          <div class="share-meta">
            <el-tag type="info">照片 {{ filteredItems.length }}</el-tag>
            <el-tag v-if="data.is_permanent" type="success">永久</el-tag>
            <el-tag v-else type="warning">到期 {{ formatDate(data.expires_at) }}</el-tag>
          </div>
        </div>
        <el-button @click="copyLink">复制链接</el-button>
      </div>

      <div class="filters">
        <el-select v-model="activeCategory" style="width: 180px">
          <el-option label="全部分类" value="all" />
          <el-option v-for="category in data.categories" :key="category" :label="category" :value="category" />
        </el-select>
        <el-select v-model="activeMonth" style="width: 180px">
          <el-option label="全部时间" value="all" />
          <el-option v-for="item in data.timeline" :key="item.month" :label="`${item.month} (${item.count})`" :value="item.month" />
        </el-select>
      </div>

      <div v-if="filteredItems.length === 0" class="state-box">
        <el-empty description="当前筛选下没有照片" />
      </div>

      <div v-else class="share-grid">
        <figure v-for="item in filteredItems" :key="item.id" class="share-item">
          <img :src="item.image_url" :alt="item.title || data.title" @click="previewUrl = item.image_url" />
          <figcaption>
            <div class="item-title">{{ item.title || '未命名照片' }}</div>
            <div class="item-meta">
              <span v-if="item.category">{{ item.category }}</span>
              <span>{{ formatDate(item.taken_at || item.created_at) }}</span>
            </div>
            <div v-if="item.description" class="item-desc">{{ item.description }}</div>
          </figcaption>
        </figure>
      </div>
    </template>

    <el-image-viewer v-if="previewUrl" :url-list="[previewUrl]" @close="previewUrl = ''" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElImageViewer, ElMessage } from 'element-plus'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const previewUrl = ref('')
const activeCategory = ref('all')
const activeMonth = ref('all')
const data = ref({
  title: '',
  items: [],
  categories: [],
  timeline: [],
  expires_at: null,
  is_permanent: false
})

const filteredItems = computed(() => {
  return (data.value.items || []).filter(item => {
    if (activeCategory.value !== 'all' && item.category !== activeCategory.value) return false
    if (activeMonth.value !== 'all' && monthOf(item.taken_at || item.created_at) !== activeMonth.value) return false
    return true
  })
})

onMounted(loadShare)

async function loadShare() {
  const id = route.params.id
  const key = route.query.key
  if (!id || !key) {
    error.value = '缺少分享密钥'
    loading.value = false
    return
  }
  try {
    const res = await fetch(`/api/photowall/share/${id}?key=${encodeURIComponent(key)}`)
    const payload = await res.json()
    if (!res.ok) throw new Error(payload.error || '加载失败')
    data.value = payload
  } catch (err) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(window.location.href)
    ElMessage.success('链接已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

function formatDate(value) {
  if (!value) return '未设置'
  return new Date(value).toLocaleString('zh-CN')
}

function monthOf(value) {
  return new Date(value).toISOString().slice(0, 7)
}
</script>

<style scoped>
.share-page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px;
}

.share-hero {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 20px;
}

.share-kicker {
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.share-hero h1 {
  margin: 6px 0 0;
  font-size: 32px;
}

.share-meta,
.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.share-meta {
  margin-top: 14px;
}

.filters {
  margin-bottom: 20px;
}

.state-box {
  min-height: 50vh;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 12px;
}

.spin {
  font-size: 32px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.share-grid {
  columns: 280px;
  column-gap: 16px;
}

.share-item {
  break-inside: avoid;
  margin: 0 0 16px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-color);
  border-radius: 18px;
  padding: 12px;
}

.share-item img {
  width: 100%;
  border-radius: 12px;
  display: block;
  cursor: zoom-in;
}

.share-item figcaption {
  margin-top: 10px;
}

.item-title {
  font-weight: 700;
  margin-bottom: 6px;
}

.item-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.item-desc {
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

@media (max-width: 768px) {
  .share-page {
    padding: 16px;
  }

  .share-hero {
    flex-direction: column;
  }
}
</style>
