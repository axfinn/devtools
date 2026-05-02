<template>
  <div class="gallery-app">
    <!-- Animated background -->
    <div class="bg-orb bg-orb-1" />
    <div class="bg-orb bg-orb-2" />
    <div class="bg-orb bg-orb-3" />

    <!-- Hero background slideshow (AI-generated) -->
    <div class="hero-bg-slideshow">
      <div
        v-for="(bg, i) in bgImages"
        :key="i"
        class="hero-bg-img"
        :class="{ active: bgIndex === i }"
        :style="{ backgroundImage: `url(${bg})` }"
      />
    </div>

    <header class="gallery-hero">
      <div class="hero-content">
        <div class="hero-badge">AI CREATIVE STUDIO</div>
        <h1 class="hero-title">
          <span class="title-line">媒体</span>
          <span class="title-line accent">画廊</span>
        </h1>
        <p class="hero-sub">AI 生成的音乐、视频与图像作品集 &mdash; 无需登录，自由欣赏</p>
        <div class="hero-stats" v-if="!loading && !error">
          <div class="stat" v-for="s in stats" :key="s.label">
            <span class="stat-num">{{ s.count }}</span>
            <span class="stat-label">{{ s.label }}</span>
          </div>
        </div>
      </div>
    </header>

    <!-- Filter bar -->
    <div class="filter-bar" v-if="!error">
      <div class="filter-pills">
        <button
          v-for="opt in typeOptions"
          :key="opt.value"
          class="pill"
          :class="{ active: filterType === opt.value }"
          @click="filterType = opt.value; loadItems()"
        >
          <el-icon v-if="opt.icon" class="pill-icon"><component :is="opt.icon" /></el-icon>
          {{ opt.label }}
        </button>
      </div>
      <div class="filter-search">
        <el-input
          v-model="keyword"
          placeholder="搜索作品标题..."
          :prefix-icon="Search"
          clearable
          class="search-input"
          @change="loadItems"
        />
      </div>
      <button v-if="playableItems.length > 0" class="pill play-all-pill" @click="playAllShuffled">
        <el-icon class="pill-icon"><VideoPlay /></el-icon>
        随机播放全部
      </button>
    </div>

    <!-- Loading skeletons -->
    <div v-if="loading" class="skeleton-grid">
      <div v-for="i in 6" :key="i" class="skeleton-card">
        <div class="skel-media" />
        <div class="skel-body">
          <div class="skel-line w-3/4" />
          <div class="skel-line w-1/2" />
          <div class="skel-line w-full" />
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="state-shell">
      <div class="state-card">
        <el-icon :size="48"><WarningFilled /></el-icon>
        <h3>加载失败</h3>
        <p>{{ error }}</p>
        <el-button type="primary" @click="loadItems">重新加载</el-button>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="items.length === 0" class="state-shell">
      <div class="state-card">
        <el-icon :size="48"><PictureFilled /></el-icon>
        <h3>暂无作品</h3>
        <p>去 MiniMax Studio 生成音乐、视频或图片吧</p>
        <el-button type="primary" @click="$router.push('/minimax-studio')">前往 Studio</el-button>
      </div>
    </div>

    <!-- Grid -->
    <template v-else>
      <div class="masonry">
        <div
          v-for="(item, idx) in items"
          :key="item.id"
          class="brick"
          :class="{
            'brick-featured': idx === 0 && filterType === '',
            'brick-playing': player.state.currentIndex >= 0 && player.state.playlist[player.state.currentIndex]?.id === item.id
          }"
          :style="{ animationDelay: `${Math.min(idx * 40, 400)}ms` }"
          @click="openDetail(item)"
        >
          <div class="brick-media">
            <!-- Image -->
            <img
              v-if="item.primaryAsset?.kind === 'image'"
              :src="item.primaryAsset.asset_url"
              :alt="item.title"
              loading="lazy"
            />
            <!-- Video -->
            <video
              v-else-if="item.primaryAsset?.kind === 'video'"
              :ref="el => setVideoRef(item.id, el)"
              :src="item.primaryAsset.asset_url"
              :loop="player.state.loopMode === 'one'"
              playsinline
              preload="metadata"
              @click.stop
            />
            <!-- Audio canvas -->
            <div v-else-if="item.primaryAsset?.kind === 'audio'" class="audio-visual">
              <canvas :ref="el => setCanvasRef(item.id, el)" class="audio-canvas" />
              <button class="play-btn-big" @click.stop="toggleItemPlay(item)">
                <el-icon :size="isItemPlaying(item) ? 24 : 36">
                  <component :is="isItemPlaying(item) ? 'VideoPause' : 'VideoPlay'" />
                </el-icon>
              </button>
            </div>
            <!-- Fallback -->
            <div v-else class="no-media">
              <el-icon :size="36"><Document /></el-icon>
            </div>

            <!-- Hover overlay -->
            <div class="brick-overlay">
              <span class="overlay-type">{{ typeLabel(item.result_type) }}</span>
              <span class="overlay-model">{{ item.model }}</span>
            </div>

            <!-- Playing indicator -->
            <div v-if="isItemPlaying(item) && item.primaryAsset?.kind === 'audio'" class="playing-eq">
              <span v-for="i in 4" :key="i" class="eq-bar" :style="{ animationDelay: `${i * 0.15}s` }" />
            </div>
          </div>

          <div class="brick-body">
            <h3 class="brick-title">{{ item.title || '未命名作品' }}</h3>
            <p v-if="item.summary" class="brick-summary">{{ item.summary }}</p>
            <div class="brick-foot">
              <span class="brick-date">{{ fmtDate(item.created_at) }}</span>
              <button class="brick-action" title="复制分享链接" @click.stop="copyShareLink(item)">
                <el-icon><Link /></el-icon>
              </button>
              <button
                v-if="item.primaryAsset?.kind === 'audio'"
                class="brick-action"
                :class="{ playing: isItemPlaying(item) }"
                :title="isItemPlaying(item) ? '暂停' : '播放'"
                @click.stop="toggleItemPlay(item)"
              >
                <el-icon><component :is="isItemPlaying(item) ? 'VideoPause' : 'VideoPlay'" /></el-icon>
              </button>
              <button
                v-if="item.primaryAsset?.kind === 'audio'"
                class="brick-action"
                title="加入播放队列"
                @click.stop="player.addToPlaylist(item)"
              >
                <el-icon><Plus /></el-icon>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Load more -->
      <div v-if="hasMore" class="load-more">
        <el-button :loading="loadingMore" size="large" round @click="loadMore">
          加载更多作品
        </el-button>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search, Link, VideoPlay, VideoPause, Document, WarningFilled, PictureFilled, Headset, VideoCamera, Picture, Plus } from '@element-plus/icons-vue'
import { useMediaPlayer } from '../../composables/useMediaPlayer'

const router = useRouter()
const player = useMediaPlayer()

// AI-generated background images
const bgImages = [
  '/gallery-bg/bg_5ea77fbec28489f84a30d1f9.jpeg',
  '/gallery-bg/bg_a01d1aab095d3c613352f508.jpeg',
  '/gallery-bg/bg_7d379f7fc43cacbe9a08acd4.jpeg',
]
const bgIndex = ref(0)
let bgTimer = null

onMounted(() => {
  loadItems()
  startCanvasLoop()
  bgTimer = setInterval(() => {
    bgIndex.value = (bgIndex.value + 1) % bgImages.length
  }, 8000)
})

onBeforeUnmount(() => {
  if (animFrame) cancelAnimationFrame(animFrame)
  if (bgTimer) clearInterval(bgTimer)
})

const typeOptions = [
  { label: '全部', value: '', icon: null },
  { label: '音乐', value: 'audio', icon: Headset },
  { label: '视频', value: 'video', icon: VideoCamera },
  { label: '图片', value: 'image', icon: Picture },
]

const filterType = ref('')
const keyword = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const items = ref([])
const offset = ref(0)
const total = ref(0)
const limit = 30

const videoRefs = {}
const canvasRefs = {}
let animFrame = null

const hasMore = computed(() => items.value.length < total.value)

const playableItems = computed(() =>
  items.value.filter(i => i.primaryAsset?.kind === 'audio' || i.primaryAsset?.kind === 'video')
)

const stats = computed(() => {
  const counts = { audio: 0, video: 0, image: 0 }
  for (const item of items.value) {
    const k = item.primaryAsset?.kind
    if (counts[k] !== undefined) counts[k]++
  }
  return [
    { label: '音乐作品', count: counts.audio },
    { label: '视频作品', count: counts.video },
    { label: '图片作品', count: counts.image },
  ].filter(s => s.count > 0)
})


function setVideoRef(id, el) { if (el) videoRefs[id] = el }
function setCanvasRef(id, el) { if (el) canvasRefs[id] = el }

function isItemPlaying(item) {
  const idx = player.state.currentIndex
  return idx >= 0 && player.state.playlist[idx]?.id === item.id && player.state.isPlaying
}

function toggleItemPlay(item) {
  if (isItemPlaying(item)) {
    player.stop()
  } else {
    // Build full playlist from current filter, then play starting at this item
    const list = player.buildPlaylist(items.value)
    const startIdx = list.findIndex(t => t.id === item.id)
    player.playAll(items.value, startIdx >= 0 ? startIdx : 0)
  }
}

function playAllShuffled() {
  player.setPlaylist(items.value)
  player.state.shuffleMode = true
  if (player.state.playlist.length > 0) {
    const idx = Math.floor(Math.random() * player.state.playlist.length)
    player.play(idx)
  }
}

function startCanvasLoop() {
  function draw() {
    animFrame = requestAnimationFrame(draw)
    for (const item of items.value) {
      if (item.primaryAsset?.kind !== 'audio') continue
      const canvas = canvasRefs[item.id]
      if (!canvas) continue
      const ctx = canvas.getContext('2d')
      const w = canvas.width || canvas.parentElement?.clientWidth || 320
      const h = canvas.height || canvas.parentElement?.clientHeight || 180
      if (canvas.width !== w) canvas.width = w
      if (canvas.height !== h) canvas.height = h

      ctx.fillStyle = 'rgba(2, 6, 23, 0.9)'
      ctx.fillRect(0, 0, w, h)

      const bars = 48
      const barW = w / bars
      const isActive = isItemPlaying(item)

      for (let i = 0; i < bars; i++) {
        let height
        if (isActive) {
          const t = Date.now() / 1000
          height = Math.abs(Math.sin(t * 2.5 + i * 0.35) * Math.cos(t * 1.7 + i * 0.21)) * h * 0.65
                 + Math.abs(Math.sin(t * 3.9 + i * 0.19)) * h * 0.2
                 + h * 0.06
        } else {
          height = Math.abs(Math.sin(i * 0.4)) * h * 0.3 + h * 0.06
        }

        const grad = ctx.createLinearGradient(0, h, 0, h - height)
        if (isActive) {
          grad.addColorStop(0, '#38bdf8')
          grad.addColorStop(0.5, '#6366f1')
          grad.addColorStop(1, '#a78bfa')
        } else {
          grad.addColorStop(0, 'rgba(56, 189, 248, 0.3)')
          grad.addColorStop(1, 'rgba(56, 189, 248, 0.15)')
        }
        ctx.fillStyle = grad
        ctx.fillRect(i * barW + 1, h - height, barW - 2, height)
      }
    }
  }
  draw()
}

async function loadItems() {
  loading.value = true
  error.value = ''
  offset.value = 0
  player.stop()
  try {
    const params = new URLSearchParams({ limit: String(limit), offset: '0' })
    // Don't send type filter to API — filter client-side by asset kind instead.
    // The backend result_type field doesn't distinguish video vs media vs audio well.
    // Video shares have result_type="media", audio shares have "audio" or "speech", etc.
    if (keyword.value) params.set('keyword', keyword.value)
    const res = await fetch(`/api/minimax/result-shares/public/list?${params}`)
    const data = await safeJson(res)
    if (!res.ok) throw new Error(data.error || '加载失败')
    let enriched = enrichItems(data.items || [])
    // Client-side filter by primary asset kind
    if (filterType.value) {
      enriched = enriched.filter(item => item.primaryAsset?.kind === filterType.value)
    }
    items.value = enriched
    total.value = data.total || items.value.length
  } catch (err) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value) return
  loadingMore.value = true
  offset.value += limit
  try {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset.value) })
    if (keyword.value) params.set('keyword', keyword.value)
    const res = await fetch(`/api/minimax/result-shares/public/list?${params}`)
    const data = await safeJson(res)
    if (!res.ok) throw new Error(data.error || '加载失败')
    let newItems = enrichItems(data.items || [])
    if (filterType.value) {
      newItems = newItems.filter(item => item.primaryAsset?.kind === filterType.value)
    }
    items.value.push(...newItems)
  } catch (err) {
    ElMessage.error(err.message || '加载失败')
  } finally {
    loadingMore.value = false
  }
}

function enrichItems(raw) {
  return raw.map(item => {
    const assets = item.assets || []
    const priority = ['audio', 'video', 'image']
    let primaryAsset = null
    for (const kind of priority) {
      primaryAsset = assets.find(a => a.kind === kind)
      if (primaryAsset) break
    }
    if (!primaryAsset && assets.length > 0) primaryAsset = assets[0]
    return { ...item, _assets: assets, primaryAsset }
  })
}

function openDetail(item) {
  if (item.share_url) {
    router.push(new URL(item.share_url).pathname)
  } else {
    router.push(`/minimax/share/${item.id}`)
  }
}

function copyShareLink(item) {
  const url = item.share_url || `${window.location.origin}/minimax/share/${item.id}`
  navigator.clipboard.writeText(url)
    .then(() => ElMessage.success('分享链接已复制'))
    .catch(() => ElMessage.error('复制失败'))
}

function fmtDate(val) {
  if (!val) return ''
  return new Date(val).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function typeLabel(type) {
  const map = { audio: '音乐', video: '视频', image: '图片', media: '媒体', text: '文本' }
  return map[type] || type || '作品'
}

async function safeJson(res) {
  try { return await res.json() } catch { return {} }
}
</script>

<style scoped>
/* ===== Base ===== */
.gallery-app {
  min-height: 100vh;
  padding: 0 20px 120px;  /* extra bottom for player bar */
  background: #0b1120;
  color: #e2e8f0;
  position: relative;
  overflow-x: hidden;
}

/* ===== Animated BG orbs ===== */
.bg-orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(120px);
  pointer-events: none;
  z-index: 0;
}
.bg-orb-1 {
  width: 600px; height: 600px;
  top: -200px; left: -150px;
  background: rgba(56, 189, 248, 0.08);
  animation: orb1 18s ease-in-out infinite;
}
.bg-orb-2 {
  width: 500px; height: 500px;
  bottom: -150px; right: -100px;
  background: rgba(167, 139, 250, 0.07);
  animation: orb2 22s ease-in-out infinite;
}
.bg-orb-3 {
  width: 350px; height: 350px;
  top: 50%; left: 55%;
  background: rgba(99, 102, 241, 0.06);
  animation: orb3 15s ease-in-out infinite;
}
@keyframes orb1 {
  0%, 100% { transform: translate(0, 0); }
  33% { transform: translate(80px, 60px); }
  66% { transform: translate(-30px, 100px); }
}
@keyframes orb2 {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(-70px, -80px); }
}
@keyframes orb3 {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(60px, -50px); }
}

/* ===== Hero background slideshow ===== */
.hero-bg-slideshow {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
}
.hero-bg-img {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  opacity: 0;
  transition: opacity 2s ease-in-out;
  filter: brightness(0.25) saturate(0.6);
}
.hero-bg-img.active {
  opacity: 1;
}

/* ===== Hero ===== */
.gallery-hero {
  position: relative;
  z-index: 1;
  max-width: 1440px;
  margin: 0 auto 32px;
  padding-top: 48px;
}
.hero-content { text-align: center; }
.hero-badge {
  display: inline-block;
  padding: 4px 14px;
  border: 1px solid rgba(56, 189, 248, 0.3);
  border-radius: 20px;
  font-size: 11px;
  letter-spacing: 0.2em;
  color: #38bdf8;
  margin-bottom: 16px;
}
.hero-title {
  margin: 0;
  font-size: clamp(40px, 7vw, 72px);
  font-weight: 800;
  line-height: 1.05;
  letter-spacing: -0.03em;
}
.title-line { display: block; color: #f1f5f9; }
.title-line.accent {
  background: linear-gradient(135deg, #38bdf8 0%, #6366f1 50%, #a78bfa 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.hero-sub {
  margin: 16px 0 0;
  color: #64748b;
  font-size: 16px;
  max-width: 480px;
  margin-left: auto;
  margin-right: auto;
}
.hero-stats { display: flex; justify-content: center; gap: 36px; margin-top: 24px; }
.stat { text-align: center; }
.stat-num {
  display: block;
  font-size: 28px;
  font-weight: 700;
  background: linear-gradient(135deg, #38bdf8, #a78bfa);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.stat-label { font-size: 12px; color: #64748b; text-transform: uppercase; letter-spacing: 0.08em; }

/* ===== Filter bar ===== */
.filter-bar {
  position: relative;
  z-index: 1;
  max-width: 1440px;
  margin: 0 auto 28px;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
  gap: 12px;
}
.filter-pills {
  display: flex;
  gap: 6px;
  padding: 4px;
  border-radius: 28px;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.1);
}
.pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 18px;
  border: none;
  border-radius: 24px;
  background: transparent;
  color: #94a3b8;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.pill:hover { color: #e2e8f0; }
.pill.active {
  background: linear-gradient(135deg, #38bdf8, #6366f1);
  color: #fff;
}
.pill-icon { font-size: 15px; }
.play-all-pill {
  background: linear-gradient(135deg, #f59e0b, #ef4444);
  color: #fff;
  border: none;
}
.play-all-pill:hover { opacity: 0.9; color: #fff; }
.search-input { width: 220px; }

/* ===== Skeleton grid ===== */
.skeleton-grid {
  position: relative;
  z-index: 1;
  max-width: 1440px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 20px;
}
.skeleton-card {
  border-radius: 20px;
  overflow: hidden;
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.06);
}
.skel-media {
  aspect-ratio: 16 / 10;
  background: linear-gradient(110deg, rgba(30,41,59,0.6) 30%, rgba(51,65,85,0.4) 50%, rgba(30,41,59,0.6) 70%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
}
.skel-body { padding: 16px; display: flex; flex-direction: column; gap: 10px; }
.skel-line {
  height: 14px;
  border-radius: 7px;
  background: linear-gradient(110deg, rgba(51,65,85,0.4) 30%, rgba(71,85,105,0.3) 50%, rgba(51,65,85,0.4) 70%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
}
.skel-line.w-3\/4 { width: 75%; }
.skel-line.w-1\/2 { width: 50%; }
.skel-line.w-full { width: 100%; }
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* ===== State shells ===== */
.state-shell {
  position: relative;
  z-index: 1;
  min-height: 50vh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.state-card {
  text-align: center;
  padding: 48px;
  border-radius: 28px;
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.08);
}
.state-card h3 { margin: 16px 0 8px; color: #f1f5f9; }
.state-card p { color: #64748b; margin-bottom: 20px; }

/* ===== Masonry grid ===== */
.masonry {
  position: relative;
  z-index: 1;
  max-width: 1440px;
  margin: 0 auto;
  columns: 3 340px;
  column-gap: 20px;
}
.brick {
  break-inside: avoid;
  margin-bottom: 20px;
  border-radius: 20px;
  overflow: hidden;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.08);
  cursor: pointer;
  transition: transform 0.25s ease, border-color 0.25s, box-shadow 0.25s;
  animation: fadeUp 0.5s ease both;
}
.brick:hover {
  transform: translateY(-4px);
  border-color: rgba(56, 189, 248, 0.3);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(56, 189, 248, 0.1);
}
.brick-featured { border-color: rgba(56, 189, 248, 0.2); }
.brick-playing {
  border-color: #38bdf8;
  box-shadow: 0 0 24px rgba(56, 189, 248, 0.2), 0 0 0 1px rgba(56, 189, 248, 0.3);
}
@keyframes fadeUp {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ===== Brick media ===== */
.brick-media {
  position: relative;
  width: 100%;
  background: #020617;
  overflow: hidden;
}
.brick-media img,
.brick-media video {
  width: 100%;
  display: block;
}
.audio-visual {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 10;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #020617;
}
.audio-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}
.play-btn-big {
  position: relative;
  z-index: 2;
  width: 64px; height: 64px;
  border: 2px solid rgba(255,255,255,0.3);
  border-radius: 50%;
  background: rgba(0,0,0,0.5);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: all 0.2s;
}
.play-btn-big:hover {
  border-color: #38bdf8;
  background: rgba(56, 189, 248, 0.2);
  transform: scale(1.08);
}
.no-media {
  aspect-ratio: 16 / 10;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #334155;
}

/* Playing EQ indicator */
.playing-eq {
  position: absolute;
  bottom: 12px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 3px;
  height: 20px;
  align-items: flex-end;
}
.eq-bar {
  width: 3px;
  border-radius: 2px;
  background: #38bdf8;
  animation: eqAnim 0.7s ease-in-out infinite alternate;
}
@keyframes eqAnim {
  0% { height: 6px; }
  100% { height: 20px; }
}

/* Overlay on hover */
.brick-overlay {
  position: absolute;
  top: 0; left: 0; right: 0;
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  background: linear-gradient(180deg, rgba(0,0,0,0.6) 0%, transparent 100%);
  opacity: 0;
  transition: opacity 0.25s;
}
.brick-media:hover .brick-overlay,
.brick:hover .brick-overlay { opacity: 1; }
.overlay-type,
.overlay-model {
  font-size: 11px;
  padding: 3px 10px;
  border-radius: 12px;
  background: rgba(0,0,0,0.5);
  backdrop-filter: blur(6px);
  color: #e2e8f0;
}
.overlay-model {
  color: #94a3b8;
  font-family: ui-monospace, monospace;
}

/* ===== Brick body ===== */
.brick-body {
  padding: 14px 16px;
}
.brick-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #f1f5f9;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.brick-summary {
  margin: 6px 0 0;
  font-size: 13px;
  color: #64748b;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.brick-foot {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.brick-date {
  font-size: 12px;
  color: #475569;
  margin-right: auto;
}
.brick-action {
  width: 32px; height: 32px;
  border: none;
  border-radius: 50%;
  background: rgba(148, 163, 184, 0.1);
  color: #94a3b8;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  font-size: 14px;
}
.brick-action:hover { background: rgba(56, 189, 248, 0.15); color: #38bdf8; }
.brick-action.playing { background: #38bdf8; color: #fff; }

/* ===== Load more ===== */
.load-more {
  position: relative;
  z-index: 1;
  text-align: center;
  margin-top: 36px;
}

/* ===== Responsive ===== */
@media (max-width: 1100px) {
  .masonry { columns: 2 300px; }
}
@media (max-width: 768px) {
  .gallery-app { padding: 0 12px 120px; }
  .gallery-hero { padding-top: 32px; }
  .hero-title { font-size: 36px; }
  .hero-stats { gap: 20px; }
  .stat-num { font-size: 22px; }
  .masonry { columns: 1; }
  .filter-bar { flex-direction: column; }
  .search-input { width: 100%; }
  .filter-pills { width: 100%; justify-content: center; }
}
</style>
