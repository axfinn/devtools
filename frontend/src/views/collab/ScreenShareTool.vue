<template>
  <div class="min-h-screen p-4 sm:p-6 max-w-5xl mx-auto">
    <header class="mb-6">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <el-icon><VideoCamera /></el-icon>
        屏幕共享 + 远程协助
      </h1>
      <p class="text-sm text-gray-500 mt-1">
        基于 WebRTC P2P,服务器只做信令转发。需先登录邮箱账号才能创建会话。
      </p>
    </header>

    <!-- 1. 未登录 -->
    <el-card v-if="!loggedIn" class="mb-4">
      <template #header>邮箱登录</template>
      <el-form :model="loginForm" label-width="80px" @submit.prevent>
        <el-form-item label="邮箱">
          <el-input v-model="loginForm.email" placeholder="you@example.com" />
        </el-form-item>
        <el-form-item label="邀请码" v-if="needInvite">
          <el-input v-model="loginForm.inviteCode" placeholder="邀请码(选填,未开放注册时必填)" />
        </el-form-item>
        <el-form-item label="验证码">
          <div class="flex gap-2 w-full">
            <el-input v-model="loginForm.code" placeholder="6 位数字" :maxlength="6" />
            <el-button :loading="sendingCode" :disabled="cooldown > 0" @click="sendCode">
              {{ cooldown > 0 ? `${cooldown}s 后重发` : '发送验证码' }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loggingIn" @click="doLogin">登录 / 注册</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 2. 已登录但未创建会话 -->
    <el-card v-else-if="!currentSession" class="mb-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span>新建会话</span>
          <div class="text-sm text-gray-500">
            {{ me.email }}
            <el-button text type="primary" @click="logout">退出</el-button>
          </div>
        </div>
      </template>
      <el-form :model="createForm" label-width="120px" @submit.prevent>
        <el-form-item label="会话标题">
          <el-input v-model="createForm.title" placeholder="例如:帮爸妈设置手机" />
        </el-form-item>
        <el-form-item label="访问密码">
          <el-input v-model="createForm.password" placeholder="留空=任何人凭链接可加入" show-password />
        </el-form-item>
        <el-form-item label="允许远程控制">
          <el-switch v-model="createForm.allowRemoteControl" />
          <span class="ml-2 text-xs text-gray-500">
            开启后,viewer 的鼠标/键盘可在你的页面生效(仅限本页面,不影响系统其他应用)
          </span>
        </el-form-item>
        <el-form-item label="持续时间">
          <el-select v-model="createForm.durationMinutes" class="w-full">
            <el-option label="不限" :value="0" />
            <el-option label="30 分钟" :value="30" />
            <el-option label="2 小时" :value="120" />
            <el-option label="8 小时" :value="480" />
            <el-option label="24 小时" :value="1440" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="creating" @click="createSession">创建并准备共享</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 3. 已创建,等待/进行中 -->
    <el-card v-else>
      <template #header>
        <div class="flex items-center justify-between">
          <span>会话进行中 · {{ currentSession.title || '(无标题)' }}</span>
          <el-tag v-if="currentSession.allow_remote_control" type="warning" size="small">允许远程控制</el-tag>
          <el-tag v-else type="info" size="small">仅观看</el-tag>
        </div>
      </template>

      <el-row :gutter="16">
        <el-col :xs="24" :md="14">
          <div class="bg-black rounded-lg overflow-hidden aspect-video relative flex items-center justify-center">
            <video v-if="localStream" ref="localVideoEl" autoplay muted playsinline class="w-full h-full object-contain" />
            <div v-else class="text-white text-center p-8">
              <el-icon size="48"><VideoPlay /></el-icon>
              <p class="mt-2 text-sm">点击下方"开始共享"按钮选择屏幕/窗口/标签页</p>
            </div>
            <!-- Viewer 远程输入反馈层:viewer 端鼠标位置映射到 host 视频预览的归一化坐标 -->
            <div
              v-show="remoteCursors.length > 0"
              class="absolute inset-0 pointer-events-none"
            >
              <div
                v-for="(c, i) in remoteCursors"
                :key="c.id"
                class="absolute"
                :style="{
                  left: (c.nx * 100) + '%',
                  top: (c.ny * 100) + '%',
                  transform: 'translate(-50%, -50%)',
                }"
              >
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M5 3 L19 12 L12 13 L9 21 Z" fill="#ef4444" stroke="white" stroke-width="1.5" stroke-linejoin="round" />
                </svg>
                <div v-if="c.clickFlash" class="absolute left-3 top-3 w-6 h-6 rounded-full bg-red-500/70 animate-ping" />
                <div v-if="c.nickname" class="absolute left-6 top-3 px-1.5 py-0.5 text-[10px] bg-red-500/80 text-white rounded whitespace-nowrap">
                  {{ c.nickname }}
                </div>
              </div>
            </div>
          </div>
          <div class="mt-3 flex gap-2 flex-wrap">
            <el-button v-if="!localStream" type="primary" @click="onStartShare" :loading="rtcState === 'connecting'">
              <el-icon><VideoCamera /></el-icon>
              开始共享
            </el-button>
            <el-button v-else type="danger" @click="onStopShare">
              <el-icon><VideoPause /></el-icon>
              停止共享
            </el-button>
            <el-button @click="copyViewerUrl" :disabled="!viewerUrl">
              <el-icon><CopyDocument /></el-icon>
              复制观众链接
            </el-button>
            <el-button type="danger" plain @click="onEndSession">
              <el-icon><CircleClose /></el-icon>
              结束会话
            </el-button>
          </div>
          <p v-if="rtcLastError" class="text-xs text-red-500 mt-2">{{ rtcLastError }}</p>
        </el-col>

        <el-col :xs="24" :md="10">
          <h3 class="font-medium text-sm mb-2">观众 ({{ viewers.length }})</h3>
          <div v-if="!viewers.length" class="text-sm text-gray-400 py-8 text-center">
            等待观众加入……
          </div>
          <ul v-else class="space-y-1">
            <li v-for="v in viewers" :key="v.peer_id" class="flex items-center px-2 py-1 hover:bg-gray-50 rounded">
              <el-icon><User /></el-icon>
              <span class="ml-2 flex-1 text-sm">{{ v.nickname || '匿名观众' }}</span>
              <el-tag v-if="v.is_anon" size="small" type="info">匿名</el-tag>
            </li>
          </ul>
          <el-divider />
          <h3 class="font-medium text-sm mb-2">观众链接</h3>
          <el-input :model-value="viewerUrl" readonly>
            <template #append>
              <el-button @click="copyViewerUrl">复制</el-button>
            </template>
          </el-input>
          <p class="text-xs text-gray-400 mt-1">
            链接带访问密码时,viewer 仍需在浏览器中输入密码。
          </p>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  VideoCamera, VideoPlay, VideoPause, CopyDocument, CircleClose, User,
} from '@element-plus/icons-vue'
import { useScreenRTC } from '../../composables/useScreenRTC'
import { useScreenRelayUpload } from '../../composables/useScreenRelay'

const API = '/api/screen'
const ASKIT = '/api/askit/v1'
const TOKEN_KEY = 'askit_screen_access'
const REFRESH_KEY = 'askit_screen_refresh'

const route = useRoute()
const router = useRouter()
const rtc = useScreenRTC()
const relayUpload = useScreenRelayUpload()

// 标签页诊断:host 端实时显示本地流 + relay 状态 + viewer 数。
// 用户只要看浏览器 tab 标题就知道是不是真在共享。
const hostDiagTitle = computed(() => {
  return `stream=${rtc.localStream.value?'Y':'N'} relay=${relayUpload.isActive.value?'Y':'N'} viewers=${viewers.value.length}`
})
let hostDiagTimer = null
onUnmounted(() => { if (hostDiagTimer) clearInterval(hostDiagTimer) })

const loggedIn = ref(false)
const me = ref({ email: '' })
const loginForm = reactive({ email: '', code: '', inviteCode: '' })
const needInvite = ref(false)
const sendingCode = ref(false)
const loggingIn = ref(false)
const cooldown = ref(0)
let cooldownTimer = null

// Viewer 远程输入反馈:每个 viewer 一个 cursor,host 在自己的视频预览上画出来。
// 纯 web 没法影响 host OS,所以这里把 viewer 的鼠标位置 / 点击做成视觉标注,
// 至少 viewer / host 双方能看到"对方输入是同步的"。
//
// 坐标体系:
//   viewer 发的是 viewer 视频元素的 client 坐标(viewer.x, viewer.y)
//   host 视频预览可能跟 host 屏幕不同分辨率 + 有 letterbox padding (object-contain)
//   host 端需要把 viewer 坐标按"viewer 看到的画面"归一化到 [0,1],host 画红圈时
//   用 (nx, ny) 乘 host 视频预览的实际视频区域尺寸即可。
//
// viewer 端的归一化发生在前端 (useScreenRelay.sendInput 调用处),host 收到的
// data 已经是 { nx, ny } 形式。如果老消息还发 { x, y },尝试兼容。
const remoteCursors = ref([])
let remoteCursorId = 0
let remoteCursorClearTimer = null

const createForm = reactive({
  title: '',
  password: '',
  allowRemoteControl: false,
  durationMinutes: 120,
})
const creating = ref(false)
const currentSession = ref(null)
const viewerUrl = ref('')
const viewers = ref([])

const localVideoEl = ref(null)
let signalingWS = null
let myPeerId = ''

// 模板里要用 localStream,但 rtc.localStream 是嵌套 ref,Vue 不会自动 unwrap。
// 暴露成顶层 computed 让模板直接写 `localStream` 就行。
const localStream = computed(() => rtc.localStream.value)
const rtcState = computed(() => rtc.state.value)
const rtcLastError = computed(() => rtc.lastError.value)

// ── askit auth ────────────────────────────────────────────
function authHeaders() {
  return { Authorization: `Bearer ${localStorage.getItem(TOKEN_KEY) || ''}` }
}

async function tryRefresh() {
  const refresh = localStorage.getItem(REFRESH_KEY)
  if (!refresh) return false
  try {
    const r = await fetch(`${ASKIT}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: refresh }),
    })
    if (!r.ok) return false
    const data = await r.json()
    if (data.accessToken) {
      localStorage.setItem(TOKEN_KEY, data.accessToken)
      if (data.refreshToken) localStorage.setItem(REFRESH_KEY, data.refreshToken)
      return true
    }
  } catch {}
  return false
}

async function fetchMe(allowRefresh = true) {
  try {
    let r = await fetch(`${ASKIT}/auth/me`, { headers: authHeaders() })
    if (r.status === 401 && allowRefresh && (await tryRefresh())) {
      r = await fetch(`${ASKIT}/auth/me`, { headers: authHeaders() })
    }
    if (!r.ok) { clearSession(); return false }
    me.value = await r.json()
    return true
  } catch { return false }
}

// authFetch:对所有需要 askit Bearer 的请求统一处理 401 → 自动 refresh → 重试一次。
// 这覆盖 access token 过期(默认 2h)的场景:用户开了 tab 半天没动,回来点按钮时
// 旧 token 已经失效;refresh token(30 天)还在,刷新后重发请求,UI 不会闪回登录页。
async function authFetch(path, init = {}, retry = true) {
  const headers = { ...(init.headers || {}), ...authHeaders() }
  let r = await fetch(path, { ...init, headers })
  if (r.status === 401 && retry && (await tryRefresh())) {
    const newHeaders = { ...(init.headers || {}), ...authHeaders() }
    r = await fetch(path, { ...init, headers: newHeaders })
  }
  return r
}

function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_KEY)
  loggedIn.value = false
  me.value = { email: '' }
}

function logout() {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    fetch(`${ASKIT}/auth/logout`, { method: 'POST', headers: authHeaders() }).catch(() => {})
  }
  clearSession()
}

async function sendCode() {
  if (!loginForm.email.trim()) { ElMessage.warning('请输入邮箱'); return }
  if (cooldown.value > 0) return
  sendingCode.value = true
  try {
    const r = await fetch(`${ASKIT}/auth/request-code`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: loginForm.email.trim(),
        inviteCode: loginForm.inviteCode.trim(),
      }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      const map = {
        invalid_email: '邮箱格式不正确',
        registration_closed: '当前未开放注册,请联系管理员',
        invalid_invite_code: '邀请码无效或已过期',
      }
      if (data.error === 'invalid_invite_code' || data.error === 'registration_closed') {
        needInvite.value = true
      }
      ElMessage.error(map[data.error] || `发送失败（${data.error || r.status}）`)
      return
    }
    startCooldown()
    ElMessage.success(data.emailSent === false ? '验证码已生成(邮件未配置)' : '验证码已发送')
  } catch {
    ElMessage.error('请求失败')
  } finally {
    sendingCode.value = false
  }
}

function startCooldown() {
  cooldown.value = 60
  clearInterval(cooldownTimer)
  cooldownTimer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0) clearInterval(cooldownTimer)
  }, 1000)
}

async function doLogin() {
  if (!loginForm.code.trim()) { ElMessage.warning('请输入验证码'); return }
  loggingIn.value = true
  try {
    const r = await fetch(`${ASKIT}/auth/login-code`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: loginForm.email.trim(),
        code: loginForm.code.trim(),
        inviteCode: loginForm.inviteCode.trim(),
      }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      const map = {
        invalid_code: '验证码错误或已过期',
        registration_closed: '当前未开放注册',
        invalid_invite_code: '邀请码无效',
      }
      ElMessage.error(map[data.error] || `登录失败（${data.error || r.status}）`)
      return
    }
    localStorage.setItem(TOKEN_KEY, data.accessToken)
    if (data.refreshToken) localStorage.setItem(REFRESH_KEY, data.refreshToken)
    if (await fetchMe(false)) {
      loggedIn.value = true
      ElMessage.success(`已登录 · ${me.value.email}`)
    }
  } catch {
    ElMessage.error('请求失败')
  } finally {
    loggingIn.value = false
  }
}

// ── session 管理 ──────────────────────────────────────────
async function createSession() {
  creating.value = true
  try {
    const r = await authFetch(`${API}/sessions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: createForm.title,
        password: createForm.password,
        allow_remote_control: createForm.allowRemoteControl,
        duration_minutes: createForm.durationMinutes,
      }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      if (r.status === 401) {
        ElMessage.error('登录已过期,请重新登录')
        clearSession()
        return
      }
      ElMessage.error(data.error || `创建失败（${r.status}）`)
      return
    }
    currentSession.value = data.session
    viewerUrl.value = data.viewer_url
    // 给 relay upload 喂 sessionId + 当前 askit access token(用于 /relay-host 鉴权)
    relayUpload.setSession(data.session.id, localStorage.getItem(TOKEN_KEY) || '')
    await connectHostWS(data.session.id)
  } catch (e) {
    ElMessage.error('请求失败: ' + (e?.message || e))
  } finally {
    creating.value = false
  }
}

async function connectHostWS(sessionId) {
  const token = localStorage.getItem(TOKEN_KEY) || ''
  // 立即预取 TURN 凭证,这样第一个 viewer 加入时 rtc 已经拿到 iceServers
  rtc.prefetchIceServers()
  const ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/screen/sessions/${sessionId}/ws?role=host&token=${encodeURIComponent(token)}`)
  signalingWS = ws
  ws.onopen = () => {
    console.log('[HOST] WS open')
    ElMessage.success('信令通道已建立')
  }
  ws.onmessage = (ev) => {
    let msg
    try { msg = JSON.parse(ev.data) } catch { return }
    console.log('[HOST] recv', msg.type, msg.from || '', msg.to || '')
    handleSignalingMessage(msg)
  }
  ws.onclose = (ev) => {
    console.log('[HOST] WS close', ev.code, ev.reason)
    if (currentSession.value) {
      ElMessage.warning('信令通道已断开')
    }
  }
  ws.onerror = (e) => {
    console.error('[HOST] WS error', e)
    ElMessage.error('信令通道连接出错')
  }
}

function wsSend(obj) {
  if (signalingWS && signalingWS.readyState === WebSocket.OPEN) {
    console.log('[HOST] send', obj.type, obj.to || '')
    signalingWS.send(JSON.stringify(obj))
  } else {
    console.warn('[HOST] send dropped (ws not open)', obj.type)
  }
}

const signaling = {
  send: wsSend,
}

function handleSignalingMessage(msg) {
  switch (msg.type) {
    case 'joined':
      myPeerId = msg.from
      viewers.value = msg.viewer_list || []
      break
    case 'viewer_joined':
      viewers.value.push(msg.viewer)
      viewers.value = [...viewers.value]
      onViewerJoined(msg.viewer)
      break
    case 'viewer_left':
      viewers.value = viewers.value.filter(v => v.peer_id !== msg.from)
      rtc.removePeer(msg.from)
      break
    case 'offer':
    case 'answer':
    case 'ice':
      // simple-peer 内部处理 SDP/ICE;useScreenRTC 会做 wire-format ↔ simple-peer 翻译
      rtc.handleSignal(msg.from, msg)
      break
    case 'error':
      ElMessage.error('信令错误: ' + (msg.error || '未知'))
      break
  }
}

function onViewerJoined(viewer) {
  // 之前这里会调 rtc.hostAttachViewer 走 P2P。但用户网络下 TURN 不可达,
  // P2P ICE 100% 失败,白白占用带宽和 CPU,且 viewer 端已经改成纯 relay。
  // 这里直接跳过 P2P 初始化,viewer 走 server relay 看画面。
  console.log('[HOST] viewer_joined', viewer.peer_id, viewer.nickname, '— relay-only mode, skip P2P attach')
}

// 把 viewer 控制消息转成 host 视频预览上的反馈标注。
// 纯 web 无法影响 host OS (Chrome 没有公开 API 模拟跨应用鼠标),所以这里
// 只是在 host 自己的视频预览上显示红圈 + nickname 提示,确认 viewer 输入
// 已经传到了 host 这边。
//
// 数据约定:
//   { nx, ny }:viewer 归一化坐标 (0..1,相对于 viewer 视频元素)
//   { x, y }:兼容老的原始 client 坐标(自动归一化)
//   { type }:click 等事件额外标注
function handleRemoteInputFeedback(kind, data) {
  if (!data) return
  let nx = Number(data.nx)
  let ny = Number(data.ny)
  // 兼容老消息(viewer 没归一化):用 window.innerWidth/Height 兜底
  if (!Number.isFinite(nx) && Number.isFinite(data.x)) nx = data.x / window.innerWidth
  if (!Number.isFinite(ny) && Number.isFinite(data.y)) ny = data.y / window.innerHeight
  if (!Number.isFinite(nx) || !Number.isFinite(ny)) return
  // 限制到 [0, 1],viewer 报错的极端值(如鼠标移到视频元素外)会被丢掉
  nx = Math.max(0, Math.min(1, nx))
  ny = Math.max(0, Math.min(1, ny))
  // 找这个 viewer 对应的 cursor(目前简单做法:用单一 cursor,多人看时再去重)
  // TODO 多人场景下 viewer 端发 join 消息带 peerId,这里按 peerId 映射。
  const id = ++remoteCursorId
  const cursor = {
    id,
    nx,
    ny,
    clickFlash: kind === 'click',
    nickname: data.nickname || 'viewer',
  }
  remoteCursors.value.push(cursor)
  // mousemove 高频,2s 后淡出;click 闪 0.4s 后移除
  const ttl = kind === 'click' ? 400 : 2000
  setTimeout(() => {
    remoteCursors.value = remoteCursors.value.filter(c => c.id !== id)
  }, ttl)
}

// ── 控制按钮 ──────────────────────────────────────────────
async function onStartShare() {
  console.log('[HOST] onStartShare click')
  try {
    await rtc.startCapture()
    nextTick(() => {
      if (localVideoEl.value && rtc.localStream.value) {
        localVideoEl.value.srcObject = rtc.localStream.value
      }
    })
    console.log('[HOST] capture ready, viewers count=' + viewers.value.length)
    // 启动 server relay fallback:P2P 失败时 viewer 走这里看画面 + 听音频。
    // 始终上传,不靠 P2P 检测触发;host 带宽代价(~500KB/s)换 100% 可达性。
    relayUpload.start(rtc.localStream.value)
    // 接收 viewer 转发过来的远程控制事件(JSON),经 server relay fanout 到 host。
    // P2P data channel 不通时这是唯一控制通道。
    relayUpload.onControl((kind, data) => {
      handleRemoteInputFeedback(kind, data)
    })
    // 已经加入的 viewer 现在补 offer
    for (const v of viewers.value) {
      onViewerJoined(v)
    }
  } catch (e) {
    console.error('[HOST] start share err', e)
    ElMessage.error('启动共享失败: ' + (e?.message || e))
  }
}

function onStopShare() {
  console.log('[HOST] onStopShare click')
  relayUpload.stop()
  rtc.stopCapture()
  // 通知 viewers:host 已停止
  for (const v of viewers.value) {
    rtc.removePeer(v.peer_id)
  }
}

function copyViewerUrl() {
  if (!viewerUrl.value) return
  navigator.clipboard.writeText(viewerUrl.value).then(
    () => ElMessage.success('已复制观众链接'),
    () => ElMessage.error('复制失败,请手动复制'),
  )
}

async function onEndSession() {
  if (!currentSession.value) return
  try {
    await authFetch(`${API}/sessions/${currentSession.value.id}`, {
      method: 'DELETE',
    })
  } catch {}
  cleanup()
}

function cleanup() {
  if (signalingWS) {
    try { signalingWS.close() } catch {}
    signalingWS = null
  }
  rtc.closeAll()
  relayUpload.stop()
  currentSession.value = null
  viewerUrl.value = ''
  viewers.value = []
}

watch(localVideoEl, (el) => {
  if (el && rtc.localStream.value) el.srcObject = rtc.localStream.value
})

// 启动 host 标签页诊断(每秒刷新)
hostDiagTimer = setInterval(() => {
  document.title = `[${hostDiagTitle.value}] 屏幕共享 · host`
}, 1000)

// ── 启动 ──────────────────────────────────────────────────
;(async () => {
  const ok = await fetchMe()
  if (ok) loggedIn.value = true
})()

onUnmounted(cleanup)
</script>

<style scoped>
.aspect-video {
  aspect-ratio: 16 / 9;
}
</style>