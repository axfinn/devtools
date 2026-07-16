// 端到端模拟:viewer 先发 offer、host 后建 peer,验证缓冲回放能让协商成功。
// 这是真实用户场景("等主机开始共享"页面上要等好几秒)的回归测试。
//
// 用法:
//   cd frontend && npm install ws simple-peer @roamhq/wrtc --no-save
//   node tests/screen-rtc-race.mjs
import SimplePeer from 'simple-peer'
import wrtc from '@roamhq/wrtc'
import { WebSocketServer, WebSocket } from 'ws'

SimplePeer.config = { wrtc }
const iceServers = [{ urls: 'stun:stun.l.google.com:19302' }]
const sleep = (ms) => new Promise(r => setTimeout(r, ms))
const log = (...a) => process.stdout.write(a.join(' ') + '\n')

function makeRTC(role) {
  const peers = new Map()
  const pendingSignals = new Map()
  let signalingWS = null
  function handleSignal(peerId, msg) {
    const entry = peers.get(peerId)
    if (!entry) {
      const buf = pendingSignals.get(peerId) || []
      buf.push(msg); pendingSignals.set(peerId, buf); return
    }
    if (msg.type === 'ice') entry.peer.signal({ type: 'candidate', candidate: msg.candidate })
    else entry.peer.signal({ type: msg.type, sdp: msg.sdp })
  }
  function makePeer({ peerId, initiator, stream }) {
    const peer = new SimplePeer({ initiator, wrtc, trickle: false, stream, offerOptions: { offerToReceiveVideo: true, offerToReceiveAudio: true }, config: { iceServers } })
    peers.set(peerId, { peer })
    peer.on('signal', (d) => {
      if (d.candidate) return
      signalingWS.send(JSON.stringify({ type: d.type, sdp: d.sdp, to: peerId === 'host' ? 'viewer' : 'host' }))
    })
    return peer
  }
  function attach(peerId, stream) {
    const peer = makePeer({ peerId, initiator: false, stream })
    if (pendingSignals.has(peerId)) {
      const buf = pendingSignals.get(peerId); pendingSignals.delete(peerId)
      for (const msg of buf) handleSignal(peerId, msg)
    }
    return peer
  }
  function startViewer(peerId) { return makePeer({ peerId, initiator: true }) }
  function setWS(ws) { signalingWS = ws }
  return { handleSignal, attach, startViewer, setWS, peers }
}

async function run() {
  const wss = new WebSocketServer({ port: 17999 })
  const clients = {}
  wss.on('connection', (ws, req) => {
    const role = new URL(req.url, 'http://x').searchParams.get('role')
    clients[role] = ws
    ws.on('message', (data) => {
      const msg = JSON.parse(data.toString())
      const dst = role === 'host' ? 'viewer' : 'host'
      if (clients[dst]) clients[dst].send(JSON.stringify({ ...msg, from: role }))
    })
  })
  await new Promise(r => wss.once('listening', r))

  const hostWS = new WebSocket('ws://localhost:17999/?role=host')
  await new Promise(r => hostWS.on('open', r))
  const hostRTC = makeRTC('HOST'); hostRTC.setWS(hostWS)
  hostWS.on('message', (data) => {
    const msg = JSON.parse(data.toString())
    if (['offer','answer','ice'].includes(msg.type)) hostRTC.handleSignal(msg.from || 'viewer', msg)
  })

  await sleep(100)
  const viewerWS = new WebSocket('ws://localhost:17999/?role=viewer')
  await new Promise(r => viewerWS.on('open', r))
  const viewerRTC = makeRTC('VIEWER'); viewerRTC.setWS(viewerWS)
  viewerWS.on('message', (data) => {
    const msg = JSON.parse(data.toString())
    if (['offer','answer','ice'].includes(msg.type)) viewerRTC.handleSignal(msg.from || 'host', msg)
  })
  viewerRTC.startViewer('host')

  let viewerGotStream = false
  viewerRTC.peers.get('host').peer.on('stream', () => { viewerGotStream = true; log('  *** VIEWER got stream ***') })

  // 模拟真实场景:viewer 进入后等 8 秒,host 才点 "开始共享"
  await sleep(8000)
  const { RTCVideoSource } = wrtc.nonstandard
  const hostStream = new wrtc.MediaStream([new RTCVideoSource().createTrack()])
  hostRTC.attach('viewer', hostStream)

  await sleep(8000)
  log('=== 结果 ==='); log('  viewerGotStream=' + viewerGotStream)
  wss.close(); process.exit(viewerGotStream ? 0 : 1)
}

run().catch(e => { console.error(e); process.exit(2) })
