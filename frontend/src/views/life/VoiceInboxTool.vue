<template>
  <div class="voice-inbox">
    <!-- Header -->
    <div class="inbox-header">
      <h1 class="inbox-title">语音收件箱</h1>
      <p class="inbox-sub">录音会归档到当前事项档案，再决定是否转成具体事项</p>
    </div>

    <div class="archive-banner" :class="{ missing: !hasPlannerArchive }">
      <div class="archive-copy">
        <strong>{{ hasPlannerArchive ? '当前事项档案已连接' : '未连接事项档案' }}</strong>
        <span v-if="hasPlannerArchive">档案 ID：{{ plannerSession.profileId }}</span>
        <span v-else>语音收件箱现在依附于事项档案，请先去事项管理创建或登录档案。</span>
      </div>
      <el-button :type="hasPlannerArchive ? 'default' : 'primary'" @click="openPlanner">
        {{ hasPlannerArchive ? '打开事项管理' : '去连接档案' }}
      </el-button>
    </div>

    <template v-if="hasPlannerArchive">

    <!-- Main Record Area -->
    <div class="record-zone">
      <div class="record-btn-wrapper" :class="{ recording: isRecording }">
        <div class="record-ripple" v-if="isRecording"></div>
        <button class="record-btn" @click="toggleRecording" :disabled="recordingReady === false">
          <el-icon :size="isRecording ? 36 : 40">
            <component :is="isRecording ? 'VideoPause' : 'Microphone'" />
          </el-icon>
        </button>
      </div>
      <div class="record-label" v-if="!isRecording">点击开始录音</div>
      <div class="record-label recording-label" v-else>
        正在录音 {{ formatDuration(recordingTime) }}
      </div>
      <div class="record-waveform" v-if="isRecording">
        <span v-for="i in 5" :key="i" class="wave-bar" :style="{ animationDelay: `${i * 0.12}s` }"></span>
      </div>
    </div>

    <!-- Search -->
    <div class="search-bar" v-if="memos.length > 0 || searchQuery">
      <el-input v-model="searchQuery" placeholder="搜索备忘..." :prefix-icon="Search" clearable size="large" />
    </div>

    <!-- Memo List -->
    <div class="memo-list" v-if="filteredMemos.length > 0">
      <TransitionGroup name="memo">
        <div
          v-for="memo in filteredMemos"
          :key="memo.id"
          class="memo-card"
          :class="{
            expanded: expandedId === memo.id,
            'is-draft': memo.status === 'draft',
            'is-expiring': isExpiring(memo)
          }"
        >
          <div class="memo-main" @click="expandMemo(memo)">
            <div class="memo-icon" :class="memo.status">
              <el-icon :size="20">
                <component :is="statusIcon(memo.status)" />
              </el-icon>
            </div>
            <div class="memo-info">
              <div class="memo-title">{{ memo.title }}</div>
              <div class="memo-meta">
                <span class="memo-date">{{ formatDate(memo.created_at) }}</span>
                <span class="memo-dur" v-if="memo.duration_sec > 0">{{ formatDuration(memo.duration_sec) }}</span>
                <span class="memo-status-tag" :class="memo.status">{{ statusLabel(memo) }}</span>
                <span class="memo-expiry" v-if="memo.status === 'draft' && memo.expires_at">
                  {{ expiryText(memo.expires_at) }}
                </span>
              </div>
            </div>
            <div class="memo-actions" @click.stop>
              <!-- Draft: show transcribe button -->
              <el-button
                v-if="memo.status === 'draft' || memo.status === 'failed'"
                circle size="small" type="primary"
                @click="transcribeMemo(memo)"
                :loading="memo.status === 'transcribing'"
              >
                <el-icon :size="16"><MagicStick /></el-icon>
              </el-button>
              <!-- Play -->
              <el-button circle size="small" @click="playMemo(memo)">
                <el-icon :size="16">
                  <component :is="playingId === memo.id && playerPlaying ? 'VideoPause' : 'VideoPlay'" />
                </el-icon>
              </el-button>
              <el-button circle size="small" type="danger" @click="deleteMemo(memo)">
                <el-icon :size="16"><Delete /></el-icon>
              </el-button>
            </div>
          </div>

          <!-- Expanded area -->
          <div class="memo-expand" v-if="expandedId === memo.id">
            <!-- Edit title -->
            <div class="memo-field">
              <label>标题</label>
              <el-input v-model="editTitle" size="default" placeholder="备忘标题" />
            </div>

            <!-- Transcript or actions -->
            <div v-if="memo.status === 'draft'" class="memo-cta">
              <p>录音已保存为草稿，14天后自动过期。点击转写按钮生成文字。</p>
              <el-button type="primary" @click="transcribeMemo(memo)" :loading="transcribingId === memo.id">
                <el-icon><MagicStick /></el-icon> 开始语音转文字
              </el-button>
            </div>

            <div v-else-if="memo.status === 'transcribing'" class="memo-transcript transcribing">
              <el-icon class="is-loading"><Loading /></el-icon> 正在语音识别中，请稍候...
            </div>

            <div v-else-if="memo.status === 'completed' || memo.status === 'saved'" class="memo-field">
              <label>转写内容</label>
              <el-input
                v-model="editTranscript"
                type="textarea"
                :rows="4"
                placeholder="转写内容，可编辑修改..."
                resize="vertical"
              />
            </div>

            <div v-else-if="memo.status === 'failed'" class="memo-transcript error">
              识别失败：{{ memo.error_message || '未知错误' }}，可重新转写
            </div>

            <!-- Audio player -->
            <div class="memo-audio-bar" v-if="playingId === memo.id">
              <input
                type="range"
                min="0"
                :max="playerDuration || 1"
                :value="playerCurrentTime"
                @input="seekAudio($event.target.value)"
                class="audio-slider"
              />
              <div class="audio-time">
                <span>{{ formatDuration(playerCurrentTime) }}</span>
                <span>{{ formatDuration(playerDuration) }}</span>
              </div>
            </div>

            <!-- Save button -->
            <div class="memo-save-row" v-if="memo.status === 'completed' || memo.status === 'saved'">
              <el-button
                v-if="memo.status !== 'saved'"
                type="success"
                @click="saveMemo(memo)"
              >
                确认入库（永久保留）
              </el-button>
              <span class="saved-badge" v-else>
                <el-icon><CircleCheckFilled /></el-icon> 已入库，永久保留
              </span>
              <el-button
                v-if="!memo.planner_task_id"
                type="primary"
                plain
                @click="createPlannerTaskFromMemo(memo)"
              >
                转入{{ plannerKindLabel }}
              </el-button>
              <span class="saved-badge planner-badge" v-else>
                <el-icon><Document /></el-icon> 已同步到事项档案
              </span>
              <el-button
                v-if="memo.planner_task_id"
                plain
                @click="openPlanner"
              >
                打开事项管理
              </el-button>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>

    <!-- Empty State -->
    <div class="empty-state" v-else-if="!isRecording && loaded">
      <el-icon :size="48" color="#94a3b8"><Microphone /></el-icon>
      <p v-if="searchQuery">没有找到匹配的备忘</p>
      <p v-else>还没有语音备忘，点击上方按钮开始录音</p>
    </div>

    </template>

    <div v-else class="empty-state archive-empty">
      <el-icon :size="48" color="#94a3b8"><Lock /></el-icon>
      <p>先进入事项管理档案，语音收件箱才会开始保存和展示内容。</p>
    </div>

    <audio ref="audioRef" @timeupdate="onAudioTime" @ended="onAudioEnd" @loadedmetadata="onAudioMeta" style="display:none" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Microphone, VideoPause, VideoPlay, Document, Loading, WarningFilled,
  Delete, Search, MagicStick, CircleCheckFilled, Lock
} from '@element-plus/icons-vue'

const STORAGE_KEY = 'voicememo_device_id'
const PLANNER_CREATOR_KEY_STORAGE_KEY = 'planner_creator_keys'
const router = useRouter()

function getDeviceId() {
  let id = localStorage.getItem(STORAGE_KEY)
  if (!id) {
    id = 'dev_' + Math.random().toString(36).slice(2, 10) + Date.now().toString(36)
    localStorage.setItem(STORAGE_KEY, id)
  }
  return id
}

const deviceId = getDeviceId()
const plannerSession = reactive({
  profileId: '',
  password: '',
  creatorKey: '',
  activeKind: 'life',
})
const hasPlannerArchive = computed(() => Boolean(plannerSession.profileId && plannerSession.password))
const plannerKindLabel = computed(() => plannerSession.activeKind === 'work' ? '工作事项' : '生活事项')

function loadPlannerSession() {
  plannerSession.profileId = localStorage.getItem('planner_profile_id') || ''
  plannerSession.password = localStorage.getItem('planner_password') || ''
  plannerSession.activeKind = localStorage.getItem('planner_active_kind') || 'life'
  try {
    const keyMap = JSON.parse(localStorage.getItem(PLANNER_CREATOR_KEY_STORAGE_KEY) || '{}')
    plannerSession.creatorKey = plannerSession.profileId ? (keyMap[plannerSession.profileId] || '') : ''
  } catch {
    plannerSession.creatorKey = ''
  }
}

function plannerHeaders(extra = {}) {
  const headers = { ...extra, 'X-Device-ID': deviceId }
  if (plannerSession.password) {
    headers['X-Password'] = plannerSession.password
  }
  if (plannerSession.creatorKey) {
    headers['X-Creator-Key'] = plannerSession.creatorKey
  }
  return headers
}

function withProfileQuery(path) {
  if (!plannerSession.profileId) return path
  const separator = path.includes('?') ? '&' : '?'
  return `${path}${separator}profile_id=${encodeURIComponent(plannerSession.profileId)}`
}

async function voiceFetch(path, options = {}) {
  const headers = plannerHeaders(options.body instanceof FormData ? (options.headers || {}) : { 'Content-Type': 'application/json', ...(options.headers || {}) })
  const response = await fetch(withProfileQuery(path), {
    ...options,
    headers,
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}))
    throw new Error(payload.error || '请求失败')
  }
  return response
}

function openPlanner() {
  router.push('/planner')
}

// ── Recording State ──
const isRecording = ref(false)
const recordingTime = ref(0)
const recordingReady = ref(true)
let mediaRecorder = null
let recordingTimer = null
let audioChunks = []
let recordingStartTime = 0

// ── Memo List ──
const memos = ref([])
const total = ref(0)
const loaded = ref(false)
const searchQuery = ref('')
const expandedId = ref(null)
const editTitle = ref('')
const editTranscript = ref('')
const transcribingId = ref(null)
let pollTimer = null

// ── Playback ──
const audioRef = ref(null)
const playingId = ref(null)
const playerPlaying = ref(false)
const playerCurrentTime = ref(0)
const playerDuration = ref(0)
let currentAudioObjectURL = ''

const filteredMemos = computed(() => {
  if (!searchQuery.value) return memos.value
  const q = searchQuery.value.toLowerCase()
  return memos.value.filter(m =>
    m.title.toLowerCase().includes(q) ||
    (m.transcript && m.transcript.toLowerCase().includes(q))
  )
})

// ── Helpers ──
function statusIcon(status) {
  return { draft: 'Headset', transcribing: 'Loading', completed: 'Document', saved: 'CircleCheckFilled', failed: 'WarningFilled' }[status] || 'Document'
}

function statusLabel(memo) {
  if (memo.status === 'draft') return '草稿'
  if (memo.status === 'transcribing') return '转写中...'
  if (memo.status === 'completed') return '待入库'
  if (memo.status === 'saved') return '已入库'
  if (memo.status === 'failed') return '转写失败'
  return memo.status
}

function isExpiring(memo) {
  if (memo.status !== 'draft' || !memo.expires_at) return false
  const days = daysLeft(memo.expires_at)
  return days <= 3
}

function daysLeft(expiresAt) {
  return Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (86400 * 1000)))
}

function expiryText(expiresAt) {
  const days = daysLeft(expiresAt)
  if (days <= 0) return '即将过期'
  return `${days}天后过期`
}

// ── Recording ──
async function toggleRecording() {
  if (isRecording.value) {
    stopRecording()
  } else {
    startRecording()
  }
}

async function startRecording() {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
      ? 'audio/webm;codecs=opus'
      : MediaRecorder.isTypeSupported('audio/webm')
        ? 'audio/webm'
        : 'audio/mp4'

    mediaRecorder = new MediaRecorder(stream, { mimeType })
    audioChunks = []

    mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) audioChunks.push(e.data)
    }

    mediaRecorder.onstop = async () => {
      stream.getTracks().forEach(t => t.stop())
      const blob = new Blob(audioChunks, { type: mimeType })
      await uploadMemo(blob)
    }

    mediaRecorder.start(1000)
    isRecording.value = true
    recordingTime.value = 0
    recordingStartTime = Date.now()

    recordingTimer = setInterval(() => {
      recordingTime.value = (Date.now() - recordingStartTime) / 1000
    }, 100)
  } catch (err) {
    ElMessage.error('无法访问麦克风，请检查权限设置')
  }
}

function stopRecording() {
  if (mediaRecorder && mediaRecorder.state !== 'inactive') {
    mediaRecorder.stop()
  }
  isRecording.value = false
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
}

async function uploadMemo(blob) {
  try {
    const form = new FormData()
    form.append('file', blob, `memo_${Date.now()}.webm`)
    form.append('device_id', deviceId)
    form.append('profile_id', plannerSession.profileId)
    form.append('duration', String(recordingTime.value))

    const resp = await voiceFetch('/api/voicememo/upload', { method: 'POST', body: form })
    const data = await resp.json()
    memos.value.unshift(data)
    total.value++
    ElMessage.success('录音已保存到当前事项档案')
  } catch (err) {
    ElMessage.error('保存失败: ' + err.message)
  }
}

// ── Transcribe ──
async function transcribeMemo(memo) {
  try {
    transcribingId.value = memo.id
    const resp = await voiceFetch(`/api/voicememo/${memo.id}/transcribe`, { method: 'POST' })
    memo.status = 'transcribing'
    ElMessage.success('已开始语音转文字...')
    startPolling()
  } catch (err) {
    ElMessage.error(err.message)
  } finally {
    transcribingId.value = null
  }
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    const transcribing = memos.value.filter(m => m.status === 'transcribing')
    if (transcribing.length === 0) {
      clearInterval(pollTimer)
      pollTimer = null
      return
    }
    for (const m of transcribing) {
      try {
        const resp = await voiceFetch(`/api/voicememo/${m.id}`)
        const data = await resp.json()
        if (data.status !== 'transcribing') {
          Object.assign(m, data)
          if (data.status === 'completed') {
            ElMessage.success(`"${m.title}" 转写完成`)
          }
        }
      } catch {}
    }
  }, 3000)
}

// ── Save ──
async function saveMemo(memo) {
  try {
    const resp = await voiceFetch(`/api/voicememo/${memo.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        title: editTitle.value || memo.title,
        transcript: editTranscript.value || memo.transcript,
        status: 'saved',
      }),
    })
    const data = await resp.json()
    Object.assign(memo, data)
    ElMessage.success('已入库，永久保留')
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function createPlannerTaskFromMemo(memo) {
  try {
    const resp = await voiceFetch(`/api/voicememo/${memo.id}/planner-task`, {
      method: 'POST',
      body: JSON.stringify({
        profile_id: plannerSession.profileId,
        kind: plannerSession.activeKind,
        bucket: 'inbox',
        title: editTitle.value || memo.title,
        transcript: editTranscript.value || memo.transcript,
      }),
    })
    const data = await resp.json()
    Object.assign(memo, data.memo || {})
    ElMessage.success(`已转入${plannerKindLabel.value}`)
  } catch (err) {
    ElMessage.error(err.message)
  }
}

// ── Expand ──
function expandMemo(memo) {
  if (expandedId.value === memo.id) {
    expandedId.value = null
    editTitle.value = ''
    editTranscript.value = ''
  } else {
    expandedId.value = memo.id
    editTitle.value = memo.title
    editTranscript.value = memo.transcript || ''
  }
}

// ── Playback ──
function playMemo(memo) {
  if (playingId.value === memo.id) {
    if (playerPlaying.value) {
      audioRef.value?.pause()
      playerPlaying.value = false
    } else {
      audioRef.value?.play().catch(() => {})
      playerPlaying.value = true
    }
    return
  }
  loadAndPlayMemoAudio(memo)
}

async function loadAndPlayMemoAudio(memo) {
  try {
    const resp = await voiceFetch(memo.audio_url, { headers: {} })
    const blob = await resp.blob()
    if (currentAudioObjectURL) {
      URL.revokeObjectURL(currentAudioObjectURL)
      currentAudioObjectURL = ''
    }
    currentAudioObjectURL = URL.createObjectURL(blob)
    if (audioRef.value) {
      audioRef.value.src = currentAudioObjectURL
      await audioRef.value.play().catch(() => {})
    }
    playingId.value = memo.id
    playerPlaying.value = true
    playerCurrentTime.value = 0
  } catch (err) {
    ElMessage.error(err.message || '音频加载失败')
  }
}

function seekAudio(val) { if (audioRef.value) audioRef.value.currentTime = parseFloat(val) }
function onAudioTime() { if (audioRef.value) playerCurrentTime.value = audioRef.value.currentTime }
function onAudioMeta() { if (audioRef.value) playerDuration.value = audioRef.value.duration }
function onAudioEnd() { playerPlaying.value = false; playerCurrentTime.value = 0 }

// ── Delete ──
async function deleteMemo(memo) {
  try {
    await ElMessageBox.confirm('确定删除这条语音备忘吗？', '确认删除', {
      type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消',
    })
  } catch { return }

  try {
    const resp = await voiceFetch(`/api/voicememo/${memo.id}`, { method: 'DELETE' })
    memos.value = memos.value.filter(m => m.id !== memo.id)
    total.value--
    if (playingId.value === memo.id) {
      audioRef.value?.pause()
      playingId.value = null
    }
    if (expandedId.value === memo.id) expandedId.value = null
    ElMessage.success('已删除')
  } catch (err) {
    ElMessage.error(err.message)
  }
}

// ── Load ──
async function loadMemos() {
  if (!hasPlannerArchive.value) {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    memos.value = []
    total.value = 0
    loaded.value = true
    return
  }
  try {
    const resp = await voiceFetch(`/api/voicememo/list?device_id=${encodeURIComponent(deviceId)}&limit=50`)
    const data = await resp.json()
    memos.value = data.items || []
    total.value = data.total || 0
    loaded.value = true
    if (memos.value.some(m => m.status === 'transcribing')) startPolling()
  } catch (err) {
    loaded.value = true
  }
}

// ── Formatting ──
function formatDuration(sec) {
  if (!sec || sec <= 0) return '0:00'
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const now = new Date()
  const diffMin = Math.floor((now - d) / 60000)
  if (diffMin < 1) return '刚刚'
  if (diffMin < 60) return `${diffMin} 分钟前`
  if (diffMin < 1440) return `${Math.floor(diffMin / 60)} 小时前`
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

function handleVisibilityChange() {
  if (document.visibilityState !== 'visible') return
  const previousProfile = plannerSession.profileId
  loadPlannerSession()
  if (plannerSession.profileId !== previousProfile) {
    expandedId.value = null
  }
  loadMemos()
}

onMounted(() => {
  loadPlannerSession()
  loadMemos()
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('storage', handleVisibilityChange)
})

onUnmounted(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('storage', handleVisibilityChange)
  if (currentAudioObjectURL) {
    URL.revokeObjectURL(currentAudioObjectURL)
    currentAudioObjectURL = ''
  }
})
</script>

<style scoped>
.voice-inbox {
  max-width: 640px;
  margin: 0 auto;
  padding: 20px 16px 40px;
}

.inbox-header { text-align: center; margin-bottom: 28px; }
.inbox-title { font-size: 22px; font-weight: 700; color: #1e293b; margin: 0 0 4px; }
.inbox-sub { font-size: 13px; color: #94a3b8; margin: 0; }

.archive-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  margin-bottom: 24px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.08), rgba(14, 165, 233, 0.08));
  border: 1px solid rgba(14, 165, 233, 0.14);
}

.archive-banner.missing {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.08), rgba(239, 68, 68, 0.06));
  border-color: rgba(245, 158, 11, 0.2);
}

.archive-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.archive-copy strong {
  font-size: 14px;
  color: #0f172a;
}

.archive-copy span {
  font-size: 12px;
  color: #64748b;
  word-break: break-all;
}

/* ── Record ── */
.record-zone { display: flex; flex-direction: column; align-items: center; margin-bottom: 24px; }
.record-btn-wrapper { position: relative; width: 80px; height: 80px; display: flex; align-items: center; justify-content: center; }
.record-btn {
  width: 72px; height: 72px; border-radius: 50%; border: none;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; cursor: pointer; display: flex; align-items: center; justify-content: center;
  transition: all 0.3s ease; box-shadow: 0 4px 20px rgba(99,102,241,0.35); z-index: 1;
}
.record-btn:hover { transform: scale(1.05); box-shadow: 0 6px 28px rgba(99,102,241,0.45); }
.recording .record-btn {
  width: 80px; height: 80px;
  background: linear-gradient(135deg, #ef4444, #f97316);
  box-shadow: 0 4px 24px rgba(239,68,68,0.4);
  animation: pulse-btn 1.5s ease infinite;
}
@keyframes pulse-btn {
  0%,100% { box-shadow: 0 4px 24px rgba(239,68,68,0.4); }
  50% { box-shadow: 0 4px 40px rgba(239,68,68,0.7); }
}
.record-ripple {
  position: absolute; top: -8px; left: -8px; right: -8px; bottom: -8px;
  border-radius: 50%; border: 2px solid rgba(239,68,68,0.3);
  animation: ripple 1.5s ease infinite;
}
@keyframes ripple {
  0% { transform: scale(1); opacity: 1; }
  100% { transform: scale(1.6); opacity: 0; }
}
.record-label { margin-top: 12px; font-size: 14px; color: #94a3b8; }
.recording-label { color: #ef4444; font-weight: 600; animation: blink-text 1s step-end infinite; }
@keyframes blink-text { 0%,100% { opacity: 1; } 50% { opacity: 0.5; } }
.record-waveform { display: flex; gap: 3px; align-items: flex-end; height: 32px; margin-top: 10px; }
.wave-bar { width: 4px; height: 8px; background: #ef4444; border-radius: 2px; animation: wave 0.6s ease-in-out infinite alternate; }
@keyframes wave { 0% { height: 8px; } 100% { height: 28px; } }

/* ── Search ── */
.search-bar { margin-bottom: 16px; }
.search-bar :deep(.el-input__wrapper) { border-radius: 12px; background: #f8fafc; }

/* ── Cards ── */
.memo-list { display: flex; flex-direction: column; gap: 10px; }
.memo-card {
  background: #fff; border-radius: 14px; border: 1px solid #f1f5f9;
  overflow: hidden; transition: box-shadow 0.2s;
}
.memo-card:hover { box-shadow: 0 2px 12px rgba(0,0,0,0.06); }
.memo-card.is-draft { border-left: 3px solid #f59e0b; }
.memo-card.is-expiring { border-left-color: #ef4444; background: #fffbfb; }
.memo-main {
  display: flex; align-items: center; padding: 14px 16px; gap: 12px; cursor: pointer;
}
.memo-icon {
  width: 40px; height: 40px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.memo-icon.draft { background: #fffbeb; color: #f59e0b; }
.memo-icon.transcribing { background: #eff6ff; color: #3b82f6; }
.memo-icon.completed { background: #ecfdf5; color: #10b981; }
.memo-icon.saved { background: #f0fdf4; color: #16a34a; }
.memo-icon.failed { background: #fef2f2; color: #ef4444; }
.memo-info { flex: 1; min-width: 0; }
.memo-title { font-size: 14px; font-weight: 600; color: #1e293b; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.memo-meta { display: flex; align-items: center; gap: 8px; margin-top: 3px; font-size: 12px; color: #94a3b8; flex-wrap: wrap; }
.memo-status-tag { font-size: 11px; padding: 1px 6px; border-radius: 4px; font-weight: 500; }
.memo-status-tag.draft { background: #fffbeb; color: #d97706; }
.memo-status-tag.transcribing { background: #eff6ff; color: #3b82f6; }
.memo-status-tag.completed { background: #ecfdf5; color: #059669; }
.memo-status-tag.saved { background: #f0fdf4; color: #16a34a; }
.memo-status-tag.failed { background: #fef2f2; color: #dc2626; }
.memo-expiry { font-size: 11px; color: #ef4444; font-weight: 500; }
.memo-actions { display: flex; gap: 6px; flex-shrink: 0; }

/* ── Expanded ── */
.memo-expand { padding: 0 16px 16px; border-top: 1px solid #f8fafc; }
.memo-field { margin-top: 12px; }
.memo-field label { display: block; font-size: 12px; color: #64748b; margin-bottom: 4px; font-weight: 500; }
.memo-cta { margin-top: 12px; padding: 16px; background: #fffbeb; border-radius: 10px; text-align: center; }
.memo-cta p { font-size: 13px; color: #92400e; margin: 0 0 10px; }
.memo-transcript {
  margin-top: 12px; padding: 12px; background: #f8fafc; border-radius: 10px;
  font-size: 13px; line-height: 1.7; color: #334155;
  white-space: pre-wrap; word-break: break-word;
}
.memo-transcript.transcribing { color: #3b82f6; display: flex; align-items: center; gap: 6px; }
.memo-transcript.error { color: #dc2626; }
.memo-audio-bar { margin-top: 10px; }
.audio-slider {
  width: 100%; height: 4px; -webkit-appearance: none; appearance: none;
  background: #e2e8f0; border-radius: 2px; outline: none; cursor: pointer;
}
.audio-slider::-webkit-slider-thumb {
  -webkit-appearance: none; width: 14px; height: 14px; border-radius: 50%; background: #6366f1; cursor: pointer;
}
.audio-time { display: flex; justify-content: space-between; font-size: 11px; color: #94a3b8; margin-top: 4px; }
.memo-save-row { margin-top: 14px; display: flex; align-items: center; gap: 8px; }
.saved-badge { font-size: 13px; color: #16a34a; display: flex; align-items: center; gap: 4px; font-weight: 500; }
.planner-badge { color: #0f766e; }

/* ── Empty ── */
.empty-state { text-align: center; padding: 60px 20px; color: #94a3b8; }
.empty-state p { margin: 12px 0 0; font-size: 14px; }
.archive-empty { padding-top: 48px; }

/* ── Transitions ── */
.memo-enter-active, .memo-leave-active { transition: all 0.3s ease; }
.memo-enter-from { opacity: 0; transform: translateY(10px); }
.memo-leave-to { opacity: 0; transform: translateX(-20px); }

@media (max-width: 640px) {
  .archive-banner {
    flex-direction: column;
    align-items: stretch;
  }

  .memo-save-row {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
