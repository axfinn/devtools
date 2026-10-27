<template>
  <div>
    <!-- 收起时的小气泡 / 宠物窗口（二选一） -->
    <button
      v-if="!visible"
      ref="launcherEl"
      class="pet-launcher"
      :style="{ left: launcherPos.x + 'px', top: launcherPos.y + 'px' }"
      @pointerdown="onLauncherPointerDown"
      @click="onLauncherClick"
      title="拖动可移动 · 点击召唤魔法宠物"
      aria-label="打开魔法宠物（可拖动）"
    >
      🐾
    </button>

    <div
      v-else
      ref="rootEl"
      class="pet-window"
      @pointerdown="onWindowPointerDown"
      :style="{
        left: position.x + 'px',
        top: position.y + 'px',
        width: size.w + 'px',
        height: size.h + 'px',
      }"
    >
      <div class="pet-drag-handle" title="按住拖动可移动宠物，避免遮挡按钮">
        <span class="handle-grip">⠿</span>
        <span class="handle-title">魔法宠物</span>
        <span class="handle-hint">拖动可移动</span>
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
          :title="'形态：' + f.label + '（同一模型，只改变配色）'"
          @click.stop="setForm(f.id)"
        >{{ f.icon }}</button>
      </div>

      <div class="pet-canvas">
        <!-- FINN-29 A6 · pet-widget 异步注入：元素未 upgrade 时浏览器把它当 HTMLUnknownElement，
             不报错；v-else 占位保证窗口尺寸仍可见，避免用户感觉"白屏" -->
        <pet-widget
          v-if="petReady"
          ref="petRef"
          :theme="themeId"
          :form="formId"
          :width="size.w"
          :height="size.h - 28"
          :voice="voiceOn ? '' : null"
          no-shell
        />
        <div v-else class="pet-placeholder">
          <div class="pet-placeholder-spinner" aria-hidden="true">🐾</div>
          <div class="pet-placeholder-text">宠物加载中…</div>
        </div>
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
import { ensurePetWidget } from '../utils/petWidgetLoader'

const STORAGE_KEY = 'pet-widget.global'
const POS_KEY = 'pet-widget.position'
const LAUNCHER_POS_KEY = 'pet-widget.launcherPos'
const THEME_KEY = 'pet-widget.theme'
const CHEER_INTERVAL_KEY = 'pet-widget.cheerInterval'
const CHEER_ENABLED_KEY = 'pet-widget.cheerEnabled'
// 全局语音总开关的持久化键（与 pet-widget.js 内保持一致）
const VOICE_ENABLED_KEY = 'pet-widget.enabled'
// pet-widget.js 在全局开关变化时广播的事件（用于图标与实际状态同步）
const VOICE_CHANGE_EVENT = 'pet-voice-change'

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

// 语音开关初值读持久化的全局开关（缺失 = 开启），保证 🔊/🔇 图标与实际发声状态一致
function readVoiceEnabled() {
  try {
    const v = localStorage.getItem(VOICE_ENABLED_KEY)
    return v === null ? true : v === '1'
  } catch { return true } // 隐私模式 / 无 localStorage ⇒ 默认开启
}
const voiceOn = ref(readVoiceEnabled())
const rootEl = ref(null)
const petRef = ref(null)
// FINN-29 A6 · pet-widget custom element 是否已 upgrade
const petReady = ref(false)

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
const launcherEl = ref(null)

const LAUNCHER_SIZE = 48

function isMobileViewport() {
  return typeof window !== 'undefined' && window.innerWidth < 768
}

// 移动端窗口按屏宽收缩（280×348 在 390px 屏上会压住整个输入区）
function responsiveSize() {
  if (isMobileViewport()) {
    const w = Math.min(232, Math.round(window.innerWidth * 0.6))
    return { w, h: Math.round(w * 1.1) + 28 }
  }
  return { w: 280, h: 320 + 28 }
}

const size = reactive(responsiveSize()) // canvas + 28 拖动条

function clamp(v, min, max) {
  return Math.max(min, Math.min(max, v))
}

// 缩略图标位置：桌面默认右下角 24px；移动端抬高，避开聊天输入框/发送按钮
function defaultLauncherPos() {
  const vw = window.innerWidth
  const vh = window.innerHeight
  if (vw < 768) {
    return {
      x: Math.max(8, vw - LAUNCHER_SIZE - 12),
      y: Math.max(80, vh - LAUNCHER_SIZE - 104),
    }
  }
  return { x: Math.max(8, vw - LAUNCHER_SIZE - 24), y: Math.max(8, vh - LAUNCHER_SIZE - 24) }
}

// 组件为异步加载（客户端），setup 期间即可读 window，避免首帧闪现错误位置
function loadLauncherPos() {
  const fallback = defaultLauncherPos()
  try {
    const raw = localStorage.getItem(LAUNCHER_POS_KEY)
    if (!raw) return fallback
    const p = JSON.parse(raw)
    if (typeof p.x !== 'number' || typeof p.y !== 'number') return fallback
    // 旧坐标可能来自桌面宽屏，落到手机上要钳制回可视区
    return {
      x: clamp(p.x, 0, Math.max(0, window.innerWidth - LAUNCHER_SIZE)),
      y: clamp(p.y, 0, Math.max(0, window.innerHeight - LAUNCHER_SIZE)),
    }
  } catch { return fallback }
}

const launcherPos = reactive(loadLauncherPos())

function saveLauncherPos() {
  try {
    localStorage.setItem(LAUNCHER_POS_KEY, JSON.stringify({ x: launcherPos.x, y: launcherPos.y }))
  } catch { /* ignore */ }
}

// 把窗口位置钳制回可视区（含移动端尺寸变化后的重算）
function clampWindowPos() {
  position.x = clamp(position.x, 0, Math.max(0, window.innerWidth - size.w))
  position.y = clamp(position.y, 0, Math.max(0, window.innerHeight - size.h))
}

function loadState() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v !== null) visible.value = v === '1'
    const pos = localStorage.getItem(POS_KEY)
    if (pos) Object.assign(position, JSON.parse(pos))
    const t = localStorage.getItem(THEME_KEY)
    if (t && THEME_ORDER.includes(t)) themeId.value = t
  } catch { /* ignore */ }

  // 默认右下角（移动端抬高，避免压住聊天输入框）
  if (position.x < 0 || position.y < 0) {
    const bottomGap = isMobileViewport() ? 104 : 24
    position.x = window.innerWidth - size.w - (isMobileViewport() ? 12 : 24)
    position.y = window.innerHeight - size.h - bottomGap
  }
  clampWindowPos()
}

function savePos() {
  try {
    localStorage.setItem(POS_KEY, JSON.stringify({ x: position.x, y: position.y }))
  } catch { /* ignore */ }
}

function show() {
  visible.value = true
  try { localStorage.setItem(STORAGE_KEY, '1') } catch { /* ignore */ }
  // FINN-29 A6 · 回头访客点 🐾 唤出时才真正加载 pet-widget.js
  if (!petReady.value) {
    ensurePetWidget()
      .then(() => { petReady.value = true })
      .catch(() => { /* ignore */ })
  }
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
  // 直接写持久化键：即使页面上还没有 pet-widget 实例，静音选择也不会丢
  try { localStorage.setItem(VOICE_ENABLED_KEY, voiceOn.value ? '1' : '0') } catch { /* ignore */ }
  // 通知所有 widget（voice 属性变化会驱动 pet-widget 写回全局开关）
  document.querySelectorAll('pet-widget').forEach((el) => {
    if (voiceOn.value) el.setAttribute('voice', '')
    else el.removeAttribute('voice')
  })
}

// pet-widget 广播全局开关变化 ⇒ 同步本组件的图标状态（页面上多个入口共用同一个开关）
function onVoiceChange(e) {
  if (e?.detail && typeof e.detail.enabled === 'boolean') voiceOn.value = e.detail.enabled
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

// ============ 拖动（指针事件，桌面/触屏通用；除按钮外整窗可拖） ============
let dragState = null
let pointerMoved = false

// 点按钮 / 表单按钮不启动拖动，让它们正常响应
function isInteractive(el) {
  return !!(el && el.closest && el.closest('button, input, textarea, select, .pet-forms, .pet-actions'))
}

function onWindowPointerDown(e) {
  if (isInteractive(e.target)) return
  e.preventDefault()
  pointerMoved = false
  dragState = {
    startX: e.clientX,
    startY: e.clientY,
    startPosX: position.x,
    startPosY: position.y,
  }
  window.addEventListener('pointermove', onPointerMove, { passive: false })
  window.addEventListener('pointerup', onPointerUp, { once: true })
  window.addEventListener('pointercancel', onPointerUp, { once: true })
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'grabbing'
}

function onPointerMove(e) {
  if (!dragState) return
  const dx = e.clientX - dragState.startX
  const dy = e.clientY - dragState.startY
  // 移动 < 4px 视为点击，不触发拖动（点宠物仍可触发其动作）
  if (!pointerMoved && Math.hypot(dx, dy) < 4) return
  pointerMoved = true
  e.preventDefault?.()
  const newX = Math.max(0, Math.min(window.innerWidth - size.w, dragState.startPosX + dx))
  const newY = Math.max(0, Math.min(window.innerHeight - size.h, dragState.startPosY + dy))
  position.x = newX
  position.y = newY
}

function onPointerUp() {
  const wasDragging = pointerMoved
  dragState = null
  pointerMoved = false
  window.removeEventListener('pointermove', onPointerMove)
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
  if (wasDragging) savePos()
}

// ============ 缩略图标（🐾）拖动：触摸/鼠标通用，点击仍是唤出 ============
let launcherDrag = null
let launcherMoved = false

function onLauncherPointerDown(e) {
  launcherMoved = false
  // 避免拖完后又触发 click 唤出窗口
  launcherDrag = {
    startX: e.clientX,
    startY: e.clientY,
    startPosX: launcherPos.x,
    startPosY: launcherPos.y,
  }
  try { e.currentTarget.setPointerCapture?.(e.pointerId) } catch { /* ignore */ }
  window.addEventListener('pointermove', onLauncherPointerMove, { passive: false })
  window.addEventListener('pointerup', onLauncherPointerUp, { once: true })
  window.addEventListener('pointercancel', onLauncherPointerUp, { once: true })
}

function onLauncherPointerMove(e) {
  if (!launcherDrag) return
  const dx = e.clientX - launcherDrag.startX
  const dy = e.clientY - launcherDrag.startY
  // 移动 < 6px 视为点击
  if (!launcherMoved && Math.hypot(dx, dy) < 6) return
  launcherMoved = true
  if (e.cancelable) e.preventDefault()
  launcherPos.x = clamp(launcherDrag.startPosX + dx, 0, Math.max(0, window.innerWidth - LAUNCHER_SIZE))
  launcherPos.y = clamp(launcherDrag.startPosY + dy, 0, Math.max(0, window.innerHeight - LAUNCHER_SIZE))
}

function onLauncherPointerUp() {
  window.removeEventListener('pointermove', onLauncherPointerMove)
  if (launcherMoved) saveLauncherPos()
  launcherDrag = null
}

function onLauncherClick() {
  if (launcherMoved) { launcherMoved = false; return } // 刚拖过，不当作点击
  show()
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
  // 订阅全局语音开关广播
  document.addEventListener(VOICE_CHANGE_EVENT, onVoiceChange)
  // FINN-29 A6 · 异步注入 pet-widget.js（首屏之后；用户首次可见窗口才真正用到）
  // 回头访客场景：localStorage['pet-widget.global'] === '0' 时 visible 保持 false，
  // 不调用 ensurePetWidget，用户点 🐾 时再 show() 也会自然加载（见 show 方法）。
  if (visible.value) {
    ensurePetWidget()
      .then(() => { petReady.value = true })
      .catch(() => { /* 失败保留占位，让用户重试 */ })
  }
})

onUnmounted(() => {
  stopCheerTimer()
  document.removeEventListener(VOICE_CHANGE_EVENT, onVoiceChange)
})

// 切主题时由 widget 内部自己播报（无 token 推送需要）
watch(themeId, () => { /* noop */ })

// 窗口尺寸变化时，确保宠物不超出（并按断点重算移动端尺寸）
function onResize() {
  const next = responsiveSize()
  if (next.w !== size.w || next.h !== size.h) {
    size.w = next.w
    size.h = next.h
  }
  clampWindowPos()
  launcherPos.x = clamp(launcherPos.x, 0, Math.max(0, window.innerWidth - LAUNCHER_SIZE))
  launcherPos.y = clamp(launcherPos.y, 0, Math.max(0, window.innerHeight - LAUNCHER_SIZE))
  if (position.x >= 0 && position.y >= 0) savePos()
  saveLauncherPos()
}
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<style scoped>
.pet-launcher {
  position: fixed;
  /* 位置由内联 left/top 控制（可拖动），不再用 bottom/right 固定死 */
  left: 0;
  top: 0;
  width: 48px;
  height: 48px;
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: linear-gradient(135deg, rgba(92, 242, 255, 0.25), rgba(184, 141, 255, 0.25));
  backdrop-filter: blur(12px);
  color: #fff;
  font-size: 22px;
  cursor: grab;
  z-index: 9999;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  transition: transform 200ms;
  touch-action: none;   /* 触屏拖动不触发页面滚动 */
  user-select: none;
  -webkit-user-select: none;
}
.pet-launcher:active {
  cursor: grabbing;
  transform: scale(0.96);
}
.pet-launcher:hover {
  transform: scale(1.1) rotate(-8deg);
}

.pet-window {
  position: fixed;
  z-index: 9998;
  display: flex;
  flex-direction: column;
  border-radius: 0;          /* 不要圆角了（透出来） */
  overflow: visible;         /* 不要裁剪（widget 透明背景要透出来） */
  background: transparent;    /* 透明！透出页面背景 */
  box-shadow: none;
  border: none;
  backdrop-filter: none;
  /* 关键：整窗默认点击穿透，否则透明画布会盖住下面的按钮（登录/发送等）无法点击。
     只有拖动条与工具按钮重新开启 pointer-events。 */
  pointer-events: none;
}
/* 拖动条是唯一的拖动入口（点击可穿透问题只影响画布，不影响这里） */
.pet-drag-handle {
  pointer-events: auto;
  touch-action: none;
}
.pet-actions,
.pet-forms {
  pointer-events: auto;
}
.pet-window:active {
  cursor: grabbing;
}

.pet-drag-handle {
  height: 26px;
  display: flex;
  align-items: center;
  padding: 0 10px;
  gap: 6px;
  background: rgba(0, 0, 0, 0.55);   /* 略深，确保文字可见 */
  border-radius: 12px;               /* 拖动条独立成胶囊 */
  cursor: grab;
  user-select: none;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.9);
  flex-shrink: 0;
  margin-bottom: 2px;
  width: 100%;
  box-sizing: border-box;
}
.pet-drag-handle:active {
  cursor: grabbing;
}
.handle-grip {
  font-size: 12px;
  letter-spacing: 1px;
  opacity: 0.8;
}
.handle-title {
  font-weight: 500;
}
.handle-hint {
  font-size: 10px;
  opacity: 0.55;
}
.handle-theme {
  font-size: 10px;
  font-family: monospace;
  opacity: 0.8;
  margin-left: auto;                 /* 主题信息推到最右 */
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
  top: 30px;
  display: flex;
  gap: 2px;
  padding: 3px 4px;
  background: rgba(0, 0, 0, 0.45);
  border-radius: 8px;
  backdrop-filter: blur(4px);
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
  pointer-events: none;      /* 透明画布不拦截页面点击 */
}
.pet-canvas >>> pet-widget {
  display: block;
  width: 100%;
  height: 100%;
}

/* FINN-29 A6 · pet-widget 未 upgrade 时的占位（首屏异步加载期间） */
.pet-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.65);
  font-size: 13px;
  background: linear-gradient(135deg, rgba(92, 242, 255, 0.08), rgba(184, 141, 255, 0.08));
  border-radius: 12px;
  user-select: none;
}
.pet-placeholder-spinner {
  font-size: 32px;
  animation: pet-placeholder-bounce 1.2s ease-in-out infinite;
}
.pet-placeholder-text {
  font-size: 12px;
  opacity: 0.85;
}
@keyframes pet-placeholder-bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}
</style>
