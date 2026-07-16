// useScreenRelay — 屏幕共享的 server-relay fallback。
//
// 与 useScreenRTC 平行:当 WebRTC P2P 在用户网络下不可达(TURN 不通 / 双端都在 NAT 后),
// host 把屏幕帧和音频块推到后端,后端 fanout 给所有 viewer。
//
// 与 useScreenRTC 互不耦合:同时打开 P2P + relay,viewer 端哪个先到画面显示哪个。
// 这里故意不走 WebRTC,跨网络时 100% 可达,代价是 server 带宽 × viewer 数。
//
// 帧格式:每条 binary WS 消息第一个字节是 type,后面是 payload,跟后端 screen.go 约定一致。
//   0x01 → JPEG 视频帧(host canvas.toBlob,viewer 用 createImageBitmap)
//   0x02 → Opus 音频块(host MediaRecorder,viewer 用 AudioContext decodeAudioData)

import { ref, shallowRef } from 'vue'

const FRAME_TYPE_VIDEO = 0x01
const FRAME_TYPE_AUDIO = 0x02
// 之前 10 FPS 0.7 画质 + 1280×720 在 server relay 转发链路下用户反馈"延迟很高"。
// server-relay 路径不能假设 P2P 直连的低延迟,得主动压缩体积。
// 提速三件套:提 FPS 到 15 → 减少画面跳变感知;画质降到 0.5 → 体积砍半;
// 最大尺寸收紧到 960×540 → 大多数屏幕内容 540p 仍清晰可读,JPEG 帧降至 30-50KB,
// 普通 LAN/隧道带宽下也能在 100-200ms 内推完。
const VIDEO_FPS = 15
const VIDEO_QUALITY = 0.5
const VIDEO_MAX_WIDTH = 960
const VIDEO_MAX_HEIGHT = 540
const AUDIO_BITRATE = 64000

function log(...args) { console.log('[RELAY]', ...args) }
function logErr(...args) { console.error('[RELAY]', ...args) }

// ── Host: 上传 ────────────────────────────────────────────────────
export function useScreenRelayUpload() {
  const isActive = ref(false)
  const ws = shallowRef(null)
  const canvas = shallowRef(null)
  const ctx = shallowRef(null)
  let frameTimer = null
  let rvfcHandle = null
  let lastSentAt = 0
  // 节流:即使 rVFC 回调来得很快,也不超过 VIDEO_FPS。
  // 屏幕源是 30/60 FPS 时,本参数把它压到 15 FPS,既省带宽又少编码热。
  const minFrameIntervalMs = 1000 / VIDEO_FPS
  let audioRecorder = null
  let audioStream = null
  let screenStream = null
  let hiddenVideo = null
  // rVFC 回调触发次数,首 3 + 每 60 报一次,用来确认 rVFC 真的有调度
  let rvfcFireCount = 0
  // captureAndSendFrame 因 videoWidth=0 / paused 静默 bail 的次数,首 3 + 每 60 报一次
  let captureBailLog = 0
  // host 端 viewer 通过 relay WS 转发来的远程控制 JSON 消息
  let onControlCb = null

  function onControl(cb) { onControlCb = cb }

  function buildVideoUrl() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    return `${proto}://${location.host}/api/screen/sessions/${currentSessionId.value}/relay-host?token=${encodeURIComponent(currentToken.value)}`
  }

  // 复用父组件已有的 session id + askit token(ref 由 useScreenRelayUpload 调用方注入)
  const currentSessionId = ref('')
  const currentToken = ref('')

  function setSession(sessionId, token) {
    currentSessionId.value = sessionId
    currentToken.value = token
  }

  function sendBinary(typeByte, payload) {
    const wsVal = ws.value
    if (!wsVal || wsVal.readyState !== WebSocket.OPEN) return
    // 1 byte type + payload — 用 concat 拼一个 ArrayBuffer 一次发送,
    // 比发两条消息快,也保证一条 WS 消息就是一帧。
    const buf = new Uint8Array(1 + payload.byteLength)
    buf[0] = typeByte
    buf.set(new Uint8Array(payload), 1)
    wsVal.send(buf)
  }

  // rVFC 调度器:每个新帧回调里跑 captureAndSendFrame,除非还没到
  // minFrameIntervalMs(限速到 VIDEO_FPS)。
  function scheduleNextFrame() {
    if (!hiddenVideo || !isActive.value) return
    if (typeof hiddenVideo.requestVideoFrameCallback === 'function') {
      rvfcHandle = hiddenVideo.requestVideoFrameCallback((_now, metadata) => {
        if (!isActive.value) return
        const now = performance.now()
        // 诊断:首 3 次 rVFC 回调 + 之后每 60 帧打印一次,确认 rVFC 真的有调度
        rvfcFireCount++
        if (rvfcFireCount <= 3 || rvfcFireCount % 60 === 0) {
          log('rVFC fired #', rvfcFireCount, 'mediaTime=', metadata?.mediaTime?.toFixed(3),
            'vw=', hiddenVideo.videoWidth, 'vh=', hiddenVideo.videoHeight)
        }
        // 节流:距上次发送时间 < 间隔就跳过。
        if (now - lastSentAt >= minFrameIntervalMs) {
          lastSentAt = now
          captureAndSendFrame()
        }
        // 继续等下一帧。
        scheduleNextFrame()
      })
    } else {
      // 降级路径:旧浏览器没有 rVFC,用 setInterval(显式 fps 上限)。
      // 但只在屏幕真正在变时才抓 — 用 hiddenVideo.currentTime 变化检测。
      frameTimer = setInterval(() => {
        if (!hiddenVideo || hiddenVideo.paused || hiddenVideo.ended) return
        const now = performance.now()
        if (now - lastSentAt >= minFrameIntervalMs) {
          lastSentAt = now
          captureAndSendFrame()
        }
      }, 1000 / VIDEO_FPS)
    }
  }

  async function captureAndSendFrame() {
    const c = canvas.value
    const cx = ctx.value
    const v = hiddenVideo
    if (!c || !cx || !v || !v.videoWidth) {
      // 不再静默:首 3 次 bail 各记一次,之后每 60 帧提醒一次。
      // 帮用户/我自己知道"画面没出但音频在推"是不是这里卡住。
      captureBailLog++
      if (captureBailLog <= 3 || captureBailLog % 60 === 0) {
        logErr('captureAndSendFrame bail #', captureBailLog,
          'canvas=' + !!c, 'ctx=' + !!cx, 'video=' + !!v,
          'videoWidth=' + (v?.videoWidth || 0),
          'paused=' + (v?.paused), 'ended=' + (v?.ended),
          'readyState=' + (v?.readyState))
      }
      return
    }
    // 等比缩放到 VIDEO_MAX_WIDTH/HEIGHT 以内,确保 canvas 输出的 JPEG 帧
    // 体积小到可以在 100ms 内推完。1080p 屏幕缩小到 960×540 后帧大小从
    // ~200KB 降到 ~40KB,带宽 / 延迟都显著改善。
    const srcW = v.videoWidth
    const srcH = v.videoHeight
    const scale = Math.min(1, VIDEO_MAX_WIDTH / srcW, VIDEO_MAX_HEIGHT / srcH)
    const dstW = Math.round(srcW * scale)
    const dstH = Math.round(srcH * scale)
    if (c.width !== dstW || c.height !== dstH) {
      c.width = dstW
      c.height = dstH
    }
    cx.drawImage(v, 0, 0, dstW, dstH)
    c.toBlob(async (blob) => {
      if (!blob) return
      const ab = await blob.arrayBuffer()
      sendBinary(FRAME_TYPE_VIDEO, ab)
      log('frame sent size=' + ab.byteLength + ' src=' + srcW + 'x' + srcH + ' dst=' + dstW + 'x' + dstH)
    }, 'image/jpeg', VIDEO_QUALITY)
  }

  async function startAudioCapture() {
    // mic 权限失败就静默降级:只传视频,不报错。host 不一定需要音频。
    try {
      audioStream = await navigator.mediaDevices.getUserMedia({
        audio: { echoCancellation: true, noiseSuppression: true },
      })
    } catch (e) {
      log('mic capture skipped:', e?.name, e?.message)
      return
    }
    // 优先用 Opus,Chrome / Safari 都支持 audio/webm;codecs=opus。
    // Fallback 到默认 mimeType。
    let mimeType = 'audio/webm;codecs=opus'
    if (typeof MediaRecorder !== 'undefined' && !MediaRecorder.isTypeSupported(mimeType)) {
      mimeType = 'audio/webm'
      if (!MediaRecorder.isTypeSupported(mimeType)) {
        log('MediaRecorder not supported, audio skipped')
        audioStream.getTracks().forEach(t => t.stop())
        audioStream = null
        return
      }
    }
    audioRecorder = new MediaRecorder(audioStream, {
      mimeType,
      audioBitsPerSecond: AUDIO_BITRATE,
    })
    audioRecorder.ondataavailable = async (e) => {
      if (!e.data || e.data.size === 0) return
      const ab = await e.data.arrayBuffer()
      sendBinary(FRAME_TYPE_AUDIO, ab)
      log('audio chunk sent size=' + ab.byteLength)
    }
    audioRecorder.start(100) // 100ms 一片
  }

  function stopAudioCapture() {
    if (audioRecorder && audioRecorder.state !== 'inactive') {
      try { audioRecorder.stop() } catch {}
    }
    audioRecorder = null
    if (audioStream) {
      audioStream.getTracks().forEach(t => t.stop())
      audioStream = null
    }
  }

  async function start(screenMediaStream) {
    if (isActive.value) return
    if (!currentSessionId.value || !currentToken.value) {
      logErr('start called without session/token')
      return
    }
    screenStream = screenMediaStream
    log('start, session=' + currentSessionId.value)

    // 准备 video 抓帧用的 hidden <video> + <canvas>
    //
    // 关键坑:不能只设 srcObject 就走人。Chromium 对离屏 <video>(没挂 DOM、没
    // 调用 play())的 rVFC 行为不一致 — 经常 videoWidth=0,导致 captureAndSendFrame
    // 静默 return(viewer 端就一直收不到 type=1 帧,只能收到 type=2 音频,页面"卡死"
    // 没有画面)。修法:挂到 DOM 上(visibility:hidden 仍计算布局/合成),autoplay+muted
    // 之外显式 play() 一把,play() 失败时记日志(可能权限/编解码问题)。
    hiddenVideo = document.createElement('video')
    hiddenVideo.autoplay = true
    hiddenVideo.muted = true
    hiddenVideo.playsInline = true
    hiddenVideo.style.position = 'fixed'
    hiddenVideo.style.left = '-9999px'
    hiddenVideo.style.top = '0'
    hiddenVideo.style.width = '1px'
    hiddenVideo.style.height = '1px'
    hiddenVideo.style.opacity = '0'
    hiddenVideo.style.pointerEvents = 'none'
    hiddenVideo.srcObject = screenMediaStream
    document.body.appendChild(hiddenVideo)
    hiddenVideo.play().catch((e) => {
      logErr('hidden video play() rejected:', e?.name, e?.message, '— rVFC may never fire, viewer will not see video')
    })
    canvas.value = document.createElement('canvas')
    ctx.value = canvas.value.getContext('2d')
    // hidden video 需要 metadata 加载完才能 drawImage;用 loadedmetadata 触发首帧.
    // 失败时(老流没 metadata)也记日志,不要静默。
    hiddenVideo.addEventListener('loadedmetadata', () => {
      log('hidden video metadata', hiddenVideo.videoWidth, 'x', hiddenVideo.videoHeight)
      if (!hiddenVideo.videoWidth) {
        logErr('hidden video metadata loaded but videoWidth=0 — capture will bail every frame')
      }
    }, { once: true })
    hiddenVideo.addEventListener('error', (e) => {
      logErr('hidden video element error', e)
    })

    const wsUrl = buildVideoUrl()
    ws.value = new WebSocket(wsUrl)
    ws.value.binaryType = 'arraybuffer'
    ws.value.onopen = () => {
      log('upload WS open')
      isActive.value = true
      // 视频抓帧:用 requestVideoFrameCallback 跟随屏幕源帧率,避免 setInterval
      // 固定间隔(67ms for 15FPS)与屏幕真实帧不同步导致的"采集空帧"或"漏帧"。
      // rVFC 的回调传 metadata.mediaTime,可计算两帧间隔做自适应 FPS 上限。
      scheduleNextFrame()
      // 音频抓帧:异步,失败也不阻塞视频。
      startAudioCapture()
    }
    ws.value.onclose = (ev) => {
      log('upload WS close', ev.code, ev.reason)
      isActive.value = false
      stop()
    }
    ws.value.onerror = (e) => {
      logErr('upload WS error', e)
    }
    // host 端:接收 viewer 转发过来的远程控制消息(viewer→host 走 TextMessage,
    // 经后端转发回来)。P2P data channel 在 NAT 下不可达时这是唯一通道。
    ws.value.onmessage = (ev) => {
      if (typeof ev.data !== 'string') return
      let msg
      try { msg = JSON.parse(ev.data) } catch { return }
      if (msg?.kind && onControlCb) onControlCb(msg.kind, msg.data)
    }
  }

  function stop() {
    if (frameTimer) { clearInterval(frameTimer); frameTimer = null }
    if (rvfcHandle && hiddenVideo && typeof hiddenVideo.cancelVideoFrameCallback === 'function') {
      try { hiddenVideo.cancelVideoFrameCallback(rvfcHandle) } catch {}
    }
    rvfcHandle = null
    stopAudioCapture()
    if (ws.value) {
      try { ws.value.close() } catch {}
      ws.value = null
    }
    if (hiddenVideo) {
      try { hiddenVideo.srcObject = null } catch {}
      // 之前 start 时把 video 挂到 body 上了,stop 时要摘掉,否则多次 start/stop
      // 会在 body 上累积 1×1 离屏 video 节点。
      try { hiddenVideo.remove() } catch {}
      hiddenVideo = null
    }
    canvas.value = null
    ctx.value = null
    screenStream = null
    lastSentAt = 0
    rvfcFireCount = 0
    captureBailLog = 0
    isActive.value = false
    log('stopped')
  }

  return { isActive, start, stop, setSession, onControl }
}

// ── Viewer: 下载 ────────────────────────────────────────────────
export function useScreenRelayDownload() {
  const isActive = ref(false)
  const lastFrameAt = ref(0)
  const ws = shallowRef(null)
  let canvasEl = null
  let canvasCtx = null
  let audioCtx = null
  let nextPlayTime = 0
  let currentSessionId = ''
  let currentPassword = ''
  let currentNickname = ''

  function setContext(sessionId, password = '', nickname = '') {
    currentSessionId = sessionId
    currentPassword = password
    currentNickname = nickname
  }

  async function ensureAudioCtx() {
    if (audioCtx) return audioCtx
    audioCtx = new (window.AudioContext || window.webkitAudioContext)()
    return audioCtx
  }

  function paintVideoFrame(jpegBytes) {
    if (!canvasEl || !canvasCtx) {
      logErr('paintVideoFrame dropped: canvasEl/canvasCtx null (ref not ready?)')
      return
    }
    // createImageBitmap 是浏览器推荐的高效路径:不阻塞主线程,GPU 解码
    createImageBitmap(new Blob([jpegBytes], { type: 'image/jpeg' })).then((bmp) => {
      if (!canvasEl || !canvasCtx) { bmp.close(); return }
      if (canvasEl.width !== bmp.width || canvasEl.height !== bmp.height) {
        canvasEl.width = bmp.width
        canvasEl.height = bmp.height
      }
      canvasCtx.drawImage(bmp, 0, 0)
      bmp.close()
      lastFrameAt.value = Date.now()
      log('frame painted, size=' + jpegBytes.byteLength)
    }).catch((e) => logErr('paintVideoFrame', e?.message))
  }

  async function playAudioChunk(_opusBytes) {
    // MediaRecorder 产出的 audio/webm;codecs=opus 是流式分片(每片 ~1KB),
    // 不是完整可解码容器;AudioContext.decodeAudioData 拿到这种片段会直接抛
    // "Unable to decode audio data"。要支持流式 Opus 需要引入 opus-decoder 这类
    // 库,体积代价不划算(屏幕分享以画面为主,声音是可选项)。
    //
    // 这里静默丢弃音频分片,host 仍然在推(viewer 仍然在收),只是 viewer 不播。
    // 静音单跳会浪费 host 上行带宽,后续要么上 opus-decoder,要么 host 端改推
    // 完整 webm 容器(需要缓冲几片)。当前目标:视频画面已通,先把音频路径关掉
    // 避免 console 一堆红字。
  }

  function start(el) {
    if (isActive.value) return
    if (!currentSessionId) {
      logErr('start called without sessionId')
      return
    }
    canvasEl = el
    canvasCtx = el ? el.getContext('2d') : null

    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const params = new URLSearchParams()
    if (currentPassword) params.set('password', currentPassword)
    if (currentNickname) params.set('nickname', currentNickname)
    const qs = params.toString() ? `?${params.toString()}` : ''
    const url = `${proto}://${location.host}/api/screen/sessions/${currentSessionId}/relay-viewer${qs}`

    ws.value = new WebSocket(url)
    ws.value.binaryType = 'arraybuffer'
    ws.value.onopen = () => {
      log('viewer WS open')
      isActive.value = true
      nextPlayTime = 0
    }
    ws.value.onmessage = async (ev) => {
      // 服务端只发 binary(latest 帧 + 后续帧都是 binary)
      if (typeof ev.data === 'string') return
      const buf = ev.data
      if (!(buf instanceof ArrayBuffer) || buf.byteLength < 2) return
      // ArrayBuffer 不能用 [] 索引,必须包成 Uint8Array 读第一个 byte
      // 之前直接 buf[0] 是 undefined,导致 type 比较永远不通过,帧被丢。
      const view = new Uint8Array(buf)
      const type = view[0]
      const payload = buf.slice(1)
      log('binary recv type=' + type + ' size=' + buf.byteLength)
      if (type === FRAME_TYPE_VIDEO) {
        paintVideoFrame(payload)
      } else if (type === FRAME_TYPE_AUDIO) {
        // 不 await:音频播放是异步链,不能让慢的解码卡住帧渲染
        playAudioChunk(payload)
      }
    }
    ws.value.onclose = (ev) => {
      log('viewer WS close', ev.code, ev.reason)
      isActive.value = false
    }
    ws.value.onerror = (e) => {
      logErr('viewer WS error', e)
    }
  }

  function stop() {
    if (ws.value) {
      try { ws.value.close() } catch {}
      ws.value = null
    }
    if (audioCtx) {
      try { audioCtx.close() } catch {}
      audioCtx = null
    }
    nextPlayTime = 0
    isActive.value = false
    canvasEl = null
    canvasCtx = null
    log('stopped')
  }

  // sendInput — 远程控制事件(viewer→host)走 viewer 这条 WS,以 TextMessage 发 JSON。
  // P2P 不通时这是唯一的远程控制通道。
  function sendInput(kind, data) {
    const wsVal = ws.value
    if (!wsVal || wsVal.readyState !== WebSocket.OPEN) return false
    wsVal.send(JSON.stringify({ kind, data }))
    return true
  }

  return { isActive, lastFrameAt, start, stop, setContext, sendInput }
}