<template>
  <div class="min-h-screen bg-gray-900 text-white">
    <header class="bg-gray-800 px-4 py-3 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <el-icon size="22"><VideoCamera /></el-icon>
        <h1 class="text-lg font-medium">屏幕共享观看</h1>
      </div>
      <div class="text-sm text-gray-300 flex items-center gap-2">
        <span v-if="session?.title">{{ session.title }}</span>
        <el-tag v-if="session?.allow_remote_control" type="warning" size="small">允许远程控制</el-tag>
        <el-tag :type="connStateTagType" size="small">{{ connStateText }}</el-tag>
        <!-- 主动离开按钮:断开 WS + 跳回 host 工具页。原代码无任何路径让 viewer 主动退出。 -->
        <el-button text type="primary" size="small" @click="leave">
          <el-icon><Back /></el-icon>
          <span class="ml-1">离开观看</span>
        </el-button>
      </div>
    </header>

    <!-- 加载/错误态 -->
    <div v-if="loadError" class="p-8 text-center">
      <el-icon size="48" color="#ef4444"><CircleClose /></el-icon>
      <p class="mt-4 text-lg">{{ loadError }}</p>
      <el-button class="mt-4" @click="loadSession">重试</el-button>
    </div>

    <!-- 自动离开倒计时:host 关闭共享 / 重连失败 / session 过期时显示,
         给用户缓冲看到提示后跳回 /screen 而不是卡在"已连接"。 -->
    <div v-else-if="leaveTimer > 0" class="max-w-md mx-auto mt-16 p-6 bg-gray-800 rounded-lg text-center">
      <el-icon size="48" color="#f59e0b"><Warning /></el-icon>
      <p class="mt-4 text-lg">{{ leaveReason }}</p>
      <p class="mt-2 text-sm text-gray-400">{{ leaveTimer }} 秒后自动返回主机工具页</p>
      <div class="mt-4 flex gap-2 justify-center">
        <el-button type="primary" @click="leave">立即离开</el-button>
        <el-button @click="cancelAutoLeave">取消 (留在页面)</el-button>
      </div>
    </div>

    <!-- 密码输入 -->
    <div v-else-if="needPassword" class="max-w-md mx-auto mt-16 p-6 bg-gray-800 rounded-lg">
      <h2 class="text-lg font-medium mb-4 flex items-center gap-2">
        <el-icon><Lock /></el-icon>
        请输入访问密码
      </h2>
      <el-input v-model="passwordInput" type="password" show-password placeholder="访问密码" @keyup.enter="onCheckPassword" />
      <el-button class="mt-3 w-full" type="primary" :loading="checkingPassword" @click="onCheckPassword">进入</el-button>
      <p v-if="passwordError" class="text-red-400 text-sm mt-2">{{ passwordError }}</p>
    </div>

    <!-- 昵称输入(可选) -->
    <div v-else-if="!connected && !waitingForHost" class="max-w-md mx-auto mt-16 p-6 bg-gray-800 rounded-lg">
      <h2 class="text-lg font-medium mb-2">即将加入</h2>
      <p class="text-sm text-gray-400 mb-4">填写昵称(可留空,留空则显示为匿名观众)</p>
      <el-input v-model="nickname" placeholder="昵称(可选)" :maxlength="32" @keyup.enter="connect" />
      <el-button class="mt-3 w-full" type="primary" :loading="connecting" @click="connect">加入观看</el-button>
    </div>

    <!-- 等待 host -->
    <div v-else-if="waitingForHost" class="p-8 text-center">
      <el-icon size="48" color="#60a5fa"><Loading /></el-icon>
      <p class="mt-4">等待主机开始共享……</p>
    </div>

    <!-- 信令重连中 -->
    <div v-else-if="reconnecting" class="p-8 text-center">
      <el-icon size="48" color="#f59e0b"><Loading /></el-icon>
      <p class="mt-4">连接中断,正在重连…… ({{ reconnectCountdown }}s)</p>
      <p class="text-xs text-gray-500 mt-2">{{ reconnectError || '' }}</p>
    </div>

    <!-- 视频画面:永远挂着,canvas 即使没帧也是黑框(用户看到的就是"等帧"而不是 v-show 隐藏整个区域)。
         原代码用 v-show="connected",意味着信令 WS 没握手成功时整个 <main> display:none,
         即使 relay 已经在画帧,viewer 啥也看不到 → "永远正在协商连接"。
         现在改成 <main> 永远在,但在它上面叠一个半透明"协商中"指示层仅在 connected=false 且
         还没收到帧时显示。一旦收到帧立刻移除。-->
    <main class="relative flex items-center justify-center p-4" style="min-height: calc(100vh - 56px);">
      <!-- 视频或 canvas 的共同容器,远程控制事件统一绑在这上面。
           注意:P2P video 和 Relay canvas 二选一显示,但 listener 绑在容器上,
           e.currentTarget 拿到的是 listener 直接挂载的元素 (<main>),不是
           video/canvas。所以 attach 路径改用 clientX/Y + currentTarget rect 都 OK,
           容器的 rect 把视频视作整体,和 viewer 视觉效果一致。-->
      <div
        ref="remoteContainerEl"
        class="relative"
        :class="{ 'cursor-none': remoteControlEnabled }"
      >
        <!-- P2P 路径(WebRTC video) -->
        <video
          v-show="hasRemoteTrack"
          ref="remoteVideoEl"
          autoplay
          muted
          playsinline
          controls
          class="max-w-full max-h-full rounded shadow-lg bg-black"
        />
        <!-- Relay 路径(server 转发 JPEG → canvas)。
             改成永远在 DOM 里,通过 opacity 切换可见性:即使 connected=false 也能显示已收到的帧 -->
        <canvas
          ref="relayCanvasEl"
          class="max-w-full max-h-full rounded shadow-lg bg-black transition-opacity duration-200"
          :class="hasRemoteTrack || !relayFrameReceived ? 'opacity-0' : 'opacity-100'"
        />
        <!-- viewer 自己看到的鼠标反馈:确认输入已被采集并发出 -->
        <div
          v-if="remoteControlEnabled && selfCursorVis"
          class="absolute pointer-events-none"
          :style="{
            left: (selfCursorVis.nx * 100) + '%',
            top: (selfCursorVis.ny * 100) + '%',
            transform: 'translate(-50%, -50%)',
          }"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M5 3 L19 12 L12 13 L9 21 Z" fill="#22d3ee" stroke="white" stroke-width="1.5" stroke-linejoin="round" />
          </svg>
        </div>
        <!-- 协商中提示层:canvas 之上、信令未握手成功时显示。
             信令一旦 connected 或 relayFrameReceived=true,指示自动消失。-->
        <div
          v-if="!hasRemoteTrack && !relayFrameReceived"
          class="absolute inset-0 flex flex-col items-center justify-center bg-black/70 text-center px-4"
        >
          <el-icon size="48" :color="reconnecting ? '#f59e0b' : '#60a5fa'">
            <Loading v-if="!reconnecting" />
            <Connection v-else />
          </el-icon>
          <p class="mt-3 text-lg">
            {{ reconnecting ? '连接中断,正在重连…' : '正在协商连接…' }}
          </p>
          <p v-if="reconnecting" class="mt-1 text-sm text-gray-400">({{ reconnectCountdown }}s) {{ reconnectError }}</p>
          <p class="mt-2 text-xs text-gray-500 max-w-sm">
            WS: <span class="font-mono">{{ connected ? 'connected' : (connecting ? 'connecting' : (reconnecting ? 'reconnecting' : 'disconnected')) }}</span>
            · relay: <span class="font-mono">{{ relayDownload.isActive ? 'on' : 'off' }}</span>
            · canvas: <span class="font-mono">{{ canvasInfo }}</span>
          </p>
        </div>
      </div>
    </main>

    <!-- 诊断信息 — 直接显示在页面上,不用开 DevTools -->
    <div class="fixed bottom-2 left-2 z-50 bg-red-900/80 text-white text-xs p-2 rounded font-mono max-w-[90vw]">
      <div>WS connected: {{ connected }}</div>
      <div>relay isActive: {{ relayDownload.isActive }}</div>
      <div>hasRemoteTrack: {{ hasRemoteTrack }}</div>
      <div>relayFrameReceived: {{ relayFrameReceived }}</div>
      <div>lastFrameAt: {{ relayDownload.lastFrameAt || '(none)' }}</div>
      <div>秒前: {{ relayDownload.lastFrameAt ? Math.round((Date.now() - relayDownload.lastFrameAt) / 1000) : '-' }}</div>
      <div>canvas: {{ canvasInfo }}</div>
    </div>

    <!-- 远程控制开关(允许时) -->
    <div v-if="connected && session?.allow_remote_control" class="fixed bottom-4 right-4 z-10">
      <el-card class="bg-gray-800 text-white border-gray-700" body-class="p-3">
        <div class="flex items-center gap-3">
          <span class="text-sm">远程协助</span>
          <el-switch v-model="remoteControlEnabled" />
          <span class="text-xs text-gray-400">
            {{ remoteControlEnabled ? '你的鼠标/键盘会同步给主机(仅影响本页面)' : '仅观看,不发送输入' }}
          </span>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  VideoCamera, CircleClose, Lock, Loading, Back, Warning,
} from '@element-plus/icons-vue'
import { useScreenRTC, useRemoteInputSender } from '../../composables/useScreenRTC'
import { useScreenRelayDownload } from '../../composables/useScreenRelay'

const API = '/api/screen'
const route = useRoute()
const router = useRouter()
const rtc = useScreenRTC()
const remoteInput = useRemoteInputSender(ref(null))
const relayDownload = useScreenRelayDownload()

const sessionId = computed(() => route.params.id)
const session = ref(null)
const loadError = ref('')

const needPassword = ref(false)
const passwordInput = ref('')
const checkingPassword = ref(false)
const passwordError = ref('')

const nickname = ref('')
const connecting = ref(false)
const connected = ref(false)
const waitingForHost = ref(false)
const hasRemoteTrack = ref(false)
const remoteControlEnabled = ref(false)

const reconnecting = ref(false)
const reconnectCountdown = ref(0)
const reconnectError = ref('')
let reconnectTimer = null
let reconnectAttempt = 0

const remoteVideoEl = ref(null)
const relayCanvasEl = ref(null)
const remoteContainerEl = ref(null)
// viewer 自己鼠标的实时反馈:确认 host 正在收控制事件。仅显示最近一次。
// mousemove 不节流会狂更新 ref,简单用最新值。
const selfCursorVis = ref(null)
let selfCursorTimer = null
const relayFrameReceived = ref(false)
const canvasInfo = ref('not ready')
let signalingWS = null
let hostPeerId = 'host'
let detachInput = null
let diagTimer = null

// 标签页标题实时显示诊断信息,用户看浏览器 tab 就能知道状态,不需要 DevTools。
// 格式:[conn=Y relay=Y frames=N sec=-] 屏幕共享观看
//   conn    = 信令 WS 是否 connected
//   relay   = relay WS 是否 active
//   frames  = 是否收到过至少一帧
//   sec     = 距离上一帧多少秒,-1 表示从未收到
const diagTitle = computed(() => {
  const sec = relayDownload.lastFrameAt.value
    ? Math.round((Date.now() - relayDownload.lastFrameAt.value) / 1000)
    : -1
  return `conn=${connected.value?'Y':'N'} relay=${relayDownload.isActive.value?'Y':'N'} frames=${relayFrameReceived.value?'Y':'N'} sec=${sec}`
})

watch([connected, () => relayDownload.isActive.value, relayFrameReceived, () => relayDownload.lastFrameAt.value], () => {
  document.title = `[${diagTitle.value}] 屏幕共享观看`
}, { immediate: true })

const connStateText = computed(() => {
  if (connected.value) return hasRemoteTrack.value ? '已连接' : '协商中'
  if (waitingForHost.value) return '等待主机'
  if (connecting.value) return '连接中'
  return '未连接'
})
const connStateTagType = computed(() => {
  if (connected.value && hasRemoteTrack.value) return 'success'
  if (connected.value) return 'warning'
  if (waitingForHost.value || connecting.value) return 'info'
  return 'info'
})

onMounted(() => {
  loadSession()
  diagTimer = setInterval(() => {
    const c = relayCanvasEl.value
    canvasInfo.value = c ? `${c.width}x${c.height}` : 'no ref'
    // 触发 diagTitle 重新计算并写回 document.title,让标签页每秒刷新诊断信息
    document.title = `[${diagTitle.value}] 屏幕共享观看`
  }, 1000)
})
onUnmounted(() => {
  cleanup()
  if (diagTimer) clearInterval(diagTimer)
})

async function loadSession() {
  loadError.value = ''
  try {
    const r = await fetch(`${API}/sessions/${sessionId.value}/info`)
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      loadError.value = data.error === 'session_unavailable' ? '会话已结束或过期' : '会话不存在'
      return
    }
    session.value = data.session
    if (data.session.has_password) {
      needPassword.value = true
    } else {
      needPassword.value = false
      // 不需要密码,直接进入昵称步骤
    }
  } catch (e) {
    loadError.value = '加载失败: ' + (e?.message || e)
  }
}

async function onCheckPassword() {
  if (!passwordInput.value.trim()) {
    passwordError.value = '请输入密码'
    return
  }
  checkingPassword.value = true
  passwordError.value = ''
  try {
    const r = await fetch(`${API}/sessions/${sessionId.value}/check-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: passwordInput.value }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      passwordError.value = '请求失败'
      return
    }
    if (data.ok) {
      needPassword.value = false
      passwordError.value = ''
    } else {
      passwordError.value = '密码错误'
    }
  } catch (e) {
    passwordError.value = '请求失败: ' + (e?.message || e)
  } finally {
    checkingPassword.value = false
  }
}

function connect() {
  if (!session.value) return
  connecting.value = true
  waitingForHost.value = false
  reconnecting.value = false

  // 启动 server relay 订阅(独立于 P2P signaling WS)。P2P 不通时也能看到画面。
  // 必须在 ws.onopen 里、connected=true 之后再 start,这样 <canvas> ref 已经
  // 渲染(虽然 v-show 是 false,vue 仍然把 ref 挂上,但 nextTick 保险)。
  relayDownload.setContext(sessionId.value, passwordInput.value || '', nickname.value)

  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const params = new URLSearchParams({ role: 'viewer', nickname: nickname.value })
  if (passwordInput.value) params.set('password', passwordInput.value)
  const ws = new WebSocket(`${proto}://${location.host}/api/screen/sessions/${sessionId.value}/ws?${params}`)
  signalingWS = ws
  ws.onopen = async () => {
    console.log('[VIEWER] WS open')
    connecting.value = false
    connected.value = true
    reconnecting.value = false
    reconnectAttempt = 0
    reconnectError.value = ''
    // connected=true 触发 <main> v-show=true,canvas 元素现在保证在 DOM 里。
    // start 在 nextTick 里确保 v-show 切换后的 ref 同步更新。
    nextTick(() => relayDownload.start(relayCanvasEl.value))
    // 之前这里会启 P2P:rtc.prefetchIceServers() + rtc.startViewerSession。
    // 但用户网络下 TURN 不可达,P2P 永远 ICE 失败 → host 收到 viewer_joined
    // 后 hostAttachViewer 也失败 → viewer 端永远拿不到 P2P 流,只能走 relay。
    // 干脆跳过 P2P,纯走 relay,排错面缩到最小。
  }
  ws.onmessage = (ev) => {
    let msg
    try { msg = JSON.parse(ev.data) } catch { return }
    console.log('[VIEWER] recv', msg.type, msg.from || '', msg.to || '')
    handleSignaling(msg)
  }
  ws.onclose = (ev) => {
    console.log('[VIEWER] WS close', ev.code, ev.reason)
    // 之前确实连上过(host 已知道我们),而不是第一次就连不上 → 自动重连
    if (connected.value) {
      ElMessage.warning('与信令服务器断开连接,正在重连……')
      scheduleReconnect()
      return
    }
    // 第一次就连不上,直接回到加入页(密码错 / session 不存在)
    connected.value = false
    waitingForHost.value = false
  }
  ws.onerror = (e) => {
    console.error('[VIEWER] WS error', e)
  }
}

function scheduleReconnect() {
  if (reconnectTimer) return
  reconnectAttempt += 1
  // 退避:1s, 2s, 4s, 上限 8s
  const delay = Math.min(8000, 1000 * Math.pow(2, reconnectAttempt - 1))
  // 上限 5 次:连续失败超过 5 次不再尝试,触发 autoLeave 让 viewer 退出
  const maxAttempts = 5
  if (reconnectAttempt > maxAttempts) {
    ElMessage.error('多次重连失败,即将离开观看')
    startAutoLeave('重连失败', 5)
    return
  }
  reconnectError.value = `第 ${reconnectAttempt}/${maxAttempts} 次重连,等待 ${Math.round(delay / 1000)}s`
  let remaining = Math.round(delay / 1000)
  reconnectCountdown.value = remaining
  reconnecting.value = true
  reconnectTimer = setInterval(() => {
    remaining -= 1
    reconnectCountdown.value = Math.max(0, remaining)
    if (remaining <= 0) {
      clearInterval(reconnectTimer)
      reconnectTimer = null
      // 重要:重连前先清掉旧 peer,避免 stale 状态
      rtc.closeAll()
      connect()
    }
  }, 1000)
}

function wsSend(obj) {
  if (signalingWS && signalingWS.readyState === WebSocket.OPEN) {
    console.log('[VIEWER] send', obj.type, obj.to || '')
    signalingWS.send(JSON.stringify(obj))
  } else {
    console.warn('[VIEWER] send dropped (ws not open)', obj.type)
  }
}

function attachInput() {
  if (detachInput) detachInput()
  // 绑在容器上而不是 video / canvas:P2P 走 video,relay 走 canvas,而 viewer 鼠标
  // 期待"在我看到的画面上" → 用容器 rect (video/canvas 共享父 div 的 box) 做归一化
  const el = remoteContainerEl.value || remoteVideoEl.value
  if (!el) return
  detachInput = remoteInput.attach(el, (kind, data) => {
    // 顺便更新本地 cursor 反馈 (mouse / click) 让 viewer 自己能看到红圈
    if ((kind === 'mousemove' || kind === 'click') && data && Number.isFinite(data.nx)) {
      selfCursorVis.value = { nx: data.nx, ny: data.ny }
      if (selfCursorTimer) clearTimeout(selfCursorTimer)
      // 600ms 后隐藏 cursor,下次 mousemove 重置
      selfCursorTimer = setTimeout(() => { selfCursorVis.value = null }, 600)
    }
    return relayDownload.sendInput(kind, data)
  })
}

function handleSignaling(msg) {
  switch (msg.type) {
    case 'joined':
      // viewer 端 joined 不会有 viewer_list,忽略
      break
    case 'answer':
    case 'offer':
    case 'ice':
      rtc.handleSignal(hostPeerId, msg)
      break
    case 'host_disconnected':
      ElMessage.warning('主机已断开或结束共享')
      connected.value = false
      hasRemoteTrack.value = false
      // 原代码只把 connected=false,viewer 还卡在"已连接"标签上没动过。新行为:
      // 倒计时 10s 后自动跳回 host 工具页 (同时清干净 WS / RTC),给用户缓冲看到提示。
      // 用户点"离开观看"立即跳,不用等倒计时。
      startAutoLeave('host 关闭共享', 10)
      break
    case 'error':
      ElMessage.error('信令错误: ' + (msg.error || '未知'))
      break
  }
}

// ── 离开观看 / 自动跳转 ──
// 原版没有任何方式让 viewer 离开会话:只能关 tab 或后退。这里加:
//   1) leave() — 用户点"离开观看"按钮,立刻断开 WS + 跳到 /screen
//   2) startAutoLeave() — host 断开后倒计时自动离开。preventLeave 标志让用户在
//      自动跳转前看到提示文案的"立即前往"按钮有取消机会(本实现暂不展开,留接口)。
const leaveTimer = ref(0)
let leaveInterval = null
const leaveReason = ref('')
const preventLeave = ref(false)

function leave() {
  cleanup()
  router.replace({ path: '/screen' })
}

function startAutoLeave(reason, seconds) {
  if (leaveInterval) return
  leaveReason.value = reason
  leaveTimer.value = seconds
  preventLeave.value = false
  leaveInterval = setInterval(() => {
    if (preventLeave.value) {
      clearInterval(leaveInterval)
      leaveInterval = null
      return
    }
    leaveTimer.value -= 1
    if (leaveTimer.value <= 0) {
      clearInterval(leaveInterval)
      leaveInterval = null
      leave()
    }
  }, 1000)
}

function cancelAutoLeave() {
  preventLeave.value = true
}

// 监听 remoteStreams:simple-peer on('stream') 会往里写,
const currentRemoteStream = computed(() => rtc.remoteStreams.value.get(hostPeerId))
watch(currentRemoteStream, (stream) => {
  if (!stream) {
    hasRemoteTrack.value = false
    return
  }
  hasRemoteTrack.value = true
  nextTick(() => {
    if (remoteVideoEl.value) {
      remoteVideoEl.value.srcObject = stream
      remoteVideoEl.value.play().catch(e => {
        console.error('[VIEWER] video.play() rejected', e?.name, e?.message)
      })
    }
  })
}, { immediate: true })

// Relay 路径:只要收到第一帧就显示 canvas,之后 P2P 有 track 时自动隐藏(canvas 在模板里 v-show 控制)
watch(() => relayDownload.lastFrameAt.value, (t) => {
  if (t > 0) {
    relayFrameReceived.value = true
    if (relayFrameReceived.value && !hasRemoteTrack.value) {
      console.log('[VIEWER] relay first frame painted')
    }
  }
})

// 监听 simple-peer 自己的 data channel 打开
watch(() => {
  const e = rtc.peers.value.get(hostPeerId)
  return e?.peer?._channel?.readyState
}, (state) => {
  console.log('[VIEWER] data channel state =', state)
  if (state === 'open') {
    if (session.value?.allow_remote_control && remoteControlEnabled.value) attachInput()
  } else if (state === 'closed') {
    if (detachInput) { detachInput(); detachInput = null }
  }
})

watch(remoteControlEnabled, (v) => {
  if (v && session.value?.allow_remote_control) attachInput()
  else if (detachInput) { detachInput(); detachInput = null }
})

function cleanup() {
  if (reconnectTimer) { clearInterval(reconnectTimer); reconnectTimer = null }
  if (detachInput) { detachInput(); detachInput = null }
  if (selfCursorTimer) { clearTimeout(selfCursorTimer); selfCursorTimer = null }
  if (leaveInterval) { clearInterval(leaveInterval); leaveInterval = null }
  if (signalingWS) {
    try { signalingWS.close() } catch {}
    signalingWS = null
  }
  rtc.closeAll()
  relayDownload.stop()
  relayFrameReceived.value = false
  // 清干净所有进入页用的状态,避免下次 onMounted(loadSession) 看到残留值
  connected.value = false
  waitingForHost.value = false
  connecting.value = false
  reconnecting.value = false
  reconnectCountdown.value = 0
  reconnectError.value = ''
  reconnectAttempt = 0
  hasRemoteTrack.value = false
  remoteControlEnabled.value = false
  selfCursorVis.value = null
  leaveTimer.value = 0
  leaveReason.value = ''
  preventLeave.value = false
  passwordInput.value = ''
  nickname.value = ''
}
</script>