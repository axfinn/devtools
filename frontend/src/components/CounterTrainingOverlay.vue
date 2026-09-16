<template>
  <div class="training-layer" role="dialog" aria-modal="true" aria-label="训练中">
    <header class="training-top">
      <div class="training-count">
        <span class="training-count-label">本次训练</span>
        <strong class="training-count-value">{{ count.toLocaleString() }}</strong>
      </div>

      <div class="training-metrics">
        <span class="training-metric training-metric-goal">{{ goalLabel }}</span>
        <span class="training-metric">实际时长 {{ durationLabel }}</span>
        <span class="training-metric">有效时长 {{ activeLabel }}</span>
        <span class="training-metric">{{ speed > 0 ? `${speed}/分` : '未开始' }}</span>
        <span class="training-metric">步长已锁定 +{{ step }}</span>
      </div>

      <div v-if="goalValue > 0" class="training-progress" aria-hidden="true">
        <div class="training-progress-fill" :style="{ width: `${goalPercent}%` }"></div>
      </div>
    </header>

    <div v-if="degraded" class="training-notice">
      <button type="button" class="training-notice-btn" @click="emit('restore-audio')">
        音效已暂停 · 点我恢复
      </button>
    </div>

    <div v-if="bannerVisible" class="training-banner" role="status">
      {{ bannerText }}
    </div>

    <div class="training-stage">
      <button
        type="button"
        class="training-strike"
        :class="{ paused }"
        @pointerdown="onStrike"
      >
        <span class="training-strike-symbol">{{ figureSymbol }}</span>
        <span class="training-strike-name">{{ figureName }}</span>
        <span class="training-strike-hint">{{ paused ? '已暂停 · 仍在计数' : '轻触即计数' }}</span>
      </button>
    </div>

    <div v-if="confirming" class="training-confirm" role="alertdialog" aria-label="结束训练确认">
      <p class="training-confirm-text">结束并保存本次训练？</p>
      <div class="training-confirm-actions">
        <button type="button" class="training-btn" @click="confirming = false">继续训练</button>
        <button type="button" class="training-btn training-btn-danger" @click="confirmEnd">结束并保存</button>
      </div>
    </div>

    <footer class="training-bottom">
      <button
        type="button"
        class="training-btn training-btn-hold"
        :class="{ armed: pauseProgress > 0 }"
        @pointerdown="beginPress('pause', $event)"
        @pointerup="cancelPress($event)"
        @pointercancel="cancelPress($event)"
        @pointerleave="cancelPress($event)"
        @pointermove="onPressMove($event)"
        @contextmenu.prevent
      >
        <span class="training-btn-label">{{ paused ? '长按继续' : '长按暂停' }}</span>
        <span
          class="training-btn-progress"
          :style="{ transform: `scaleX(${pauseProgress})` }"
          aria-hidden="true"
        ></span>
      </button>

      <button
        type="button"
        class="training-btn training-btn-danger training-btn-hold"
        :class="{ armed: exitProgress > 0 }"
        @pointerdown="beginPress('exit', $event)"
        @pointerup="cancelPress($event)"
        @pointercancel="cancelPress($event)"
        @pointerleave="cancelPress($event)"
        @pointermove="onPressMove($event)"
        @contextmenu.prevent
      >
        <span class="training-btn-label">长按结束训练</span>
        <span
          class="training-btn-progress"
          :style="{ transform: `scaleX(${exitProgress})` }"
          aria-hidden="true"
        ></span>
      </button>
    </footer>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import {
  EXIT_LONG_PRESS_MS,
  LONG_PRESS_MS,
  LONG_PRESS_MOVE_TOLERANCE_PX,
  createLongPressGuard
} from '../utils/counterTapGuard'

const props = defineProps({
  count: { type: Number, default: 0 },
  goalValue: { type: Number, default: 0 },
  elapsedMs: { type: Number, default: 0 },
  activeMs: { type: Number, default: 0 },
  speed: { type: Number, default: 0 },
  step: { type: Number, default: 1 },
  paused: { type: Boolean, default: false },
  degraded: { type: Boolean, default: false },
  reached: { type: Boolean, default: false },
  /** 一次性提示文案（如「自动连点已关闭」）—— 训练层内不能用 EP Toast，只能自绘横幅 */
  notice: { type: String, default: '' },
  figureSymbol: { type: String, default: '木' },
  figureName: { type: String, default: '木鱼' }
})

const emit = defineEmits(['tap', 'pause', 'resume', 'end', 'restore-audio'])

/**
 * ⚠️ 训练层存活期间**禁止**调用任何 Element Plus 全局弹层 API
 * （消息提示 / 确认框 / 通知 / 对话框 / 抽屉）——
 * 训练层 z-index 10050 高于 EP 弹层（popup manager 从 2000 起），弹出来用户看不见，
 * 「按了没反应」在训练中比报错更糟。所有确认 UI 都在本组件内自绘。
 * L2 #42b 负向 grep 会按这几个 API 名断言本文件无命中。
 */

// ── 长按守卫（只用于破坏性操作；计数面一击即计，没有任何门限） ──
const pauseGuard = createLongPressGuard({ thresholdMs: LONG_PRESS_MS })
const exitGuard = createLongPressGuard({ thresholdMs: EXIT_LONG_PRESS_MS })

const press = ref(null)
const pauseProgress = ref(0)
const exitProgress = ref(0)
const confirming = ref(false)

let commitTimer = null
let progressTimer = null

const goalPercent = computed(() => {
  if (props.goalValue <= 0) return 0
  return Math.min(100, Math.round((props.count / props.goalValue) * 100))
})

const goalLabel = computed(() => {
  if (props.goalValue <= 0) return '不设目标'
  return `目标 ${props.count}/${props.goalValue}`
})

function formatDuration(ms) {
  const total = Math.max(0, Math.floor((Number(ms) || 0) / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
  }
  return `${minutes}:${String(seconds).padStart(2, '0')}`
}

const durationLabel = computed(() => formatDuration(props.elapsedMs))
const activeLabel = computed(() => formatDuration(props.activeMs))

const showReachedBanner = ref(props.reached)
const showNoticeBanner = ref(!!props.notice)
let bannerTimer = null
let noticeTimer = null

function scheduleBannerHide(clear) {
  window.clearTimeout(bannerTimer)
  bannerTimer = window.setTimeout(() => {
    clear()
  }, 3000)
}

watch(
  () => props.reached,
  (value) => {
    if (!value) {
      showReachedBanner.value = false
      return
    }
    showReachedBanner.value = true
    scheduleBannerHide(() => {
      showReachedBanner.value = false
    })
  }
)

watch(
  () => props.notice,
  (value) => {
    if (!value) {
      showNoticeBanner.value = false
      return
    }
    showNoticeBanner.value = true
    window.clearTimeout(noticeTimer)
    noticeTimer = window.setTimeout(() => {
      showNoticeBanner.value = false
    }, 3000)
  }
)

/** 达标横幅优先（它同时带震动反馈）；两者都不会阻塞敲击面 */
const bannerVisible = computed(() => showReachedBanner.value || showNoticeBanner.value)
const bannerText = computed(() =>
  showReachedBanner.value ? '目标达成 · 继续保持' : props.notice
)

/** 计数面：pointerdown 一击即计，不做位移 / 时长 / 主指针标志 / 去抖任何判定（方案 §D2① 硬口径） */
function onStrike() {
  emit('tap')
}

function guardFor(action) {
  return action === 'exit' ? exitGuard : pauseGuard
}

function clearCommitTimer() {
  if (commitTimer) {
    window.clearTimeout(commitTimer)
    commitTimer = null
  }
}

function stopProgressLoop() {
  if (progressTimer) {
    window.clearInterval(progressTimer)
    progressTimer = null
  }
}

function beginPress(action, event) {
  if (confirming.value) return
  if (event.pointerType === 'mouse' && event.button !== 0 && event.button != null) return

  const pointerId = event.pointerId
  const startedAt = performance.now()
  press.value = { action, pointerId, startedAt, x: event.clientX, y: event.clientY }

  guardFor(action).start(pointerId)

  const threshold = action === 'exit' ? EXIT_LONG_PRESS_MS : LONG_PRESS_MS

  clearCommitTimer()
  commitTimer = window.setTimeout(() => {
    commitTimer = null
    const current = press.value
    if (!current || current.pointerId !== pointerId) return
    if (guardFor(action).takeCommit(pointerId)) {
      press.value = null
      stopProgressLoop()
      pauseProgress.value = 0
      exitProgress.value = 0
      commitAction(action)
    }
  }, threshold)

  stopProgressLoop()
  progressTimer = window.setInterval(() => {
    const current = press.value
    if (!current) return
    const ratio = Math.min(1, (performance.now() - current.startedAt) / threshold)
    if (current.action === 'exit') {
      exitProgress.value = ratio
    } else {
      pauseProgress.value = ratio
    }
  }, 40)

  try {
    event.currentTarget?.setPointerCapture?.(pointerId)
  } catch (_) {
    // 指针捕获失败不影响长按
  }
}

function cancelPress(event) {
  const current = press.value
  if (current) {
    guardFor(current.action).cancel(current.pointerId)
  }
  if (event?.currentTarget && current && event.pointerId != null) {
    try {
      event.currentTarget.releasePointerCapture?.(event.pointerId)
    } catch (_) {
      // ignore
    }
  }
  commitTimer && clearCommitTimer()
  stopProgressLoop()
  press.value = null
  pauseProgress.value = 0
  exitProgress.value = 0
}

/** 手指滑出容差即取消长按 —— 只作用于破坏性操作 */
function onPressMove(event) {
  const current = press.value
  if (!current) return
  const dx = (event.clientX || 0) - current.x
  const dy = (event.clientY || 0) - current.y
  if (Math.hypot(dx, dy) > LONG_PRESS_MOVE_TOLERANCE_PX) {
    cancelPress(event)
  }
}

function commitAction(action) {
  if (action === 'pause') {
    emit(props.paused ? 'resume' : 'pause')
    return
  }
  // 结束训练：二次确认**层内自绘**（不能用 EP 确认框 —— 会被训练层自己盖住）
  confirming.value = true
}

function confirmEnd() {
  confirming.value = false
  emit('end')
}

onBeforeUnmount(() => {
  clearCommitTimer()
  stopProgressLoop()
  window.clearTimeout(bannerTimer)
  window.clearTimeout(noticeTimer)
})
</script>

<style scoped>
.training-layer {
  position: fixed;
  inset: 0;
  /* ⚠️ 10050 —— 必须高于全仓库所有常驻 fixed 浮层（.player-bar 10000 / .pet-launcher 9999
     / .pet-window 9998 / .cheer-toast 10000），否则「结束训练」（层内唯一退出路径）会被压住。
     代价：也高于 Element Plus 弹层（2000 起），故层内不得使用任何 EP 全局弹层 API。 */
  z-index: 10050;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: max(12px, env(safe-area-inset-top)) max(24px, env(safe-area-inset-right))
    calc(12px + env(safe-area-inset-bottom)) max(24px, env(safe-area-inset-left));
  background: var(--counter-bg, #0b1120);
  color: var(--counter-text, #e2e8f0);
  /* 层内不可能产生滚动 / 缩放手势 —— 也意味着层内不能放需要滚动的内容 */
  touch-action: none;
  overscroll-behavior: none;
  user-select: none;
  -webkit-user-select: none;
  -webkit-tap-highlight-color: transparent;
}

.training-top {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 18px;
  background: var(--counter-card, rgba(255, 255, 255, 0.06));
  border: 1px solid var(--counter-border, rgba(255, 255, 255, 0.12));
}

.training-count {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.training-count-label {
  color: var(--counter-muted, rgba(226, 232, 240, 0.7));
  font-size: 12px;
  letter-spacing: 0.08em;
}

.training-count-value {
  font-size: 40px;
  font-weight: 900;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.training-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.training-metric {
  padding: 5px 10px;
  border-radius: 999px;
  background: var(--counter-card-strong, rgba(255, 255, 255, 0.1));
  color: var(--counter-muted, rgba(226, 232, 240, 0.78));
  font-size: 12px;
  white-space: nowrap;
}

.training-metric-goal {
  color: var(--counter-accent, #38bdf8);
}

.training-progress {
  height: 6px;
  border-radius: 999px;
  background: var(--counter-track, rgba(255, 255, 255, 0.14));
  overflow: hidden;
}

.training-progress-fill {
  height: 100%;
  border-radius: 999px;
  background: var(--counter-accent, #38bdf8);
  transition: width 0.2s ease;
}

.training-notice {
  display: flex;
  justify-content: center;
}

.training-notice-btn {
  min-height: 48px;
  padding: 0 18px;
  border: 1px solid var(--counter-border, rgba(255, 255, 255, 0.18));
  border-radius: 999px;
  background: var(--counter-card-strong, rgba(255, 255, 255, 0.1));
  color: var(--counter-text, #e2e8f0);
  font-size: 13px;
  cursor: pointer;
}

.training-banner {
  padding: 10px 14px;
  border-radius: 14px;
  background: var(--counter-accent-soft, rgba(56, 189, 248, 0.18));
  color: var(--counter-accent, #38bdf8);
  font-size: 14px;
  font-weight: 800;
  text-align: center;
}

.training-stage {
  display: flex;
  flex: 1 1 auto;
  min-height: 200px;
}

.training-strike {
  position: relative;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  min-height: 200px;
  border: 1px solid var(--counter-border, rgba(255, 255, 255, 0.14));
  border-radius: 28px;
  background: linear-gradient(180deg, var(--counter-card-strong, rgba(255, 255, 255, 0.1)), transparent);
  color: inherit;
  cursor: pointer;
  touch-action: none;
}

.training-strike.paused {
  opacity: 0.72;
}

.training-strike-symbol {
  font-size: 68px;
  font-weight: 900;
  line-height: 1;
}

.training-strike-name {
  font-size: 18px;
  font-weight: 700;
}

.training-strike-hint {
  color: var(--counter-muted, rgba(226, 232, 240, 0.66));
  font-size: 12px;
}

.training-confirm {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border-radius: 18px;
  background: var(--counter-card-strong, rgba(255, 255, 255, 0.12));
  border: 1px solid var(--counter-border, rgba(255, 255, 255, 0.2));
}

.training-confirm-text {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  text-align: center;
}

.training-confirm-actions {
  display: flex;
  gap: 10px;
}

.training-confirm-actions .training-btn {
  flex: 1 1 0;
  min-height: 48px;
}

.training-bottom {
  display: flex;
  gap: 10px;
}

.training-btn {
  position: relative;
  overflow: hidden;
  flex: 1 1 0;
  min-width: 132px;
  min-height: 56px;
  padding: 0 14px;
  border: 1px solid var(--counter-border, rgba(255, 255, 255, 0.16));
  border-radius: 18px;
  background: var(--counter-card, rgba(255, 255, 255, 0.08));
  color: var(--counter-text, #e2e8f0);
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  touch-action: none;
}

.training-btn-danger {
  border-color: rgba(239, 68, 68, 0.5);
  color: #fca5a5;
}

.training-btn-label {
  position: relative;
  z-index: 1;
}

.training-btn-progress {
  position: absolute;
  inset: 0;
  transform-origin: left center;
  transform: scaleX(0);
  background: var(--counter-accent-soft, rgba(56, 189, 248, 0.3));
}

.training-btn-hold.armed .training-btn-progress {
  background: var(--counter-accent-soft, rgba(56, 189, 248, 0.42));
}
</style>
