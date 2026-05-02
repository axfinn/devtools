<template>
  <div class="arcade-page">
    <section class="hero-shell">
      <div class="hero-main panel">
        <p class="eyebrow">Multiplayer Arcade</p>
        <h1>多人街机房</h1>
        <p class="hero-copy">
          旧小游戏不修了，直接换新。现在是支持分享链接、实时同步和键盘操作的多人街机房。
        </p>

        <div class="game-switches">
          <button
            v-for="mode in gameModes"
            :key="mode.id"
            class="mode-card"
            :class="{ active: selectedGame === mode.id }"
            @click="selectedGame = mode.id"
          >
            <div class="mode-top">
              <span class="mode-icon">{{ mode.icon }}</span>
              <span class="mode-tag">{{ mode.tag }}</span>
            </div>
            <h2>{{ mode.name }}</h2>
            <p>{{ mode.description }}</p>
            <div class="mode-meta">
              <span>{{ mode.pace }}</span>
              <span>{{ mode.players }}</span>
            </div>
          </button>
        </div>
      </div>

      <div class="hero-side panel">
        <div class="utility-row">
          <button class="toggle-sound" :class="{ off: !soundEnabled }" @click="toggleSound">
            {{ soundEnabled ? '音效开' : '音效关' }}
          </button>
          <span>Space / Enter 热键已启用</span>
        </div>

        <template v-if="!session.active">
          <h3>开房 / 进房</h3>
          <label class="field">
            <span>昵称</span>
            <input
              v-model.trim="nickname"
              maxlength="12"
              type="text"
              placeholder="来个响亮点的名字"
              @keyup.enter="createRoom(selectedGame)"
            >
          </label>

          <div class="create-actions">
            <button
              v-for="mode in gameModes"
              :key="mode.id"
              class="create-btn"
              :class="mode.id"
              :disabled="loading"
              @click="createRoom(mode.id)"
            >
              创建{{ mode.name }}
            </button>
          </div>

          <div class="join-box">
            <label class="field">
              <span>房间码</span>
              <input
                v-model.trim="joinRoomCode"
                maxlength="6"
                type="text"
                placeholder="例如 A1B2C3"
                @keyup.enter="joinRoom"
              >
            </label>
            <button class="join-btn" :disabled="loading" @click="joinRoom">加入房间</button>
          </div>

          <div class="side-note">
            <p>支持 2-8 人。</p>
            <p>房主开局后，所有人实时同步，不需要刷新轮询。</p>
          </div>
        </template>

        <template v-else>
          <h3>{{ currentMode.name }}</h3>
          <div class="room-head">
            <div>
              <p class="room-label">房间码</p>
              <strong>{{ state.room_id || session.roomId }}</strong>
            </div>
            <div class="room-actions">
              <button class="ghost-btn" @click="copyRoomCode">复制</button>
              <button class="ghost-btn" @click="copyInviteLink">邀请</button>
              <button class="ghost-btn danger" @click="leaveRoom">退出</button>
            </div>
          </div>

          <div class="status-pill" :class="state.phase || 'lobby'">
            {{ phaseText }}
          </div>

          <p class="room-message">{{ state.message || '正在同步房间状态…' }}</p>

          <div class="host-actions">
            <button
              v-if="isHost && state.phase === 'lobby'"
              class="primary-btn"
              @click="sendEvent('start_match')"
            >
              开始比赛
            </button>
            <button
              v-if="isHost && state.phase === 'finished'"
              class="primary-btn secondary"
              @click="sendEvent('restart_match')"
            >
              再来一局
            </button>
          </div>
        </template>
      </div>
    </section>

    <section v-if="session.active" class="arena-shell">
      <aside class="scoreboard panel">
        <div class="scoreboard-head">
          <div>
            <p class="mini-label">实时排行</p>
            <h3>第 {{ displayRound }} / {{ totalRounds }} 回合</h3>
          </div>
          <div class="sync-dot" :class="{ online: isConnected }">
            {{ isConnected ? '在线' : '重连中' }}
          </div>
        </div>

        <div class="player-list">
          <div
            v-for="player in state.players"
            :key="player.id"
            class="player-card"
            :class="{ self: player.id === session.playerId }"
          >
            <div class="player-top">
              <div class="player-badge" :style="{ background: player.color }"></div>
              <div class="player-info">
                <strong>{{ player.nickname }}</strong>
                <span>{{ player.is_host ? '房主' : '玩家' }} · {{ player.connected ? '在线' : '离线' }}</span>
              </div>
              <div class="player-score">{{ player.score }}</div>
            </div>
            <div class="player-round">
              {{ playerRoundText(player) }}
            </div>
          </div>
        </div>
      </aside>

      <main class="arena-stage panel">
        <header class="stage-head">
          <div>
            <p class="mini-label">{{ currentMode.name }}</p>
            <h2>{{ currentMode.icon }} {{ currentMode.headline }}</h2>
          </div>
          <div class="timer-block">
            <div class="time-value">{{ countdownText }}</div>
            <div class="time-bar">
              <div class="time-fill" :style="{ width: `${timerProgress}%` }"></div>
            </div>
          </div>
        </header>

        <section v-if="state.game === 'reaction'" class="reaction-game">
          <div class="reaction-signal" :class="reactionVisualClass">
            <div class="signal-ring"></div>
            <div class="signal-core">
              <span>{{ reactionHeadline }}</span>
              <strong>{{ reactionSubline }}</strong>
            </div>
          </div>

          <button
            class="reaction-button"
            :class="reactionVisualClass"
            @click="sendEvent('react')"
          >
            {{ reactionButtonText }}
          </button>

          <div class="reaction-tips">
            <span>抢跑会扣 1 分</span>
            <span>第一名 5 分</span>
            <span>Space / Enter 可直接拍</span>
          </div>
        </section>

        <section v-else-if="state.game === 'beat'" class="beat-game">
          <div class="beat-arena">
            <div class="beat-track">
              <div class="beat-target-zone"></div>
              <div class="beat-marker" :style="{ left: `${beatMarkerLeft}%` }"></div>
            </div>
            <div class="beat-scale">
              <span>太早</span>
              <span>完美点</span>
              <span>太晚</span>
            </div>
          </div>

          <button class="reaction-button go" @click="sendEvent('tap_beat')">
            卡点出手
          </button>

          <div class="reaction-tips">
            <span>越接近中心越高分</span>
            <span>Perfect 5 分</span>
            <span>Space / Enter 可直接拍</span>
          </div>
        </section>

        <section v-else-if="state.game === 'sequence'" class="sequence-game">
          <div class="sequence-banner">
            <div>
              <span class="target-label">你下一个该点</span>
              <strong>{{ sequenceNextValue }}</strong>
            </div>
            <div class="sequence-progress">
              {{ selfPlayer?.progress || 0 }} / {{ state.sequence_goal || 0 }}
            </div>
          </div>

          <div class="sequence-grid">
            <button
              v-for="(item, index) in state.grid"
              :key="`seq-${index}-${item}`"
              class="sequence-cell"
              :class="sequenceCellClass(index, item)"
              :disabled="!canPickSequence(index)"
              @click="sendSequence(index)"
            >
              {{ item }}
            </button>
          </div>

          <div class="reaction-tips">
            <span>从 1 一直点到最后</span>
            <span>Tab 聚焦格子后可用 Space / Enter</span>
            <span>回合结束自动下一局</span>
          </div>
        </section>

        <section v-else class="hunt-game">
          <div class="hunt-header">
            <div class="hunt-target">
              <span class="target-label">找这个</span>
              <strong>{{ state.target || '❔' }}</strong>
            </div>
            <div class="hunt-note">
              <p>先点到先得高分，点错就锁定本回合。</p>
            </div>
          </div>

          <div class="hunt-grid">
            <button
              v-for="(item, index) in state.grid"
              :key="`${index}-${item}`"
              class="hunt-cell"
              :class="cellClass(index)"
              :disabled="!canPickTile(index)"
              @click="sendPick(index)"
            >
              {{ revealCell(index) }}
            </button>
          </div>
        </section>

        <footer class="stage-foot">
          <div class="foot-card">
            <span>玩法说明</span>
            <p>{{ currentMode.rules }}</p>
          </div>
          <div class="foot-card">
            <span>当前状态</span>
            <p>{{ state.message || '等待同步' }}</p>
          </div>
        </footer>
      </main>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

const STORAGE_KEY = 'devtools_arcade_session_v2'
const NICKNAME_KEY = 'devtools_arcade_nickname_v1'
const SOUND_KEY = 'devtools_arcade_sound_v1'

const route = useRoute()
const router = useRouter()

const gameModes = [
  {
    id: 'reaction',
    name: '反应竞速',
    icon: '⚡',
    tag: '手速局',
    pace: '10 秒进入状态',
    players: '2-8 人',
    headline: '红灯忍住，绿灯拍下去',
    description: '所有人一起等信号，谁先点中发光按钮谁吃分。抢跑直接扣分。',
    rules: '倒计时结束后不会立刻亮灯，等它真的发光再拍。抢跑扣 1 分，越快得分越高。'
  },
  {
    id: 'beat',
    name: '节拍判定',
    icon: '🎵',
    tag: '卡点局',
    pace: '一键拼准度',
    players: '2-8 人',
    headline: '不是快，是准',
    description: '光标会不断扫过判定条，卡在中心蓝区出手，分差一眼就看出来。',
    rules: '光标扫到中间蓝区时按下最好。越靠近中心分越高，适合直接用空格和回车拍。'
  },
  {
    id: 'sequence',
    name: '数字接力',
    icon: '🔢',
    tag: '竞速局',
    pace: '持续手忙脚乱',
    players: '2-8 人',
    headline: '从 1 点到最后',
    description: '同一张乱序数字盘，所有人同时开跑，谁先清完顺序谁拿高分。',
    rules: '只能按 1、2、3…… 的顺序点，点错不淘汰，但会白白浪费节奏。'
  },
  {
    id: 'hunt',
    name: '表情猎场',
    icon: '🎯',
    tag: '眼力局',
    pace: '8 秒一回合',
    players: '2-8 人',
    headline: '在人群里秒抓目标',
    description: '目标表情每回合都在变，越快找到它，分越高；点错就直接罚站。',
    rules: '看清顶部目标表情，在表情阵里第一时间点中它。点错后本回合不能再答。'
  }
]

const nickname = ref('')
const joinRoomCode = ref('')
const selectedGame = ref('reaction')
const loading = ref(false)
const nowTs = ref(Date.now())
const soundEnabled = ref(localStorage.getItem(SOUND_KEY) !== 'off')

const session = reactive({
  active: false,
  roomId: '',
  playerId: '',
  secret: ''
})

const state = reactive({
  room_id: '',
  game: 'reaction',
  phase: 'lobby',
  round: 0,
  total_rounds: 5,
  message: '',
  prompt: '',
  players: [],
  countdown_at: 0,
  signal_at: 0,
  live_at: 0,
  round_ends_at: 0,
  result_ends_at: 0,
  target: '',
  grid: [],
  winner_ids: [],
  can_start: false,
  max_players: 8,
  beat_cycle_ms: 0,
  beat_window_ms: 0,
  sequence_goal: 0
})

let ws = null
let reconnectTimer = null
let clockTimer = null
let audioContext = null
let lastSnapshot = {
  roomId: '',
  phase: '',
  round: 0,
  myScore: 0
}

const currentMode = computed(() => {
  return gameModes.find((mode) => mode.id === (state.game || selectedGame.value)) || gameModes[0]
})

const isConnected = computed(() => !!ws && ws.readyState === WebSocket.OPEN)
const selfPlayer = computed(() => state.players.find((player) => player.id === session.playerId) || null)
const isHost = computed(() => !!selfPlayer.value?.is_host)
const totalRounds = computed(() => state.total_rounds || 5)
const shareRoomCode = computed(() => state.room_id || session.roomId || normalizeRoomCode(route.query.room))
const inviteUrl = computed(() => {
  if (!shareRoomCode.value) return ''
  return `${window.location.origin}${route.path}?room=${encodeURIComponent(shareRoomCode.value)}`
})
const displayRound = computed(() => {
  if (!state.round) {
    return 0
  }
  return Math.min(state.round, totalRounds.value)
})

const phaseText = computed(() => {
  const map = {
    lobby: '等人进房',
    countdown: '倒计时',
    arming: '别抢跑',
    live: '进行中',
    result: '结算中',
    finished: '本局结束'
  }
  return map[state.phase] || '同步中'
})

const countdownText = computed(() => {
  if (state.phase === 'countdown') {
    return formatRemaining(state.countdown_at)
  }
  if (state.phase === 'arming') {
    return formatRemaining(state.signal_at)
  }
  if (state.phase === 'live') {
    return formatRemaining(state.round_ends_at)
  }
  if (state.phase === 'result') {
    return formatRemaining(state.result_ends_at)
  }
  return '--'
})

const timerProgress = computed(() => {
  if (state.phase === 'countdown') {
    return progressBetween(nowTs.value, state.countdown_at - 3000, state.countdown_at)
  }
  if (state.phase === 'arming') {
    return progressBetween(nowTs.value, state.signal_at - 2500, state.signal_at)
  }
  if (state.phase === 'live') {
    const liveStart = state.live_at || (state.round_ends_at ? state.round_ends_at - 8000 : 0)
    return progressBetween(nowTs.value, liveStart, state.round_ends_at)
  }
  if (state.phase === 'result') {
    return progressBetween(nowTs.value, state.result_ends_at - 2600, state.result_ends_at)
  }
  return 0
})

const reactionVisualClass = computed(() => {
  if (state.phase === 'live') return 'go'
  if (state.phase === 'arming') return 'tense'
  if (state.phase === 'countdown') return 'countdown'
  if (state.phase === 'result') return 'result'
  if (state.phase === 'finished') return 'finished'
  return 'idle'
})

const reactionHeadline = computed(() => {
  if (state.phase === 'live') return 'GO'
  if (state.phase === 'arming') return 'WAIT'
  if (state.phase === 'countdown') return '3…2…1…'
  if (state.phase === 'result') return 'ROUND'
  if (state.phase === 'finished') return 'DONE'
  return 'LOBBY'
})

const reactionSubline = computed(() => {
  if (state.phase === 'live') return '现在点'
  if (state.phase === 'arming') return '别抢跑'
  if (state.phase === 'countdown') return '准备'
  if (state.phase === 'result') return '结算中'
  if (state.phase === 'finished') return '看排名'
  return '等房主开局'
})

const reactionButtonText = computed(() => {
  if (state.phase === 'live') return '拍下去'
  if (state.phase === 'arming') return '先忍住'
  if (state.phase === 'countdown') return '热身中'
  if (state.phase === 'result') return '本回合结束'
  if (state.phase === 'finished') return '比赛结束'
  return '等房主开始'
})

const beatMarkerLeft = computed(() => {
  if (state.game !== 'beat' || state.phase !== 'live' || !state.live_at || !state.beat_cycle_ms) {
    return 50
  }
  const elapsed = Math.max(0, nowTs.value - state.live_at)
  const progress = (elapsed % state.beat_cycle_ms) / state.beat_cycle_ms
  return progress * 100
})

const sequenceNextValue = computed(() => {
  const next = (selfPlayer.value?.progress || 0) + 1
  if (!state.sequence_goal || next > state.sequence_goal) {
    return '完成'
  }
  return next
})

function createEmptyState() {
  return {
    room_id: '',
    game: selectedGame.value,
    phase: 'lobby',
    round: 0,
    total_rounds: 5,
    message: '',
    prompt: '',
    players: [],
    countdown_at: 0,
    signal_at: 0,
    live_at: 0,
    round_ends_at: 0,
    result_ends_at: 0,
    target: '',
    grid: [],
    winner_ids: [],
    can_start: false,
    max_players: 8,
    beat_cycle_ms: 0,
    beat_window_ms: 0,
    sequence_goal: 0
  }
}

function resetState() {
  Object.assign(state, createEmptyState())
}

function normalizeRoomCode(value) {
  return String(value || '').trim().toUpperCase()
}

function persistSession() {
  if (!session.active) {
    sessionStorage.removeItem(STORAGE_KEY)
    return
  }
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify({
    roomId: session.roomId,
    playerId: session.playerId,
    secret: session.secret
  }))
}

function persistNickname() {
  const safeNickname = nickname.value.trim()
  if (safeNickname) {
    localStorage.setItem(NICKNAME_KEY, safeNickname)
  }
}

function resetSnapshotMemory() {
  lastSnapshot = {
    roomId: '',
    phase: '',
    round: 0,
    myScore: 0
  }
}

function ensureAudioContext() {
  if (!soundEnabled.value) {
    return null
  }
  const Ctx = window.AudioContext || window.webkitAudioContext
  if (!Ctx) {
    return null
  }
  if (!audioContext) {
    audioContext = new Ctx()
  }
  if (audioContext.state === 'suspended') {
    audioContext.resume().catch(() => {})
  }
  return audioContext
}

function playTone(frequency, duration = 0.12, type = 'sine', gainValue = 0.04, delay = 0) {
  const ctx = ensureAudioContext()
  if (!ctx) {
    return
  }
  const startAt = ctx.currentTime + delay
  const oscillator = ctx.createOscillator()
  const gainNode = ctx.createGain()
  oscillator.type = type
  oscillator.frequency.setValueAtTime(frequency, startAt)
  gainNode.gain.setValueAtTime(0.0001, startAt)
  gainNode.gain.exponentialRampToValueAtTime(gainValue, startAt + 0.01)
  gainNode.gain.exponentialRampToValueAtTime(0.0001, startAt + duration)
  oscillator.connect(gainNode)
  gainNode.connect(ctx.destination)
  oscillator.start(startAt)
  oscillator.stop(startAt + duration + 0.02)
}

function playSoundPreset(name) {
  if (!soundEnabled.value) {
    return
  }
  switch (name) {
    case 'join':
      playTone(440, 0.08, 'triangle', 0.035)
      playTone(660, 0.12, 'triangle', 0.03, 0.05)
      break
    case 'start':
      playTone(330, 0.08, 'square', 0.03)
      playTone(494, 0.1, 'square', 0.03, 0.06)
      playTone(659, 0.12, 'square', 0.03, 0.12)
      break
    case 'score':
      playTone(523.25, 0.09, 'triangle', 0.035)
      playTone(783.99, 0.14, 'triangle', 0.03, 0.05)
      break
    case 'finish':
      playTone(392, 0.1, 'sine', 0.03)
      playTone(523.25, 0.12, 'sine', 0.03, 0.07)
      playTone(783.99, 0.16, 'sine', 0.03, 0.14)
      break
    case 'toggle':
      playTone(soundEnabled.value ? 720 : 260, 0.08, 'sine', 0.03)
      break
    default:
      break
  }
}

function toggleSound() {
  soundEnabled.value = !soundEnabled.value
  localStorage.setItem(SOUND_KEY, soundEnabled.value ? 'on' : 'off')
  if (soundEnabled.value) {
    ensureAudioContext()
  }
  playSoundPreset('toggle')
}

function syncRoomQuery(roomId = '') {
  const query = { ...route.query }
  if (roomId) {
    query.room = roomId
  } else {
    delete query.room
  }
  router.replace({ path: route.path, query }).catch(() => {})
}

function loadSession() {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function applySession(payload) {
  session.active = true
  session.roomId = payload.room_id
  session.playerId = payload.player_id
  session.secret = payload.secret
  persistNickname()
  persistSession()
  syncRoomQuery(payload.room_id)
}

function leaveRoom() {
  session.active = false
  session.roomId = ''
  session.playerId = ''
  session.secret = ''
  persistSession()
  syncRoomQuery('')
  resetState()
  resetSnapshotMemory()
  disconnectSocket()
}

function disconnectSocket() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (ws) {
    const socket = ws
    ws = null
    socket.close()
  }
}

async function createRoom(game) {
  const safeNickname = nickname.value.trim()
  if (!safeNickname) {
    ElMessage.warning('先填昵称')
    return
  }
  loading.value = true
  try {
    const res = await fetch('/api/game/arcade/rooms', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        game,
        nickname: safeNickname
      })
    })
    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || '创建房间失败')
    }
    persistNickname()
    selectedGame.value = game
    applySession(data)
    connectSocket()
  } catch (error) {
    ElMessage.error(error.message || '创建房间失败')
  } finally {
    loading.value = false
  }
}

async function joinRoom() {
  const safeNickname = nickname.value.trim()
  const roomId = normalizeRoomCode(joinRoomCode.value)
  if (!safeNickname) {
    ElMessage.warning('先填昵称')
    return
  }
  if (!roomId) {
    ElMessage.warning('请输入房间码')
    return
  }
  loading.value = true
  try {
    const res = await fetch(`/api/game/arcade/rooms/${encodeURIComponent(roomId)}/join`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nickname: safeNickname })
    })
    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || '加入失败')
    }
    persistNickname()
    applySession(data)
    connectSocket()
  } catch (error) {
    ElMessage.error(error.message || '加入失败')
  } finally {
    loading.value = false
  }
}

function buildWsUrl() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const params = new URLSearchParams({
    player_id: session.playerId,
    secret: session.secret
  })
  return `${protocol}//${window.location.host}/api/game/arcade/ws/${encodeURIComponent(session.roomId)}?${params.toString()}`
}

function handleSnapshotEffects(nextState) {
  const myPlayer = nextState.players?.find((player) => player.id === session.playerId)
  if (nextState.room_id && nextState.room_id !== lastSnapshot.roomId) {
    playSoundPreset('join')
  }
  if (nextState.phase !== lastSnapshot.phase) {
    if (nextState.phase === 'live') {
      playSoundPreset('start')
    } else if (nextState.phase === 'finished') {
      playSoundPreset('finish')
    }
  }
  if (myPlayer && myPlayer.score > lastSnapshot.myScore) {
    playSoundPreset('score')
  }
  lastSnapshot = {
    roomId: nextState.room_id || '',
    phase: nextState.phase || '',
    round: nextState.round || 0,
    myScore: myPlayer?.score || 0
  }
}

function connectSocket() {
  if (!session.active || !session.roomId || !session.playerId || !session.secret) {
    return
  }
  disconnectSocket()
  ws = new WebSocket(buildWsUrl())
  ws.onopen = () => {}
  ws.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data)
      if (payload.type === 'snapshot' && payload.state) {
        handleSnapshotEffects(payload.state)
        Object.assign(state, payload.state)
        selectedGame.value = payload.state.game || selectedGame.value
      } else if (payload.type === 'error') {
        ElMessage.error(payload.message || '房间操作失败')
      }
    } catch (error) {
      console.error('Failed to parse arcade message', error)
    }
  }
  ws.onclose = () => {
    const shouldReconnect = session.active
    ws = null
    if (shouldReconnect) {
      reconnectTimer = window.setTimeout(() => {
        connectSocket()
      }, 1500)
    }
  }
  ws.onerror = () => {
    ElMessage.error('多人连接断开了')
  }
}

function sendEvent(type) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    ElMessage.warning('连接还没准备好')
    return
  }
  ws.send(JSON.stringify({ type }))
}

function sendPick(index) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    return
  }
  ws.send(JSON.stringify({ type: 'pick_tile', index }))
}

function sendSequence(index) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    return
  }
  ws.send(JSON.stringify({ type: 'pick_sequence', index }))
}

function playerRoundText(player) {
  if (state.game === 'sequence') {
    if (player.round_state === 'finished') {
      return `${player.metric_ms} ms · +${player.round_award} 分`
    }
    if (player.round_state === 'timeout') {
      return `停在 ${player.progress}/${state.sequence_goal}`
    }
    return `进度 ${player.progress}/${state.sequence_goal || 0}`
  }

  const stateMap = {
    '': '等待动作',
    false_start: '抢跑 -1',
    reacted: `${player.metric_ms} ms`,
    hit: `+${player.round_award} 分`,
    miss: '未得分',
    correct: player.round_award > 0 ? `+${player.round_award} 分` : '命中目标',
    wrong: '点错了',
    perfect: `Perfect · +${player.round_award} 分`,
    good: `Good · +${player.round_award} 分`,
    ok: `OK · +${player.round_award} 分`
  }
  return stateMap[player.round_state] || '等待动作'
}

function formatRemaining(target) {
  if (!target) return '--'
  const diff = Math.max(0, target - nowTs.value)
  return `${(diff / 1000).toFixed(diff > 1000 ? 1 : 2)}s`
}

function progressBetween(now, start, end) {
  if (!start || !end || end <= start) {
    return 0
  }
  const value = ((now - start) / (end - start)) * 100
  return Math.max(0, Math.min(100, value))
}

function canPickTile(index) {
  if (state.game !== 'hunt' || state.phase !== 'live') {
    return false
  }
  const me = selfPlayer.value
  if (!me || me.round_state) {
    return false
  }
  return index >= 0
}

function canPickSequence(index) {
  if (state.game !== 'sequence' || state.phase !== 'live') {
    return false
  }
  const me = selfPlayer.value
  if (!me || me.round_state === 'finished' || me.round_state === 'timeout') {
    return false
  }
  return index >= 0
}

function cellClass(index) {
  const me = selfPlayer.value
  const isMine = me?.picked_index === index
  const isTarget = state.grid[index] === state.target
  return {
    mine: isMine,
    locked: !canPickTile(index),
    reveal: state.phase === 'result' || state.phase === 'finished',
    target: (state.phase === 'result' || state.phase === 'finished') && isTarget
  }
}

function revealCell(index) {
  return state.grid[index] || '·'
}

function sequenceCellClass(index, item) {
  const me = selfPlayer.value
  const value = Number(item)
  const progress = me?.progress || 0
  return {
    mine: me?.picked_index === index,
    next: value === progress + 1,
    done: value <= progress,
    locked: !canPickSequence(index)
  }
}

async function copyRoomCode() {
  try {
    await navigator.clipboard.writeText(state.room_id || session.roomId)
    ElMessage.success('房间码已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

async function copyInviteLink() {
  if (!inviteUrl.value) {
    ElMessage.warning('还没有可分享的房间')
    return
  }
  try {
    await navigator.clipboard.writeText(inviteUrl.value)
    ElMessage.success('邀请链接已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制链接')
  }
}

async function hydrateSharedRoom(roomCode, autoJoin = false) {
  const normalized = normalizeRoomCode(roomCode)
  if (!normalized) {
    return
  }
  joinRoomCode.value = normalized
  try {
    const res = await fetch(`/api/game/arcade/rooms/${encodeURIComponent(normalized)}`)
    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || '房间不存在')
    }
    selectedGame.value = data.game || selectedGame.value
    if (!session.active) {
      Object.assign(state, data)
    }
    if (autoJoin && nickname.value.trim() && !session.active) {
      await joinRoom()
    }
  } catch (error) {
    ElMessage.error(error.message || '无法打开分享房间')
  }
}

function isInputLikeTarget(target) {
  if (!(target instanceof HTMLElement)) {
    return false
  }
  const tagName = target.tagName
  return tagName === 'INPUT' || tagName === 'TEXTAREA' || target.isContentEditable
}

function onGlobalKeydown(event) {
  if (event.defaultPrevented || event.repeat) {
    return
  }
  if (event.key !== ' ' && event.key !== 'Enter' && event.key !== 'Spacebar') {
    return
  }
  if (isInputLikeTarget(event.target)) {
    return
  }
  if (event.target instanceof HTMLButtonElement) {
    return
  }

  let handled = false
  if (!session.active) {
    if (event.key === 'Enter') {
      handled = true
      if (joinRoomCode.value.trim()) {
        joinRoom()
      } else {
        createRoom(selectedGame.value)
      }
    }
  } else if (state.phase === 'lobby' && isHost.value) {
    sendEvent('start_match')
    handled = true
  } else if (state.game === 'reaction' && ['countdown', 'arming', 'live'].includes(state.phase)) {
    sendEvent('react')
    handled = true
  } else if (state.game === 'beat' && state.phase === 'live') {
    sendEvent('tap_beat')
    handled = true
  }

  if (handled) {
    event.preventDefault()
  }
}

function restoreSession() {
  const savedNickname = localStorage.getItem(NICKNAME_KEY)
  if (savedNickname && !nickname.value) {
    nickname.value = savedNickname
  }
  const saved = loadSession()
  if (!saved?.roomId || !saved?.playerId || !saved?.secret) {
    return
  }
  if (route.query.room && normalizeRoomCode(route.query.room) !== normalizeRoomCode(saved.roomId)) {
    return
  }
  session.active = true
  session.roomId = saved.roomId
  session.playerId = saved.playerId
  session.secret = saved.secret
  connectSocket()
}

onMounted(() => {
  clockTimer = window.setInterval(() => {
    nowTs.value = Date.now()
  }, 80)
  window.addEventListener('keydown', onGlobalKeydown)
  restoreSession()
  if (!session.active && route.query.room) {
    hydrateSharedRoom(route.query.room, !!nickname.value.trim())
  } else if (route.query.room) {
    joinRoomCode.value = normalizeRoomCode(route.query.room)
  }
})

onBeforeUnmount(() => {
  if (clockTimer) {
    clearInterval(clockTimer)
  }
  window.removeEventListener('keydown', onGlobalKeydown)
  disconnectSocket()
})

watch(
  () => route.query.room,
  (nextRoom, prevRoom) => {
    if (nextRoom === prevRoom) {
      return
    }
    if (nextRoom) {
      hydrateSharedRoom(nextRoom, false)
      return
    }
    if (!session.active) {
      joinRoomCode.value = ''
    }
  }
)
</script>

<style scoped>
.arcade-page {
  max-width: 1380px;
  margin: 0 auto;
  padding: 24px;
  color: #ecf2ff;
}

.panel {
  border-radius: 28px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background:
    radial-gradient(circle at top left, rgba(64, 213, 255, 0.18), transparent 32%),
    radial-gradient(circle at bottom right, rgba(255, 123, 77, 0.14), transparent 24%),
    linear-gradient(180deg, rgba(12, 18, 36, 0.92), rgba(9, 14, 28, 0.96));
  box-shadow: 0 24px 60px rgba(3, 8, 20, 0.45);
  backdrop-filter: blur(18px);
}

.hero-shell {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(330px, 0.85fr);
  gap: 20px;
  margin-bottom: 20px;
}

.hero-main,
.hero-side,
.scoreboard,
.arena-stage {
  padding: 28px;
}

.eyebrow,
.mini-label {
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  font-size: 0.76rem;
  color: #63d7ff;
  font-weight: 700;
}

.hero-main h1 {
  margin: 10px 0 0;
  font-size: clamp(2.6rem, 4vw, 4.2rem);
  line-height: 0.98;
  color: #fff;
}

.hero-copy {
  max-width: 720px;
  margin: 16px 0 0;
  line-height: 1.8;
  color: rgba(231, 238, 255, 0.8);
  font-size: 1rem;
}

.game-switches {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 24px;
}

.mode-card {
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 22px;
  padding: 18px;
  background: rgba(255, 255, 255, 0.04);
  color: #eef4ff;
  text-align: left;
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.mode-card:hover,
.mode-card.active {
  transform: translateY(-2px);
  border-color: rgba(99, 215, 255, 0.55);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.25);
}

.mode-top,
.mode-meta,
.room-head,
.room-actions,
.scoreboard-head,
.player-top,
.hunt-header,
.stage-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.mode-icon {
  font-size: 2rem;
}

.mode-tag {
  border-radius: 999px;
  padding: 6px 10px;
  background: rgba(99, 215, 255, 0.14);
  color: #7fe3ff;
  font-size: 0.78rem;
  font-weight: 700;
}

.mode-card h2 {
  margin: 16px 0 8px;
  font-size: 1.2rem;
}

.mode-card p,
.room-message,
.side-note p,
.player-info span,
.player-round,
.foot-card p,
.hunt-note p {
  margin: 0;
  color: rgba(231, 238, 255, 0.74);
  line-height: 1.65;
}

.mode-meta {
  margin-top: 14px;
  color: rgba(231, 238, 255, 0.56);
  font-size: 0.86rem;
}

.hero-side h3,
.scoreboard h3 {
  margin: 0 0 16px;
  font-size: 1.15rem;
  color: #fff;
}

.utility-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.utility-row span {
  color: rgba(231, 238, 255, 0.6);
  font-size: 0.84rem;
}

.toggle-sound {
  border: 1px solid rgba(99, 215, 255, 0.28);
  border-radius: 999px;
  padding: 10px 14px;
  background: rgba(99, 215, 255, 0.1);
  color: #9be9ff;
  font-weight: 700;
  cursor: pointer;
}

.toggle-sound.off {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.05);
  color: rgba(231, 238, 255, 0.68);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 14px;
}

.field span,
.room-label,
.target-label,
.foot-card span {
  font-size: 0.84rem;
  color: rgba(231, 238, 255, 0.62);
}

.field input {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 14px 16px;
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  font-size: 0.98rem;
  outline: none;
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.field input:focus {
  border-color: rgba(99, 215, 255, 0.6);
  box-shadow: 0 0 0 4px rgba(99, 215, 255, 0.12);
}

.create-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.create-btn,
.join-btn,
.primary-btn,
.ghost-btn {
  border: 0;
  border-radius: 16px;
  padding: 14px 16px;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.18s ease, opacity 0.18s ease, box-shadow 0.18s ease;
}

.create-btn:disabled,
.join-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.create-btn:hover,
.join-btn:hover,
.primary-btn:hover,
.ghost-btn:hover {
  transform: translateY(-1px);
}

.create-btn.reaction,
.primary-btn {
  background: linear-gradient(135deg, #00d4ff, #3269ff);
  color: #fff;
  box-shadow: 0 14px 30px rgba(0, 105, 255, 0.35);
}

.create-btn.beat {
  background: linear-gradient(135deg, #7c3aed, #22d3ee);
  color: #fff;
  box-shadow: 0 14px 30px rgba(124, 58, 237, 0.3);
}

.create-btn.sequence,
.create-btn.hunt,
.primary-btn.secondary {
  background: linear-gradient(135deg, #ff9f43, #ff5b5b);
  color: #fff;
  box-shadow: 0 14px 30px rgba(255, 91, 91, 0.28);
}

.join-box {
  margin-top: 18px;
}

.join-btn,
.ghost-btn {
  width: 100%;
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.ghost-btn.danger {
  background: rgba(255, 91, 91, 0.12);
  color: #ffb0b0;
}

.side-note {
  display: grid;
  gap: 8px;
  margin-top: 18px;
  padding: 16px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.04);
}

.room-label {
  display: block;
  margin-bottom: 4px;
}

.room-head strong {
  font-size: 1.9rem;
  color: #fff;
  letter-spacing: 0.08em;
}

.room-actions {
  flex-direction: column;
  width: 120px;
}

.status-pill {
  display: inline-flex;
  margin-top: 18px;
  border-radius: 999px;
  padding: 8px 14px;
  font-size: 0.82rem;
  font-weight: 700;
}

.status-pill.lobby,
.status-pill.finished {
  background: rgba(255, 255, 255, 0.08);
}

.status-pill.countdown,
.status-pill.arming {
  background: rgba(255, 184, 77, 0.18);
  color: #ffd389;
}

.status-pill.live {
  background: rgba(0, 232, 136, 0.16);
  color: #8cffc2;
}

.status-pill.result {
  background: rgba(99, 215, 255, 0.16);
  color: #8ee7ff;
}

.room-message {
  margin-top: 14px;
}

.host-actions {
  margin-top: 18px;
}

.arena-shell {
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  gap: 20px;
}

.sync-dot {
  border-radius: 999px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(231, 238, 255, 0.7);
  font-size: 0.8rem;
}

.sync-dot.online {
  color: #83ffc7;
}

.player-list {
  display: grid;
  gap: 12px;
  margin-top: 18px;
}

.player-card {
  border-radius: 20px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid transparent;
}

.player-card.self {
  border-color: rgba(99, 215, 255, 0.35);
  box-shadow: inset 0 0 0 1px rgba(99, 215, 255, 0.08);
}

.player-badge {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  flex-shrink: 0;
}

.player-info {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.player-info strong,
.player-score {
  color: #fff;
}

.player-score {
  font-size: 1.8rem;
  font-weight: 800;
}

.player-round {
  margin-top: 10px;
  font-size: 0.9rem;
}

.stage-head {
  align-items: flex-start;
}

.stage-head h2 {
  margin: 10px 0 0;
  font-size: 1.8rem;
  color: #fff;
}

.timer-block {
  min-width: 180px;
}

.time-value {
  text-align: right;
  font-size: 1.35rem;
  font-weight: 800;
  color: #fff;
}

.time-bar {
  overflow: hidden;
  height: 8px;
  border-radius: 999px;
  margin-top: 10px;
  background: rgba(255, 255, 255, 0.08);
}

.time-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #00d4ff, #5e7bff, #ff9f43);
}

.reaction-game,
.hunt-game,
.beat-game,
.sequence-game {
  margin-top: 28px;
}

.reaction-signal {
  position: relative;
  display: grid;
  place-items: center;
  min-height: 250px;
}

.signal-ring {
  width: 240px;
  height: 240px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: radial-gradient(circle, rgba(255, 255, 255, 0.08), transparent 68%);
}

.signal-core {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.signal-core span {
  color: rgba(231, 238, 255, 0.72);
  font-size: 0.82rem;
  letter-spacing: 0.24em;
}

.signal-core strong {
  font-size: 2.6rem;
  color: #fff;
}

.reaction-button {
  width: 100%;
  max-width: 420px;
  display: block;
  margin: 0 auto;
  border: 0;
  border-radius: 28px;
  padding: 24px 20px;
  font-size: 1.4rem;
  font-weight: 800;
  color: #fff;
  cursor: pointer;
  transition: transform 0.12s ease, box-shadow 0.18s ease, filter 0.18s ease;
}

.reaction-button:hover {
  transform: translateY(-2px);
}

.reaction-button.idle,
.reaction-button.finished,
.reaction-button.result {
  background: linear-gradient(135deg, rgba(255,255,255,0.08), rgba(255,255,255,0.06));
}

.reaction-button.countdown,
.reaction-button.tense {
  background: linear-gradient(135deg, #7c2d12, #dc2626);
  box-shadow: 0 16px 35px rgba(220, 38, 38, 0.28);
}

.reaction-button.go {
  background: linear-gradient(135deg, #00cc7a, #00d4ff);
  box-shadow: 0 18px 40px rgba(0, 212, 255, 0.28);
}

.reaction-signal.go .signal-ring {
  box-shadow: 0 0 50px rgba(0, 212, 255, 0.35), inset 0 0 30px rgba(0, 212, 255, 0.18);
}

.reaction-signal.tense .signal-ring,
.reaction-signal.countdown .signal-ring {
  box-shadow: 0 0 40px rgba(255, 91, 91, 0.25), inset 0 0 24px rgba(255, 91, 91, 0.16);
}

.reaction-tips {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
}

.reaction-tips span,
.foot-card {
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.05);
}

.reaction-tips span {
  padding: 10px 14px;
  color: rgba(231, 238, 255, 0.74);
  font-size: 0.88rem;
}

.beat-arena {
  padding: 18px 20px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.05);
}

.beat-track {
  position: relative;
  height: 24px;
  border-radius: 999px;
  background:
    linear-gradient(90deg, rgba(255, 120, 120, 0.8), rgba(99, 215, 255, 0.8) 45%, rgba(99, 215, 255, 0.95) 55%, rgba(255, 120, 120, 0.8));
  box-shadow: inset 0 0 18px rgba(0, 0, 0, 0.2);
}

.beat-target-zone {
  position: absolute;
  left: calc(50% - 8%);
  top: -4px;
  width: 16%;
  height: 32px;
  border-radius: 14px;
  border: 2px solid rgba(255, 255, 255, 0.95);
  background: rgba(255, 255, 255, 0.12);
}

.beat-marker {
  position: absolute;
  top: -10px;
  width: 12px;
  height: 44px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 0 18px rgba(255, 255, 255, 0.45);
  transform: translateX(-50%);
}

.beat-scale {
  display: flex;
  justify-content: space-between;
  margin-top: 14px;
  color: rgba(231, 238, 255, 0.6);
  font-size: 0.86rem;
}

.sequence-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
  padding: 18px 20px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.05);
}

.sequence-banner strong,
.sequence-progress {
  color: #fff;
}

.sequence-banner strong {
  display: block;
  margin-top: 6px;
  font-size: 2rem;
}

.sequence-progress {
  font-size: 1.2rem;
  font-weight: 800;
}

.sequence-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.sequence-cell {
  aspect-ratio: 1 / 1;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  font-size: clamp(1.35rem, 3vw, 2rem);
  font-weight: 800;
  cursor: pointer;
  transition: transform 0.12s ease, border-color 0.18s ease, background 0.18s ease, opacity 0.18s ease;
}

.sequence-cell:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: rgba(99, 215, 255, 0.45);
}

.sequence-cell.next {
  border-color: rgba(255, 184, 77, 0.7);
  box-shadow: inset 0 0 0 1px rgba(255, 184, 77, 0.32);
}

.sequence-cell.done {
  background: rgba(0, 232, 136, 0.16);
  border-color: rgba(0, 232, 136, 0.45);
}

.sequence-cell.mine {
  background: rgba(99, 215, 255, 0.18);
}

.sequence-cell.locked,
.hunt-cell.locked {
  opacity: 0.88;
  cursor: not-allowed;
}

.reaction-button:focus-visible,
.hunt-cell:focus-visible,
.sequence-cell:focus-visible,
.mode-card:focus-visible,
.create-btn:focus-visible,
.join-btn:focus-visible,
.ghost-btn:focus-visible,
.toggle-sound:focus-visible {
  outline: 2px solid rgba(99, 215, 255, 0.95);
  outline-offset: 3px;
}

.hunt-header {
  margin-bottom: 18px;
}

.hunt-target {
  display: flex;
  align-items: center;
  gap: 14px;
  border-radius: 24px;
  padding: 18px 20px;
  background: rgba(255, 255, 255, 0.05);
}

.hunt-target strong {
  font-size: 2.4rem;
  color: #fff;
}

.hunt-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.hunt-cell {
  aspect-ratio: 1 / 1;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  font-size: clamp(1.7rem, 3vw, 2.3rem);
  cursor: pointer;
  transition: transform 0.12s ease, border-color 0.18s ease, background 0.18s ease;
}

.hunt-cell:hover:not(:disabled) {
  transform: translateY(-1px) scale(1.01);
  border-color: rgba(99, 215, 255, 0.45);
}

.hunt-cell.mine {
  background: rgba(255, 184, 77, 0.18);
  border-color: rgba(255, 184, 77, 0.5);
}

.hunt-cell.reveal.target {
  background: rgba(0, 232, 136, 0.16);
  border-color: rgba(0, 232, 136, 0.55);
}

.hunt-cell.locked {
  cursor: not-allowed;
}

.stage-foot {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 24px;
}

.foot-card {
  padding: 16px 18px;
}

.foot-card p {
  margin-top: 8px;
}

@media (max-width: 1100px) {
  .hero-shell,
  .arena-shell {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 780px) {
  .arcade-page {
    padding: 16px;
  }

  .hero-main,
  .hero-side,
  .scoreboard,
  .arena-stage {
    padding: 20px;
  }

  .game-switches,
  .create-actions,
  .stage-foot {
    grid-template-columns: 1fr;
  }

  .hunt-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .sequence-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .room-head,
  .stage-head,
  .hunt-header,
  .sequence-banner,
  .utility-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .room-actions {
    width: 100%;
    flex-direction: row;
  }
}
</style>
