<template>
  <div class="minimax-share-page">
    <div v-if="loading" class="status-shell">
      <el-icon class="is-loading" style="font-size: 34px;"><Loading /></el-icon>
      <span>加载分享内容...</span>
    </div>

    <div v-else-if="error" class="status-shell error-shell">
      <el-icon style="font-size: 34px;"><CircleCloseFilled /></el-icon>
      <h3>{{ error }}</h3>
      <el-button type="primary" @click="$router.push('/minimax-studio')">返回 MiniMax Studio</el-button>
    </div>

    <template v-else>
      <section class="share-hero">
        <div class="hero-main">
          <p class="eyebrow">MINIMAX RESULT SHARE</p>
          <h1>{{ share.title || '未命名结果分享' }}</h1>
          <p class="hero-summary">{{ share.summary || '这个分享已被保存到本地，可公开查看，也可直接播放媒体产物。' }}</p>
          <div class="hero-meta">
            <el-tag effect="dark" type="success">{{ share.result_type || 'result' }}</el-tag>
            <el-tag>{{ share.model || '未标注模型' }}</el-tag>
            <span>{{ formatTime(share.created_at) }}</span>
            <span>{{ share.assets?.length || 0 }} 个资产</span>
          </div>
        </div>
        <div class="hero-actions">
          <el-button type="primary" @click="copyText(currentPageURL)">复制页面链接</el-button>
          <el-button @click="copyText(prettyPayload)">复制原始结果 JSON</el-button>
        </div>
      </section>

      <section class="content-grid">
        <el-card class="share-card" shadow="never">
          <template #header>
            <div class="card-head">
              <span>结果摘要</span>
              <el-tag size="small">{{ summaryLabel }}</el-tag>
            </div>
          </template>

          <div v-if="primaryText" class="text-block">{{ primaryText }}</div>
          <div v-else-if="featureId" class="feature-block">{{ featureId }}</div>
          <div v-else class="empty-block">该结果没有可直接提取的文本摘要，请查看下方原始结果。</div>
        </el-card>

        <el-card class="share-card" shadow="never">
          <template #header>
            <div class="card-head">
              <span>公开资产</span>
              <span class="asset-tip">保存后走本地分享资产，可直接预览或播放</span>
            </div>
          </template>

          <div v-if="share.assets?.length" class="asset-grid">
            <div v-for="asset in share.assets" :key="asset.id" class="asset-card">
              <img v-if="asset.kind === 'image'" :src="asset.asset_url" :alt="asset.filename" />
              <video v-else-if="asset.kind === 'video'" :src="asset.asset_url" controls playsinline />
              <audio v-else-if="asset.kind === 'audio'" :src="asset.asset_url" controls />
              <div v-else class="file-card">
                <strong>{{ asset.filename }}</strong>
                <p>{{ asset.content_type || 'application/octet-stream' }}</p>
              </div>
              <div class="asset-actions">
                <span class="asset-name">{{ asset.filename }}</span>
                <el-button text type="primary" @click="openLink(asset.asset_url)">打开</el-button>
              </div>
            </div>
          </div>
          <div v-else class="empty-block">这个分享不包含媒体资产。</div>
        </el-card>
      </section>

      <el-card class="share-card" shadow="never">
        <template #header>
          <div class="card-head">
            <span>原始结果</span>
            <el-button text @click="copyText(prettyPayload)">复制 JSON</el-button>
          </div>
        </template>
        <pre class="payload-block">{{ prettyPayload }}</pre>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const share = ref({})
const currentPageURL = window.location.href

const payload = computed(() => share.value?.payload || {})
const prettyPayload = computed(() => JSON.stringify(payload.value || {}, null, 2))
const primaryText = computed(() => {
  const raw = payload.value || {}
  return raw.text || raw.lyrics || raw.full_lyrics || raw.summary || raw.prompt || ''
})
const featureId = computed(() => {
  const raw = payload.value || {}
  return raw.cover_feature_id || raw.feature_id || ''
})
const summaryLabel = computed(() => {
  if (primaryText.value) return '文本摘要'
  if (featureId.value) return 'Feature ID'
  return '结构化结果'
})

onMounted(() => {
  void loadShare()
})

async function loadShare() {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch(`/api/minimax/result-shares/${encodeURIComponent(route.params.id)}`)
    const data = await safeJson(res)
    if (!res.ok) throw new Error(data.error || '加载分享失败')
    share.value = data
  } catch (err) {
    error.value = err.message || '加载分享失败'
  } finally {
    loading.value = false
  }
}

function formatTime(value) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

function copyText(text) {
  navigator.clipboard.writeText(text || '')
    .then(() => ElMessage.success('已复制'))
    .catch(() => ElMessage.error('复制失败'))
}

function openLink(url) {
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

async function safeJson(res) {
  try {
    return await res.json()
  } catch {
    return {}
  }
}
</script>

<style scoped>
.minimax-share-page {
  min-height: 100vh;
  padding: 18px;
  background:
    radial-gradient(circle at top left, rgba(14, 116, 144, 0.12), transparent 24%),
    radial-gradient(circle at top right, rgba(249, 115, 22, 0.12), transparent 24%),
    linear-gradient(180deg, #f8fafc 0%, #eef6ff 100%);
}

.status-shell {
  min-height: calc(100vh - 36px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  color: #334155;
}

.error-shell {
  color: #b91c1c;
}

.share-hero {
  display: grid;
  grid-template-columns: 1.4fr minmax(240px, 360px);
  gap: 18px;
  margin-bottom: 18px;
}

.hero-main,
.hero-actions,
.share-card {
  border: 1px solid rgba(15, 23, 42, 0.08);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.06);
}

.hero-main {
  padding: 26px;
  border-radius: 28px;
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.95), rgba(30, 64, 175, 0.94));
  color: #f8fafc;
}

.eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.18em;
  opacity: 0.8;
}

.hero-main h1 {
  margin: 0;
  font-size: 34px;
  line-height: 1.15;
}

.hero-summary {
  margin: 14px 0 0;
  line-height: 1.8;
  color: rgba(248, 250, 252, 0.92);
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
  color: rgba(248, 250, 252, 0.85);
  font-size: 13px;
}

.hero-actions {
  padding: 20px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.92);
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 12px;
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-bottom: 18px;
}

.share-card {
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.92);
}

.card-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.asset-tip {
  color: #64748b;
  font-size: 12px;
}

.text-block,
.feature-block,
.payload-block,
.empty-block,
.file-card {
  border-radius: 18px;
}

.text-block,
.feature-block {
  min-height: 140px;
  padding: 18px;
  background: #f8fafc;
  white-space: pre-wrap;
  color: #0f172a;
  line-height: 1.85;
}

.feature-block {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  word-break: break-all;
}

.payload-block {
  margin: 0;
  padding: 16px;
  background: #0f172a;
  color: #dbeafe;
  font-size: 12px;
  line-height: 1.6;
  overflow: auto;
}

.empty-block,
.file-card {
  padding: 18px;
  background: #f8fafc;
  color: #64748b;
  text-align: center;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.asset-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border-radius: 20px;
  background: #f8fafc;
  overflow: hidden;
}

.asset-card img,
.asset-card video,
.asset-card audio {
  width: 100%;
  border-radius: 16px;
  background: #e2e8f0;
}

.asset-card img,
.asset-card video {
  min-height: 220px;
  object-fit: cover;
}

.asset-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.asset-name {
  color: #334155;
  font-size: 13px;
  word-break: break-all;
}

@media (max-width: 900px) {
  .share-hero,
  .content-grid,
  .asset-grid {
    grid-template-columns: 1fr;
  }

  .hero-main h1 {
    font-size: 28px;
  }
}

@media (max-width: 640px) {
  .minimax-share-page {
    padding: 12px;
  }

  .hero-main,
  .hero-actions {
    padding: 18px;
    border-radius: 22px;
  }

  .asset-actions,
  .card-head {
    flex-wrap: wrap;
  }
}
</style>
