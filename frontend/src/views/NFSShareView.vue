<template>
  <div class="nfs-share-page" :class="{ 'with-chat': watchConnected }">
    <!-- 密码验证弹窗 -->
    <el-dialog v-model="passwordDialogVisible" title="访问密码" width="340px" :show-close="false" :close-on-click-modal="false">
      <el-form @submit.prevent="confirmPassword">
        <p style="color:#999;font-size:13px;margin:0 0 16px">该分享需要密码才能访问</p>
        <el-form-item>
          <el-input
            v-model="inputPassword"
            type="password"
            placeholder="输入访问密码"
            show-password
            autofocus
            @keydown.enter.prevent="confirmPassword"
          />
        </el-form-item>
        <p v-if="passwordError" style="color:#f56c6c;font-size:12px;margin:4px 0 0">{{ passwordError }}</p>
      </el-form>
      <template #footer>
        <el-button type="primary" :loading="passwordChecking" @click="confirmPassword">进入</el-button>
      </template>
    </el-dialog>

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

              <!-- 视频元素（ArtPlayer 挂载点） -->
              <div
                v-show="!transcoding"
                ref="artRef"
                class="art-player-container"
              ></div>
            </div>

            <div class="video-actions">
              <!-- 一起看：已连接时只显示退出，未连接时显示加入和管理员登录 -->
              <el-button
                v-if="watchConnected"
                type="danger"
                size="small"
                @click="disconnectWatch"
              >
                <el-icon><VideoPause /></el-icon>
                退出一起看
              </el-button>
              <template v-else>
                <el-button
                  v-if="!info.watch_enabled"
                  type="primary"
                  size="small"
                  @click="toggleWatch"
                >
                  <el-icon><VideoPlay /></el-icon>
                  一起看
                </el-button>
                <el-button
                  size="small"
                  @click="openAdminLogin"
                >
                  管理员登录
                </el-button>
              </template>

              <!-- 语音控制：房主显示开关+静音；观众只能退出 -->
              <template v-if="watchConnected">
                <template v-if="isHost">
                  <el-button
                    :type="voiceChannelEnabled ? 'warning' : 'success'"
                    size="small"
                    @click="toggleVoiceChannel"
                  >
                    {{ voiceChannelEnabled ? '🔇 关闭语音' : '🎤 开启语音' }}
                  </el-button>
                  <el-button
                    v-if="voiceChannelEnabled"
                    :type="voiceMuted ? 'warning' : ''"
                    size="small"
                    @click="toggleMute"
                  >
                    {{ voiceMuted ? '🔇 已静音' : '🔊 静音' }}
                  </el-button>
                </template>
              </template>

              <!-- 播放模式切换（原生视频且 HLS 可用时显示） -->
              <el-button-group v-if="isNativeVideo && qualityList.length > 0" size="small">
                <el-button :type="playMode === 'hls' ? 'primary' : ''" @click="switchMode('hls')">HLS</el-button>
                <el-button :type="playMode === 'native' ? 'primary' : ''" @click="switchMode('native')">原生</el-button>
              </el-button-group>

              <!-- 清晰度选择器（HLS 模式且有清晰度时显示） -->
              <template v-if="playMode === 'hls' && qualityList.length > 0">
                <el-select
                  v-model="currentQuality"
                  size="small"
                  style="width: 90px"
                  @change="onQualityChange"
                >
                  <el-option
                    v-for="q in qualityList"
                    :key="q.name"
                    :label="q.name + (q.ready ? '' : ' ⏳')"
                    :value="q.name"
                  />
                </el-select>
              </template>

              <el-button v-if="!info.disable_video_download && !info.disable_download" size="small" :href="downloadUrl" tag="a" target="_blank">
                <el-icon><Download /></el-icon>
                下载
              </el-button>
            </div>

            <!-- 加入一起看 - 管理员登录弹窗 -->
            <el-dialog v-model="joinDialogVisible" title="管理员登录" width="360px" :show-close="true" :close-on-click-modal="true">
              <el-form @submit.prevent="confirmJoin">
                <el-form-item label="昵称">
                  <el-input v-model="joinNickname" placeholder="输入你的昵称" maxlength="20" autofocus />
                </el-form-item>
                <el-form-item label="管理密码">
                  <el-input v-model="joinAdminPwd" type="password" placeholder="管理密码" show-password />
                </el-form-item>
              </el-form>
              <template #footer>
                <el-button @click="joinDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="confirmJoin">登录</el-button>
              </template>
            </el-dialog>
          </div>

          <!-- 非视频文件：预览/下载页 -->
          <div v-else class="preview-area">
            <!-- 文件头部信息 -->
            <div class="preview-header">
              <div class="preview-title-row">
                <el-icon class="preview-file-icon" :style="{ color: fileIconColor }">
                  <component :is="fileIconComponent" />
                </el-icon>
                <h2 class="preview-title">{{ info.name }}</h2>
              </div>
              <div class="preview-meta">
                <el-tag type="info" size="small">{{ formatSize(info.file_size) }}</el-tag>
                <el-tag type="info" size="small">{{ info.mime_type }}</el-tag>
                <el-tag :type="remainingTag" size="small">剩余 {{ info.remaining_views }} 次</el-tag>
              </div>
              <div class="preview-actions">
                <el-button v-if="!info.disable_download" type="primary" :href="downloadUrl" tag="a" download>
                  <el-icon><Download /></el-icon>
                  下载文件
                </el-button>
              </div>
            </div>

            <!-- 图片预览 -->
            <div v-if="info.is_image" class="preview-image-wrap">
              <img :src="downloadUrl" :alt="info.name" class="preview-image" @error="previewError = true" />
              <p v-if="previewError" class="preview-error">图片加载失败，请直接下载</p>
            </div>

            <!-- 音频播放器 -->
            <div v-else-if="info.is_audio" class="preview-audio-wrap">
              <audio :src="downloadUrl" controls class="preview-audio" @error="previewError = true"></audio>
              <p v-if="previewError" class="preview-error">音频加载失败，请直接下载</p>
            </div>

            <!-- PDF 预览 -->
            <div v-else-if="info.is_pdf" class="preview-pdf-wrap">
              <iframe :src="downloadUrl" class="preview-pdf" @error="previewError = true"></iframe>
              <p v-if="previewError" class="preview-error">PDF 加载失败，请直接下载</p>
            </div>

            <!-- 文本/代码预览 -->
            <div v-else-if="info.is_text" class="preview-text-wrap">
              <div v-if="textLoading" class="preview-text-loading">
                <el-icon class="is-loading"><Loading /></el-icon> 加载中...
              </div>
              <pre v-else-if="textContent !== null" class="preview-text-content"><code>{{ textContent }}</code></pre>
              <p v-if="previewError" class="preview-error">文本加载失败，请直接下载</p>
            </div>

            <!-- 其他文件：仅下载 -->
            <div v-else class="preview-generic">
              <el-icon style="font-size:80px;color:#c0c4cc"><Document /></el-icon>
              <p style="color:#909399;margin-top:12px">该文件类型暂不支持预览</p>
            </div>
          </div>
        </div>

        <!-- 右侧：聊天栏（一起看时显示） -->
        <div v-if="watchConnected" class="chat-panel">
          <div class="chat-header">
            <span>聊天室</span>
            <span class="viewer-count">{{ viewerCount }} 人在线</span>
          </div>

          <!-- 语音区：仅显示参与者列表 -->
          <div v-if="voiceChannelEnabled || voiceParticipants.length > 0" class="voice-area">
            <div v-if="voiceParticipants.length > 0" class="voice-participants">
              <div
                v-for="p in voiceParticipants"
                :key="p.peerID"
                class="voice-user"
                :class="{ speaking: p.speaking }"
              >
                <span class="voice-icon">🎤</span>
                <span class="voice-name">{{ p.nickname }}</span>
              </div>
            </div>
            <div v-else-if="voiceChannelEnabled" class="voice-empty">语音已开启，等待成员加入...</div>
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
import { ElMessage } from 'element-plus'
import Hls from 'hls.js'
import Artplayer from 'artplayer'

const route = useRoute()
const id = route.params.id

// ---- 状态 ----
const loading = ref(true)
const error = ref('')
const info = ref({})
const transcoding = ref(false)
const videoEl = ref(null)   // 指向 ArtPlayer 底层 video 元素
const artRef = ref(null)    // ArtPlayer 挂载 DOM
let art = null              // ArtPlayer 实例
const playerWrapper = ref(null)
const danmakuLayer = ref(null)

// ---- 密码 ----
const password = ref('')
const passwordDialogVisible = ref(false)
const inputPassword = ref('')
const passwordError = ref('')
const passwordChecking = ref(false)

// ---- 文件预览 ----
const previewError = ref(false)
const textContent = ref(null)
const textLoading = ref(false)

const fileIconComponent = computed(() => {
  if (info.value.is_image) return 'Picture'
  if (info.value.is_audio) return 'Headset'
  if (info.value.is_pdf) return 'Document'
  if (info.value.is_text) return 'Document'
  return 'Document'
})
const fileIconColor = computed(() => {
  if (info.value.is_image) return '#67c23a'
  if (info.value.is_audio) return '#e6a23c'
  if (info.value.is_pdf) return '#f56c6c'
  if (info.value.is_text) return '#409eff'
  return '#909399'
})

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
let syncLockCount = 0 // 防止收到 sync 时触发重复 sync（计数器，比 boolean 更可靠）
let watchReconnectTimer = null
let watchManualClose = false // 用户主动断开时不重连
let pendingSync = null // 播放器未就绪时缓存最新的 sync 消息
let pendingWs = null  // 匿名监听连接，用于接收 force_watch 信号

// ---- Voice Chat (WebRTC) ----
const voiceActive = ref(false)
const voiceMuted = ref(false)
const voiceChannelEnabled = ref(false) // 语音频道是否开启（由房主控制）
const localStreamActive = ref(false)   // 是否持有麦克风流
const voiceParticipants = ref([]) // [{ peerID, nickname, speaking }]
let myPeerID = ''
let localStream = null
const peerConns = new Map() // peerID → RTCPeerConnection
let rtcConfig = null // 动态从后端获取

async function fetchRtcConfig() {
  if (rtcConfig) return rtcConfig
  try {
    const res = await fetch('/api/nfsshare/turn-credentials')
    if (res.ok) {
      const d = await res.json()
      rtcConfig = {
        iceServers: [
          { urls: d.stun },
          { urls: d.turn, username: d.username, credential: d.credential },
        ],
      }
    }
  } catch (_) {}
  // fallback：无 TURN 时仅用公共 STUN
  if (!rtcConfig) {
    rtcConfig = { iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] }
  }
  return rtcConfig
}

// ---- 清晰度 ----
const qualityList = ref([])   // [{ name, height, ready }]
const currentQuality = ref('')
const playMode = ref('hls')   // 'hls' | 'native'，默认 HLS

// ---- HLS ----
let hls = null
let pollTimer = null

const downloadUrl = computed(() => {
  const base = `/api/nfsshare/${id}`
  return password.value ? `${base}?password=${encodeURIComponent(password.value)}` : base
})
const streamUrl = computed(() => {
  const base = `/api/nfsshare/${id}/stream`
  return password.value ? `${base}?password=${encodeURIComponent(password.value)}` : base
})
const hlsUrl = computed(() => {
  const q = currentQuality.value
  if (!q) return ''
  const base = `/api/nfsshare/${id}/hls/${q}/index.m3u8`
  return password.value ? `${base}?password=${encodeURIComponent(password.value)}` : base
})
// 浏览器原生支持的视频格式，直接流式播放，无需 HLS 转码
const isNativeVideo = computed(() => !!info.value.is_native_video)

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
    loading.value = false
    await nextTick()
    if (data.has_password) {
      // 需要密码才能继续，弹出密码框（密码框里也包含昵称和管理密码输入）
      passwordDialogVisible.value = true
    } else if (data.is_video) {
      if (data.watch_enabled) {
        // 视频+一起看模式：先申请麦克风，拒绝则禁止播放
        try {
          localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false })
          await fetchRtcConfig()
        } catch (e) {
          error.value = '需要麦克风权限才能观看此视频，请允许后刷新页面'
          loading.value = false
          return
        }
      }
      await initPlayer()
      if (data.watch_enabled) connectPendingWatch()
    } else if (data.is_text) {
      loadTextPreview()
    }
  } catch {
    error.value = '加载失败，请检查网络'
  } finally {
    loading.value = false
  }
}

// ---- 密码验证 ----
async function confirmPassword() {
  const pwd = inputPassword.value.trim()
  if (!pwd) {
    passwordError.value = '请输入密码'
    return
  }
  passwordChecking.value = true
  passwordError.value = ''
  try {
    // 密码验证：用 info 接口（不消耗次数），但 info 不校验密码，改用专用接口 HEAD 试探
    const testUrl = info.value.is_native_video
      ? `/api/nfsshare/${id}/stream?password=${encodeURIComponent(pwd)}`
      : info.value.is_video
        ? `/api/nfsshare/${id}/qualities?password=${encodeURIComponent(pwd)}`
        : `/api/nfsshare/${id}?password=${encodeURIComponent(pwd)}`
    const res = await fetch(testUrl, { method: 'HEAD' })
    if (res.status === 401 || res.status === 403) {
      passwordError.value = '密码错误，请重试'
      return
    }
    // 密码正确
    password.value = pwd
    passwordDialogVisible.value = false
    await nextTick()
    if (info.value.is_video) {
      if (info.value.watch_enabled) {
        try {
          localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false })
          await fetchRtcConfig()
        } catch (e) {
          error.value = '需要麦克风权限才能观看此视频，请允许后刷新页面'
          return
        }
      }
      await initPlayer()
      if (info.value.watch_enabled) connectPendingWatch()
    } else if (info.value.is_text) {
      loadTextPreview()
    }
  } catch {
    passwordError.value = '验证失败，请重试'
  } finally {
    passwordChecking.value = false
  }
}

// ---- 文本预览 ----
async function loadTextPreview() {
  if (info.value.file_size > 512 * 1024) {
    // 超过 512KB 不预览，只提供下载
    textContent.value = null
    return
  }
  textLoading.value = true
  try {
    const res = await fetch(downloadUrl.value)
    if (!res.ok) { previewError.value = true; return }
    textContent.value = await res.text()
  } catch {
    previewError.value = true
  } finally {
    textLoading.value = false
  }
}

// ---- 清晰度加载 ----
async function loadQualities() {
  try {
    const pwdParam = password.value ? `?password=${encodeURIComponent(password.value)}` : ''
    const res = await fetch(`/api/nfsshare/${id}/qualities${pwdParam}`)
    if (!res.ok) return
    const data = await res.json()
    qualityList.value = data.qualities || []
    if (qualityList.value.length > 0) {
      currentQuality.value = qualityList.value[0].name // 默认最高清晰度
    }
  } catch (_) {}
}

// ---- 视频播放器 ----
async function initPlayer() {
  await loadQualities()
  if (qualityList.value.length > 0) {
    playMode.value = 'hls'
    transcoding.value = true
    await nextTick()
    createArtPlayer(hlsUrl.value, 'hls')
  } else if (isNativeVideo.value) {
    playMode.value = 'native'
    await nextTick()
    createArtPlayer(streamUrl.value, 'native')
  } else {
    error.value = '无法获取视频清晰度信息，请稍后重试'
  }
}

function createArtPlayer(src, mode) {
  if (art) { art.destroy(); art = null }
  if (hls) { hls.destroy(); hls = null }

  const isHls = mode === 'hls'

  art = new Artplayer({
    container: artRef.value,
    url: src,
    type: isHls ? 'm3u8' : '',
    volume: 0.8,
    autoplay: false,
    pip: true,
    fullscreen: true,
    fullscreenWeb: true,
    playbackRate: true,
    aspectRatio: true,
    setting: true,
    hotkey: true,
    theme: '#409eff',
    lang: 'zh-cn',
    moreVideoAttr: {
      preload: 'metadata',
    },
    customType: isHls ? {
      m3u8: function (video, url) {
        if (Hls.isSupported()) {
          if (hls) { hls.destroy() }
          hls = new Hls({
            manifestLoadingTimeOut: 30 * 60 * 1000,
            manifestLoadingRetryDelay: 2000,
            manifestLoadingMaxRetry: 900,
          })
          hls.loadSource(url)
          hls.attachMedia(video)
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
        } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
          video.src = url
          video.addEventListener('loadedmetadata', () => { transcoding.value = false }, { once: true })
        } else {
          error.value = '浏览器不支持 HLS 播放'
          transcoding.value = false
        }
      }
    } : {},
  })

  // 拿到底层 video 元素，挂同步事件
  art.on('ready', () => {
    videoEl.value = art.video
    if (!isHls) transcoding.value = false
    art.video.addEventListener('error', onVideoError)
    // 直接监听原生 video 事件，ArtPlayer 的封装事件在 programmatic 操作时也会触发
    art.video.addEventListener('play', onHostPlay)
    art.video.addEventListener('pause', onHostPause)
    art.video.addEventListener('seeked', onHostSeek)
    // 播放器就绪后应用缓存的 sync 消息（新加入者场景）
    if (pendingSync && !isHost.value) {
      const msg = pendingSync
      pendingSync = null
      applySyncMsg(msg)
    }
  })
}

async function switchMode(mode) {
  if (mode === playMode.value) return
  const savedTime = art?.video?.currentTime || 0
  playMode.value = mode
  if (hls) { hls.destroy(); hls = null }
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  if (mode === 'hls') {
    transcoding.value = true
    createArtPlayer(hlsUrl.value, 'hls')
    // seek after manifest
  } else {
    transcoding.value = false
    createArtPlayer(streamUrl.value, 'native')
  }
  if (savedTime > 0) {
    art?.once('ready', () => { if (art?.video) art.video.currentTime = savedTime })
  }
}

async function onQualityChange(quality) {
  const savedTime = art?.video?.currentTime || 0
  currentQuality.value = quality
  transcoding.value = true
  if (hls) { hls.destroy(); hls = null }
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  createArtPlayer(hlsUrl.value, 'hls')
  if (savedTime > 0) {
    art?.once('ready', () => { if (art?.video) art.video.currentTime = savedTime })
  }
}

function startNative() {
  createArtPlayer(streamUrl.value, 'native')
}

function startHLS() {
  createArtPlayer(hlsUrl.value, 'hls')
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

async function onVideoError() {
  if (transcoding.value) return
  // 尝试获取服务端的具体错误信息
  try {
    const pwdParam = password.value ? `?password=${encodeURIComponent(password.value)}` : ''
    const res = await fetch(`/api/nfsshare/${id}/stream${pwdParam}`, { method: 'HEAD' })
    if (res.status === 404) {
      error.value = '源文件不存在，可能已被清理，请联系分享者重新上传'
      return
    }
  } catch (_) {}
  error.value = '视频播放出错'
}

// ---- 同步应用（viewer 端） ----
// syncLockCount: 每次 programmatic 操作前 +1，对应事件触发后 -1
// 这样无论事件多快触发都能正确屏蔽，不依赖 setTimeout
function applySyncMsg(msg) {
  const video = art.video
  const needSeek = msg.action === 'seek' || Math.abs(video.currentTime - msg.time) > 2
  if (needSeek) {
    syncLockCount++ // 屏蔽即将触发的 seeked 事件
    video.currentTime = msg.time
  }
  if (msg.action === 'play') {
    syncLockCount++ // 屏蔽即将触发的 play 事件
    video.play().catch(() => {})
  } else if (msg.action === 'pause') {
    syncLockCount++ // 屏蔽即将触发的 pause 事件
    video.pause()
  }
}

// ---- 房主同步事件 ----
function onHostPlay() {
  if (!isHost.value || !ws || ws.readyState !== WebSocket.OPEN) {
    // 观众在一起看中不能自己控制播放，强制暂停
    if (watchConnected.value && !isHost.value && syncLockCount === 0) {
      art?.video?.pause()
    }
    return
  }
  if (syncLockCount > 0) { syncLockCount--; return }
  wsSend({ type: 'sync', action: 'play', time: art?.video?.currentTime ?? 0 })
}
function onHostPause() {
  if (!isHost.value || !ws || ws.readyState !== WebSocket.OPEN) return
  if (syncLockCount > 0) { syncLockCount--; return }
  wsSend({ type: 'sync', action: 'pause', time: art?.video?.currentTime ?? 0 })
}
function onHostSeek() {
  if (!isHost.value || !ws || ws.readyState !== WebSocket.OPEN) return
  if (syncLockCount > 0) { syncLockCount--; return }
  wsSend({ type: 'sync', action: 'seek', time: art?.video?.currentTime ?? 0 })
}

// ---- 一起看 ----
function toggleWatch() {
  if (watchConnected.value) {
    disconnectWatch()
  } else {
    connectWatch('观众', '')
  }
}

function openAdminLogin() {
  joinAdminPwd.value = ''
  joinNickname.value = ''
  joinDialogVisible.value = true
}

// 建立匿名监听连接，仅用于接收 force_watch 信号，不算正式加入
function connectPendingWatch() {
  if (pendingWs) return
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  let url = `${proto}://${location.host}/api/nfsshare/${id}/watch/ws?nickname=__pending__`
  if (password.value) url += `&password=${encodeURIComponent(password.value)}`
  pendingWs = new WebSocket(url)
  pendingWs.onmessage = async (e) => {
    try {
      const msg = JSON.parse(e.data)
      if (msg.type === 'force_watch') {
        if (msg.host_active && !watchConnected.value) {
          // 管理员上线：暂停视频，申请麦克风，直接自动加入
          if (art?.video) art.video.pause()
          if (!localStream) {
            try {
              localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false })
              localStreamActive.value = true
              await fetchRtcConfig()
            } catch (e) {
              ElMessage.error('需要麦克风权限才能加入一起看')
              return
            }
          }
          if (pendingWs) { pendingWs.onclose = null; pendingWs.close(); pendingWs = null }
          connectWatch(joinNickname.value.trim() || '观众', '')
        } else if (!msg.host_active) {
          // 管理员下线：恢复视频可播
          joinDialogVisible.value = false
        }
      }
    } catch (_) {}
  }
  pendingWs.onclose = () => {
    // 如果不是因为正式加入而关闭，尝试重连
    if (!watchConnected.value && pendingWs) {
      pendingWs = null
      setTimeout(connectPendingWatch, 3000)
    }
  }
  pendingWs.onerror = () => {}
}

async function toggleVoiceChannel() {
  // 房主开关语音频道
  if (!voiceChannelEnabled.value && !voiceActive.value) {
    // 开启语音频道时，房主自动加入语音
    await startVoice()
  } else if (voiceChannelEnabled.value && voiceActive.value) {
    stopVoice()
  }
  wsSend({ type: 'voice_toggle' })
}

async function confirmJoin() {
  // 关闭 pending 监听连接，切换为正式连接
  if (pendingWs) { pendingWs.onclose = null; pendingWs.close(); pendingWs = null }
  joinDialogVisible.value = false
  const nick = joinNickname.value.trim() || '匿名用户'
  connectWatch(nick, joinAdminPwd.value)
}

function connectWatch(nickname, adminPwd) {
  myNickname.value = nickname
  isHost.value = !!adminPwd
  watchManualClose = false
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  let url = `${proto}://${location.host}/api/nfsshare/${id}/watch/ws?nickname=${encodeURIComponent(nickname)}`
  if (adminPwd) url += `&admin_password=${encodeURIComponent(adminPwd)}`
  if (password.value) url += `&password=${encodeURIComponent(password.value)}`
  ws = new WebSocket(url)
  ws.onopen = () => {
    watchConnected.value = true
    if (watchReconnectTimer) { clearTimeout(watchReconnectTimer); watchReconnectTimer = null }
    addSystemMsg('已加入观看室')
  }
  ws.onmessage = (e) => {
    try { handleWsMsg(JSON.parse(e.data)) } catch (_) {}
  }
  ws.onclose = () => {
    watchConnected.value = false
    if (!watchManualClose) {
      addSystemMsg('连接断开，3秒后重连...')
      watchReconnectTimer = setTimeout(() => connectWatch(nickname, adminPwd), 3000)
    } else {
      addSystemMsg('已断开连接')
    }
  }
  ws.onerror = () => {
    // onerror 后会触发 onclose，由 onclose 处理重连
  }
}

function disconnectWatch() {
  watchManualClose = true
  if (watchReconnectTimer) { clearTimeout(watchReconnectTimer); watchReconnectTimer = null }
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
    case 'sync': {
      if (isHost.value) break
      // 播放器未就绪时缓存，等 ready 后应用
      if (!art?.video) { pendingSync = msg; break }
      applySyncMsg(msg)
      break
    }

    // ---- WebRTC 语音信令 ----
    case 'voice_peers':
      // 服务端返回已在语音的成员列表，我主动向每人发 offer
      if (voiceActive.value && msg.peers) {
        for (const p of msg.peers) initiateCall(p.peer_id, p.nickname)
      }
      break
    case 'voice_join':
      // 其他人加入了语音，如果我也在语音里，主动向他发 offer
      if (voiceActive.value && msg.peer_id) {
        initiateCall(msg.peer_id, msg.nickname)
      }
      break
    case 'voice_leave':
      if (msg.peer_id) removeVoiceParticipant(msg.peer_id)
      break
    case 'voice_offer':
      if (voiceActive.value && msg.from) handleVoiceOffer(msg.from, msg.sdp, msg.nickname)
      break
    case 'voice_answer':
      if (msg.from) handleVoiceAnswer(msg.from, msg.sdp)
      break
    case 'voice_ice':
      if (msg.from) handleVoiceIce(msg.from, msg.candidate)
      break
    case 'voice_state':
      voiceChannelEnabled.value = !!msg.voice_enabled
      // 语音频道关闭时，强制断开本地语音
      if (!msg.voice_enabled && voiceActive.value) {
        leaveVoice()
        addSystemMsg('房主已关闭语音频道')
      } else if (msg.voice_enabled) {
        addSystemMsg('房主已开启语音频道')
      }
      break
    case 'force_watch':
      if (!msg.host_active) {
        addSystemMsg('管理员已离开，一起看模式结束')
      }
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

// ---- Voice: 开关语音 ----
async function toggleVoice() {
  if (voiceActive.value) {
    stopVoice()
  } else {
    await startVoice()
  }
}

async function startVoice() {
  if (!localStream) {
    try {
      localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false })
      localStreamActive.value = true
    } catch (e) {
      ElMessage.error('无法获取麦克风：' + (e.message || e))
      return
    }
  }
  await fetchRtcConfig()
  myPeerID = Math.random().toString(36).slice(2, 10)
  voiceActive.value = true
  voiceMuted.value = false
  wsSend({ type: 'voice_join' })
}

function stopVoice() {
  wsSend({ type: 'voice_leave' })
  leaveVoice()
}

function leaveVoice() {
  // 关闭所有 peer 连接
  for (const [, pc] of peerConns) pc.close()
  peerConns.clear()
  voiceParticipants.value = []
  // 停止麦克风
  if (localStream) {
    localStream.getTracks().forEach(t => t.stop())
    localStream = null
    localStreamActive.value = false
  }
  voiceActive.value = false
}

function toggleMute() {
  voiceMuted.value = !voiceMuted.value
  if (localStream) {
    localStream.getAudioTracks().forEach(t => { t.enabled = !voiceMuted.value })
  }
}

// ---- Voice: 创建 RTCPeerConnection ----
function createPC(peerID) {
  if (peerConns.has(peerID)) return peerConns.get(peerID)
  const pc = new RTCPeerConnection(rtcConfig || { iceServers: [] })

  // 把本地音轨加入
  if (localStream) {
    localStream.getTracks().forEach(t => pc.addTrack(t, localStream))
  }

  // 收到对端音频
  pc.ontrack = (e) => {
    const audio = new Audio()
    audio.srcObject = e.streams[0]
    audio.autoplay = true
    // 用 AudioContext 检测说话（音量）
    try {
      const ctx = new AudioContext()
      const src = ctx.createMediaStreamSource(e.streams[0])
      const analyser = ctx.createAnalyser()
      analyser.fftSize = 256
      src.connect(analyser)
      const buf = new Uint8Array(analyser.frequencyBinCount)
      const checkSpeaking = () => {
        if (!peerConns.has(peerID)) { ctx.close(); return }
        analyser.getByteFrequencyData(buf)
        const avg = buf.reduce((a, b) => a + b, 0) / buf.length
        const p = voiceParticipants.value.find(p => p.peerID === peerID)
        if (p) p.speaking = avg > 15
        requestAnimationFrame(checkSpeaking)
      }
      checkSpeaking()
    } catch (_) {}
  }

  // ICE 候选
  pc.onicecandidate = (e) => {
    if (e.candidate) {
      wsSend({ type: 'voice_ice', to: peerID, candidate: JSON.stringify(e.candidate) })
    }
  }

  pc.onconnectionstatechange = () => {
    if (pc.connectionState === 'failed' || pc.connectionState === 'closed') {
      pc.close()
      peerConns.delete(peerID)
      voiceParticipants.value = voiceParticipants.value.filter(p => p.peerID !== peerID)
    }
  }

  peerConns.set(peerID, pc)
  return pc
}

// ---- Voice: 主动发起 offer（我向对方呼叫）----
async function initiateCall(peerID, nickname) {
  addVoiceParticipant(peerID, nickname)
  const pc = createPC(peerID)
  const offer = await pc.createOffer()
  await pc.setLocalDescription(offer)
  wsSend({ type: 'voice_offer', to: peerID, sdp: offer.sdp })
}

// ---- Voice: 收到 offer，回 answer ----
async function handleVoiceOffer(from, sdp, nickname) {
  addVoiceParticipant(from, nickname || from)
  const pc = createPC(from)
  await pc.setRemoteDescription({ type: 'offer', sdp })
  const answer = await pc.createAnswer()
  await pc.setLocalDescription(answer)
  wsSend({ type: 'voice_answer', to: from, sdp: answer.sdp })
}

// ---- Voice: 收到 answer ----
async function handleVoiceAnswer(from, sdp) {
  const pc = peerConns.get(from)
  if (!pc) return
  await pc.setRemoteDescription({ type: 'answer', sdp })
}

// ---- Voice: 收到 ICE candidate ----
async function handleVoiceIce(from, candidateStr) {
  const pc = peerConns.get(from)
  if (!pc) return
  try {
    await pc.addIceCandidate(JSON.parse(candidateStr))
  } catch (_) {}
}

function addVoiceParticipant(peerID, nickname) {
  if (!voiceParticipants.value.find(p => p.peerID === peerID)) {
    voiceParticipants.value.push({ peerID, nickname, speaking: false })
  }
}

function removeVoiceParticipant(peerID) {
  const pc = peerConns.get(peerID)
  if (pc) { pc.close(); peerConns.delete(peerID) }
  voiceParticipants.value = voiceParticipants.value.filter(p => p.peerID !== peerID)
}

onMounted(() => { loadInfo() })
onUnmounted(() => {
  if (art) { art.destroy(); art = null }
  if (hls) { hls.destroy(); hls = null }
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  watchManualClose = true
  if (watchReconnectTimer) { clearTimeout(watchReconnectTimer); watchReconnectTimer = null }
  if (ws) { ws.close(); ws = null }
  if (voiceActive.value) stopVoice()
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

.art-player-container {
  width: 100%;
  height: 100%;
  display: block;
}

/* ArtPlayer 全局样式覆盖 */
:global(.art-video-player) {
  width: 100% !important;
  height: 100% !important;
}

/* 弹幕层 */
.danmaku-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 30;
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

/* ===== 文件预览区 ===== */
.preview-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0;
  min-height: 0;
}

.preview-header {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #2a2a2a;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.preview-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.preview-file-icon {
  font-size: 32px;
  flex-shrink: 0;
}

.preview-title {
  margin: 0;
  font-size: 18px;
  color: #e0e0e0;
  word-break: break-all;
}

.preview-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.preview-actions {
  display: flex;
  gap: 8px;
}

.preview-image-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #111;
  overflow: auto;
}

.preview-image {
  max-width: 100%;
  max-height: 80vh;
  object-fit: contain;
  border-radius: 4px;
  box-shadow: 0 4px 24px rgba(0,0,0,0.5);
}

.preview-audio-wrap {
  padding: 40px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.preview-audio {
  width: 100%;
  max-width: 600px;
}

.preview-pdf-wrap {
  flex: 1;
  min-height: 600px;
}

.preview-pdf {
  width: 100%;
  height: 100%;
  min-height: 600px;
  border: none;
  background: #fff;
}

.preview-text-wrap {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.preview-text-loading {
  padding: 40px;
  text-align: center;
  color: #909399;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.preview-text-content {
  margin: 0;
  padding: 20px 24px;
  font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4d4d4;
  background: #1e1e1e;
  white-space: pre-wrap;
  word-break: break-all;
  min-height: 200px;
}

.preview-generic {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 16px;
}

.preview-error {
  color: #f56c6c;
  font-size: 13px;
  margin-top: 8px;
}

/* ===== 手机端适配 ===== */
@media (max-width: 768px) {
  /* 布局：纵向堆叠 */
  .content-layout {
    flex-direction: column;
    height: auto;
    min-height: 100vh;
    overflow: visible;
  }

  .main-area {
    padding: 8px;
    overflow-y: visible;
  }

  /* 视频区域撑满宽度 */
  .video-area {
    gap: 8px;
    max-width: 100%;
  }

  .video-header {
    flex-direction: column;
    gap: 6px;
  }

  .video-title {
    font-size: 14px;
  }

  .video-meta {
    flex-wrap: wrap;
    gap: 4px;
  }

  /* 播放器 16:9 自适应 */
  .player-wrapper {
    border-radius: 4px;
  }

  /* 操作按钮适当缩小 */
  .video-actions {
    gap: 6px;
    flex-wrap: wrap;
  }

  /* 聊天面板放在视频下方，高度适中 */
  .chat-panel {
    width: 100%;
    height: 45vh;
    min-height: 220px;
    border-left: none;
    border-top: 1px solid #2a2a2a;
    /* 避免高度撑满整屏 */
    max-height: 50vh;
  }

  /* 下载页内边距减小 */
  .download-area {
    padding: 30px 12px;
    gap: 14px;
  }

  .download-area h2 {
    font-size: 16px;
  }

  .file-meta {
    font-size: 13px;
    gap: 8px;
  }

  /* 弹幕字体稍小 */
  :global(.danmaku-item) {
    font-size: 14px;
  }
}

/* 语音区 */
.voice-area {
  border-bottom: 1px solid #2a2a2a;
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.voice-bar {
  display: flex;
  gap: 6px;
  align-items: center;
}

.voice-participants {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.voice-user {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #1e2a1e;
  border: 1px solid #2a4a2a;
  border-radius: 12px;
  padding: 2px 8px;
  font-size: 12px;
  color: #aaa;
  transition: all 0.2s;
}

.voice-user.speaking {
  background: #1a3a1a;
  border-color: #67c23a;
  color: #67c23a;
}

.voice-icon {
  font-size: 11px;
}

.voice-empty {
  font-size: 11px;
  color: #555;
  font-style: italic;
}

/* 超小屏（≤480px） */
@media (max-width: 480px) {
  .main-area {
    padding: 4px;
  }

  .video-title {
    font-size: 13px;
  }

  .chat-panel {
    height: 40vh;
    min-height: 180px;
  }

  .chat-header {
    padding: 8px 10px;
    font-size: 13px;
  }

  .chat-messages {
    padding: 6px 8px;
  }

  .chat-input-area {
    padding: 6px 8px;
  }

  .danmaku-input-area {
    padding: 4px 8px 8px;
  }
}
</style>
