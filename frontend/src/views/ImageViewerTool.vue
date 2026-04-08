<template>
  <div class="image-viewer-tool">
    <!-- 输入区 -->
    <div class="input-section">
      <div class="input-header">
        <span class="label">图片 URL（每行一个）</span>
        <div class="actions">
          <el-checkbox v-model="proxyFallback" size="small">跨域走代理</el-checkbox>
          <el-button size="small" @click="clearAll">清空</el-button>
          <el-button size="small" @click="shareLink" :disabled="!urlInput.trim()">
            <el-icon><Share /></el-icon> 分享
          </el-button>
          <el-button size="small" type="primary" @click="loadImages">加载图片</el-button>
        </div>
      </div>
      <el-input
        v-model="urlInput"
        type="textarea"
        :rows="4"
        placeholder="https://example.com/a.jpg&#10;https://example.com/b.png&#10;..."
      />
    </div>

    <!-- 对照列表 -->
    <div v-if="images.length" class="image-list">
      <div class="list-header">
        <span>共 {{ images.length }} 张 · 成功 {{ successCount }} · 失败 {{ failCount }}</span>
        <el-button v-if="failCount > 0" size="small" @click="retryFailed">重试失败</el-button>
      </div>

      <div v-for="(img, idx) in images" :key="img.url" class="image-row">
        <!-- 左：序号 + URL -->
        <div class="row-meta">
          <span class="row-index">#{{ idx + 1 }}</span>
          <div class="row-url-wrap">
            <span class="row-url" :title="img.url">{{ img.url }}</span>
            <el-button size="small" text @click="copyUrl(img.url)">
              <el-icon><CopyDocument /></el-icon>
            </el-button>
          </div>
          <div class="row-status">
            <el-tag v-if="img.status === 'ok'" type="success" size="small">OK</el-tag>
            <el-tag v-else-if="img.status === 'error'" type="danger" size="small">失败</el-tag>
            <el-tag v-else type="info" size="small">加载中</el-tag>
            <span v-if="img.proxied" class="proxy-badge">代理</span>
          </div>
        </div>

        <!-- 右：图片 -->
        <div class="row-image" @click="openLightbox(idx)">
          <template v-if="img.status !== 'error'">
            <img
              :src="img.displaySrc"
              :alt="img.url"
              @load="onLoad(img)"
              @error="onError(img)"
              loading="lazy"
            />
            <div v-if="img.status === 'loading'" class="img-overlay">
              <el-icon class="spin"><Loading /></el-icon>
            </div>
          </template>
          <div v-else class="img-error">
            <el-icon size="32"><Picture /></el-icon>
            <span>加载失败</span>
            <el-button size="small" @click.stop="retryOne(img)">重试</el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="empty-state">
      <el-icon size="56" color="#ccc"><Picture /></el-icon>
      <p>输入图片 URL，点击「加载图片」</p>
    </div>

    <!-- Lightbox -->
    <div v-if="lightboxIdx !== null" class="lightbox" @click.self="closeLightbox">
      <button class="lb-close" @click="closeLightbox">✕</button>
      <button class="lb-prev" @click="prevImage" :disabled="lightboxIdx === 0">‹</button>
      <button class="lb-next" @click="nextImage" :disabled="lightboxIdx === images.length - 1">›</button>
      <div class="lb-body">
        <img :src="currentLightboxSrc" />
        <div class="lb-info">
          <span>{{ lightboxIdx + 1 }} / {{ images.length }}</span>
          <span class="lb-url">{{ images[lightboxIdx]?.url }}</span>
          <el-button size="small" @click="copyUrl(images[lightboxIdx]?.url)">复制链接</el-button>
          <a :href="images[lightboxIdx]?.url" target="_blank">
            <el-button size="small">新标签打开</el-button>
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading, CopyDocument, Picture, Share } from '@element-plus/icons-vue'

const urlInput = ref('')
const images = ref([])
const proxyFallback = ref(true)
const lightboxIdx = ref(null)

const successCount = computed(() => images.value.filter(i => i.status === 'ok').length)
const failCount = computed(() => images.value.filter(i => i.status === 'error').length)
const currentLightboxSrc = computed(() => {
  if (lightboxIdx.value === null) return ''
  return images.value[lightboxIdx.value]?.displaySrc || ''
})

function loadImages() {
  const urls = urlInput.value
    .split('\n')
    .map(u => u.trim())
    .filter(u => u && /^https?:\/\//i.test(u))
  if (!urls.length) {
    ElMessage.warning('请输入有效的图片 URL')
    return
  }
  images.value = urls.map(url => ({
    url,
    displaySrc: url,
    status: 'loading',
    proxied: false
  }))
}

function onLoad(img) {
  img.status = 'ok'
}

function onError(img) {
  if (proxyFallback.value && !img.proxied) {
    img.proxied = true
    img.status = 'loading'
    img.displaySrc = `/api/proxy-image?url=${encodeURIComponent(img.url)}`
  } else {
    img.status = 'error'
  }
}

function retryOne(img) {
  img.status = 'loading'
  img.proxied = false
  img.displaySrc = img.url
}

function retryFailed() {
  images.value.filter(i => i.status === 'error').forEach(retryOne)
}

function clearAll() {
  urlInput.value = ''
  images.value = []
}

async function shareLink() {
  const urls = urlInput.value.trim()
  if (!urls) return
  // 先压缩编码，构造完整页面链接
  const encoded = await compressToBase64(urls)
  const fullUrl = `${location.origin}/image-viewer?q=${encoded}`
  // 调短链接口
  try {
    const res = await fetch('/api/shorturl', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ original_url: fullUrl })
    })
    const data = await res.json()
    if (data.short_url) {
      navigator.clipboard.writeText(data.short_url).then(() => ElMessage.success('短链已复制：' + data.short_url))
    } else {
      // 短链失败降级：直接复制长链
      navigator.clipboard.writeText(fullUrl).then(() => ElMessage.warning('短链生成失败，已复制完整链接'))
    }
  } catch {
    navigator.clipboard.writeText(fullUrl).then(() => ElMessage.warning('短链生成失败，已复制完整链接'))
  }
}

async function compressToBase64(str) {
  const stream = new Blob([str]).stream().pipeThrough(new CompressionStream('gzip'))
  const buf = await new Response(stream).arrayBuffer()
  return btoa(String.fromCharCode(...new Uint8Array(buf)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

async function decompressFromBase64(b64) {
  const bin = atob(b64.replace(/-/g, '+').replace(/_/g, '/'))
  const buf = Uint8Array.from(bin, c => c.charCodeAt(0))
  const stream = new Blob([buf]).stream().pipeThrough(new DecompressionStream('gzip'))
  return new Response(stream).text()
}

function copyUrl(url) {
  navigator.clipboard.writeText(url).then(() => ElMessage.success('已复制'))
}

function openLightbox(idx) {
  if (images.value[idx]?.status === 'error') return
  lightboxIdx.value = idx
}

function closeLightbox() {
  lightboxIdx.value = null
}

function prevImage() {
  if (lightboxIdx.value > 0) lightboxIdx.value--
}

function nextImage() {
  if (lightboxIdx.value < images.value.length - 1) lightboxIdx.value++
}

// 键盘导航
function onKeydown(e) {
  if (lightboxIdx.value === null) return
  if (e.key === 'ArrowLeft') prevImage()
  if (e.key === 'ArrowRight') nextImage()
  if (e.key === 'Escape') closeLightbox()
}

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  // 从分享链接恢复
  const q = new URLSearchParams(location.search).get('q')
  if (q) {
    try {
      urlInput.value = await decompressFromBase64(q)
      loadImages()
    } catch {
      ElMessage.error('分享链接解析失败')
    }
  }
})
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.image-viewer-tool {
  padding: 16px;
  max-width: 1200px;
  margin: 0 auto;
}

.input-section {
  margin-bottom: 16px;
}

.input-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.label {
  font-weight: 500;
  font-size: 14px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 列表 */
.image-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: #666;
  padding-bottom: 8px;
  border-bottom: 1px solid #eee;
}

.image-row {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  padding: 12px;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  background: #fafafa;
  transition: box-shadow 0.2s;
}

.image-row:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

/* 左侧 meta */
.row-meta {
  flex: 0 0 280px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.row-index {
  font-size: 12px;
  color: #999;
  font-weight: 600;
}

.row-url-wrap {
  display: flex;
  align-items: flex-start;
  gap: 4px;
}

.row-url {
  font-size: 12px;
  color: #333;
  word-break: break-all;
  line-height: 1.5;
  flex: 1;
}

.row-status {
  display: flex;
  align-items: center;
  gap: 6px;
}

.proxy-badge {
  font-size: 11px;
  color: #e6a23c;
  background: #fdf6ec;
  border: 1px solid #f5dab1;
  border-radius: 3px;
  padding: 0 4px;
}

/* 右侧图片 */
.row-image {
  flex: 1;
  min-height: 120px;
  max-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  cursor: pointer;
  border-radius: 6px;
  overflow: hidden;
  background: #f0f0f0;
}

.row-image img {
  max-width: 100%;
  max-height: 300px;
  object-fit: contain;
  display: block;
  border-radius: 4px;
}

.img-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255,255,255,0.6);
}

.img-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #999;
  font-size: 13px;
  padding: 24px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
.spin { animation: spin 1s linear infinite; }

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 60px 0;
  color: #aaa;
}

/* Lightbox */
.lightbox {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.88);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
}

.lb-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  max-width: 90vw;
  max-height: 90vh;
  gap: 12px;
}

.lb-body img {
  max-width: 90vw;
  max-height: 80vh;
  object-fit: contain;
  border-radius: 4px;
}

.lb-info {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #fff;
  font-size: 13px;
  flex-wrap: wrap;
  justify-content: center;
}

.lb-url {
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  opacity: 0.7;
}

.lb-close, .lb-prev, .lb-next {
  position: fixed;
  background: rgba(255,255,255,0.15);
  border: none;
  color: #fff;
  cursor: pointer;
  border-radius: 50%;
  width: 44px;
  height: 44px;
  font-size: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.lb-close:hover, .lb-prev:hover, .lb-next:hover {
  background: rgba(255,255,255,0.3);
}

.lb-close { top: 20px; right: 20px; }
.lb-prev { left: 20px; top: 50%; transform: translateY(-50%); font-size: 30px; }
.lb-next { right: 20px; top: 50%; transform: translateY(-50%); font-size: 30px; }

.lb-prev:disabled, .lb-next:disabled {
  opacity: 0.25;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .image-row {
    flex-direction: column;
  }
  .row-meta {
    flex: none;
    width: 100%;
  }
}
</style>
