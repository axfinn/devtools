<template>
  <div class="nfs-share-page" :class="{ 'with-chat': watchConnected }">
    <!-- Loading -->
    <div v-if="loading" class="status-container">
      <el-icon class="is-loading" style="font-size:36px"><Loading /></el-icon>
      <span>加载分享信息...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="status-container error">
      <el-icon style="font-size:36px;color:#f56c6c"><CircleClose /></el-icon>
      <h3>{{ error }}</h3>
    </div>

    <!-- 内容 -->
    <template v-else>
      <div class="content-layout">
        <!-- 左/主区域 -->
        <div class="main-area">

          <!-- 视频播放器 -->
          <div v-if="info.is_video" class="video-area">
            <div class="video-header">
              <h2 class="video-title">{{ info.name }}</h2>
              <div class="video-meta">
                <el-tag type="info" size="small">{{ formatSize(info.file_size) }}</el-tag>
                <el-tag :type="remainingTag" size="small">剩余 {{ info.remaining_views }} 次</el-tag>
                <el-tag v-if="watchConnected" type="success" size="small">
                  {{ viewerCount }} 人在看
                </el-tag>
              </div>
            </div>

            <div class="player-wrapper" ref="playerWrapper">
              <!-- 弹幕层 -->
              <div class="danmaku-layer" ref="danmakuLayer"></div>

              <!-- 转码中提示 -->
              <div v-if="transcoding" class="transcoding-overlay">
                <el-icon class="is-loading" style="font-size:40px;color:#409eff"><Loading /></el-icon>
                <p>视频转码中，请稍候...</p>
                <p class="hint">首次播放需要转码，大文件可能需要几分钟</p>
              </div>

              <!-- 视频元素 -->
              <video
                v-show="!transcoding"
                ref="videoEl"
                class="video-player"
                controls
                preload="metadata"
                @play="onHostPlay"
                @pause="onHostPause"
                @seeked="onHostSeek"
                @error="onVideoError"
              ></video>
            </div>

            <div class="video-actions">
              <el-button
                :type="watchConnected ? 'danger' : 'primary'"
                size="small"
                @click="toggleWatch"
              >
                <el-icon><VideoPlay v-if="!watchConnected" /><VideoPause v-else /></el-icon>
                {{ watchConnected ? '退出一起看' : '一起看' }}
              </el-button>
              <el-button v-if="!info.disable_video_download" size="small" :href="downloadUrl" tag="a" target="_blank">
                <el-icon><Download /></el-icon>
                下载
              </el-button>
            </div>

            <!-- 加入一起看 - 昵称输入弹窗 -->
            <el-dialog v-model="joinDialogVisible" title="加入一起看" width="360px" :show-close="false">
              <el-form @submit.prevent="confirmJoin">
                <el-form-item label="昵称">
                  <el-input v-model="joinNickname" placeholder="输入你的昵称" maxlength="20" autofocus />
                </el-form-item>
                <el-form-item label="管理密码（可选）">
                  <el-input v-model="joinAdminPwd" type="password" placeholder="填入则成为房主" show-password />
                </el-form-item>
              </el-form>
              <template #footer>
                <el-button @click="joinDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmJoin">加入</el-button>
              </template>
            </el-dialog>
          </div>

          <!-- 非视频文件：下载页 -->
          <div v-else class="download-area">
            <el-icon style="font-size:72px;color:#409eff"><Document /></el-icon>
            <h2>{{ info.name }}</h2>
            <div class="file-meta">
              <span>大小：{{ formatSize(info.file_size) }}</span>
              <span>类型：{{ info.mime_type }}</span>
              <el-tag :type="remainingTag" size="small">剩余 {{ info.remaining_views }} 次</el-tag>
            </div>
            <el-button type="primary" size="large" :href="downloadUrl" tag="a">
              <el-icon><Download /></el-icon>
              下载文件
            </el-button>
          </div>
        </div>

        <!-- 右侧：聊天栏（一起看时显示） -->
        <div v-if="watchConnected" class="chat-panel">
          <div class="chat-header">
            <span>聊天室</span>
            <span class="viewer-count">{{ viewerCount }} 人在线</span>
          </div>
          <div class="chat-messages" ref="chatMessagesEl">
            <div
              v-for="(msg, i) in chatMessages"
              :key="i"
              class="chat-msg"
              :class="{ 'is-host': msg.isHost, 'is-system': msg.isSystem }"
            >
              <template v-if="msg.isSystem">
                <span class="system-text">{{ msg.text }}</span>
              </template>
              <template v-else>
                <span class="msg-nick" :class="{ host: msg.isHost }">{{ msg.nickname }}</span>
                <span class="msg-text">{{ msg.text }}</span>
              </template>
            </div>
          </div>
          <div class="chat-input-area">
            <el-input
              v-model="chatInput"
              placeholder="发送消息..."
              maxlength="200"
              @keydown.enter.exact.prevent="sendChat"
            />
            <el-button type="primary" size="small" @click="sendChat">发送</el-button>
          </div>
          <div class="danmaku-input-area">
            <el-input
              v-model="danmakuInput"
              placeholder="发弹幕..."
              maxlength="50"
              @keydown.enter.exact.prevent="sendDanmaku"
            >
              <template #append>
                <el-button @click="sendDanmaku">发射</el-button>
              </template>
            </el-input>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import Hls from 'hls.js'

const route = useRoute()
const id = route.params.id

// ---- 状态 ----
const loading = ref(true)
const error = ref('')
const info = ref({})
const transcoding = ref(false)
const videoEl = ref(null)
const playerWrapper = ref(null)
const danmakuLayer = ref(null)

// ---- Watch Party ----
const watchConnected = ref(false)
const joinDialogVisible = ref(false)
const joinNickname = ref('')
const joinAdminPwd = ref('')
const chatMessages = ref([])
const chatInput = ref('')
const danmakuInput = ref('')
const chatMessagesEl = ref(null)
const viewerCount = ref(0)
const isHost = ref(false)
const myNickname = ref('')
let ws = null
let syncLock = false // 防止收到 sync 时触发重复 sync

// ---- HLS ----
let hls = null
let pollTimer = null

const downloadUrl = computed(() => `/api/nfsshare/${id}`)
const hlsUrl = computed(() => `/api/nfsshare/${id}/hls/index.m3u8`)

const remainingTag = computed(() => {
  const r = info.value.remaining_views
  if (r <= 1) return 'danger'
  if (r <= 3) return 'warning'
  return 'success'
})

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0, v = bytes
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return v.toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}

// ---- 加载信息 ----
async function loadInfo() {
  try {
    const res = await fetch(`/api/nfsshare/${id}/info`)
    if (!res.ok) {
      const data = await res.json()
      error.value = data.error || '分享不存在或已失效'
      return
    }
    const data = await res.json()
    if (data.expired) { error.value = '该分享已过期'; return }
    if (data.exhausted) { error.value = '该分享次数已用完'; return }
    info.value = data
    if (data.is_video) {
      await initPlayer()
    }
  } catch {
    error.value = '加载失败，请检查网络'
  } finally {
    loading.value = false
  }
}

// ---- 视频播放器 ----
async function initPlayer() {
  transcoding.value = true
  startHLS()
}

function startHLS() {
  const m3u8 = hlsUrl.value
  if (Hls.isSupported()) {
    if (hls) { hls.destroy(); hls = null }
    hls = new Hls({ manifestLoadingTimeOut: 30 * 60 * 1000 })
    hls.loadSource(m3u8)
    hls.attachMedia(videoEl.value)
    hls.on(Hls.Events.MANIFEST_PARSED, () => {
      transcoding.value = false
    })
    hls.on(Hls.Events.ERROR, (_, data) => {
      if (data.fatal && data.type === Hls.ErrorTypes.NETWORK_ERROR) {
        pollTranscoding()
      } else if (data.fatal) {
        error.value = '视频加载失败'
        transcoding.value = false
      }
    })
  } else if (videoEl.value.canPlayType('application/vnd.apple.mpegurl')) {
    videoEl.value.src = m3u8
    videoEl.value.addEventListener('loadedmetadata', () => { transcoding.value = false }, { once: true })
  } else {
    error.value = '浏览器不支持 HLS 播放'
    transcoding.value = false
  }
}

function pollTranscoding() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    try {
      const res = await fetch(hlsUrl.value, { method: 'HEAD' })
      if (res.ok) {
        clearInterval(pollTimer); pollTimer = null
        startHLS()
      }
    } catch (_) {}
  }, 5000)
}

function onVideoError() {
  if (!transcoding.value) error.value = '视频播放出错'
}

// ---- 房主同步事件 ----
function onHostPlay() {
  if (!isHost.value || syncLock || !ws || ws.readyState !== WebSocket.OPEN) return
  wsSend({ type: 'sync', action: 'play', time: videoEl.value.currentTime })
}
function onHostPause() {
  if (!isHost.value || syncLock || !ws || ws.readyState !== WebSocket.OPEN) return
  wsSend({ type: 'sync', action: 'pause', time: videoEl.value.currentTime })
}
function onHostSeek() {
  if (!isHost.value || syncLock || !ws || ws.readyState !== WebSocket.OPEN) return
  wsSend({ type: 'sync', action: 'seek', time: videoEl.value.currentTime })
}

// ---- 一起看 ----
function toggleWatch() {
  if (watchConnected.value) {
    disconnectWatch()
  } else {
    // 预填昵称
    const savedPwd = sessionStorage.getItem('nfs_admin_password') || ''
    joinAdminPwd.value = savedPwd
    joinNickname.value = ''
    joinDialogVisible.value = true
  }
}

function confirmJoin() {
  joinDialogVisible.value = false
  const nick = joinNickname.value.trim() || '匿名用户'
  connectWatch(nick, joinAdminPwd.value)
}

function connectWatch(nickname, adminPwd) {
  myNickname.value = nickname
  isHost.value = !!adminPwd
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  let url = `${proto}://${location.host}/api/nfsshare/${id}/watch/ws?nickname=${encodeURIComponent(nickname)}`
  if (adminPwd) url += `&admin_password=${encodeURIComponent(adminPwd)}`
  ws = new WebSocket(url)
  ws.onopen = () => {
    watchConnected.value = true
    addSystemMsg('已加入观看室')
  }
  ws.onmessage = (e) => {
    try { handleWsMsg(JSON.parse(e.data)) } catch (_) {}
  }
  ws.onclose = () => {
    watchConnected.value = false
    addSystemMsg('已断开连接')
  }
  ws.onerror = () => {
    addSystemMsg('连接出错')
  }
}

function disconnectWatch() {
  if (ws) { ws.close(); ws = null }
  watchConnected.value = false
}

function wsSend(obj) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(obj))
  }
}

function handleWsMsg(msg) {
  switch (msg.type) {
    case 'joined':
      viewerCount.value = msg.count
      addSystemMsg(`${msg.nickname}${msg.is_host ? '（房主）' : ''} 加入了`)
      break
    case 'left':
      viewerCount.value = msg.count
      addSystemMsg(`${msg.nickname} 离开了`)
      break
    case 'chat':
      chatMessages.value.push({ nickname: msg.nickname, text: msg.text, isHost: msg.is_host })
      scrollChat()
      break
    case 'danmaku':
      fireDanmaku(msg.text)
      break
    case 'sync':
      if (!videoEl.value || isHost.value) break
      syncLock = true
      if (msg.action === 'seek' || Math.abs(videoEl.value.currentTime - msg.time) > 2) {
        videoEl.value.currentTime = msg.time
      }
      if (msg.action === 'play') videoEl.value.play().catch(() => {})
      else if (msg.action === 'pause') videoEl.value.pause()
      setTimeout(() => { syncLock = false }, 500)
      break
  }
}

function sendChat() {
  const text = chatInput.value.trim()
  if (!text) return
  wsSend({ type: 'chat', text })
  chatInput.value = ''
}

function sendDanmaku() {
  const text = danmakuInput.value.trim()
  if (!text) return
  wsSend({ type: 'danmaku', text })
  danmakuInput.value = ''
  fireDanmaku(text) // 自己也显示
}

function addSystemMsg(text) {
  chatMessages.value.push({ text, isSystem: true })
  scrollChat()
}

function scrollChat() {
  nextTick(() => {
    if (chatMessagesEl.value) {
      chatMessagesEl.value.scrollTop = chatMessagesEl.value.scrollHeight
    }
  })
}

// ---- 弹幕 ----
const danmakuColors = ['#fff', '#fe0', '#0ff', '#f0f', '#0f0', '#f90', '#09f']
function fireDanmaku(text) {
  if (!danmakuLayer.value) return
  const el = document.createElement('div')
  el.className = 'danmaku-item'
  el.textContent = text
  el.style.color = danmakuColors[Math.floor(Math.random() * danmakuColors.length)]
  const top = Math.random() * 70 // 0-70% 高度
  el.style.top = top + '%'
  el.style.right = '-300px'
  danmakuLayer.value.appendChild(el)
  // 动画
  const duration = 6000 + Math.random() * 4000
  el.animate([
    { right: '-300px', opacity: 1 },
    { right: '110%', opacity: 0.8 }
  ], { duration, easing: 'linear', fill: 'forwards' }).onfinish = () => el.remove()
}

onMounted(() => { loadInfo() })
onUnmounted(() => {
  if (hls) { hls.destroy(); hls = null }
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  if (ws) { ws.close(); ws = null }
})
</script>

<style scoped>
.nfs-share-page {
  min-height: 100vh;
  background: #0f0f0f;
  color: #e0e0e0;
}

.status-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  min-height: 60vh;
  color: #999;
  font-size: 16px;
}

.content-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding: 20px;
}

/* 视频区域 */
.video-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.video-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.video-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #e0e0e0;
  flex: 1;
  min-width: 0;
  word-break: break-word;
}

.video-meta {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-shrink: 0;
}

.player-wrapper {
  position: relative;
  width: 100%;
  aspect-ratio: 16/9;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.video-player {
  width: 100%;
  height: 100%;
  display: block;
}

/* 弹幕层 */
.danmaku-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
  overflow: hidden;
}

:global(.danmaku-item) {
  position: absolute;
  white-space: nowrap;
  font-size: 18px;
  font-weight: bold;
  text-shadow: 1px 1px 2px #000, -1px -1px 2px #000;
  pointer-events: none;
  user-select: none;
}

.transcoding-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(0,0,0,.85);
  color: #e0e0e0;
  gap: 12px;
  font-size: 15px;
  z-index: 10;
}

.transcoding-overlay .hint {
  font-size: 12px;
  color: #888;
  margin: 0;
}

.video-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 聊天面板 */
.chat-panel {
  width: 280px;
  flex-shrink: 0;
  border-left: 1px solid #2a2a2a;
  background: #141414;
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.chat-header {
  padding: 12px 16px;
  border-bottom: 1px solid #2a2a2a;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  font-weight: 600;
  color: #e0e0e0;
}

.viewer-count {
  font-size: 12px;
  color: #67c23a;
  font-weight: normal;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.chat-msg {
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
}

.chat-msg.is-system .system-text {
  color: #666;
  font-size: 11px;
  font-style: italic;
}

.msg-nick {
  color: #888;
  margin-right: 4px;
  font-size: 11px;
}

.msg-nick.host {
  color: #f0a020;
}

.msg-text {
  color: #d0d0d0;
}

.chat-input-area {
  padding: 8px 12px;
  border-top: 1px solid #2a2a2a;
  display: flex;
  gap: 6px;
}

.chat-input-area .el-input {
  flex: 1;
}

.danmaku-input-area {
  padding: 6px 12px 12px;
}

/* 下载页 */
.download-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 20px;
  padding: 60px 16px;
  text-align: center;
}

.download-area h2 {
  margin: 0;
  font-size: 20px;
  color: #e0e0e0;
  word-break: break-all;
  max-width: 600px;
}

.file-meta {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  justify-content: center;
  color: #999;
  font-size: 14px;
}

@media (max-width: 768px) {
  .content-layout { flex-direction: column; height: auto; }
  .chat-panel { width: 100%; height: 40vh; border-left: none; border-top: 1px solid #2a2a2a; }
  .main-area { padding: 10px 8px; }
  .video-title { font-size: 15px; }
}
</style>
