<template>
  <div class="tool-container">
    <div class="tool-header news-header">
      <div>
        <h2>每日新闻</h2>
        <span class="news-sub" v-if="config">{{ config.today }} · 每日 {{ config.schedule }} 自动生成{{ config.has_ai ? ' · AI 精编' : '' }}</span>
      </div>
      <div class="news-actions">
        <el-button :icon="Refresh" :loading="loading" @click="refresh">刷新</el-button>
        <el-button type="primary" :icon="Calendar" :loading="generating" @click="openGenerate">立即生成</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" class="news-tip">
      <template #title>
        <el-icon><InfoFilled /></el-icon>
        <span>后端每日定时抓取 RSS 资讯并(可选)经 AI 精编成「今日要闻」；历史各日期归档均保留可查。手动生成需服务端开启本模块。</span>
      </template>
    </el-alert>

    <div class="news-body">
      <!-- 归档侧栏(桌面端) -->
      <aside class="news-archive" v-if="!isMobile">
        <div class="archive-title">历史归档</div>
        <el-scrollbar height="calc(100vh - 280px)">
          <div
            v-for="d in dates"
            :key="d.date"
            class="archive-item"
            :class="{ active: currentDate === d.date }"
            @click="loadByDate(d.date)"
          >
            <div class="archive-date">{{ d.date_label || d.date }}</div>
            <div class="archive-meta">{{ d.item_count }} 条 · {{ d.source_count }} 源</div>
          </div>
          <el-empty v-if="dates.length === 0" description="暂无归档" :image-size="60" />
        </el-scrollbar>
      </aside>

      <!-- 移动端归档选择 -->
      <div v-else class="news-archive-mobile">
        <el-select v-model="currentDate" placeholder="选择日期" @change="loadByDate" style="width: 100%">
          <el-option v-for="d in dates" :key="d.date" :label="(d.date_label || d.date) + ' (' + d.item_count + '条)'" :value="d.date" />
        </el-select>
      </div>

      <!-- 内容区 -->
      <section class="news-content">
        <el-skeleton v-if="loading" :rows="10" animated />

        <el-empty v-else-if="!current" description="今日新闻尚未生成，点「立即生成」或在每日定时任务后自动出现" />

        <template v-else>
          <div class="news-date-head">
            <h3>{{ current.date_label || current.date }}</h3>
            <span class="news-status" :class="current.status">
              {{ statusText(current.status) }}
              <template v-if="current.ai_model"> · {{ current.ai_model }}</template>
            </span>
          </div>

          <!-- AI 今日要闻 -->
          <el-card v-if="current.summary" class="summary-card" shadow="never">
            <template #header><span class="summary-title">今日要闻 · AI 简报</span></template>
            <div class="summary-body">{{ current.summary }}</div>
          </el-card>

          <!-- 按来源分组 -->
          <div v-for="(group, src) in groupedItems" :key="src" class="source-group">
            <div class="source-name"><el-tag size="small" type="info">{{ src }}</el-tag></div>
            <el-card v-for="(it, i) in group" :key="i" class="news-item" shadow="hover">
              <a :href="it.link" target="_blank" rel="noopener noreferrer" class="news-title">{{ it.title }}</a>
              <div class="news-summary" v-if="it.summary">{{ it.summary }}</div>
              <div class="news-meta" v-if="it.published">{{ formatTime(it.published) }}</div>
            </el-card>
          </div>
        </template>
      </section>
    </div>

    <!-- 立即生成弹窗 -->
    <el-dialog v-model="genVisible" title="立即生成今日新闻" width="420px">
      <p class="gen-hint">调用后端手动生成当前日期的新闻归档(等同于每日定时任务)。</p>
      <el-input
        v-model="adminPw"
        type="password"
        show-password
        placeholder="管理员密码(若服务端未设则可留空)"
        style="margin-top: 8px"
      />
      <template #footer>
        <el-button @click="genVisible = false">取消</el-button>
        <el-button type="primary" :loading="generating" @click="doGenerate">生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Refresh, Calendar, InfoFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const config = ref(null)
const dates = ref([])
const current = ref(null)
const currentDate = ref('')
const loading = ref(false)
const generating = ref(false)
const genVisible = ref(false)
const adminPw = ref(localStorage.getItem('news_admin_pw') || '')
const isMobile = ref(window.innerWidth <= 860)

let resizeHandler = null

function statusText(s) {
  return { ok: '已生成', partial: '已生成(无AI精编)', empty: '当日无新内容' }[s] || s || '已生成'
}

function formatTime(iso) {
  const d = new Date(iso)
  if (isNaN(d)) return ''
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const groupedItems = computed(() => {
  if (!current.value || !current.value.items) return {}
  let items = []
  try {
    items = typeof current.value.items === 'string' ? JSON.parse(current.value.items) : current.value.items
  } catch (e) {
    items = []
  }
  const groups = {}
  for (const it of items) {
    const src = it.source || '未知来源'
    if (!groups[src]) groups[src] = []
    groups[src].push(it)
  }
  return groups
})

async function loadConfig() {
  try {
    const res = await fetch('/api/news/config')
    if (res.ok) config.value = await res.json()
  } catch (e) { /* 模块未启用时忽略 */ }
}

async function loadDates() {
  try {
    const res = await fetch('/api/news/list?limit=60')
    if (!res.ok) return
    const data = await res.json()
    dates.value = data.list || []
  } catch (e) { /* ignore */ }
}

async function loadLatest() {
  loading.value = true
  try {
    const res = await fetch('/api/news/latest')
    const data = await res.json()
    if (data.config) config.value = data.config
    if (data.digest) {
      current.value = data.digest
      currentDate.value = data.digest.date
    } else {
      current.value = null
    }
  } catch (e) {
    current.value = null
  } finally {
    loading.value = false
  }
}

async function loadByDate(date) {
  if (!date) return
  loading.value = true
  try {
    const res = await fetch(`/api/news/${date}`)
    if (!res.ok) {
      ElMessage.warning('该日期暂无新闻归档')
      return
    }
    const data = await res.json()
    current.value = data.digest
    currentDate.value = date
  } catch (e) {
    ElMessage.error('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await Promise.all([loadConfig(), loadDates(), loadLatest()])
}

function openGenerate() {
  genVisible.value = true
}

async function doGenerate() {
  generating.value = true
  try {
    const headers = {}
    if (adminPw.value) {
      headers['X-Admin-Password'] = adminPw.value
      localStorage.setItem('news_admin_pw', adminPw.value)
    }
    const res = await fetch('/api/news/generate', { method: 'POST', headers })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      ElMessage.error(err.error || '生成失败')
      return
    }
    ElMessage.success('已生成今日新闻')
    genVisible.value = false
    await Promise.all([loadDates(), loadLatest()])
  } catch (e) {
    ElMessage.error('生成失败: ' + e.message)
  } finally {
    generating.value = false
  }
}

onMounted(() => {
  refresh()
  resizeHandler = () => { isMobile.value = window.innerWidth <= 860 }
  window.addEventListener('resize', resizeHandler)
})

onBeforeUnmount(() => {
  if (resizeHandler) window.removeEventListener('resize', resizeHandler)
})
</script>

<style scoped>
.news-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.news-sub {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
.news-actions {
  display: flex;
  gap: 8px;
}
.news-tip {
  margin-bottom: 16px;
}
.news-body {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.news-archive {
  width: 240px;
  flex-shrink: 0;
  background: var(--el-bg-color-page, #f5f7fa);
  border-radius: 10px;
  padding: 12px;
  max-height: calc(100vh - 260px);
}
.archive-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 10px;
}
.archive-item {
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 6px;
  transition: background 0.15s;
}
.archive-item:hover { background: var(--el-fill-color-light); }
.archive-item.active {
  background: var(--el-color-primary-light-9, #ecf5ff);
  outline: 1px solid var(--el-color-primary-light-5, #a0cfff);
}
.archive-date { font-size: 14px; font-weight: 500; }
.archive-meta { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; }
.news-content { flex: 1; min-width: 0; }
.news-date-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.news-date-head h3 { margin: 0; }
.news-status { font-size: 12px; color: var(--el-text-color-secondary); }
.news-status.empty { color: var(--el-color-warning); }
.news-status.partial { color: var(--el-color-warning); }
.summary-card {
  margin-bottom: 18px;
  border-left: 3px solid var(--el-color-primary);
}
.summary-title { font-weight: 600; }
.summary-body {
  white-space: pre-wrap;
  line-height: 1.7;
  font-size: 14px;
}
.source-group { margin-bottom: 18px; }
.source-name { margin-bottom: 8px; }
.news-item { margin-bottom: 10px; }
.news-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-color-primary);
  text-decoration: none;
  line-height: 1.5;
}
.news-title:hover { text-decoration: underline; }
.news-summary {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-top: 6px;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.news-meta { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 6px; }
.news-archive-mobile { margin-bottom: 12px; }

@media (max-width: 860px) {
  .news-body { flex-direction: column; }
  .news-archive { width: 100%; max-height: none; }
}
</style>
