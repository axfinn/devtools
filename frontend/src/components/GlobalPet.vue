<template>
  <div>
    <!-- 收起时的小气泡 / 宠物窗口（二选一） -->
    <button
      v-if="!visible"
      class="pet-launcher"
      @click="show"
      title="召唤魔法宠物"
      aria-label="打开魔法宠物"
    >
      🐾
    </button>

    <div
      v-else
      ref="rootEl"
      class="pet-window"
      :style="{
        left: position.x + 'px',
        top: position.y + 'px',
        width: size.w + 'px',
        height: size.h + 'px',
      }"
    >
      <div class="pet-drag-handle" @mousedown="onDragStart" @touchstart="onDragStart" title="拖动">
        <span class="handle-grip">⋮⋮</span>
        <span class="handle-title">魔法宠物</span>
        <span class="handle-theme" :style="{ color: theme.accent }">{{ theme.name }} · {{ formLabel }}</span>
      </div>

      <div class="pet-actions">
        <button
          class="pet-action-btn"
          :title="voiceOn ? '静音' : '取消静音'"
          @click="toggleVoice"
        >
          {{ voiceOn ? '🔊' : '🔇' }}
        </button>
        <button class="pet-action-btn" title="切主题" @click="cycleTheme">🎨</button>
        <button class="pet-action-btn cheer" title="给你打打气" @click="cheerNow">💪</button>
        <button class="pet-action-btn pet-close" title="隐藏（点右下角 🐾 唤回）" @click="hide">×</button>
      </div>

      <!-- 变身按钮行：放在顶部工具条下方 -->
      <div
        class="pet-forms"
        @mousedown.stop
        @touchstart.stop
      >
        <button
          v-for="f in FORMS"
          :key="f.id"
          :class="['form-btn', { active: formId === f.id }]"
          :title="'变身：' + f.label"
          @click.stop="setForm(f.id)"
        >{{ f.icon }}</button>
      </div>

      <div class="pet-canvas">
        <pet-widget
          ref="petRef"
          :theme="themeId"
          :form="formId"
          :width="size.w"
          :height="size.h - 28"
          :voice="voiceOn ? '' : null"
          no-shell
        />
      </div>
    </div>

    <!-- 打气 Toast（独立于 v-if/v-else） -->
    <transition name="cheer">
      <div v-if="cheerText" class="cheer-toast" :key="cheerKey">
        <div class="cheer-emoji">💪</div>
        <div class="cheer-content">{{ cheerText }}</div>
        <button class="cheer-close" @click="cheerText = ''">×</button>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { THEMES, THEME_ORDER } from '../constants/pet-themes'

const STORAGE_KEY = 'pet-widget.global'
const POS_KEY = 'pet-widget.position'
const THEME_KEY = 'pet-widget.theme'
const CHEER_INTERVAL_KEY = 'pet-widget.cheerInterval'
const CHEER_ENABLED_KEY = 'pet-widget.cheerEnabled'

const CHEERS = [
  '今天也是元气满满的一天！',
  '别忘了喝水 💧 写代码也要照顾自己',
  '每一个 bug 都是变强的机会',
  '你的代码已经很棒了，再坚持一下',
  '累了就休息，代码永远写不完',
  '想想下班后吃什么，就有动力了',
  '你已经比昨天的自己更厉害',
  '编译报错？深呼吸，再看一遍',
  '不要和别人比，只和昨天的自己比',
  '慢就是快，稳就是强',
  'debug 时的你已经比 99% 的人更耐心',
  '今天的 commit 都很 OK',
  '坚持下去，结果会来',
  '把大问题拆成小问题',
  '写代码是马拉松，不是短跑',
  '休息一下眼睛，看看窗外吧',
  '你正在做一件很酷的事',
  '代码 review 不是批评，是共同进步',
  '一行一行写，bug 一个一个修',
  '记得起身活动一下肩膀',
  '咖啡 ☕ 续杯不？',
  '再难的 bug 也有人修过',
  '相信自己，你就是下一个 10x 开发者',
  '写不出来就先睡一觉，醒来就有灵感',
  '把 TODO 写下来，别全靠脑子记',
]

const visible = ref(true)
const themeId = ref('prism')
const formId = ref('default')
const voiceOn = ref(true)
const rootEl = ref(null)
const petRef = ref(null)

const FORMS = [
  { id: 'default', icon: '🔵', label: '默认' },
  { id: 'cat',     icon: '🐱', label: '小猫' },
  { id: 'robot',   icon: '🤖', label: '机械' },
  { id: 'ghost',   icon: '👻', label: '幽灵' },
  { id: 'star',    icon: '⭐', label: '星灵' },
]
const formLabel = computed(() => FORMS.find(f => f.id === formId.value)?.label || '')
const FORM_KEY = 'pet-widget.form'
function setForm(id) {
  formId.value = id
  try { localStorage.setItem(FORM_KEY, id) } catch { /* ignore */ }
  document.querySelectorAll('pet-widget').forEach((el) => el.setForm?.(id))
}

// 打气
const cheerText = ref('')
const cheerKey = ref(0)
const cheerEnabled = ref(true)
let cheerTimer = null

const position = reactive({ x: -1, y: -1 })
const size = reactive({ w: 280, h: 320 + 28 }) // canvas + 28 拖动条

function loadState() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v !== null) visible.value = v === '1'
    const pos = localStorage.getItem(POS_KEY)
    if (pos) Object.assign(position, JSON.parse(pos))
    const t = localStorage.getItem(THEME_KEY)
    if (t && THEME_ORDER.includes(t)) themeId.value = t
  } catch { /* ignore */ }

  // 默认右下角
  if (position.x < 0 || position.y < 0) {
    position.x = window.innerWidth - size.w - 24
    position.y = window.innerHeight - size.h - 24
  }
}

function savePos() {
  try {
    localStorage.setItem(POS_KEY, JSON.stringify({ x: position.x, y: position.y }))
  } catch { /* ignore */ }
}

function show() {
  visible.value = true
  try { localStorage.setItem(STORAGE_KEY, '1') } catch { /* ignore */ }
}

function hide() {
  visible.value = false
  try { localStorage.setItem(STORAGE_KEY, '0') } catch { /* ignore */ }
}

function cycleTheme() {
  const idx = THEME_ORDER.indexOf(themeId.value)
  themeId.value = THEME_ORDER[(idx + 1) % THEME_ORDER.length]
  try { localStorage.setItem(THEME_KEY, themeId.value) } catch { /* ignore */ }
}

function toggleVoice() {
  voiceOn.value = !voiceOn.value
  // 通知所有 widget
  document.querySelectorAll('pet-widget').forEach((el) => {
    if (voiceOn.value) el.setAttribute('voice', '')
    else el.removeAttribute('voice')
  })
}

// ============ 打气 ============

function pickCheer() {
  return CHEERS[Math.floor(Math.random() * CHEERS.length)]
}

function cheerNow() {
  const text = pickCheer()
  cheerText.value = text
  cheerKey.value++
  // 8 秒后自动消失
  setTimeout(() => {
    if (cheerText.value === text) cheerText.value = ''
  }, 8000)
  // 播放预录 cheer 语音（如果有）
  if (voiceOn.value) {
    document.querySelectorAll('pet-widget').forEach((el) => {
      el.speak?.('cheer')
    })
  }
}

function startCheerTimer() {
  stopCheerTimer()
  if (!cheerEnabled.value) return
  // 12 分钟一次
  cheerTimer = window.setInterval(() => {
    cheerNow()
  }, 12 * 60 * 1000)
  // 首次进来 60 秒后先来一条暖场
  window.setTimeout(() => {
    if (cheerEnabled.value) cheerNow()
  }, 60 * 1000)
}

function stopCheerTimer() {
  if (cheerTimer) {
    window.clearInterval(cheerTimer)
    cheerTimer = null
  }
}

// ============ 拖动 ============
let dragState = null

function onDragStart(e) {
  e.preventDefault()
  const isTouch = e.type === 'touchstart'
  const point = isTouch ? e.touches[0] : e
  dragState = {
    startX: point.clientX,
    startY: point.clientY,
    startPosX: position.x,
    startPosY: position.y,
  }
  document.addEventListener(isTouch ? 'touchmove' : 'mousemove', onDragMove, { passive: false })
  document.addEventListener(isTouch ? 'touchend' : 'mouseup', onDragEnd, { once: true })
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'grabbing'
}

function onDragMove(e) {
  if (!dragState) return
  e.preventDefault?.()
  const point = e.touches ? e.touches[0] : e
  const dx = point.clientX - dragState.startX
  const dy = point.clientY - dragState.startY
  const newX = Math.max(0, Math.min(window.innerWidth - size.w, dragState.startPosX + dx))
  const newY = Math.max(0, Math.min(window.innerHeight - size.h, dragState.startPosY + dy))
  position.x = newX
  position.y = newY
}

function onDragEnd() {
  dragState = null
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('touchmove', onDragMove)
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
  savePos()
}

const theme = computed(() => THEMES[themeId.value])

// 凭证相关已删除 — 预录 MP3 不需要 token
function pushToken() {
  /* noop */
}

onMounted(() => {
  loadState()
  nextTick(() => {
    /* 静态音频无需推送 token */
  })
  // 启动打气定时器
  try {
    const enabled = localStorage.getItem(CHEER_ENABLED_KEY)
    if (enabled !== null) cheerEnabled.value = enabled === '1'
  } catch { /* ignore */ }
  startCheerTimer()
})

onUnmounted(() => {
  stopCheerTimer()
})

// 切主题时由 widget 内部自己播报（无 token 推送需要）
watch(themeId, () => { /* noop */ })

// 窗口尺寸变化时，确保宠物不超出
function onResize() {
  position.x = Math.max(0, Math.min(window.innerWidth - size.w, position.x))
  position.y = Math.max(0, Math.min(window.innerHeight - size.h, position.y))
  savePos()
}
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<style scoped>
.pet-launcher {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 48px;
  height: 48px;
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: linear-gradient(135deg, rgba(92, 242, 255, 0.25), rgba(184, 141, 255, 0.25));
  backdrop-filter: blur(12px);
  color: #fff;
  font-size: 22px;
  cursor: pointer;
  z-index: 9999;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  transition: transform 200ms;
}
.pet-launcher:hover {
  transform: scale(1.1) rotate(-8deg);
}

.pet-window {
  position: fixed;
  z-index: 9998;
  display: flex;
  flex-direction: column;
  border-radius: 14px;
  overflow: hidden;
  background: rgba(8, 10, 18, 0.7);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(8px);
}

.pet-drag-handle {
  height: 28px;
  display: flex;
  align-items: center;
  padding: 0 8px;
  gap: 6px;
  background: rgba(0, 0, 0, 0.3);
  cursor: grab;
  user-select: none;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
  flex-shrink: 0;
}
.pet-drag-handle:active {
  cursor: grabbing;
}
.handle-grip {
  font-size: 10px;
  letter-spacing: 2px;
}
.handle-title {
  flex: 1;
  font-weight: 500;
}
.handle-theme {
  font-size: 10px;
  font-family: monospace;
  opacity: 0.8;
}

.pet-actions {
  position: absolute;
  top: 2px;
  right: 4px;
  display: flex;
  gap: 2px;
  z-index: 10;
}
.pet-action-btn {
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.06);
  color: #fff;
  font-size: 11px;
  line-height: 22px;
  padding: 0;
  cursor: pointer;
  transition: background 150ms;
}
.pet-action-btn:hover {
  background: rgba(255, 255, 255, 0.15);
}
.pet-action-btn.pet-close {
  background: rgba(255, 80, 100, 0.18);
  color: rgba(255, 180, 200, 0.95);
}
.pet-action-btn.cheer {
  background: rgba(255, 200, 100, 0.18);
}

/* 变身按钮行：放在右上 actions 下方 */
.pet-forms {
  position: absolute;
  right: 6;
  top: 28px;
  display: flex;
  gap: 2px;
  padding: 2px 3px;
  background: rgba(8, 10, 18, 0.7);
  border-radius: 6px;
  backdrop-filter: blur(6px);
  z-index: 5;
  pointer-events: auto;
}
.form-btn {
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 11px;
  background: transparent;
  font-size: 12px;
  line-height: 22px;
  padding: 0;
  cursor: pointer;
  opacity: 0.55;
  transition: all 180ms;
}
.form-btn:hover { opacity: 0.85; transform: scale(1.1); }
.form-btn.active {
  opacity: 1;
  background: rgba(92, 242, 255, 0.25);
  box-shadow: 0 0 8px rgba(92, 242, 255, 0.4);
}

/* 打气 Toast */
.cheer-toast {
  position: fixed;
  bottom: 32px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 18px;
  background: linear-gradient(135deg, rgba(92, 242, 255, 0.18), rgba(184, 141, 255, 0.18));
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 14px;
  backdrop-filter: blur(20px);
  color: #fff;
  font-size: 14px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.4);
  z-index: 10000;
  max-width: 420px;
  animation: cheer-bounce 0.4s ease;
}
.cheer-emoji {
  font-size: 24px;
  animation: cheer-pulse 1.6s ease infinite;
}
.cheer-content {
  flex: 1;
  line-height: 1.5;
}
.cheer-close {
  background: rgba(255,255,255,0.1);
  border: none;
  color: #fff;
  font-size: 14px;
  width: 22px;
  height: 22px;
  border-radius: 11px;
  cursor: pointer;
}
.cheer-enter-active, .cheer-leave-active {
  transition: all 0.3s ease;
}
.cheer-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}
.cheer-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}
@keyframes cheer-bounce {
  0% { transform: translateX(-50%) scale(0.8); }
  60% { transform: translateX(-50%) scale(1.05); }
  100% { transform: translateX(-50%) scale(1); }
}
@keyframes cheer-pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.2) rotate(-8deg); }
}

.pet-canvas {
  flex: 1;
  position: relative;
  overflow: hidden;
  background: transparent;
}
.pet-canvas >>> pet-widget {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
