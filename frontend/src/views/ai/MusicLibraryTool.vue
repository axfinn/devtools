<template>
  <div class="musiclib-app">
    <div class="bg-orb bg-orb-1" />
    <div class="bg-orb bg-orb-2" />

    <!-- Hero -->
    <header class="ml-hero">
      <div class="ml-badge">MY MUSIC LIBRARY</div>
      <h1 class="ml-title">音<span class="accent">乐</span>库</h1>
      <p class="ml-sub">收藏我自己的 AI 创作 — 上传、收藏、看歌词，想听就听</p>
      <div class="ml-stats" v-if="stats">
        <div class="stat"><span class="stat-num">{{ stats.total_count }}</span><span class="stat-label">曲目</span></div>
        <div class="stat"><span class="stat-num">{{ fmtTotalDuration }}</span><span class="stat-label">总时长</span></div>
        <div class="stat"><span class="stat-num">{{ stats.favorite_count }}</span><span class="stat-label">收藏</span></div>
        <div class="stat"><span class="stat-num">{{ stats.total_plays }}</span><span class="stat-label">累计播放</span></div>
      </div>
    </header>

    <!-- Toolbar -->
    <div class="ml-toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="keyword"
          placeholder="搜索标题 / 歌手 / 专辑 / 备注..."
          :prefix-icon="Search"
          clearable
          class="search-input"
          @input="onSearchInput"
        />
        <el-select v-model="sortBy" class="sort-select" @change="loadTracks">
          <el-option label="最新添加" value="created" />
          <el-option label="标题 A-Z" value="title" />
          <el-option label="最多播放" value="plays" />
        </el-select>
        <button class="pill" :class="{ active: favOnly }" @click="favOnly = !favOnly; loadTracks()">
          <svg viewBox="0 0 24 24" width="13" height="13" :fill="favOnly ? '#fff' : 'none'" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M12 21s-7.5-4.9-9.7-9.2C.9 8.9 2.7 5.5 6 5.5c2 0 3.4 1.1 4.2 2.4L12 9l1.8-1.1c.8-1.3 2.2-2.4 4.2-2.4 3.3 0 5.1 3.4 3.7 6.3C19.5 16.1 12 21 12 21z" />
          </svg>
          只看收藏
        </button>
      </div>
      <button class="upload-pill" @click="uploadDialog = true">
        <el-icon><UploadFilled /></el-icon>
        上传音乐
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="track-skeletons">
      <div v-for="i in 8" :key="i" class="skel-row" />
    </div>

    <!-- Empty -->
    <div v-else-if="tracks.length === 0" class="ml-empty">
      <el-icon :size="48"><Headset /></el-icon>
      <h3>曲库还是空的</h3>
      <p>把 AI 创作的 mp3 / flac / wav 拖进来，或者点右上角「上传音乐」</p>
      <el-button type="primary" round @click="uploadDialog = true">上传第一首歌</el-button>
    </div>

    <!-- Track list -->
    <template v-else>
      <div class="track-list">
        <div
          v-for="(t, idx) in tracks"
          :key="t.id"
          class="track-row"
          :class="{ playing: isCurrent(t) }"
          @click="playTrack(t, idx)"
        >
          <span class="row-index">
            <template v-if="isCurrent(t) && playerState.isPlaying">
              <span v-for="i in 3" :key="i" class="eq-dot" :style="{ animationDelay: `${i * 0.15}s` }" />
            </template>
            <template v-else>{{ idx + 1 }}</template>
          </span>
          <div class="row-cover" @click.stop="playTrack(t, idx)">
            <img v-if="t.cover_filename" :src="`/api/music/cover/${t.id}`" :alt="t.title" loading="lazy" />
            <span v-else class="cover-fallback">{{ (t.title || '?')[0].toUpperCase() }}</span>
            <button class="cover-play" @click.stop="playTrack(t, idx)">
              <el-icon :size="18"><component :is="isCurrent(t) && playerState.isPlaying ? 'VideoPause' : 'VideoPlay'" /></el-icon>
            </button>
          </div>
          <div class="row-meta">
            <span class="row-title">{{ t.title }}</span>
            <span class="row-sub">{{ t.artist || '未知歌手' }}<template v-if="t.album"> · {{ t.album }}</template></span>
          </div>
          <span v-if="t.lyrics" class="row-lrc-tag" title="带歌词">词</span>
          <span class="row-plays" :title="`播放 ${t.play_count} 次`">{{ t.play_count }} 次播放</span>
          <span class="row-duration">{{ fmtTime(t.duration_sec) }}</span>
          <div class="row-actions" @click.stop>
            <button class="row-btn" :class="{ faved: t.favorite }" title="收藏" @click="toggleFav(t)">
              <svg viewBox="0 0 24 24" width="15" height="15" :fill="t.favorite ? '#f59e0b' : 'none'" stroke="currentColor" stroke-width="2">
                <path d="M12 21s-7.5-4.9-9.7-9.2C.9 8.9 2.7 5.5 6 5.5c2 0 3.4 1.1 4.2 2.4L12 9l1.8-1.1c.8-1.3 2.2-2.4 4.2-2.4 3.3 0 5.1 3.4 3.7 6.3C19.5 16.1 12 21 12 21z" />
              </svg>
            </button>
            <button class="row-btn" title="歌词" @click="openLyrics(t)">
              <el-icon :size="15"><Document /></el-icon>
            </button>
            <button class="row-btn" title="编辑信息" @click="openEdit(t)">
              <el-icon :size="15"><Edit /></el-icon>
            </button>
            <a class="row-btn" title="下载" :href="`/api/music/stream/${t.id}`" :download="t.original_name || t.title">
              <el-icon :size="15"><Download /></el-icon>
            </a>
            <button class="row-btn danger" title="删除" @click="removeTrack(t)">
              <el-icon :size="15"><Delete /></el-icon>
            </button>
          </div>
        </div>
      </div>
      <div v-if="tracks.length < total" class="load-more">
        <el-button :loading="loadingMore" round @click="loadMore">还有 {{ total - tracks.length }} 首，加载更多</el-button>
      </div>
    </template>

    <!-- Upload dialog -->
    <el-dialog v-model="uploadDialog" title="上传音乐" width="560px" :close-on-click-modal="false">
      <el-upload
        ref="uploadRef"
        drag multiple
        :auto-upload="false"
        :on-change="onFileAdd"
        :show-file-list="false"
        accept=".mp3,.wav,.flac,.m4a,.aac,.ogg,.oga,.opus,.webm,.wma,.aiff,.aif,.ape"
      >
        <el-icon :size="40" class="upload-icon"><UploadFilled /></el-icon>
        <div class="el-upload__text">拖音频到这里，或 <em>点击选择</em></div>
        <template #tip>
          <div class="el-upload__tip">支持 mp3 / wav / flac / m4a / ogg / opus 等，单曲最大 200MB</div>
        </template>
      </el-upload>

      <div v-if="pendingFiles.length" class="pending-list">
        <div class="pending-head">
          <span>共 {{ pendingFiles.length }} 个文件</span>
          <el-button text size="small" @click="pendingFiles = []">清空</el-button>
        </div>
        <div class="pending-item" v-for="(p, i) in pendingFiles" :key="i">
          <el-icon class="pending-icon"><Headset /></el-icon>
          <div class="pending-meta">
            <span class="pending-name">{{ p.title }}</span>
            <span class="pending-sub">{{ p.humanSize }}<template v-if="p.duration"> · {{ fmtTime(p.duration) }}</template></span>
          </div>
          <el-button v-if="!p.uploading" text size="small" type="danger" @click="pendingFiles.splice(i, 1)">移除</el-button>
          <el-icon v-else class="is-loading"><Loading /></el-icon>
        </div>
      </div>

      <div class="upload-fields">
        <el-input v-model="uploadArtist" placeholder="歌手 / 来源(可选,应用到全部)" />
        <el-input v-model="uploadAlbum" placeholder="专辑 / 合集(可选,应用到全部)" />
      </div>

      <template #footer>
        <el-button @click="uploadDialog = false">取消</el-button>
        <el-button type="primary" :loading="uploading" :disabled="!pendingFiles.length" @click="doUpload">
          开始上传{{ pendingFiles.length ? `(${pendingFiles.length})` : '' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Edit dialog -->
    <el-dialog v-model="editDialog" title="编辑曲目信息" width="560px">
      <div class="edit-form" v-if="editForm">
        <div class="edit-cover-row">
          <div class="edit-cover">
            <img v-if="editCoverPreview" :src="editCoverPreview" />
            <span v-else-if="editForm.cover_filename" class="cover-img-wrap"><img :src="`/api/music/cover/${editForm.id}`" /></span>
            <span v-else class="cover-fallback big">{{ (editForm.title || '?')[0].toUpperCase() }}</span>
          </div>
          <div class="edit-cover-actions">
            <el-upload :auto-upload="false" :show-file-list="false" accept="image/*" :on-change="onCoverPick">
              <el-button size="small">{{ editCoverPreview || editForm.cover_filename ? '更换封面' : '上传封面' }}</el-button>
            </el-upload>
            <span class="edit-hint">可选，方形图片效果最佳</span>
          </div>
        </div>
        <el-input v-model="editForm.title" placeholder="标题(必填)" />
        <el-input v-model="editForm.artist" placeholder="歌手 / 来源" />
        <el-input v-model="editForm.album" placeholder="专辑 / 合集" />
        <el-input v-model="editForm.note" placeholder="备注(会参与搜索)" />
        <el-input
          v-model="editForm.lyrics"
          type="textarea" :rows="8"
          placeholder="歌词：支持纯文本或 LRC 时间戳格式，LRC 会逐句同步高亮"
        />
      </div>
      <template #footer>
        <el-button @click="editDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- Lyrics drawer -->
    <el-drawer v-model="lyricsDrawer" size="420px" :with-header="false" class="lyrics-drawer">
      <div class="lyrics-pane" v-if="lyricsTrack">
        <div class="lyrics-head">
          <div class="lyrics-cover">
            <img v-if="lyricsTrack.cover_filename" :src="`/api/music/cover/${lyricsTrack.id}`" />
            <span v-else class="cover-fallback big">{{ (lyricsTrack.title || '?')[0].toUpperCase() }}</span>
          </div>
          <div class="lyrics-titles">
            <h3>{{ lyricsTrack.title }}</h3>
            <p>{{ lyricsTrack.artist || '未知歌手' }}</p>
          </div>
        </div>
        <div class="lyrics-body" ref="lyricsBodyRef">
          <div
            v-for="(line, i) in lyricsLines"
            :key="i"
            class="lyrics-line"
            :class="{ active: i === activeLyricIndex }"
            @click="seekLyric(i)"
          >
            {{ line.text }}
          </div>
          <div v-if="!lyricsLines.length" class="lyrics-empty">这首歌还没有歌词，点列表上的「编辑」补上吧</div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script>
// 页面用 <script setup> 写业务,这里只留说明:
// 全局播放/播放条由 composables/useMediaPlayer.js + App.vue 的 player-bar 承担,
// 本页只负责:曲库管理(上传/编辑/删除/收藏/统计) + 把曲目喂给全局播放器 + 歌词同步。
export default { name: 'MusicLibraryTool' }
</script>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, UploadFilled, Headset, Document, Edit, Download, Delete, Loading } from '@element-plus/icons-vue'
import { useMediaPlayer } from '../../composables/useMediaPlayer'
import { parseLyrics, detectMode, findActiveIndex, computeSeekTime } from '../../components/MusicLyricsPlayer.vue'

const player = useMediaPlayer()
const playerState = player.state

// ── 列表状态 ──
const tracks = ref([])
const total = ref(0)
const stats = ref(null)
const loading = ref(true)
const loadingMore = ref(false)
const keyword = ref('')
const favOnly = ref(false)
const sortBy = ref('created')
const limit = 100
const offset = ref(0)

let searchTimer = null
function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadTracks, 350)
}

onMounted(() => { loadTracks() })
onBeforeUnmount(() => clearTimeout(searchTimer))

async function loadTracks() {
  loading.value = true
  offset.value = 0
  try {
    const data = await apiGet(listParams(0))
    tracks.value = data.items || []
    total.value = Number(data.total || 0)
  } catch (err) {
    ElMessage.error(err.message || '加载失败')
  } finally {
    loading.value = false
  }
  refreshStats()
}

async function loadMore() {
  if (loadingMore.value) return
  loadingMore.value = true
  offset.value += limit
  try {
    const data = await apiGet(listParams(offset.value))
    tracks.value.push(...(data.items || []))
  } catch (err) {
    ElMessage.error(err.message || '加载失败')
  } finally {
    loadingMore.value = false
  }
}

function listParams(off) {
  const p = new URLSearchParams({ limit: String(limit), offset: String(off), sort: sortBy.value })
  if (keyword.value.trim()) p.set('keyword', keyword.value.trim())
  if (favOnly.value) p.set('fav', '1')
  return `/api/music/tracks?${p}`
}

async function refreshStats() {
  try {
    const data = await apiGet('/api/music/stats')
    stats.value = data.stats
  } catch {}
}

async function apiGet(url) {
  const res = await fetch(url)
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error || `请求失败(${res.status})`)
  return data
}

const fmtTotalDuration = computed(() => {
  if (!stats.value) return '0'
  const h = Math.floor(stats.value.total_duration / 3600)
  const m = Math.round((stats.value.total_duration % 3600) / 60)
  return h > 0 ? `${h}时${m}分` : `${m}分`
})

// ── 播放 ──
function playerTrack(t) {
  return { id: t.id, title: t.title, artist: t.artist, primaryAsset: { kind: 'audio', asset_url: `/api/music/stream/${t.id}` } }
}

function isCurrent(t) {
  const cur = playerState.playlist[playerState.currentIndex]
  return cur?.id === t.id
}

function playTrack(t, idx) {
  if (isCurrent(t)) {
    player.togglePlay()
    return
  }
  const mapped = tracks.value.filter(x => x.id).map(playerTrack)
  const realIdx = mapped.findIndex(x => x.id === t.id)
  player.playAll(mapped, realIdx >= 0 ? realIdx : 0)
}

// 切歌时给服务端记一次播放数(只记音乐库自己的曲目)
let lastCountedId = ''
watch(() => playerState.currentIndex, () => {
  const cur = playerState.playlist[playerState.currentIndex]
  if (!cur?.id?.startsWith('mus_') || cur.id === lastCountedId) return
  lastCountedId = cur.id
  fetch(`/api/music/tracks/${cur.id}/played`, { method: 'POST' })
    .then(() => refreshStats())
    .catch(() => {})
})

// ── 收藏 / 删除 ──
async function toggleFav(t) {
  try {
    const res = await fetch(`/api/music/tracks/${t.id}/favorite`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ favorite: !t.favorite }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || '操作失败')
    t.favorite = data.favorite
    refreshStats()
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function removeTrack(t) {
  try {
    await ElMessageBox.confirm(`确定删除「${t.title}」？音频文件会一并删除，不可恢复。`, '删除曲目', {
      type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消',
    })
  } catch { return }
  try {
    const res = await fetch(`/api/music/tracks/${t.id}`, { method: 'DELETE' })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || '删除失败')
    ElMessage.success('已删除')
    tracks.value = tracks.value.filter(x => x.id !== t.id)
    total.value = Math.max(0, total.value - 1)
    refreshStats()
  } catch (err) {
    ElMessage.error(err.message)
  }
}

// ── 上传 ──
const uploadDialog = ref(false)
const uploadRef = ref(null)
const pendingFiles = reactive([])
const uploadArtist = ref('')
const uploadAlbum = ref('')
const uploading = ref(false)

function onFileAdd(file) {
  const raw = file.raw
  if (!raw) return
  const base = raw.name.replace(/\.[^.]+$/, '')
  const entry = reactive({
    raw,
    title: base,
    duration: 0,
    humanSize: humanSize(raw.size),
    uploading: false,
  })
  pendingFiles.push(entry)
  // 前端读时长:走 <audio> 元数据,失败就留 0(后端可后续补)
  const url = URL.createObjectURL(raw)
  const probe = new Audio()
  probe.preload = 'metadata'
  probe.src = url
  probe.addEventListener('loadedmetadata', () => {
    entry.duration = isFinite(probe.duration) ? probe.duration : 0
    URL.revokeObjectURL(url)
  })
  probe.addEventListener('error', () => URL.revokeObjectURL(url))
}

function humanSize(bytes) {
  if (bytes >= 1 << 30) return (bytes / (1 << 30)).toFixed(2) + ' GB'
  if (bytes >= 1 << 20) return (bytes / (1 << 20)).toFixed(1) + ' MB'
  return Math.max(1, Math.round(bytes / 1024)) + ' KB'
}

async function doUpload() {
  if (!pendingFiles.length) return
  uploading.value = true
  let okCount = 0
  for (const p of pendingFiles) {
    p.uploading = true
    try {
      const fd = new FormData()
      fd.append('file', p.raw)
      fd.append('title', p.title)
      if (uploadArtist.value.trim()) fd.append('artist', uploadArtist.value.trim())
      if (uploadAlbum.value.trim()) fd.append('album', uploadAlbum.value.trim())
      if (p.duration > 0) fd.append('duration', String(p.duration))
      const res = await fetch('/api/music/tracks', { method: 'POST', body: fd })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || '上传失败')
      okCount++
      p.uploading = false
    } catch (err) {
      p.uploading = false
      ElMessage.error(`「${p.title}」上传失败：${err.message}`)
    }
  }
  uploading.value = false
  if (okCount > 0) {
    ElMessage.success(`已入库 ${okCount} 首`)
    pendingFiles.splice(0)
    uploadArtist.value = ''
    uploadAlbum.value = ''
    uploadDialog.value = false
    loadTracks()
  }
}

// ── 编辑 ──
const editDialog = ref(false)
const editForm = ref(null)
const editCoverFile = ref(null)
const editCoverPreview = ref('')
const saving = ref(false)

function openEdit(t) {
  editForm.value = { ...t }
  editCoverFile.value = null
  editCoverPreview.value = ''
  editDialog.value = true
}

function onCoverPick(file) {
  const raw = file.raw
  if (!raw) return
  editCoverFile.value = raw
  editCoverPreview.value = URL.createObjectURL(raw)
}

async function saveEdit() {
  const f = editForm.value
  if (!f?.title?.trim()) {
    ElMessage.warning('标题不能为空')
    return
  }
  saving.value = true
  try {
    const res = await fetch(`/api/music/tracks/${f.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: f.title, artist: f.artist, album: f.album,
        lyrics: f.lyrics || '', note: f.note,
        duration_sec: f.duration_sec > 0 ? f.duration_sec : undefined,
      }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || '保存失败')
    // 封面
    if (editCoverFile.value) {
      const fd = new FormData()
      fd.append('cover', editCoverFile.value)
      const cres = await fetch(`/api/music/tracks/${f.id}/cover`, { method: 'POST', body: fd })
      if (!cres.ok) {
        const cdata = await cres.json().catch(() => ({}))
        throw new Error(cdata.error || '封面上传失败')
      }
    }
    const idx = tracks.value.findIndex(x => x.id === f.id)
    if (idx >= 0) Object.assign(tracks.value[idx], data.item || f)
    editDialog.value = false
    ElMessage.success('已保存')
  } catch (err) {
    ElMessage.error(err.message)
  } finally {
    saving.value = false
  }
}

// ── 歌词抽屉(与全局播放器同步) ──
const lyricsDrawer = ref(false)
const lyricsTrack = ref(null)
const lyricsBodyRef = ref(null)

const lyricsLines = computed(() => parseLyrics(lyricsTrack.value?.lyrics || ''))
const lyricsMode = computed(() => detectMode(lyricsLines.value))
const activeLyricIndex = computed(() =>
  findActiveIndex(lyricsLines.value, playerState.currentTime, playerState.duration, lyricsMode.value)
)

function openLyrics(t) {
  lyricsTrack.value = t
  lyricsDrawer.value = true
  nextTick(scrollLyrics)
}

function seekLyric(idx) {
  const target = computeSeekTime(lyricsLines.value[idx], idx, lyricsLines.value.length, playerState.duration, lyricsMode.value)
  player.seek(target)
  // 还没在放这首歌?直接播
  if (!isCurrent(lyricsTrack.value)) {
    const realIdx = tracks.value.findIndex(x => x.id === lyricsTrack.value.id)
    playTrack(lyricsTrack.value, realIdx)
    nextTick(() => player.seek(target))
  }
}

watch(activeLyricIndex, () => nextTick(scrollLyrics))
watch(lyricsDrawer, (open) => { if (open) nextTick(scrollLyrics) })

function scrollLyrics() {
  const body = lyricsBodyRef.value
  if (!body || activeLyricIndex.value < 0) return
  const el = body.children[activeLyricIndex.value]
  if (el) body.scrollTo({ top: el.offsetTop - body.clientHeight / 2 + el.clientHeight / 2, behavior: 'smooth' })
}
</script>

<style scoped>
/* ===== Base(与媒体画廊同款暗色氛围) ===== */
.musiclib-app {
  min-height: 100vh;
  padding: 0 24px 140px;
  background: #0b1120;
  color: #e2e8f0;
  position: relative;
  overflow-x: hidden;
}
.bg-orb { position: fixed; border-radius: 50%; filter: blur(120px); pointer-events: none; z-index: 0; }
.bg-orb-1 { width: 560px; height: 560px; top: -180px; left: -120px; background: rgba(56, 189, 248, 0.08); animation: orb1 20s ease-in-out infinite; }
.bg-orb-2 { width: 480px; height: 480px; bottom: -140px; right: -100px; background: rgba(245, 158, 11, 0.06); animation: orb2 24s ease-in-out infinite; }
@keyframes orb1 { 0%, 100% { transform: translate(0,0); } 50% { transform: translate(70px, 80px); } }
@keyframes orb2 { 0%, 100% { transform: translate(0,0); } 50% { transform: translate(-60px, -70px); } }

/* ===== Hero ===== */
.ml-hero { position: relative; z-index: 1; text-align: center; padding: 48px 0 24px; }
.ml-badge {
  display: inline-block; padding: 4px 14px; border: 1px solid rgba(245, 158, 11, 0.35);
  border-radius: 20px; font-size: 11px; letter-spacing: 0.2em; color: #f59e0b; margin-bottom: 14px;
}
.ml-title { margin: 0; font-size: clamp(38px, 6vw, 60px); font-weight: 800; letter-spacing: -0.03em; color: #f1f5f9; }
.ml-title .accent {
  background: linear-gradient(135deg, #f59e0b, #ef4444, #38bdf8);
  -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent;
}
.ml-sub { margin: 12px 0 0; color: #64748b; font-size: 15px; }
.ml-stats { display: flex; justify-content: center; gap: 36px; margin-top: 22px; flex-wrap: wrap; }
.stat { text-align: center; }
.stat-num {
  display: block; font-size: 24px; font-weight: 700;
  background: linear-gradient(135deg, #f59e0b, #38bdf8);
  -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent;
}
.stat-label { font-size: 12px; color: #64748b; text-transform: uppercase; letter-spacing: 0.08em; }

/* ===== Toolbar ===== */
.ml-toolbar {
  position: relative; z-index: 1;
  max-width: 1080px; margin: 0 auto 24px;
  display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap;
}
.toolbar-left { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.search-input { width: 260px; }
.sort-select { width: 130px; }
.pill {
  display: inline-flex; align-items: center; gap: 6px; padding: 7px 14px;
  border: 1px solid rgba(148, 163, 184, 0.2); border-radius: 20px;
  background: rgba(30, 41, 59, 0.5); color: #94a3b8; font-size: 13px; cursor: pointer; transition: all 0.2s;
}
.pill:hover { color: #e2e8f0; }
.pill.active { background: linear-gradient(135deg, #f59e0b, #ef4444); border-color: transparent; color: #fff; }
.upload-pill {
  display: inline-flex; align-items: center; gap: 8px; padding: 9px 20px;
  border: none; border-radius: 24px; cursor: pointer;
  background: linear-gradient(135deg, #f59e0b, #ef4444); color: #fff; font-size: 14px; font-weight: 600;
  transition: all 0.2s;
}
.upload-pill:hover { opacity: 0.9; transform: translateY(-1px); }

/* ===== Skeleton / Empty ===== */
.track-skeletons { position: relative; z-index: 1; max-width: 1080px; margin: 0 auto; display: flex; flex-direction: column; gap: 10px; }
.skel-row {
  height: 68px; border-radius: 16px;
  background: linear-gradient(110deg, rgba(30,41,59,0.6) 30%, rgba(51,65,85,0.4) 50%, rgba(30,41,59,0.6) 70%);
  background-size: 200% 100%; animation: shimmer 1.5s ease-in-out infinite;
}
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
.ml-empty { position: relative; z-index: 1; text-align: center; padding: 80px 0; color: #64748b; }
.ml-empty h3 { margin: 16px 0 8px; color: #f1f5f9; }

/* ===== Track list ===== */
.track-list { position: relative; z-index: 1; max-width: 1080px; margin: 0 auto; display: flex; flex-direction: column; gap: 8px; }
.track-row {
  display: flex; align-items: center; gap: 14px;
  padding: 10px 16px 10px 10px; border-radius: 16px;
  background: rgba(15, 23, 42, 0.7); border: 1px solid rgba(148, 163, 184, 0.07);
  cursor: pointer; transition: all 0.2s; animation: fadeUp 0.4s ease both;
}
.track-row:hover { border-color: rgba(245, 158, 11, 0.3); background: rgba(30, 41, 59, 0.7); }
.track-row.playing { border-color: #f59e0b; box-shadow: 0 0 20px rgba(245, 158, 11, 0.15); }
@keyframes fadeUp { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: none; } }
.row-index { width: 28px; text-align: center; color: #475569; font-size: 13px; font-variant-numeric: tabular-nums; flex-shrink: 0; }
.eq-dot { display: inline-block; width: 3px; height: 12px; margin: 0 1px; border-radius: 2px; background: #f59e0b; animation: eqb 0.7s ease-in-out infinite alternate; vertical-align: middle; }
@keyframes eqb { 0% { height: 4px; } 100% { height: 14px; } }
.row-cover {
  position: relative; width: 48px; height: 48px; border-radius: 12px; overflow: hidden;
  background: linear-gradient(135deg, #1e293b, #334155); flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
}
.row-cover img { width: 100%; height: 100%; object-fit: cover; display: block; }
.cover-fallback {
  font-size: 20px; font-weight: 700; color: #94a3b8;
}
.cover-fallback.big { font-size: 34px; }
.cover-play {
  position: absolute; inset: 0; border: none; cursor: pointer;
  background: rgba(2, 6, 23, 0.55); color: #fff;
  display: flex; align-items: center; justify-content: center;
  opacity: 0; transition: opacity 0.2s;
}
.row-cover:hover .cover-play { opacity: 1; }
.row-meta { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 2px; }
.row-title { font-size: 15px; font-weight: 600; color: #f1f5f9; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.row-sub { font-size: 12px; color: #64748b; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.row-lrc-tag {
  flex-shrink: 0; font-size: 10px; color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.4);
  padding: 1px 6px; border-radius: 8px;
}
.row-plays { flex-shrink: 0; font-size: 12px; color: #475569; width: 76px; text-align: right; }
.row-duration { flex-shrink: 0; font-size: 13px; color: #94a3b8; font-variant-numeric: tabular-nums; width: 44px; text-align: right; }
.row-actions { flex-shrink: 0; display: flex; gap: 4px; opacity: 0; transition: opacity 0.2s; }
.track-row:hover .row-actions { opacity: 1; }
.row-btn {
  width: 30px; height: 30px; border: none; border-radius: 50%;
  background: rgba(148, 163, 184, 0.12); color: #94a3b8; cursor: pointer;
  display: inline-flex; align-items: center; justify-content: center; transition: all 0.15s;
  text-decoration: none;
}
.row-btn:hover { background: rgba(56, 189, 248, 0.18); color: #38bdf8; }
.row-btn.faved { color: #f59e0b; }
.row-btn.danger:hover { background: rgba(239, 68, 68, 0.18); color: #ef4444; }

.load-more { position: relative; z-index: 1; text-align: center; margin-top: 28px; }

/* ===== Upload dialog ===== */
.upload-icon { color: #f59e0b; }
.pending-list { margin-top: 14px; border: 1px solid rgba(148, 163, 184, 0.15); border-radius: 12px; overflow: hidden; }
.pending-head { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; background: rgba(148, 163, 184, 0.08); font-size: 13px; }
.pending-item { display: flex; align-items: center; gap: 10px; padding: 8px 12px; border-top: 1px solid rgba(148, 163, 184, 0.08); }
.pending-icon { color: #f59e0b; }
.pending-meta { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.pending-name { font-size: 13px; color: #e2e8f0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pending-sub { font-size: 11px; color: #64748b; }
.upload-fields { margin-top: 12px; display: flex; gap: 10px; }

/* ===== Edit dialog ===== */
.edit-form { display: flex; flex-direction: column; gap: 12px; }
.edit-cover-row { display: flex; gap: 14px; align-items: center; }
.edit-cover {
  width: 84px; height: 84px; border-radius: 14px; overflow: hidden; flex-shrink: 0;
  background: linear-gradient(135deg, #1e293b, #334155);
  display: flex; align-items: center; justify-content: center;
}
.edit-cover img { width: 100%; height: 100%; object-fit: cover; display: block; }
.cover-img-wrap { width: 100%; height: 100%; display: block; }
.edit-cover-actions { display: flex; flex-direction: column; gap: 6px; }
.edit-hint { font-size: 11px; color: #64748b; }

/* ===== Lyrics drawer ===== */
.lyrics-pane { display: flex; flex-direction: column; height: 100%; color: #e2e8f0; }
.lyrics-head { display: flex; gap: 14px; align-items: center; padding: 8px 4px 18px; }
.lyrics-cover {
  width: 72px; height: 72px; border-radius: 14px; overflow: hidden; flex-shrink: 0;
  background: linear-gradient(135deg, #1e293b, #334155);
  display: flex; align-items: center; justify-content: center;
}
.lyrics-cover img { width: 100%; height: 100%; object-fit: cover; display: block; }
.lyrics-titles h3 { margin: 0 0 4px; font-size: 18px; color: #f1f5f9; }
.lyrics-titles p { margin: 0; font-size: 13px; color: #64748b; }
.lyrics-body {
  flex: 1; overflow-y: auto; padding: 8px 4px 40px;
  mask-image: linear-gradient(180deg, transparent 0, #000 8%, #000 92%, transparent 100%);
  -webkit-mask-image: linear-gradient(180deg, transparent 0, #000 8%, #000 92%, transparent 100%);
}
.lyrics-line {
  padding: 10px 8px; border-radius: 10px; cursor: pointer;
  font-size: 16px; line-height: 1.6; color: #64748b; transition: all 0.2s;
}
.lyrics-line:hover { color: #94a3b8; background: rgba(148, 163, 184, 0.06); }
.lyrics-line.active { color: #f59e0b; font-weight: 600; transform: scale(1.02); }
.lyrics-empty { padding: 40px 0; text-align: center; color: #475569; }

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .musiclib-app { padding: 0 12px 140px; }
  .ml-toolbar { flex-direction: column; align-items: stretch; }
  .search-input { width: 100%; }
  .row-plays { display: none; }
  .row-actions { opacity: 1; }
}
</style>
