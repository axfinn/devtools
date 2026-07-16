// useScreenRTC — 屏幕共享的 WebRTC 抽象。
//
// 用 simple-peer 替代手写的 RTCPeerConnection,避免一系列坑:
//   - offer/answer 时序
//   - ICE 候选乱序(addIceCandidate 在 setRemoteDescription 之前调用会丢)
//   - renegotiation
//   - 数据通道生命周期
//   - 媒体协商(默认 offer 没有 m=video 时 Chrome 不可靠地自动加 transceiver)
//
// simple-peer 内部已经处理了以上所有,外部只需要:
//   - host:new Peer({initiator:false, stream}) 让 viewer 发 offer 过来
//   - viewer:new Peer({initiator:true}) 自动 createOffer
//   - peer.on('signal') 把 SDP/ICE 转发给对端
//   - 收到 signal 调 peer.signal(data) 喂回去
//   - peer.on('stream') 拿到对方的 track流
//   - peer.send(data) 通过内置 data channel 发控制消息
//
// 只需要关心:
//   - 信令收发(纯透传)
//   - TURN 凭证获取
//   - 屏幕捕获生命周期

import { ref, shallowRef } from 'vue'
import { Buffer } from 'buffer'
import SimplePeer from 'simple-peer'

// simple-peer 内部 import 'buffer' / 'process' / 'global',浏览器没这些。
// vite.config.js 的 simplePeerProcessShim 在构建时把所有裸 process.X / global / Buffer
// 替换成 shim。Buffer 的 shim 引用 globalThis.Buffer,所以这里挂一下。
if (typeof globalThis !== 'undefined' && !globalThis.Buffer) {
  globalThis.Buffer = Buffer
}
if (typeof window !== 'undefined') {
  if (!window.global) window.global = window
}

export function useScreenRTC() {
  const peers = ref(new Map()) // peerId → { peer, role, signaling }
  const localStream = shallowRef(null)
  const remoteStreams = ref(new Map()) // peerId → MediaStream
  const state = ref('idle') // idle | connecting | connected | failed | closed
  const lastError = ref('')

  let iceServers = [{ urls: 'stun:stun.l.google.com:19302' }]
  let iceServersPromise = null

  // 收 signal 时 peer 还没创建就排队,等 hostAttachViewer / startViewerSession
  // 创建 peer 后回放(关键:host 还没开始共享时 viewer 的 offer 会先到)
  const pendingSignals = new Map() // peerId → [{type, sdp, candidate}]

  function log(...args) { console.log('[RTC]', ...args) }
  function logErr(...args) { console.error('[RTC]', ...args) }

  // Chrome 屏幕选择器 / LastPass 等扩展会在 unhandledrejection 抛
  // "Could not establish connection. Receiving end does not exist." 等噪声,
  // 与 WebRTC 流程无关,preventDefault 让控制台干净。
  if (typeof window !== 'undefined' && !window.__screenRtcRejInstalled) {
    window.__screenRtcRejInstalled = true
    const isChromeInternalMsg = (msg) =>
      /Could not establish connection\. Receiving end does not exist/.test(msg)
      || /Receiving end does not exist/.test(msg)
      || /Attempting to use a disconnected port object/.test(msg)
      || /The message port closed before a response was received/.test(msg)
      || /Called encrypt\(\) without a session key/.test(msg)
    window.addEventListener('unhandledrejection', (ev) => {
      const r = ev.reason
      const msg = r?.message || String(r)
      if (isChromeInternalMsg(msg)) {
        logErr('UNHANDLED REJECTION (chrome internal, suppressed)', 'name=' + r?.name, 'message=' + msg)
        ev.preventDefault(); return
      }
      logErr('UNHANDLED REJECTION', 'name=' + r?.name, 'message=' + msg)
    })
    window.addEventListener('error', (ev) => {
      const msg = ev.message || ''
      if (isChromeInternalMsg(msg)) {
        logErr('WINDOW ERROR (chrome internal, suppressed)', msg)
        ev.preventDefault(); return
      }
      logErr('WINDOW ERROR', msg)
    })
  }

  // Host WS 建连后立即调用,避免 viewer 第一个 offer 到达时 iceServers 还没好。
  function prefetchIceServers() {
    if (iceServersPromise) return iceServersPromise
    iceServersPromise = (async () => {
      try {
        const r = await fetch('/api/screen/turn-credentials')
        if (!r.ok) { logErr('turn-credentials HTTP', r.status); return }
        const c = await r.json()
        iceServers = [
          { urls: c.stun },
          { urls: c.turn, username: c.username, credential: c.credential },
        ]
        log('iceServers ready, stun=' + c.stun, 'turn=' + c.turn)
      } catch (e) { logErr('turn-credentials fetch err', e?.message) }
    })()
    return iceServersPromise
  }

  // 创建一个 simple-peer 实例并 wire 事件
  function makePeer({ initiator, signaling, peerId, stream, offerOptions }) {
    log('makePeer', peerId, 'initiator=' + initiator, 'hasStream=' + !!stream, 'offerOptions=' + JSON.stringify(offerOptions || {}))
    const peer = new SimplePeer({
      initiator,
      trickle: false,            // 等 ICE 收集完再 emit signal,杜绝乱序
      config: { iceServers },
      stream,                    // host 传屏幕流;viewer 留空
      offerOptions,              // viewer 必传 { offerToReceiveVideo: true }
    })
    peers.value.set(peerId, { peer, role: initiator ? 'viewer' : 'host', signaling })
    peers.value = new Map(peers.value)

    peer.on('signal', (data) => {
      log('signal', peerId, data.type, data.type === 'candidate'
        ? data.candidate?.candidate?.split(' ')[7]
        : 'sdpLen=' + (data.sdp?.length || 0))
      // simple-peer → wire: candidate 翻译成 ice (与后端 screen.go 一致)
      const msg = data.type === 'candidate'
        ? { type: 'ice', candidate: data.candidate, to: peerId }
        : data.type === 'transceiverRequest'
          // 非发起方请求对端加 transceiver:后端 screen.go 不识别这个 type,
          // 因此不能依赖这个机制做 renegotiation,改用 offerToReceiveVideo 在 offer 里直接加
          ? null
          : { type: data.type, sdp: data.sdp, to: peerId }
      if (msg) signaling.send(msg)
    })
    peer.on('connect', () => log('peer connect (data channel open)', peerId))
    peer.on('stream', (remoteStream) => {
      log('peer stream', peerId, 'tracks=' + remoteStream.getTracks().map(t => t.kind).join(','))
      remoteStreams.value.set(peerId, remoteStream)
      remoteStreams.value = new Map(remoteStreams.value)
    })
    peer.on('data', (data) => {
      // 来自 viewer 的远程控制事件(JSON)
      let msg
      try { msg = JSON.parse(data.toString()) } catch { return }
      if (msg?.kind) dispatchSyntheticEvent(msg)
    })
    peer.on('error', (err) => {
      logErr('peer error', peerId, err?.code, err?.message)
      lastError.value = err?.message || String(err)
    })
    peer.on('close', () => {
      log('peer close', peerId)
      peers.value.delete(peerId)
      remoteStreams.value.delete(peerId)
      peers.value = new Map(peers.value)
      remoteStreams.value = new Map(remoteStreams.value)
    })

    return peer
  }

  // Host: viewer 加入时调用。host 是 answerer,等 viewer 发 offer。
  // 必须先 await TURN 凭证:connectHostWS 里 prefetchIceServers 是 fire-and-forget,
  // viewer 加入可能比 TURN 凭证 fetch 早几百毫秒 → 这时 iceServers 还是 STUN-only。
  // RTCPeerConnection 的 iceServers 在 new 时就冻住,后改 iceServers 变量名无效。
  // viewer 侧 await 过了,host 侧容易漏这一步 → host 拿不到 relay candidate,
  // 双方都在 NAT 后时 ICE 直接 ERR_CONNECTION_FAILURE。
  async function hostAttachViewer(peerId, signaling) {
    if (!localStream.value) {
      logErr('hostAttachViewer no localStream', peerId)
      throw new Error('no_local_stream')
    }
    await prefetchIceServers()
    const peer = makePeer({
      initiator: false,
      signaling,
      peerId,
      stream: localStream.value,
    })
    // replay buffer:viewer 的 offer 可能在我们创建 peer 之前就到了,
    // 没 peer 时 handleSignal 直接丢了。现在 peer 存在了,回放
    if (pendingSignals.has(peerId)) {
      const buf = pendingSignals.get(peerId)
      pendingSignals.delete(peerId)
      log('replay pending signals for', peerId, 'count=' + buf.length)
      for (const msg of buf) handleSignal(peerId, msg)
    }
    return peer
  }

  // Viewer: 自己启动时调用。viewer 是 initiator,主动发 offer。
  // 必须传 offerToReceiveVideo:不加的话 simple-peer 默认 offer 里只有 m=application
  // (数据通道),没有 m=video → host 无法在 answer 里给 video → viewer 永远拿不到视频流
  function startViewerSession(peerId, signaling) {
    return makePeer({
      initiator: true,
      signaling,
      peerId,
      offerOptions: { offerToReceiveVideo: true, offerToReceiveAudio: true },
    })
  }

  // 收到对端发来的 SDP / ICE (wire-format),喂回 simple-peer
  function handleSignal(peerId, msg) {
    const entry = peers.value.get(peerId)
    if (!entry) {
      // peer 还没建好(viewer 的 offer 比 host 启动共享快):
      // 排队,等 hostAttachViewer 创建 peer 后回放
      log('handleSignal: no peer yet, buffering', peerId, msg.type)
      const buf = pendingSignals.get(peerId) || []
      if (buf.length < 8) {  // 防失控
        buf.push(msg)
        pendingSignals.set(peerId, buf)
      }
      return
    }
    if (msg.type === 'ice') {
      log('handleSignal ice', peerId)
      // wire ice → simple-peer candidate
      entry.peer.signal({ type: 'candidate', candidate: msg.candidate })
    } else {
      log('handleSignal', peerId, msg.type, 'sdpLen=' + (msg.sdp?.length || 0))
      entry.peer.signal({ type: msg.type, sdp: msg.sdp })
    }
  }

  // Host: 把流里新加的 track 喂给已有 peer,触发 renegotiation。
  function addLocalTrackToPeer(peerId, track, stream) {
    const entry = peers.value.get(peerId)
    if (!entry) return
    log('addLocalTrackToPeer', peerId, track.kind)
    try { entry.peer.addTrack(track, stream) } catch (e) { logErr('addTrack err', e?.message) }
  }

  function removePeer(peerId) {
    const entry = peers.value.get(peerId)
    if (!entry) return
    log('removePeer', peerId)
    try { entry.peer.destroy() } catch {}
    peers.value.delete(peerId)
    remoteStreams.value.delete(peerId)
    pendingSignals.delete(peerId)
    peers.value = new Map(peers.value)
    remoteStreams.value = new Map(remoteStreams.value)
  }

  function closeAll() {
    log('closeAll, peers=' + peers.value.size)
    for (const entry of peers.value.values()) {
      try { entry.peer.destroy() } catch {}
    }
    peers.value.clear()
    remoteStreams.value.clear()
    pendingSignals.clear()
    peers.value = new Map(peers.value)
    remoteStreams.value = new Map(remoteStreams.value)
    if (localStream.value) {
      localStream.value.getTracks().forEach(t => t.stop())
      localStream.value = null
    }
    state.value = 'closed'
  }

  // ── 屏幕捕获 ──
  async function startCapture() {
    log('startCapture')
    state.value = 'connecting'
    try {
      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: { frameRate: 30 },
        audio: false,
      })
      log('startCapture ok, tracks=' + stream.getTracks().map(t => `${t.kind}:${t.id.slice(0,8)}`).join(','))
      localStream.value = stream
      stream.getVideoTracks()[0].onended = () => {
        log('user stopped sharing via browser UI')
        state.value = 'closed'
        localStream.value = null
        closeAll()
      }
      state.value = 'connected'
      return stream
    } catch (e) {
      logErr('startCapture failed', e?.name, e?.message)
      state.value = 'failed'
      // 友好错误信息:用户取消 / 权限拒绝 / 安全限制 等情况各自有不同原因,
      // 尤其 macOS Sequoia 上 Chrome 需要 系统设置 → 隐私与安全 → 屏幕录制 → 勾上 Chrome,
      // 不然直接 NotAllowedError,用户根本看不到 picker。
      lastError.value = friendlyCaptureError(e)
      throw e
    }
  }

  function friendlyCaptureError(e) {
    const name = e?.name || ''
    const msg = e?.message || String(e)
    if (name === 'NotAllowedError' || /denied|permission/i.test(msg)) {
      return '屏幕共享被拒绝。如果是 macOS Sequoia,需要 系统设置 → 隐私与安全 → 屏幕录制 → 勾上 Chrome,然后重启浏览器。'
    }
    if (name === 'NotFoundError') return '没找到可共享的屏幕源。'
    if (name === 'NotReadableError') return '屏幕源被其他程序占用,请关闭录屏软件后重试。'
    if (/secure|https|insecure/i.test(msg)) {
      return '浏览器要求 HTTPS 才能共享屏幕(localhost 除外)。'
    }
    return msg
  }

  function stopCapture() { closeAll() }

  // ── 派发 viewer → host 的合成事件(只 host 端用) ──
  //
  // 重要现实:纯 web 没法影响 host 操作系统的桌面其他应用。Chrome / Safari 没有
  // 公开 API 让 JS 把鼠标移到 host 桌面 / 模拟键盘事件给非自己 tab 的应用。
  // chrome.desktopCapture / CDP 的 Input.dispatchMouseEvent 是 native/扩展 API,
  // 普通 web tab 拿不到。
  //
  // 所以这里实际能做的是"在 host 视频预览上显示 viewer 的输入反馈",告诉用户
  // 控制消息已经传到了 host 端。鼠标点哪、看哪、点几下,在 host 视频框里有视觉反馈。
  // 如果用户希望真正"远程控 host 桌面",那是另一个产品形态(需要 helper app 或扩展),
  // 当前的屏幕共享产品定位是"远程展示 + 协作说明",viewer 输入作为 hover 高亮 +
  // 标注提示,不期望真正影响 host OS。
  //
  // data 字段(viewer 发来的):{ x, y } 是 viewer 端相对于其视频元素的 client 坐标。
  // host 端需要按"host 视频预览的 rect"做反向映射,得到 host viewport 坐标用于
  // 在 host 视频预览上叠一层 canvas 高亮。
  function dispatchSyntheticEvent(msg) {
    const { kind, data } = msg
    if (!data) return
    // 触发一个全局事件,host 的 ScreenShareTool 注册监听,负责在视频预览上画反馈
    window.dispatchEvent(new CustomEvent('__remote_input_feedback__', {
      detail: { kind, data },
    }))
  }

  return {
    peers, localStream, remoteStreams, state, lastError,
    startCapture, stopCapture, prefetchIceServers,
    hostAttachViewer, startViewerSession, handleSignal,
    addLocalTrackToPeer, removePeer, closeAll,
    dispatchSyntheticEvent,
  }
}

// ── viewer 端 input 事件采集 ──
// 默认走 P2P data channel(simple-peer 内置 _channel);如果外部传入 sendFn
// (如 relayDownload.sendInput),则用 sendFn 走 server relay,适用于 P2P 不通的场景。
export function useRemoteInputSender(getPeer) {
  let customSend = null

  function setSender(fn) {
    customSend = fn
  }

  function send(kind, data) {
    if (customSend) {
      customSend(kind, data)
      return
    }
    const peer = typeof getPeer === 'function' ? getPeer() : getPeer
    if (!peer || !peer._channel || peer._channel.readyState !== 'open') return
    peer._channel.send(JSON.stringify({ kind, data }))
  }

  // 取得 rootEl(remote video / canvas)的 bounding rect,client 坐标 → 归一化坐标。
  // host 端的反馈层只需 nx/ny,不知道 host viewport 也能正确映射到 host 的视频预览框。
  function toNormalized(e) {
    const rect = e.currentTarget?.getBoundingClientRect?.()
    if (!rect || !rect.width || !rect.height) return null
    const nx = (e.clientX - rect.left) / rect.width
    const ny = (e.clientY - rect.top) / rect.height
    if (nx < 0 || nx > 1 || ny < 0 || ny > 1) return null  // 鼠标在视频元素外
    return { nx, ny }
  }

  function onMouseMove(e) {
    const v = toNormalized(e)
    if (!v) return
    send('mousemove', v)
  }
  function onClick(e) {
    const v = toNormalized(e)
    if (!v) return
    send('click', v)
  }
  function onWheel(e) {
    const v = toNormalized(e)
    if (!v) return
    send('wheel', { ...v, deltaY: e.deltaY })
  }
  function onKeyDown(e) { send('key', { type: 'keydown', key: e.key, code: e.code, keyCode: e.keyCode }) }
  function onKeyUp(e) { send('key', { type: 'keyup', key: e.key, code: e.code, keyCode: e.keyCode }) }

  function attach(rootEl, sendFn) {
    if (sendFn) customSend = sendFn
    if (!rootEl) return () => {}
    rootEl.addEventListener('mousemove', onMouseMove)
    rootEl.addEventListener('click', onClick)
    rootEl.addEventListener('wheel', onWheel, { passive: true })
    window.addEventListener('keydown', onKeyDown)
    window.addEventListener('keyup', onKeyUp)
    return () => {
      rootEl.removeEventListener('mousemove', onMouseMove)
      rootEl.removeEventListener('click', onClick)
      rootEl.removeEventListener('wheel', onWheel)
      window.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('keyup', onKeyUp)
      customSend = null
    }
  }

  return { send, attach, setSender }
}
