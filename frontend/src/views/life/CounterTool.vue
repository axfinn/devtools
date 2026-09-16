<template>
  <div
    class="counter-app"
    :class="[isDark ? 'is-dark' : 'is-light', `figure-${activeFigure.id}`, `scene-${activeScene.id}`]"
    :style="themeVars"
  >
    <div class="ambient-layer" aria-hidden="true"></div>

    <div class="counter-shell">
      <section class="hero-card glass-card">
        <div class="hero-top">
          <div>
            <p class="hero-kicker">{{ todayLabel }}</p>
            <h1 class="hero-title">敲击计数器</h1>
            <p class="hero-copy">
              {{ activeFigure.description }}，支持多形象、多音色和氛围主题，深浅模式自动跟随全站主题。
            </p>
          </div>
          <div class="hero-pills">
            <span class="hero-pill">{{ themeModeLabel }}</span>
            <span class="hero-pill">{{ activeScene.name }}</span>
          </div>
        </div>

        <div class="stats-grid">
          <article class="stat-card highlight">
            <span class="stat-label">今日次数</span>
            <strong class="stat-value main" :class="{ bump: isBumping }">
              {{ todayCount.toLocaleString() }}
            </strong>
          </article>
          <article class="stat-card">
            <span class="stat-label">累计总数</span>
            <strong class="stat-value">{{ grandTotal.toLocaleString() }}</strong>
          </article>
          <article class="stat-card">
            <span class="stat-label">连续达标</span>
            <strong class="stat-value">{{ streak }} 天</strong>
          </article>
          <article class="stat-card">
            <span class="stat-label">当前节奏</span>
            <strong class="stat-value">{{ speed > 0 ? `${speed}/分` : '未开始' }}</strong>
          </article>
        </div>

        <div class="goal-block">
          <div class="goal-meta">
            <div>
              <span class="section-caption">每日目标</span>
              <strong class="goal-title">
                {{ dailyGoal > 0 ? `${todayCount}/${dailyGoal}` : '未设置' }}
              </strong>
            </div>
            <div class="goal-status" :class="{ reached: dailyGoal > 0 && todayCount >= dailyGoal }">
              <span v-if="dailyGoal <= 0">自由计数</span>
              <span v-else-if="todayCount >= dailyGoal">已超出 {{ todayCount - dailyGoal }} 次</span>
              <span v-else>还差 {{ dailyGoal - todayCount }} 次</span>
            </div>
          </div>
          <div class="progress-track">
            <div class="progress-fill" :style="{ width: `${goalPercent}%` }"></div>
          </div>
        </div>
      </section>

      <section class="preset-grid">
        <article class="glass-card preset-panel">
          <div class="panel-head">
            <div>
              <h2>形象</h2>
              <p>切换主按钮造型和音色反馈</p>
            </div>
            <span class="panel-head-meta">{{ activeFigure.name }}</span>
          </div>

          <div class="preset-list">
            <button
              v-for="figure in figurePresets"
              :key="figure.id"
              type="button"
              class="preset-card"
              :class="{ active: figure.id === figureId }"
              @click="figureId = figure.id"
            >
              <span class="preset-symbol">{{ figure.symbol }}</span>
              <span class="preset-name">{{ figure.name }}</span>
              <span class="preset-desc">{{ figure.summary }}</span>
            </button>
          </div>
        </article>

        <article class="glass-card preset-panel">
          <div class="panel-head">
            <div>
              <h2>主题</h2>
              <p>氛围主题与深浅模式分离，自动跟随全站</p>
            </div>
            <span class="panel-head-meta">{{ activeScene.name }}</span>
          </div>

          <div class="preset-list scene-list">
            <button
              v-for="scene in scenePresets"
              :key="scene.id"
              type="button"
              class="preset-card scene-card"
              :class="{ active: scene.id === sceneId }"
              @click="sceneId = scene.id"
            >
              <span class="scene-swatches">
                <i :style="{ background: scene.swatches?.[0] || '' }"></i>
                <i :style="{ background: scene.swatches?.[1] || '' }"></i>
              </span>
              <span class="preset-name">{{ scene.name }}</span>
              <span class="preset-desc">{{ scene.summary }}</span>
            </button>
          </div>
        </article>
      </section>

      <section class="work-grid">
        <article class="glass-card tap-panel">
          <div class="panel-head">
            <div>
              <h2>点击区</h2>
              <p>{{ activeFigure.subtitle }}</p>
            </div>
            <span class="panel-head-meta">{{ autoMode ? `自动 ${autoInterval}ms` : `步长 +${step}` }}</span>
          </div>

          <div class="mobile-focus-strip">
            <div class="mobile-focus-count">
              <span class="mobile-focus-date">{{ todayLabel }}</span>
              <span class="mobile-focus-label">今日次数</span>
              <strong :class="{ bump: isBumping }">{{ todayCount.toLocaleString() }}</strong>
            </div>
            <div class="mobile-focus-meta">
              <span>{{ dailyGoal > 0 ? `目标 ${goalPercent}%` : '自由计数' }}</span>
              <span>{{ speed > 0 ? `${speed}/分` : '未开始' }}</span>
            </div>
          </div>

          <div class="step-strip">
            <button
              v-for="option in stepOptions"
              :key="option"
              type="button"
              class="step-chip"
              :class="{ active: step === option }"
              @click="step = option"
            >
              +{{ option }}
            </button>
          </div>

          <div v-if="audioDegraded" class="audio-notice">
            <button type="button" class="audio-notice-btn" @click="restoreAudio">
              音效已暂停 · 点我恢复
            </button>
          </div>

          <div class="tap-stage">
            <el-button
              class="side-round"
              :icon="Minus"
              circle
              size="large"
              :disabled="todayCount <= 0"
              @pointerdown.prevent="decrement"
            />

            <button
              type="button"
              class="figure-button"
              :class="{ bump: isBumping }"
              @pointerdown="handleFigureTap"
            >
              <span class="figure-orbit orbit-a"></span>
              <span class="figure-orbit orbit-b"></span>
              <span class="step-float">+{{ step }}</span>
              <span class="figure-shell" :class="`shell-${activeFigure.id}`">
                <span class="figure-mark">{{ activeFigure.symbol }}</span>
                <span class="figure-name">{{ activeFigure.name }}</span>
                <span class="figure-hit">{{ activeFigure.hitText }}</span>
              </span>
            </button>

            <el-button
              class="side-round side-plus"
              :icon="Plus"
              circle
              size="large"
              @pointerdown.prevent="increment"
            />
          </div>

          <div class="training-entry">
            <button type="button" class="training-start" @click="openStartPanel">
              {{ hasActiveSession ? '返回训练' : '开始训练' }}
            </button>
            <button type="button" class="training-records" @click="openSessionSummary()">
              训练记录
            </button>
          </div>

          <div class="mobile-main-actions">
            <el-button text size="small" :icon="RefreshLeft" :disabled="!lastActionDelta" @click="undoLast">
              撤销
            </el-button>
            <el-button
              text
              size="small"
              :type="autoMode ? 'primary' : 'default'"
              :icon="autoMode ? VideoPause : VideoPlay"
              @click="toggleAuto"
            >
              {{ autoMode ? '停止' : '自动' }}
            </el-button>
            <el-button text size="small" :icon="Setting" @click="showSettings = true">功能</el-button>
          </div>

          <div class="tap-hints">
            <span>空格 / ↑ 加 {{ step }}</span>
            <span>退格 / ↓ 减 {{ step }}</span>
            <span>Ctrl/Cmd + Z 撤销</span>
          </div>
        </article>

        <article class="glass-card insight-panel">
          <div class="panel-head">
            <div>
              <h2>本周节奏</h2>
              <p>周热力和快捷操作放在一起，单手也顺手</p>
            </div>
            <span class="panel-head-meta">{{ weekTotal }} 次</span>
          </div>

          <div class="week-heatmap">
            <div
              v-for="day in weekData"
              :key="day.key"
              class="heat-cell"
              :class="{ today: day.isToday, future: day.isFuture }"
              :style="day.isFuture ? undefined : { opacity: heatOpacity(day.count) }"
              :title="`${day.label} · ${day.count} 次`"
            >
              <span class="heat-day">{{ day.day }}</span>
              <span class="heat-count">{{ day.isFuture ? '-' : day.count }}</span>
            </div>
          </div>

          <div class="mini-stats">
            <div class="mini-stat">
              <span>周均</span>
              <strong>{{ weekAverage }}</strong>
            </div>
            <div class="mini-stat">
              <span>峰值速度</span>
              <strong>{{ bestSpeed > 0 ? `${bestSpeed}/分` : '--' }}</strong>
            </div>
            <div class="mini-stat">
              <span>当前反馈</span>
              <strong>{{ soundOn ? '音效开' : '静音' }} / {{ vibeOn ? '震动开' : '震动关' }}</strong>
            </div>
          </div>

          <div class="toolbar">
            <el-button text size="small" :icon="RefreshLeft" :disabled="!lastActionDelta" @click="undoLast">
              撤销
            </el-button>
            <el-button
              text
              size="small"
              :type="autoMode ? 'primary' : 'default'"
              :icon="autoMode ? VideoPause : VideoPlay"
              @click="toggleAuto"
            >
              {{ autoMode ? '停止连点' : '自动节奏' }}
            </el-button>
            <el-button text size="small" @click="soundOn = !soundOn">
              {{ soundOn ? '音效开' : '音效关' }}
            </el-button>
            <el-button text size="small" @click="vibeOn = !vibeOn">
              {{ vibeOn ? '震动开' : '震动关' }}
            </el-button>
            <el-button text size="small" :icon="Calendar" @click="openCalendarPanel">日历</el-button>
            <el-button text size="small" :icon="Clock" @click="showHistory = true">历史</el-button>
            <el-button text size="small" :icon="Setting" @click="showSettings = true">设置</el-button>
          </div>
        </article>
      </section>
    </div>

    <el-dialog
      v-model="goalReached"
      title="目标达成"
      width="320px"
      center
      :close-on-click-modal="false"
    >
      <div class="goal-dialog-content">
        <div class="goal-dialog-icon">{{ activeFigure.symbol }}</div>
        <p>今天已经完成 <strong>{{ dailyGoal }}</strong> 次。</p>
      </div>
      <template #footer>
        <el-button type="primary" @click="goalReached = false">继续计数</el-button>
      </template>
    </el-dialog>

    <div v-if="showSettings" class="panel-overlay" @click.self="showSettings = false">
      <section class="panel-card">
        <div class="panel-card-head">
          <h3>计数器设置</h3>
          <button type="button" class="panel-close" @click="showSettings = false">关闭</button>
        </div>

        <el-form label-position="top" size="small" class="settings-form">
          <el-form-item label="每日目标">
            <el-input-number v-model="dailyGoal" :min="0" :max="999999" :step="10" controls-position="right" />
            <span class="field-note">设为 0 表示不限制目标。</span>
          </el-form-item>

          <el-form-item label="自动节奏">
            <el-input-number v-model="autoInterval" :min="120" :max="5000" :step="20" controls-position="right" />
            <span class="field-note">用于自动连点，单位毫秒。</span>
          </el-form-item>

          <el-form-item label="反馈开关">
            <div class="switch-row">
              <div class="switch-item">
                <span>音效</span>
                <el-switch v-model="soundOn" />
              </div>
              <div class="switch-item">
                <span>震动</span>
                <el-switch v-model="vibeOn" />
              </div>
            </div>
          </el-form-item>

          <el-form-item label="音量">
            <el-slider v-model="volume" :min="0" :max="100" :step="5" show-input />
          </el-form-item>

          <el-form-item label="训练时保持屏幕常亮（耗电）">
            <el-switch :model-value="sessionWakeLockEnabled" @update:model-value="setWakeLockEnabled" />
            <span class="field-note">
              不支持或被系统拒绝的设备会自动忽略。训练数据只存本机，不支持多标签页同时使用（后保存的标签页会覆盖另一个）。
            </span>
          </el-form-item>

          <el-form-item label="当前形象">
            <div class="setting-pills">
              <button
                v-for="figure in figurePresets"
                :key="figure.id"
                type="button"
                class="setting-pill"
                :class="{ active: figure.id === figureId }"
                @click="figureId = figure.id"
              >
                {{ figure.name }}
              </button>
            </div>
          </el-form-item>

          <el-form-item label="当前主题">
            <div class="setting-pills">
              <button
                v-for="scene in scenePresets"
                :key="scene.id"
                type="button"
                class="setting-pill"
                :class="{ active: scene.id === sceneId }"
                @click="sceneId = scene.id"
              >
                {{ scene.name }}
              </button>
            </div>
          </el-form-item>

          <el-form-item label="查看记录">
            <div class="danger-actions">
              <el-button size="small" @click="showSettings = false; openCalendarPanel()">日历记录</el-button>
              <el-button size="small" @click="showSettings = false; showHistory = true">详细历史</el-button>
            </div>
          </el-form-item>

          <el-divider />

          <div class="danger-actions">
            <el-button @click="resetToday" size="small">重置今日</el-button>
            <el-button @click="clearHistory" size="small">清空历史</el-button>
            <el-button @click="resetAll" type="danger" size="small">清除全部数据</el-button>
          </div>
        </el-form>
      </section>
    </div>

    <div v-if="showCalendar" class="panel-overlay" @click.self="showCalendar = false">
      <section class="panel-card">
        <div class="panel-card-head">
          <div class="calendar-nav">
            <el-button circle size="small" @click="shiftCalendar(-1)">‹</el-button>
            <h3>{{ calYear }} 年 {{ calMonth }} 月</h3>
            <el-button circle size="small" @click="shiftCalendar(1)">›</el-button>
          </div>
          <button type="button" class="panel-close" @click="showCalendar = false">关闭</button>
        </div>

        <div class="calendar-grid">
          <span v-for="label in weekHeaders" :key="label" class="calendar-head">{{ label }}</span>
          <div
            v-for="cell in calCells"
            :key="cell.key"
            class="calendar-cell"
            :class="{
              dim: !cell.currentMonth,
              today: cell.isToday,
              filled: cell.count > 0,
              hit: cell.count > 0 && cell.goal > 0 && cell.count >= cell.goal
            }"
          >
            <span class="calendar-day">{{ cell.day || '' }}</span>
            <span v-if="cell.count > 0" class="calendar-count">{{ cell.count }}</span>
          </div>
        </div>
      </section>
    </div>

    <div v-if="showHistory" class="panel-overlay" @click.self="showHistory = false">
      <section class="panel-card">
        <div class="panel-card-head">
          <h3>历史记录</h3>
          <button type="button" class="panel-close" @click="showHistory = false">关闭</button>
        </div>

        <div v-if="dailyRecords.length > 0" class="history-list">
          <article v-for="record in dailyRecords" :key="record.date" class="history-item">
            <div>
              <strong class="history-date">{{ record.date }}</strong>
              <p class="history-day">{{ record.dayName }}</p>
            </div>
            <div class="history-meta">
              <span class="history-count">{{ record.count.toLocaleString() }} 次</span>
              <span class="history-speed">{{ record.peakSpeed > 0 ? `${record.peakSpeed}/分` : '--' }}</span>
              <el-tag
                v-if="record.dailyGoal > 0 && record.count >= record.dailyGoal"
                type="success"
                effect="plain"
                size="small"
              >
                达标
              </el-tag>
            </div>
          </article>
        </div>
        <div v-else class="empty-state">还没有历史记录。</div>
      </section>
    </div>

    <div v-if="showStartPanel" class="panel-overlay" @click.self="showStartPanel = false">
      <section class="panel-card">
        <div class="panel-card-head">
          <h3>开始训练</h3>
          <button type="button" class="panel-close" @click="showStartPanel = false">关闭</button>
        </div>

        <div class="start-block">
          <span class="section-caption">本次目标</span>
          <div class="setting-pills">
            <button
              v-for="preset in goalPresets"
              :key="`goal-${preset}`"
              type="button"
              class="setting-pill"
              :class="{ active: startGoal === preset }"
              @click="startGoal = preset"
            >
              {{ preset > 0 ? `${preset} 次` : '不设目标' }}
            </button>
          </div>
          <p class="field-note">
            每次训练单独设目标，与每日目标无关。
            <template v-if="sessionLastGoal > 0">上次：{{ sessionLastGoal }} 次。</template>
          </p>
        </div>

        <el-form label-position="top" size="small" class="settings-form">
          <el-form-item label="自定义目标（0 = 自由训练）">
            <el-input-number v-model="startGoal" :min="0" :max="99999" :step="10" controls-position="right" />
          </el-form-item>

          <el-form-item label="参考（只读）">
            <span class="field-note">{{ dailyGoalReference }}</span>
          </el-form-item>

          <el-form-item label="训练时保持屏幕常亮">
            <el-switch :model-value="sessionWakeLockEnabled" @update:model-value="setWakeLockEnabled" />
            <span class="field-note">不支持或被系统拒绝的设备会自动忽略（开启会略增耗电）。</span>
          </el-form-item>
        </el-form>

        <div class="danger-actions">
          <el-button type="primary" @click="beginTraining">进入训练</el-button>
          <el-button @click="showStartPanel = false">取消</el-button>
        </div>
      </section>
    </div>

    <div v-if="showSessionSummary" class="panel-overlay" @click.self="showSessionSummary = false">
      <section class="panel-card">
        <div class="panel-card-head">
          <h3>训练记录</h3>
          <button type="button" class="panel-close" @click="showSessionSummary = false">关闭</button>
        </div>

        <div v-if="sessionList.length > 0" class="session-list">
          <article
            v-for="item in sessionList"
            :key="item.id"
            class="session-item"
            :class="{ active: item.id === selectedSessionId }"
            @click="selectedSessionId = item.id"
          >
            <div class="session-item-head">
              <strong class="session-range">{{ formatSessionRange(item) }}</strong>
              <el-tag v-if="item.reached" type="success" effect="plain" size="small">达标</el-tag>
            </div>
            <div class="session-item-meta">
              <span>{{ item.count }} 次</span>
              <span>实际 {{ formatDuration(item.durationMs) }}</span>
              <span>有效 {{ formatDuration(item.activeMs) }}</span>
              <span>{{ endReasonLabel(item.endReason) }}</span>
            </div>
          </article>
        </div>
        <div v-else class="empty-state">还没有训练记录。训练结束后会自动保存。</div>

        <div v-if="selectedSession" class="session-timeline">
          <div class="session-timeline-head">
            <span>敲击时间轴</span>
            <span v-if="selectedTimeline.dropped > 0" class="session-timeline-note">
              仅显示最近 {{ sessionMaxTaps }} 次，更早的 {{ selectedTimeline.dropped }} 次未列出
            </span>
          </div>

          <div v-if="selectedTimeline.taps.length > 0" class="timeline-list">
            <div
              v-for="(entry, index) in selectedTimeline.taps"
              :key="`${entry.at}-${index}`"
              class="timeline-row"
            >
              <span class="timeline-time">{{ formatClock(entry.at) }}</span>
              <span class="timeline-delta" :class="{ minus: entry.delta < 0 }">
                {{ entry.delta > 0 ? `+${entry.delta}` : entry.delta }}
              </span>
              <span class="timeline-kind">{{ kindLabel(entry.kind) }}</span>
              <button
                v-if="canEditSelectedTimeline"
                type="button"
                class="timeline-delete"
                @click="deleteTimelineEntry(index)"
              >
                删除
              </button>
            </div>
          </div>
          <div v-else class="empty-state">这次训练没有可回溯的敲击记录（早于本版本，或已被截断）。</div>

          <p v-if="!canEditSelectedTimeline" class="field-note">
            跨天会话不可回溯修改（改动会落到当天计数上）。
          </p>
        </div>
      </section>
    </div>

    <!-- 训练层：挂在 .counter-app 内作为最后一个子节点（不用 Teleport，否则丢失 themeVars 变量继承） -->
    <CounterTrainingOverlay
      v-if="trainingMode"
      :count="sessionCount"
      :goal-value="sessionGoalValue"
      :elapsed-ms="sessionElapsedMs"
      :active-ms="sessionActiveMs"
      :speed="sessionSpeed"
      :step="step"
      :paused="sessionPaused"
      :degraded="audioDegraded"
      :reached="sessionReached"
      :notice="trainingNotice"
      :figure-symbol="activeFigure.symbol"
      :figure-name="activeFigure.name"
      @tap="handleFigureTap"
      @pause="pauseTraining"
      @resume="resumeTraining"
      @end="finishTraining('manual')"
      @restore-audio="restoreAudio"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Calendar,
  Clock,
  Minus,
  Plus,
  RefreshLeft,
  Setting,
  VideoPause,
  VideoPlay
} from '@element-plus/icons-vue'
import { useTheme } from '../../composables/useTheme'
import { createCounterAudio } from '../../composables/useCounterAudio'
import { createCounterSession, isSessionReached } from '../../composables/useCounterSession'
import { createCounterWakeLock } from '../../composables/useCounterWakeLock'
import CounterTrainingOverlay from '../../components/CounterTrainingOverlay.vue'
import {
  SHORTCUT_DECREMENT,
  SHORTCUT_INCREMENT,
  SHORTCUT_UNDO,
  resolveShortcut
} from '../../utils/counterTapGuard'

const STORAGE_KEY = 'counter_v4'
const LEGACY_KEY = 'counter_v3'
const MAX_HISTORY_DAYS = 365
const SPEED_WINDOW_MS = 15000
const stepOptions = [1, 3, 5, 10, 20]
const weekHeaders = ['一', '二', '三', '四', '五', '六', '日']

/** 首次训练的默认目标（jaxiu 2026-09-16 裁决：独立按次目标，首次默认 100；此后默认取「上次」） */
const DEFAULT_SESSION_GOAL = 100
/** 会话落盘节流（方案 §C2：禁止每次敲击写盘） */
const SESSION_FLUSH_INTERVAL_MS = 5000
/** 后台超时收尾、训练层仍存活时的一次性层内横幅（B2；训练层内禁止 EP 弹层，只能自绘横幅） */
const IDLE_LAYER_NOTICE = '上次会话已超时收尾 · 本次仅累计今日次数'

const figurePresets = [
  {
    id: 'mokugyo',
    name: '木鱼',
    symbol: '木',
    subtitle: '沉稳清脆，适合冥想和节律训练。',
    summary: '木质回响',
    description: '木鱼模式带一点沉稳敲感，适合稳定节奏的重复计数',
    hitText: '轻敲一下',
    sound: 'mokugyo',
    vibration: [16]
  },
  {
    id: 'bell',
    name: '铃铛',
    symbol: '铃',
    subtitle: '清亮回声，更轻快也更醒神。',
    summary: '明亮叮当',
    description: '铃铛模式更轻盈，适合提醒型或打卡型计数',
    hitText: '摇响一下',
    sound: 'bell',
    vibration: [12, 20]
  },
  {
    id: 'drum',
    name: '鼓点',
    symbol: '鼓',
    subtitle: '下潜鼓点，更适合运动和力量训练。',
    summary: '低频节拍',
    description: '鼓点模式更有冲击力，适合运动时快速连点',
    hitText: '敲出节拍',
    sound: 'drum',
    vibration: [24]
  },
  {
    id: 'chime',
    name: '星愿',
    symbol: '星',
    subtitle: '短促悦耳，适合轻松刷目标。',
    summary: '清脆铃音',
    description: '星愿模式更轻快，适合日常习惯和轻量目标追踪',
    hitText: '点亮一下',
    sound: 'chime',
    vibration: [10, 24, 10]
  }
]

const scenePresets = [
  {
    id: 'zen',
    name: '禅木',
    summary: '温润木色',
    swatches: ['#c68445', '#f4e1bf'],
    light: {
      bg: 'linear-gradient(180deg, #f8f1e6 0%, #f0e2c6 100%)',
      glowA: 'rgba(198, 132, 69, 0.22)',
      glowB: 'rgba(255, 255, 255, 0.8)',
      card: 'rgba(255, 250, 242, 0.8)',
      cardStrong: 'rgba(255, 249, 236, 0.92)',
      border: 'rgba(126, 92, 48, 0.16)',
      text: '#3b2a18',
      muted: '#735e46',
      subtle: '#a58d72',
      accent: '#c68445',
      accentSoft: 'rgba(198, 132, 69, 0.16)',
      accentContrast: '#fffaf0',
      success: '#2d8a57',
      shadow: '0 24px 60px rgba(122, 88, 42, 0.12)',
      track: 'rgba(89, 67, 33, 0.1)',
      ring: 'rgba(198, 132, 69, 0.24)',
      shell: 'linear-gradient(180deg, #f3dfb9 0%, #dca86c 100%)',
      shellSoft: 'rgba(247, 225, 188, 0.62)'
    },
    dark: {
      bg: 'linear-gradient(180deg, #161109 0%, #241b12 52%, #18120d 100%)',
      glowA: 'rgba(198, 132, 69, 0.3)',
      glowB: 'rgba(255, 214, 167, 0.08)',
      card: 'rgba(34, 24, 15, 0.82)',
      cardStrong: 'rgba(43, 31, 20, 0.96)',
      border: 'rgba(210, 159, 101, 0.16)',
      text: '#f5eadb',
      muted: '#cfbb9d',
      subtle: '#9d8466',
      accent: '#e09a57',
      accentSoft: 'rgba(224, 154, 87, 0.18)',
      accentContrast: '#1a120b',
      success: '#54c789',
      shadow: '0 28px 80px rgba(0, 0, 0, 0.28)',
      track: 'rgba(255, 236, 205, 0.08)',
      ring: 'rgba(224, 154, 87, 0.28)',
      shell: 'linear-gradient(180deg, #5a3d20 0%, #2f1f10 100%)',
      shellSoft: 'rgba(111, 73, 36, 0.34)'
    }
  },
  {
    id: 'ocean',
    name: '海雾',
    summary: '冷静蓝调',
    swatches: ['#3d8dff', '#a9d8ff'],
    light: {
      bg: 'linear-gradient(180deg, #eef7ff 0%, #dceefe 100%)',
      glowA: 'rgba(61, 141, 255, 0.2)',
      glowB: 'rgba(255, 255, 255, 0.78)',
      card: 'rgba(250, 253, 255, 0.78)',
      cardStrong: 'rgba(245, 250, 255, 0.92)',
      border: 'rgba(72, 127, 187, 0.16)',
      text: '#173355',
      muted: '#59718d',
      subtle: '#88a0bd',
      accent: '#3d8dff',
      accentSoft: 'rgba(61, 141, 255, 0.16)',
      accentContrast: '#f4f9ff',
      success: '#2b8d7c',
      shadow: '0 24px 60px rgba(47, 108, 165, 0.14)',
      track: 'rgba(30, 73, 120, 0.08)',
      ring: 'rgba(61, 141, 255, 0.24)',
      shell: 'linear-gradient(180deg, #c6e6ff 0%, #7bb7ff 100%)',
      shellSoft: 'rgba(180, 220, 255, 0.5)'
    },
    dark: {
      bg: 'linear-gradient(180deg, #0d1a2d 0%, #102544 52%, #0f1930 100%)',
      glowA: 'rgba(61, 141, 255, 0.24)',
      glowB: 'rgba(146, 198, 255, 0.08)',
      card: 'rgba(17, 31, 53, 0.82)',
      cardStrong: 'rgba(22, 38, 63, 0.96)',
      border: 'rgba(119, 171, 235, 0.16)',
      text: '#edf5ff',
      muted: '#bdd6f5',
      subtle: '#7da3d0',
      accent: '#67a8ff',
      accentSoft: 'rgba(103, 168, 255, 0.18)',
      accentContrast: '#0d1a2d',
      success: '#52cbb5',
      shadow: '0 28px 80px rgba(0, 0, 0, 0.26)',
      track: 'rgba(224, 238, 255, 0.08)',
      ring: 'rgba(103, 168, 255, 0.28)',
      shell: 'linear-gradient(180deg, #29538d 0%, #18345d 100%)',
      shellSoft: 'rgba(64, 112, 176, 0.34)'
    }
  },
  {
    id: 'sunset',
    name: '晚霞',
    summary: '柔暖橘红',
    swatches: ['#ff7f50', '#ffd07a'],
    light: {
      bg: 'linear-gradient(180deg, #fff5ed 0%, #ffe3cc 100%)',
      glowA: 'rgba(255, 127, 80, 0.22)',
      glowB: 'rgba(255, 252, 246, 0.76)',
      card: 'rgba(255, 250, 245, 0.8)',
      cardStrong: 'rgba(255, 247, 239, 0.92)',
      border: 'rgba(176, 102, 65, 0.16)',
      text: '#552918',
      muted: '#8a6455',
      subtle: '#be8d79',
      accent: '#ff7f50',
      accentSoft: 'rgba(255, 127, 80, 0.16)',
      accentContrast: '#fff7f3',
      success: '#3f9b63',
      shadow: '0 24px 60px rgba(177, 99, 65, 0.14)',
      track: 'rgba(111, 59, 39, 0.08)',
      ring: 'rgba(255, 127, 80, 0.24)',
      shell: 'linear-gradient(180deg, #ffd6aa 0%, #ff9669 100%)',
      shellSoft: 'rgba(255, 206, 155, 0.46)'
    },
    dark: {
      bg: 'linear-gradient(180deg, #22120f 0%, #351814 52%, #241310 100%)',
      glowA: 'rgba(255, 127, 80, 0.28)',
      glowB: 'rgba(255, 208, 154, 0.08)',
      card: 'rgba(51, 24, 20, 0.82)',
      cardStrong: 'rgba(63, 29, 24, 0.96)',
      border: 'rgba(238, 154, 116, 0.16)',
      text: '#fff0ea',
      muted: '#edc0af',
      subtle: '#b78674',
      accent: '#ff976d',
      accentSoft: 'rgba(255, 151, 109, 0.18)',
      accentContrast: '#2b1511',
      success: '#60cd86',
      shadow: '0 28px 80px rgba(0, 0, 0, 0.28)',
      track: 'rgba(255, 234, 223, 0.08)',
      ring: 'rgba(255, 151, 109, 0.28)',
      shell: 'linear-gradient(180deg, #7b3a28 0%, #472118 100%)',
      shellSoft: 'rgba(135, 76, 57, 0.34)'
    }
  },
  {
    id: 'forest',
    name: '青森',
    summary: '清爽绿调',
    swatches: ['#3aa575', '#9ce0b1'],
    light: {
      bg: 'linear-gradient(180deg, #eef9f1 0%, #dff3e4 100%)',
      glowA: 'rgba(58, 165, 117, 0.18)',
      glowB: 'rgba(255, 255, 255, 0.78)',
      card: 'rgba(248, 255, 250, 0.8)',
      cardStrong: 'rgba(245, 253, 247, 0.92)',
      border: 'rgba(75, 133, 101, 0.16)',
      text: '#183827',
      muted: '#5c7c69',
      subtle: '#8db39d',
      accent: '#3aa575',
      accentSoft: 'rgba(58, 165, 117, 0.16)',
      accentContrast: '#f5fff8',
      success: '#2a8b5e',
      shadow: '0 24px 60px rgba(62, 118, 90, 0.14)',
      track: 'rgba(37, 74, 54, 0.08)',
      ring: 'rgba(58, 165, 117, 0.24)',
      shell: 'linear-gradient(180deg, #c9edcf 0%, #74c992 100%)',
      shellSoft: 'rgba(177, 230, 192, 0.5)'
    },
    dark: {
      bg: 'linear-gradient(180deg, #0f1d17 0%, #123126 52%, #11231b 100%)',
      glowA: 'rgba(58, 165, 117, 0.24)',
      glowB: 'rgba(159, 235, 187, 0.08)',
      card: 'rgba(17, 45, 34, 0.82)',
      cardStrong: 'rgba(20, 55, 41, 0.96)',
      border: 'rgba(120, 215, 159, 0.16)',
      text: '#ecfff2',
      muted: '#bfebcd',
      subtle: '#77ae92',
      accent: '#52c28e',
      accentSoft: 'rgba(82, 194, 142, 0.18)',
      accentContrast: '#102118',
      success: '#68d99f',
      shadow: '0 28px 80px rgba(0, 0, 0, 0.28)',
      track: 'rgba(231, 255, 239, 0.08)',
      ring: 'rgba(82, 194, 142, 0.28)',
      shell: 'linear-gradient(180deg, #2e7756 0%, #174530 100%)',
      shellSoft: 'rgba(64, 141, 104, 0.34)'
    }
  }
]

const { currentTheme, themeMode } = useTheme()
const isDark = computed(() => currentTheme.value === 'dark')

const currentTime = ref(new Date())
const currentDate = ref(formatDateKey(currentTime.value))
const todayCount = ref(0)
const dailyGoal = ref(0)
const step = ref(1)
const autoMode = ref(false)
const autoInterval = ref(500)
const soundOn = ref(true)
const vibeOn = ref(true)
const volume = ref(75)
const figureId = ref('mokugyo')
const sceneId = ref('zen')
const speed = ref(0)
const bestSpeed = ref(0)
const lastActionDelta = ref(0)
const isBumping = ref(false)
const showSettings = ref(false)
const showCalendar = ref(false)
const showHistory = ref(false)
const goalReached = ref(false)
const dailyRecords = ref([])

const calYear = ref(currentTime.value.getFullYear())
const calMonth = ref(currentTime.value.getMonth() + 1)

// ── 训练（运动会话）相关状态 ────────────────────────────────────────────────
const showStartPanel = ref(false)
const showSessionSummary = ref(false)
const selectedSessionId = ref('')
const startGoal = ref(DEFAULT_SESSION_GOAL)
/** 训练层内的一次性提示（层内不能用 EP Toast，改由 CounterTrainingOverlay 自绘横幅） */
const trainingNotice = ref('')
/** 1s 时钟推进的时间戳，供「实际时长 / 有效时长」实时刷新（不额外起定时器） */
const nowTick = ref(Date.now())
/**
 * keep-alive 守卫（方案 §C4）：本组件在 keep-alive 下常驻，
 * 切走后 `isActive === false`，键盘 / 自动连点 / 1s 时钟都不得再改计数。
 */
const isActive = ref(true)

/**
 * 音频生命周期交给 useCounterAudio（方案 §B2 硬口径：`resume()` / `close()` /
 * `new AudioContext()` 只允许出现在它的 `unlock()` / `play()` / `rebuild()` / `dispose()` 内）。
 */
const counterAudio = createCounterAudio({
  ctxFactory: () => new (window.AudioContext || window.webkitAudioContext)(),
  createBuffers: (ctx) => {
    const map = new Map()
    const sr = ctx.sampleRate
    map.set('mokugyo', buildMokugyo(ctx, sr))
    map.set('bell', buildBell(ctx, sr))
    map.set('drum', buildDrum(ctx, sr))
    map.set('chime', buildChime(ctx, sr))
    return map
  }
})

/** 会话状态（独立键 counter_v4_training；本模块不读 dailyGoal，见方案 §C1.5） */
const session = createCounterSession()

/** 训练期屏幕常亮（jaxiu 裁决第 5 条）；不支持 / 被拒绝一律静默降级 */
const wakeLock = createCounterWakeLock()

let goalFired = false
let tapEvents = []
let autoTimer = null
let bumpTimer = null
let clockTimer = null
let listenersBound = false
let lastSessionFlushAt = 0
let lastSessionFlushCount = -1

const activeFigure = computed(() => {
  return figurePresets.find((item) => item.id === figureId.value) || figurePresets[0]
})

const activeScene = computed(() => {
  return scenePresets.find((item) => item.id === sceneId.value) || scenePresets[0]
})

const themeVars = computed(() => {
  const palette = activeScene.value[isDark.value ? 'dark' : 'light']
  return {
    '--counter-bg': palette.bg,
    '--counter-glow-a': palette.glowA,
    '--counter-glow-b': palette.glowB,
    '--counter-card': palette.card,
    '--counter-card-strong': palette.cardStrong,
    '--counter-border': palette.border,
    '--counter-text': palette.text,
    '--counter-muted': palette.muted,
    '--counter-subtle': palette.subtle,
    '--counter-accent': palette.accent,
    '--counter-accent-soft': palette.accentSoft,
    '--counter-accent-contrast': palette.accentContrast,
    '--counter-success': palette.success,
    '--counter-shadow': palette.shadow,
    '--counter-track': palette.track,
    '--counter-ring': palette.ring,
    '--counter-shell': palette.shell,
    '--counter-shell-soft': palette.shellSoft
  }
})

const themeModeLabel = computed(() => {
  if (themeMode.value === 'auto') return '深浅跟随全站'
  return themeMode.value === 'dark' ? '全站深色' : '全站浅色'
})

const todayLabel = computed(() => {
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'long',
    day: 'numeric',
    weekday: 'long'
  }).format(currentTime.value)
})

const recordMap = computed(() => {
  const map = new Map()
  dailyRecords.value.forEach((record) => {
    map.set(record.date, record)
  })
  return map
})

const historyTotal = computed(() => {
  return dailyRecords.value
    .filter((record) => record.date !== currentDate.value)
    .reduce((sum, record) => sum + record.count, 0)
})

const grandTotal = computed(() => {
  return historyTotal.value + todayCount.value
})

const goalPercent = computed(() => {
  if (dailyGoal.value <= 0) return 0
  return Math.min(100, Math.round((todayCount.value / dailyGoal.value) * 100))
})

// ── 训练 / 会话视图模型 ────────────────────────────────────────────────────
const trainingMode = computed(() => session.trainingMode.value)
const audioDegraded = computed(() => counterAudio.isDegraded.value)
const hasActiveSession = computed(() => session.active.value !== null)
const sessionLastGoal = computed(() => Number(session.prefs.value.lastGoalValue) || 0)
const sessionWakeLockEnabled = computed(() => session.prefs.value.wakeLockEnabled !== false)
const sessionList = computed(() => session.sessions.value)

/**
 * 层内计数基线（B2）：训练层打开 / 刷新恢复时对齐到当前会话的 `countAtStart`，
 * 无会话时对齐到当时的今日计数；跨零点后归 0。
 */
const layerBaselineCount = ref(0)

/**
 * 层内计数 = todayCount − 基线（撤销 / 重置自动同步，下限 0）。
 * 有会话时基线就是该会话的 `countAtStart`，与 `session.countOf()` 等价；
 * `active === null` 但训练层仍存活（后台超时被 idle 收尾、跨零点未续开 —— 方案 §D2⑦-b.3
 * 第 5 行已定这是允许状态）时 `countOf()` 恒返回 0，层内「本次训练」会一动不动 ——
 * 改用基线增量，敲击照常让层内计数增长（B2）。
 */
const sessionCount = computed(() => {
  if (session.active.value) return session.countOf(todayCount.value)
  return Math.max(0, todayCount.value - layerBaselineCount.value)
})
const sessionGoalValue = computed(() => Number(session.active.value?.goal?.value) || 0)

/** 实际时长 = 墙钟（主口径）；有效时长 = 扣掉暂停与后台（次要口径），两者同时展示 */
const sessionElapsedMs = computed(() => {
  const current = session.active.value
  if (!current) return 0
  return Math.max(0, nowTick.value - current.startedAt)
})

const sessionActiveMs = computed(() => {
  const current = session.active.value
  if (!current) return 0
  let total = Number(current.activeMs) || 0
  if (current.activeSinceMs != null) {
    total += Math.max(0, nowTick.value - current.activeSinceMs)
  }
  return Math.round(total)
})

/** 会话节奏（方案 §C1.6：count / (activeMs/60000)，与 15s 窗口的 speed 不同口径） */
const sessionSpeed = computed(() => {
  const minutes = sessionActiveMs.value / 60000
  if (minutes <= 0) return 0
  return Math.round(sessionCount.value / minutes)
})

const sessionPaused = computed(() => session.isPaused())

const sessionReached = computed(() =>
  isSessionReached(session.active.value?.goal, sessionCount.value, sessionActiveMs.value)
)

/** 本次可选目标：预设 + 「上次目标」一键选项；0 = 不设目标 */
const goalPresets = computed(() => {
  const presets = [100, 200, 500, 0]
  const last = sessionLastGoal.value
  if (last > 0 && !presets.includes(last)) presets.splice(3, 0, last)
  return presets
})

/** 只读参考行 —— 绝不参与会话目标的默认值或计算（jaxiu 裁决第 2 条） */
const dailyGoalReference = computed(() => {
  if (dailyGoal.value <= 0) return '未设每日目标'
  return `每日目标 ${dailyGoal.value} 次 · 今日已完成 ${todayCount.value} 次`
})

const selectedSession = computed(() => {
  const list = session.sessions.value
  if (!list.length) return null
  return list.find((item) => item.id === selectedSessionId.value) || list[0]
})

const selectedTimeline = computed(() => {
  const target = selectedSession.value
  if (!target) return { taps: [], total: 0, dropped: 0 }
  return session.tapsForSession(target.id) || { taps: [], total: 0, dropped: 0 }
})

/** 回溯修改只对「进行中的会话」或「今天的会话」开放（改的是今天的计数） */
const canEditSelectedTimeline = computed(() => {
  const target = selectedSession.value
  if (!target) return false
  if (session.active.value?.id === target.id) return true
  return target.startDate === currentDate.value
})

const sessionMaxTaps = computed(() => session.maxTaps)

const streak = computed(() => {
  if (dailyGoal.value <= 0) return 0
  const todayKey = formatDateKey(currentTime.value)
  let days = 0

  for (let offset = 0; offset < MAX_HISTORY_DAYS; offset += 1) {
    const targetDate = shiftDate(parseDateKey(todayKey), -offset)
    const key = formatDateKey(targetDate)
    const count = key === todayKey
      ? todayCount.value
      : (recordMap.value.get(key)?.count || 0)

    if (count >= dailyGoal.value) {
      days += 1
      continue
    }

    break
  }

  return days
})

const weekData = computed(() => {
  const today = currentTime.value
  const start = startOfWeek(today)

  return Array.from({ length: 7 }, (_, index) => {
    const date = shiftDate(start, index)
    const key = formatDateKey(date)
    const isToday = key === currentDate.value
    const isFuture = key > currentDate.value
    const count = isFuture
      ? 0
      : (isToday ? todayCount.value : (recordMap.value.get(key)?.count || 0))

    return {
      key,
      count,
      isToday,
      isFuture,
      day: weekHeaders[index],
      label: `${date.getMonth() + 1}/${date.getDate()}`
    }
  })
})

const maxWeekCount = computed(() => {
  return Math.max(1, ...weekData.value.map((day) => day.count || 0))
})

const weekTotal = computed(() => {
  return weekData.value
    .filter((day) => !day.isFuture)
    .reduce((sum, day) => sum + day.count, 0)
})

const weekAverage = computed(() => {
  const completedDays = weekData.value.filter((day) => !day.isFuture).length || 1
  return Math.round(weekTotal.value / completedDays)
})

const calCells = computed(() => {
  const firstDay = new Date(calYear.value, calMonth.value - 1, 1)
  const startOffset = (firstDay.getDay() + 6) % 7
  const daysInMonth = new Date(calYear.value, calMonth.value, 0).getDate()
  const cells = []

  for (let index = 0; index < startOffset; index += 1) {
    cells.push(blankCalendarCell(`blank-${index}`))
  }

  for (let day = 1; day <= daysInMonth; day += 1) {
    const key = `${calYear.value}-${String(calMonth.value).padStart(2, '0')}-${String(day).padStart(2, '0')}`
    const record = recordMap.value.get(key)
    const isToday = key === currentDate.value
    cells.push({
      key,
      day,
      currentMonth: true,
      isToday,
      count: isToday ? todayCount.value : (record?.count || 0),
      goal: isToday ? dailyGoal.value : (record?.dailyGoal || 0)
    })
  }

  while (cells.length % 7 !== 0) {
    cells.push(blankCalendarCell(`tail-${cells.length}`))
  }

  return cells
})

watch([autoMode, autoInterval], ([enabled]) => {
  if (enabled) {
    startAuto()
  } else {
    stopAuto()
  }
})

watch([dailyGoal, todayCount], () => {
  goalFired = dailyGoal.value > 0 && todayCount.value >= dailyGoal.value
})

watch(
  [todayCount, dailyGoal, step, autoMode, autoInterval, soundOn, vibeOn, volume, figureId, sceneId, bestSpeed],
  () => {
    saveState()
  }
)

watch(dailyRecords, () => {
  saveRecords()
}, { deep: true })

watch([calYear, calMonth], ([year, month]) => {
  if (month < 1) {
    calYear.value = year - 1
    calMonth.value = 12
  }

  if (month > 12) {
    calYear.value = year + 1
    calMonth.value = 1
  }
})

/**
 * 会话目标达成：训练层内用横幅 + 震动脉冲（不弹模态，避免挡住敲击面）。
 * 日间模式的「每日目标达成」模态保持不变（见 applyDelta）。
 */
watch(sessionReached, (reached) => {
  if (!reached || !session.trainingMode.value) return
  if (!vibeOn.value || !navigator.vibrate) return
  try {
    navigator.vibrate([30, 60, 30])
  } catch (_) {
    // 忽略震动失败
  }
})

function formatDateKey(date = new Date()) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function parseDateKey(dateKey) {
  const [year, month, day] = String(dateKey).split('-').map(Number)
  return new Date(year, (month || 1) - 1, day || 1)
}

function shiftDate(date, days) {
  const next = new Date(date)
  next.setDate(next.getDate() + days)
  return next
}

function startOfWeek(date) {
  const current = new Date(date)
  const weekday = current.getDay() || 7
  current.setHours(0, 0, 0, 0)
  current.setDate(current.getDate() - weekday + 1)
  return current
}

function getDayName(dateKey) {
  const date = parseDateKey(dateKey)
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]
}

function blankCalendarCell(key) {
  return {
    key,
    day: '',
    currentMonth: false,
    isToday: false,
    count: 0,
    goal: 0
  }
}

function heatOpacity(count) {
  if (!count) return 0.16
  return 0.2 + (count / maxWeekCount.value) * 0.8
}

function normalizeRecords(records) {
  const merged = new Map()

  if (!Array.isArray(records)) return []

  records.forEach((record) => {
    if (!record || typeof record.date !== 'string') return

    const key = record.date
    const count = Number(record.count) || 0
    const peakSpeed = Number(record.peakSpeed ?? record.maxSpeed) || 0
    const goal = Number(record.dailyGoal) || 0

    if (count <= 0 && peakSpeed <= 0) return

    if (!merged.has(key)) {
      merged.set(key, {
        date: key,
        dayName: record.dayName || getDayName(key),
        count,
        peakSpeed,
        dailyGoal: goal
      })
      return
    }

    const existing = merged.get(key)
    existing.count = Math.max(existing.count, count)
    existing.peakSpeed = Math.max(existing.peakSpeed, peakSpeed)
    existing.dailyGoal = existing.dailyGoal || goal
  })

  return [...merged.values()]
    .sort((left, right) => right.date.localeCompare(left.date))
    .slice(0, MAX_HISTORY_DAYS)
}

function readStorage(key) {
  try {
    const value = localStorage.getItem(key)
    return value ? JSON.parse(value) : null
  } catch (_) {
    return null
  }
}

function saveState() {
  const payload = {
    currentDate: currentDate.value,
    todayCount: todayCount.value,
    dailyGoal: dailyGoal.value,
    step: step.value,
    autoMode: false,
    autoInterval: autoInterval.value,
    soundOn: soundOn.value,
    vibeOn: vibeOn.value,
    volume: volume.value,
    figureId: figureId.value,
    sceneId: sceneId.value,
    bestSpeed: bestSpeed.value,
    goalFired
  }

  localStorage.setItem(`${STORAGE_KEY}_state`, JSON.stringify(payload))
}

function saveRecords() {
  localStorage.setItem(`${STORAGE_KEY}_records`, JSON.stringify(dailyRecords.value))
}

function persistAll() {
  saveState()
  saveRecords()
}

function loadAll() {
  const state = readStorage(`${STORAGE_KEY}_state`) || readStorage(`${LEGACY_KEY}_state`)
  const records = normalizeRecords(
    readStorage(`${STORAGE_KEY}_records`) || readStorage(`${LEGACY_KEY}_records`) || []
  )

  dailyRecords.value = records

  // 会话数据走独立键；旧数据不存在时等价于「无会话、无偏好」，功能与今天一致
  session.init()

  if (state && typeof state === 'object') {
    currentDate.value = state.currentDate || currentDate.value
    todayCount.value = Number(state.todayCount) || 0
    dailyGoal.value = Number(state.dailyGoal) || 0
    step.value = stepOptions.includes(Number(state.step)) ? Number(state.step) : 1
    autoMode.value = false
    autoInterval.value = clampNumber(state.autoInterval, 120, 5000, 500)
    soundOn.value = state.soundOn !== false
    vibeOn.value = state.vibeOn !== false
    volume.value = clampNumber(state.volume, 0, 100, 75)
    figureId.value = figurePresets.some((item) => item.id === state.figureId) ? state.figureId : 'mokugyo'
    sceneId.value = scenePresets.some((item) => item.id === state.sceneId) ? state.sceneId : 'zen'
    bestSpeed.value = Number(state.bestSpeed) || 0
    goalFired = Boolean(state.goalFired)
  }

  const todayKey = formatDateKey(currentTime.value)
  if (currentDate.value === todayKey) {
    dailyRecords.value = dailyRecords.value.filter((record) => record.date !== todayKey)
  }

  // B2：先按落盘恢复出来的会话对齐层内计数基线（= 该会话 `countAtStart`），再走回前台结算 ——
  // 这样「会话被 idle 收尾」后层内计数从会话已有进度继续涨，而不是跳回 0。
  alignLayerCountBaseline()

  // 冷启动等价于一次「回前台」：先按 idle 结算 → 再补跑跨零点 → 最后消费 pendingSplitFrom。
  // 直接调 checkDayRollover() 会把「昨天 23:50 离开、今天 07:50 打开」当成前台跨零点而假续开（方案 §C1.4）。
  session.handleForeground(Date.now(), {
    todayCount: todayCount.value,
    runRollover: checkDayRollover
  })
  noticeLayerWithoutSession()
  refreshSpeed()
  nowTick.value = Date.now()
  persistAll()
  // 刷新 / 冷启动时若会话仍 active（会直接进入训练层），补一次屏幕常亮申请。
  // Wake Lock 的重新申请不需要用户手势；失败照常静默降级（方案 §D4 第 4 条）。
  syncWakeLock()
}

function clampNumber(value, min, max, fallback) {
  const numeric = Number(value)
  if (Number.isNaN(numeric)) return fallback
  return Math.min(max, Math.max(min, numeric))
}

function archiveDay(dateKey, count, peakSpeed) {
  if (!dateKey || count <= 0) return

  const next = normalizeRecords([
    ...dailyRecords.value,
    {
      date: dateKey,
      dayName: getDayName(dateKey),
      count,
      peakSpeed,
      dailyGoal: dailyGoal.value
    }
  ])

  dailyRecords.value = next
}

function checkDayRollover() {
  const now = new Date()
  currentTime.value = now
  const todayKey = formatDateKey(now)

  if (currentDate.value === todayKey) return

  // ① 会话收尾 / 拆分必须发生在 todayCount 清零【之前】（方案 §C1.8）：
  //    此处 todayCount.value 仍是「昨天」的值，就是会话计数需要的快照；
  //    顺序写反（先清零再取快照）会让跨零点后的会话计数恒为 0。
  //    只在前台看到翻转时才自动续开 —— 1s 时钟在后台仍在跑（setInterval 不受后台影响）。
  const autoContinue = isActive.value && document.visibilityState === 'visible'
  session.handleRollover(now.getTime(), {
    todayCount: todayCount.value,
    autoContinue
  })

  // ② 现有日归档 + 清零，语义与改动前完全一致
  archiveDay(currentDate.value, todayCount.value, bestSpeed.value)
  currentDate.value = todayKey
  todayCount.value = 0
  speed.value = 0
  bestSpeed.value = 0
  lastActionDelta.value = 0
  tapEvents = []
  goalFired = false
  // B2：跨零点后层内计数基线归 0（今日计数已清零）—— 训练层不因跨零点关闭，
  // 此时层内「本次训练」= 新一天已累计的次数；续开的新段 `countAtStart` 同样是 0，两种口径一致。
  layerBaselineCount.value = 0
}

function bumpFigure() {
  isBumping.value = true
  window.clearTimeout(bumpTimer)
  bumpTimer = window.setTimeout(() => {
    isBumping.value = false
  }, 150)
}

/**
 * 主敲击面：**一击即计**（方案 §D2① 硬口径）。
 * 顺序必须是 `increment()` → `unlockAudio()`：先计数（同步、瞬时），再唤醒音频，且不 await。
 * 计数面不得施加位移 / 时长 / 主指针标志 / 去抖 / 长按任何门槛 —— 那会吞掉真实敲击。
 */
function handleFigureTap() {
  increment()
  unlockAudio()
}

function increment() {
  applyDelta(step.value, true, 'tap')
}

function decrement() {
  applyDelta(-step.value, false, 'minus')
}

function applyDelta(delta, shouldFeedback, kind = 'tap') {
  if (!isActive.value) return

  checkDayRollover()
  const nextValue = Math.max(0, todayCount.value + delta)
  const actualDelta = nextValue - todayCount.value

  if (!actualDelta) return

  todayCount.value = nextValue
  lastActionDelta.value = actualDelta

  // 会话敲击日志的唯一写入点（方案 §C1.7 边界 1）：写在 actualDelta !== 0 的分支里，
  // 因此「末条 delta === lastActionDelta」恒成立，undoLast() 弹出末条永远精确。
  session.recordTap({ at: Date.now(), delta: actualDelta, kind })

  if (actualDelta > 0 && shouldFeedback) {
    registerHit(actualDelta)
  }

  if (dailyGoal.value > 0 && todayCount.value >= dailyGoal.value && !goalFired) {
    goalFired = true
    // 训练层存活期间禁止任何 EP 全局弹层（会被 z-index 10050 盖住 → 用户以为「按了没反应」），
    // 训练中的达标反馈改为层内横幅 + 震动（见 CounterTrainingOverlay 与 sessionReached 监听）。
    if (!session.trainingMode.value) {
      nextTick(() => {
        goalReached.value = true
      })
    }
  }
}

function undoLast() {
  if (!lastActionDelta.value) return

  todayCount.value = Math.max(0, todayCount.value - lastActionDelta.value)
  lastActionDelta.value = 0
  goalFired = dailyGoal.value > 0 && todayCount.value >= dailyGoal.value
  // 会话侧同步弹出末条；session.count 是派生值（todayCount − countAtStart）会自动跟随
  session.popLastTap()
  ElMessage.success('已撤销上一步')
}

function registerHit(delta) {
  bumpFigure()
  playFigureSound()
  vibrate()
  tapEvents.push({ time: Date.now(), delta })
  refreshSpeed()
  session.noteSpeedPeak(speed.value)
}

function refreshSpeed() {
  const now = Date.now()
  tapEvents = tapEvents.filter((event) => now - event.time <= SPEED_WINDOW_MS)

  if (tapEvents.length < 2) {
    speed.value = 0
    return
  }

  const first = tapEvents[0].time
  const last = tapEvents[tapEvents.length - 1].time
  const total = tapEvents.reduce((sum, event) => sum + event.delta, 0)
  const spanMinutes = Math.max((last - first) / 60000, 0.03)
  speed.value = Math.max(0, Math.round(total / spanMinutes))
  bestSpeed.value = Math.max(bestSpeed.value, speed.value)
}

function toggleAuto() {
  autoMode.value = !autoMode.value
}

function startAuto() {
  stopAuto()
  autoTimer = window.setInterval(() => {
    applyDelta(step.value, true, 'auto')
  }, autoInterval.value)
}

function stopAuto() {
  if (autoTimer) {
    window.clearInterval(autoTimer)
    autoTimer = null
  }
}

/**
 * 音频解锁（方案 §B4）：必须在**用户手势内同步调用**，且不 await、不阻塞计数。
 * `counterAudio.unlock()` 内部把 ctxFactory / resume 都放在手势调用栈里，返回的 Promise 不予等待。
 */
function unlockAudio() {
  if (!soundOn.value) return
  counterAudio.unlock()
}

/**
 * 「音效已暂停 · 点我恢复」胶囊：脱离降级态的唯一入口。
 * ⚠️ 该按钮可出现在训练层内，而训练层存活期间禁止任何 EP 全局弹层 → 这里不做 Toast 反馈。
 */
function restoreAudio() {
  counterAudio.rebuild()
}

// Create a buffer from a generator function that fills sample data
function makeBuffer(ctx, sr, duration, fillFn) {
  const len = Math.ceil(sr * duration)
  const buf = ctx.createBuffer(1, len, sr)
  const data = buf.getChannelData(0)
  fillFn(data, sr, len)
  return buf
}

// Mix multiple signals into one buffer
function mixBuffers(buffers) {
  const maxLen = Math.max(...buffers.map(b => b.length))
  const result = new Float32Array(maxLen)
  for (const buf of buffers) {
    for (let i = 0; i < buf.length; i++) {
      result[i] += buf[i]
    }
  }
  // Normalize
  let peak = 0
  for (let i = 0; i < result.length; i++) {
    const abs = Math.abs(result[i])
    if (abs > peak) peak = abs
  }
  if (peak > 0.98) {
    const scale = 0.98 / peak
    for (let i = 0; i < result.length; i++) {
      result[i] *= scale
    }
  }
  return result
}

// ============ Sound generators ============

// Soft clipping for warmth
function softClip(x) {
  return Math.tanh(x * 1.2)
}

// Exponential decay envelope
function decayEnv(i, len, attackLen, peakLevel) {
  if (i < attackLen) return (i / attackLen) * peakLevel
  const t = (i - attackLen) / (len - attackLen)
  return peakLevel * Math.exp(-t * 6)
}

// Wooden fish: sharp resonant knock
function buildMokugyo(ctx, sr) {
  const dur = 0.09
  const len = Math.ceil(sr * dur)
  const buf = ctx.createBuffer(1, len, sr)
  const d = buf.getChannelData(0)
  const attackLen = Math.ceil(sr * 0.0003)
  const f1 = 560, f2 = 1240, f3 = 280

  for (let i = 0; i < len; i++) {
    const t = i / sr
    const env = decayEnv(i, len, attackLen, 1)
    // Fundamental + 2 harmonics with detune
    let s = Math.sin(2 * Math.PI * f1 * t) * 0.7
    s += Math.sin(2 * Math.PI * f2 * t) * 0.2 * Math.exp(-t * 40)
    s += Math.sin(2 * Math.PI * f3 * t) * 0.35 * Math.exp(-t * 18)
    // Very short noise attack
    if (t < 0.004) {
      s += (Math.random() * 2 - 1) * 0.25 * (1 - t / 0.004)
    }
    d[i] = softClip(s * env)
  }
  return buf
}

// Bell: bright metallic ping
function buildBell(ctx, sr) {
  const dur = 0.22
  const len = Math.ceil(sr * dur)
  const buf = ctx.createBuffer(1, len, sr)
  const d = buf.getChannelData(0)
  const attackLen = Math.ceil(sr * 0.0005)
  // Musical intervals: fundamental + major third + fifth
  const partials = [
    { f: 880, amp: 1, decay: 12 },
    { f: 1100, amp: 0.55, decay: 15 },
    { f: 1320, amp: 0.3, decay: 18 },
    { f: 660, amp: 0.4, decay: 10 },
    { f: 1760, amp: 0.12, decay: 24 }
  ]

  for (let i = 0; i < len; i++) {
    const t = i / sr
    const env = decayEnv(i, len, attackLen, 1)
    let s = 0
    for (const p of partials) {
      s += Math.sin(2 * Math.PI * p.f * t) * p.amp * Math.exp(-t * p.decay)
    }
    d[i] = softClip(s * env * 0.7)
  }
  return buf
}

// Drum: deep thump
function buildDrum(ctx, sr) {
  const dur = 0.16
  const len = Math.ceil(sr * dur)
  const buf = ctx.createBuffer(1, len, sr)
  const d = buf.getChannelData(0)
  const attackLen = Math.ceil(sr * 0.0002)

  for (let i = 0; i < len; i++) {
    const t = i / sr
    const env = decayEnv(i, len, attackLen, 1)
    // Frequency sweep: 140Hz → 45Hz
    const freq = 140 * Math.exp(-t * 14) + 45
    let s = Math.sin(2 * Math.PI * freq * t) * 0.8
    // Second harmonic
    s += Math.sin(2 * Math.PI * freq * 2.2 * t) * 0.25 * Math.exp(-t * 22)
    // Noise - drum head
    if (t < 0.03) {
      const noiseEnv = Math.exp(-t * 60)
      s += (Math.random() * 2 - 1) * 0.3 * noiseEnv
    }
    d[i] = softClip(s * env * 0.85)
  }
  return buf
}

// Chime: sparkly short tink
function buildChime(ctx, sr) {
  const dur = 0.16
  const len = Math.ceil(sr * dur)
  const buf = ctx.createBuffer(1, len, sr)
  const d = buf.getChannelData(0)
  const attackLen = Math.ceil(sr * 0.0003)
  // Major chord: root + third + fifth + octave
  const partials = [
    { f: 1047, amp: 1, decay: 16 },
    { f: 1319, amp: 0.5, decay: 20 },
    { f: 1568, amp: 0.3, decay: 24 },
    { f: 2093, amp: 0.1, decay: 30 }
  ]

  for (let i = 0; i < len; i++) {
    const t = i / sr
    const env = decayEnv(i, len, attackLen, 1)
    let s = 0
    for (const p of partials) {
      s += Math.sin(2 * Math.PI * p.f * t) * p.amp * Math.exp(-t * p.decay)
    }
    // Subtle sparkle noise
    if (t < 0.02) {
      s += (Math.random() * 2 - 1) * 0.12 * Math.exp(-t * 80)
    }
    d[i] = softClip(s * env * 0.65)
  }
  return buf
}

/**
 * 播放当前形象的音色。**永不抛异常**（`play()` 内部吞掉全部失败），
 * 音频失败绝不冒泡到 applyDelta 这条计数路径。
 */
function playFigureSound() {
  if (!soundOn.value || volume.value <= 0) return
  counterAudio.play(activeFigure.value.sound, volume.value / 100)
}

function vibrate() {
  if (!vibeOn.value || !navigator.vibrate) return

  try {
    navigator.vibrate(activeFigure.value.vibration)
  } catch (_) {
    // ignore vibration failures
  }
}

/**
 * 键盘入口的**唯一派发点**（方案 §D2⑦-b.1 / L2 #42c-i）。
 * 三个守卫的先后顺序是硬性的：`isActive` → 可编辑目标 → `resolveShortcut`。
 * 训练层存活时 `resolveShortcut` 对 `Cmd/Ctrl+Z` 与 `ArrowDown` / `Minus` / `Backspace`
 * 返回 `blocked` —— 本函数据此**不派发任何动作**（因而 `undoLast()` 里的 EP 弹层不可达），
 * 但仍 `preventDefault()`：`Backspace` 的浏览器历史后退 / 方向键的页面滚动不得漏出去。
 */
function onKeyDown(event) {
  // keep-alive 守卫（方案 §C4）：切走之后残留监听不得再改计数
  if (!isActive.value) return

  const target = event.target
  if (target && typeof target.closest === 'function' && target.closest('input, textarea, [contenteditable="true"]')) {
    return
  }

  const action = resolveShortcut(event, { trainingMode: session.trainingMode.value })
  if (!action) return

  event.preventDefault()

  if (action === SHORTCUT_INCREMENT) {
    increment()
    unlockAudio()
  } else if (action === SHORTCUT_DECREMENT) {
    decrement()
  } else if (action === SHORTCUT_UNDO) {
    undoLast()
  }
  // action === SHORTCUT_BLOCKED（训练层存活期）→ 到此为止：只吞默认行为，不派发任何动作
}

function shiftCalendar(delta) {
  calMonth.value += delta
}

function openCalendarPanel() {
  const date = parseDateKey(currentDate.value)
  calYear.value = date.getFullYear()
  calMonth.value = date.getMonth() + 1
  showCalendar.value = true
}

function resetToday() {
  // ⚠️ 顺序硬约束（方案 §D3 / §D2⑦）：必须先关训练层再弹确认框 ——
  // 否则确认框会被 z-index 10050 的训练层盖住，用户看不到任何反馈。
  leaveTrainingLayerForDialog()
  ElMessageBox.confirm('重置今日计数将结束当前训练会话，确认？', '确认', {
    type: 'warning',
    confirmButtonText: '确认',
    cancelButtonText: '取消'
  }).then(() => {
    stopAuto()
    autoMode.value = false
    // 先结会话再清零：会话记录保留真实次数（endReason='reset'）
    endSessionForReset()
    todayCount.value = 0
    speed.value = 0
    bestSpeed.value = 0
    lastActionDelta.value = 0
    tapEvents = []
    goalFired = false
    goalReached.value = false
    persistAll()
    ElMessage.success('今日数据已重置')
  }).catch(() => {})
}

function clearHistory() {
  ElMessageBox.confirm('清空全部历史记录（含训练会话），但保留今天的计数与进行中的训练？', '确认', {
    type: 'warning',
    confirmButtonText: '确认',
    cancelButtonText: '取消'
  }).then(() => {
    dailyRecords.value = []
    saveRecords()
    // 会话历史与 dailyRecords 同属「历史数据」；不动进行中的 active（方案 §D3）
    session.clearSessions()
    ElMessage.success('历史记录已清空')
  }).catch(() => {})
}

function resetAll() {
  leaveTrainingLayerForDialog()
  ElMessageBox.confirm('清除今日计数、全部历史记录和训练记录（含训练设置）？此操作不可恢复。', '确认', {
    type: 'warning',
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    confirmButtonClass: 'el-button--danger'
  }).then(() => {
    stopAuto()
    autoMode.value = false
    endSessionForReset()
    todayCount.value = 0
    speed.value = 0
    bestSpeed.value = 0
    lastActionDelta.value = 0
    tapEvents = []
    dailyRecords.value = []
    goalFired = false
    goalReached.value = false
    localStorage.removeItem(`${STORAGE_KEY}_state`)
    localStorage.removeItem(`${STORAGE_KEY}_records`)
    // 训练数据：内存态回到默认（prefs 一并重置，与「全部数据」语义一致）
    session.resetAll()
    // 显式抹掉训练键字面量（与 session.resetAll() 同一语义；保留字面量便于静态断言，方案 L2 #47）
    localStorage.removeItem('counter_v4_training')
    showSessionSummary.value = false
    ElMessage.success('计数器数据已清除')
  }).catch(() => {})
}

// ───────────────────────── 训练 / 运动会话 ─────────────────────────

/** 任何「先关训练层再弹确认框」的路径都必须走这里（方案 §D3 硬约束） */
function leaveTrainingLayerForDialog() {
  if (!session.trainingMode.value) return
  session.setTrainingMode(false)
  wakeLock.release()
}

/** 会话因重置而中断：endReason='reset'，先结会话再清零以便保留真实次数 */
function endSessionForReset() {
  if (session.active.value) {
    session.endSession({ todayCount: todayCount.value, reason: 'reset' })
  } else {
    session.setTrainingMode(false)
  }
  wakeLock.release()
}

function closeAllPanels() {
  showSettings.value = false
  showCalendar.value = false
  showHistory.value = false
  showSessionSummary.value = false
  showStartPanel.value = false
  goalReached.value = false
}

function openStartPanel() {
  // 会话还在（例如刷新后训练层被关掉）→ 直接回到训练层，不重开一段
  if (session.active.value) {
    trainingNotice.value = ''
    enterTrainingLayer()
    return
  }
  // 默认 = 上一次的目标；首次默认 100（jaxiu 裁决第 2 条：独立按次目标，绝不由 dailyGoal 派生）
  const last = sessionLastGoal.value
  startGoal.value = last > 0 ? last : DEFAULT_SESSION_GOAL
  closeAllPanels()
  showStartPanel.value = true
}

function setWakeLockEnabled(value) {
  session.updatePrefs({ wakeLockEnabled: value === true })
  syncWakeLock()
}

function beginTraining() {
  if (session.active.value) return

  // 自动连点不是真实敲击，进训练前强制关掉（方案 §D2③）；层内不能弹 Toast，改横幅提示
  const autoWasOn = autoMode.value
  stopAuto()
  autoMode.value = false
  trainingNotice.value = autoWasOn ? '自动连点已关闭 · 训练只计真实敲击' : ''

  session.startSession({
    todayCount: todayCount.value,
    goalValue: startGoal.value
  })
  nowTick.value = Date.now()
  enterTrainingLayer()
}

/**
 * 把层内计数基线对齐到「现在」（B2）：有会话 → 该会话的 `countAtStart`（无缝接续已有进度）；
 * 无会话 → 当前今日计数（层内从 0 起重新累计）。
 * 只在「打开训练层」与「刷新恢复到训练层」两个时机对齐 —— 会话被 idle 收尾时**不**重设
 * （基线本来就是那一段的 `countAtStart`，重设会让层内数字跳回 0）；跨零点由
 * `checkDayRollover()` 单独归零（那是真的换了一天）。
 */
function alignLayerCountBaseline() {
  layerBaselineCount.value = session.active.value
    ? session.active.value.countAtStart
    : todayCount.value
}

function enterTrainingLayer() {
  closeAllPanels()
  session.setTrainingMode(true)
  alignLayerCountBaseline()
  // ⚠️ 屏幕常亮的首次申请必须在用户手势内；失败静默降级，不阻塞训练
  if (session.prefs.value.wakeLockEnabled) wakeLock.request()
  syncWakeLock()
}

/**
 * B2：训练层仍存活但会话已不在（后台超时被 idle 收尾 / 跨零点未续开 / 刷新后只剩 prefs）——
 * 给一次性层内说明横幅；**不自动关闭训练层**（`active === null` 是方案 §D2⑦-b.3 第 5 行允许的状态）。
 * 训练层内禁止 EP 弹层，横幅只能自绘；同一文案重复赋值不会再次触发横幅（层内只 watch 文案变化），
 * 因此每个训练层最多出现一次。
 */
function noticeLayerWithoutSession() {
  if (!session.trainingMode.value || session.active.value) return
  trainingNotice.value = IDLE_LAYER_NOTICE
}

function pauseTraining() {
  if (session.pauseSession()) syncWakeLock()
}

function resumeTraining() {
  if (session.resumeSession()) syncWakeLock()
}

/** 结束训练：手动 / 空闲 / 跨零点 / 重置四种原因之一（方案 §C1.9） */
function finishTraining(reason = 'manual') {
  if (!session.active.value) {
    session.setTrainingMode(false)
    return
  }

  const finished = session.endSession({
    todayCount: todayCount.value,
    reason
  })

  wakeLock.release()
  nowTick.value = Date.now()
  trainingNotice.value = ''
  persistAll()

  // 训练层已随 trainingMode 关闭 → 在日间模式用现有面板样式展示总结 + 时间轴
  if (finished) {
    selectedSessionId.value = finished.id
    showSessionSummary.value = true
  }
}

function openSessionSummary(sessionId = '') {
  if (sessionId) {
    selectedSessionId.value = sessionId
  } else if (!session.sessions.value.some((item) => item.id === selectedSessionId.value)) {
    selectedSessionId.value = session.sessions.value?.[0]?.id || ''
  }
  showSessionSummary.value = true
}

/**
 * 时间轴回溯删除单条：`todayCount` 同步扣减（delta 带符号，minus/auto 条目为负）。
 * 与 `undoLast()` 是两条独立路径（方案 §D2⑥）：撤销只弹末条，这里可删任意一条。
 */
function deleteTimelineEntry(index) {
  const target = selectedSession.value
  if (!target) return

  const live = session.active.value?.id === target.id
  const removed = live
    ? session.removeTapAt(index)
    : session.removeSessionTap(target.id, index)
  if (!removed) return

  todayCount.value = Math.max(0, todayCount.value - (Number(removed.delta) || 0))
  // 删完之后「末条 delta === lastActionDelta」不再保证 → 关掉撤销入口，避免撤销金额错位
  lastActionDelta.value = 0
  goalFired = dailyGoal.value > 0 && todayCount.value >= dailyGoal.value
  persistAll()
}

/** 屏幕常亮：只在「训练层存活 且 会话进行中 且 未暂停」时持有 */
function syncWakeLock() {
  if (!session.trainingMode.value || !session.active.value || session.isPaused()) {
    wakeLock.release()
    return
  }
  if (session.prefs.value.wakeLockEnabled) wakeLock.reacquire()
}

/** 会话落盘节流（方案 §C2：禁止每次敲击写 counter_v4_training） */
function flushSessionThrottled() {
  if (!session.active.value) return
  const stamp = Date.now()
  const count = sessionCount.value
  if (stamp - lastSessionFlushAt < SESSION_FLUSH_INTERVAL_MS) return
  if (count === lastSessionFlushCount) return
  lastSessionFlushAt = stamp
  lastSessionFlushCount = count
  session.flush()
}

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

function formatClock(ms) {
  const date = new Date(Number(ms) || 0)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
}

function formatSessionRange(item) {
  if (!item) return ''
  const start = new Date(Number(item.startedAt) || 0)
  const day = `${start.getMonth() + 1}/${start.getDate()}`
  return `${day} ${formatClock(item.startedAt).slice(0, 5)} - ${formatClock(item.endedAt).slice(0, 5)}`
}

const END_REASON_LABELS = {
  manual: '手动结束',
  idle: '超时未归',
  midnight: '跨零点拆分',
  reset: '被重置打断'
}

function endReasonLabel(reason) {
  return END_REASON_LABELS[reason] || '手动结束'
}

const TAP_KIND_LABELS = {
  tap: '敲击',
  minus: '减号',
  auto: '自动',
  undo: '撤销'
}

function kindLabel(kind) {
  return TAP_KIND_LABELS[kind] || '敲击'
}

/**
 * 回前台 / bfcache 恢复。
 * 音频侧**只置 needsUnlock**（方案 §B2 硬口径：绝不在 visibilitychange / pageshow 里 resume / close / new ctx），
 * 真正的恢复只发生在下一次用户手势的 `unlock()` / `play()` 里。
 * 会话侧走严格三步：先结算 idle → 再补跨零点 → 最后消费 pendingSplitFrom（由模块内部保证顺序）。
 */
function handleReturnToForeground() {
  if (!isActive.value) return

  counterAudio.handleForeground()
  session.handleForeground(Date.now(), {
    todayCount: todayCount.value,
    runRollover: checkDayRollover
  })
  noticeLayerWithoutSession()
  nowTick.value = Date.now()
  syncWakeLock()
  persistAll()
}

function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') {
    // 只结算活跃段 + 落盘；不结束会话、不关闭 AudioContext（下次前台还要用）
    session.handleBackground(Date.now())
    persistAll()
    return
  }
  handleReturnToForeground()
}

/** bfcache 恢复：休眠期间可能已跨零点，补跑一次 1s 时钟逻辑（幂等） */
function handlePageShow() {
  if (!isActive.value) return
  handleReturnToForeground()
  counterAudio.handleForegroundClock(() => {
    if (!isActive.value) return
    checkDayRollover()
    refreshSpeed()
    nowTick.value = Date.now()
  })
}

function handlePageHide() {
  session.handleBackground(Date.now())
  persistAll()
}

/**
 * keep-alive 下 `currentViewKey` 变化会重建实例，而旧实例的 document 监听不会自动摘除
 * （方案 §C4）：注册 / 摘除都做成幂等的，`onMounted` + `onActivated` 各注册一次。
 */
function addGlobalListeners() {
  if (listenersBound) return
  listenersBound = true
  document.addEventListener('keydown', onKeyDown)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('beforeunload', handlePageHide)
  window.addEventListener('pagehide', handlePageHide)
  window.addEventListener('pageshow', handlePageShow)
}

function removeGlobalListeners() {
  if (!listenersBound) return
  listenersBound = false
  document.removeEventListener('keydown', onKeyDown)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('beforeunload', handlePageHide)
  window.removeEventListener('pagehide', handlePageHide)
  window.removeEventListener('pageshow', handlePageShow)
}

onMounted(() => {
  isActive.value = true
  addGlobalListeners()
  loadAll()
  nowTick.value = Date.now()
  clockTimer = window.setInterval(() => {
    if (!isActive.value) return
    checkDayRollover()
    refreshSpeed()
    nowTick.value = Date.now()
    flushSessionThrottled()
  }, 1000)
})

onActivated(() => {
  isActive.value = true
  addGlobalListeners()
  // keep-alive 切回 = 一次「回前台」（B3）：必须补跑会话结算。`onDeactivated` 里的
  // `handleBackground()` 会结算活跃段并把 `activeSinceMs` 置 null，而只有 `handleForeground()`
  // 的 resumed 分支会把它重新打开 —— 不补这一步，`sessionActiveMs` 在离开过一次路由之后**永久冻结**，
  // 落盘的 `activeMs` 系统性偏小。`handleReturnToForeground()` 内部已含
  // `checkDayRollover` / `nowTick` / `syncWakeLock` / `persistAll`。
  // 音频恢复仍只走用户手势路径（`handleForeground()` 只置 needsUnlock，绝不 resume）。
  handleReturnToForeground()
  refreshSpeed()
})

onDeactivated(() => {
  isActive.value = false
  stopAuto()
  // 路由离开与切后台同一套规则（方案 §C1.4）：不结束会话，只结算活跃段并记下时间点
  session.handleBackground(Date.now())
  removeGlobalListeners()
  wakeLock.release()
  persistAll()
})

onBeforeUnmount(() => {
  stopAuto()
  window.clearTimeout(bumpTimer)
  window.clearInterval(clockTimer)
  removeGlobalListeners()
  wakeLock.release()
  persistAll()
  counterAudio.dispose()
})
</script>

<style scoped>
.counter-app {
  position: relative;
  /* 方案 §A2：桌面下父容器（.main-content）显式定高，百分比可依赖；
     移动端父容器高度由内容决定（indefinite）→ 百分比 min-height 按 CSS 2.1 §10.7 当 0 处理，
     所以那个断点里改用 calc(100dvh - var(--ct-chrome-top)) 的显式算式。 */
  min-height: 100%;
  padding: 24px 16px 40px;
  background: var(--counter-bg);
  color: var(--counter-text);
  /* 全断点保留：只为裁剪 .ambient-layer 向右出血 60px 的光斑（方案 §A2(5)，移动端同样生效） */
  overflow: hidden;
}

/* 方案 §A7：safe-area 内边距只在桌面 / 横屏分支生效。
   ⚠️ 必须包在 min-width: 768px 里 —— 裸写会与基类同选择器同特异性，
   把移动端 §A2 的 padding（顶部固定 8px 不含 safe-area-inset-top）一起覆盖掉。 */
@media (min-width: 768px) {
  .counter-app {
    padding:
      max(12px, env(safe-area-inset-top))
      max(12px, env(safe-area-inset-right))
      calc(16px + env(safe-area-inset-bottom))
      max(12px, env(safe-area-inset-left));
  }
}

.ambient-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.ambient-layer::before,
.ambient-layer::after {
  content: '';
  position: absolute;
  border-radius: 999px;
  filter: blur(24px);
}

.ambient-layer::before {
  top: -80px;
  left: -40px;
  width: 260px;
  height: 260px;
  background: var(--counter-glow-a);
}

.ambient-layer::after {
  right: -60px;
  top: 180px;
  width: 220px;
  height: 220px;
  background: var(--counter-glow-b);
}

.counter-shell {
  position: relative;
  z-index: 1;
  width: min(1040px, 100%);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.glass-card {
  background: var(--counter-card);
  border: 1px solid var(--counter-border);
  border-radius: 24px;
  box-shadow: var(--counter-shadow);
  backdrop-filter: blur(18px);
}

.hero-card {
  padding: 24px;
}

.hero-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.hero-kicker {
  margin: 0 0 10px;
  color: var(--counter-muted);
  font-size: 13px;
  letter-spacing: 0.08em;
}

.hero-title {
  margin: 0;
  font-size: clamp(28px, 4vw, 40px);
  line-height: 1.06;
  letter-spacing: -0.04em;
}

.hero-copy {
  margin: 12px 0 0;
  max-width: 720px;
  color: var(--counter-muted);
  line-height: 1.7;
}

.hero-pills {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.hero-pill {
  padding: 8px 12px;
  border-radius: 999px;
  background: var(--counter-accent-soft);
  color: var(--counter-accent);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-top: 22px;
}

.stat-card {
  padding: 18px;
  border-radius: 20px;
  background: var(--counter-card-strong);
  border: 1px solid rgba(255, 255, 255, 0.04);
}

.stat-card.highlight {
  background:
    linear-gradient(135deg, var(--counter-accent-soft), transparent 64%),
    var(--counter-card-strong);
}

.stat-label {
  display: block;
  color: var(--counter-muted);
  font-size: 12px;
  letter-spacing: 0.06em;
}

.stat-value {
  display: block;
  margin-top: 10px;
  font-size: clamp(22px, 3vw, 30px);
  font-weight: 800;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.stat-value.main {
  font-size: clamp(34px, 5vw, 54px);
  letter-spacing: -0.05em;
}

.stat-value.bump {
  animation: count-bump 0.16s ease;
}

@keyframes count-bump {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.04);
  }
  100% {
    transform: scale(1);
  }
}

.goal-block {
  margin-top: 18px;
  padding: 18px 20px;
  border-radius: 20px;
  background: var(--counter-card-strong);
}

.goal-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.section-caption {
  display: block;
  color: var(--counter-subtle);
  font-size: 12px;
}

.goal-title {
  display: block;
  margin-top: 6px;
  font-size: 20px;
}

.goal-status {
  color: var(--counter-muted);
  font-size: 13px;
  text-align: right;
}

.goal-status.reached {
  color: var(--counter-success);
}

.progress-track {
  position: relative;
  height: 10px;
  margin-top: 16px;
  border-radius: 999px;
  background: var(--counter-track);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--counter-accent), rgba(255, 255, 255, 0.85));
  transition: width 0.24s ease;
}

.preset-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.preset-panel {
  padding: 20px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.panel-head h2 {
  margin: 0;
  font-size: 18px;
}

.panel-head p {
  margin: 6px 0 0;
  color: var(--counter-muted);
  font-size: 13px;
  line-height: 1.6;
}

.panel-head-meta {
  padding: 7px 12px;
  border-radius: 999px;
  background: var(--counter-card-strong);
  color: var(--counter-accent);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.preset-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.preset-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--counter-border);
  border-radius: 18px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
  transition: transform 0.18s ease, border-color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease;
}

.preset-card:hover {
  transform: translateY(-1px);
  border-color: var(--counter-accent);
}

.preset-card.active {
  background: var(--counter-accent-soft);
  border-color: var(--counter-accent);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
}

.preset-symbol {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 14px;
  background: var(--counter-card-strong);
  color: var(--counter-accent);
  font-size: 20px;
  font-weight: 800;
}

.preset-name {
  font-size: 15px;
  font-weight: 700;
}

.preset-desc {
  color: var(--counter-muted);
  font-size: 12px;
  line-height: 1.5;
}

.scene-card .preset-symbol {
  display: none;
}

.scene-swatches {
  display: flex;
  gap: 6px;
}

.scene-swatches i {
  display: block;
  width: 32px;
  height: 18px;
  border-radius: 999px;
}

.work-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(0, 0.9fr);
  gap: 20px;
}

.tap-panel,
.insight-panel {
  padding: 22px;
}

.step-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
}

.mobile-focus-strip,
.mobile-main-actions {
  display: none;
}

.step-chip {
  min-width: 58px;
  padding: 10px 14px;
  border: 1px solid var(--counter-border);
  border-radius: 999px;
  background: transparent;
  color: var(--counter-muted);
  font-weight: 700;
  cursor: pointer;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
  transition: all 0.18s ease;
}

.step-chip.active {
  background: var(--counter-accent);
  border-color: var(--counter-accent);
  color: var(--counter-accent-contrast);
}

/* 音频降级胶囊：日间模式与训练层共用的「脱离降级态」入口（方案 §B1 层 4） */
.audio-notice {
  display: flex;
  justify-content: center;
  margin-top: 10px;
}

.audio-notice-btn {
  min-height: 48px;
  padding: 0 18px;
  border: 1px solid var(--counter-border);
  border-radius: 999px;
  background: var(--counter-card-strong);
  color: var(--counter-text);
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
}

.tap-stage {
  display: grid;
  grid-template-columns: 68px minmax(0, 1fr) 68px;
  align-items: center;
  gap: 14px;
  margin-top: 22px;
  touch-action: manipulation;
  user-select: none;
  -webkit-user-select: none;
}

.side-round {
  width: 56px !important;
  height: 56px !important;
  background: var(--counter-card-strong);
  border-color: var(--counter-border) !important;
  color: var(--counter-text) !important;
  touch-action: manipulation;
  user-select: none;
  -webkit-user-select: none;
  -webkit-tap-highlight-color: transparent;
}

.side-plus {
  background: var(--counter-accent) !important;
  border-color: var(--counter-accent) !important;
  color: var(--counter-accent-contrast) !important;
}

/* 训练入口：日间模式进入全屏训练层 / 查看训练记录（方案 §D2②） */
.training-entry {
  display: flex;
  gap: 10px;
  margin-top: 14px;
}

.training-start,
.training-records {
  flex: 1 1 0;
  min-height: 48px;
  padding: 0 14px;
  border: 1px solid var(--counter-border);
  border-radius: 16px;
  background: var(--counter-card-strong);
  color: var(--counter-text);
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
}

.training-start {
  border-color: var(--counter-accent);
  background: var(--counter-accent);
  color: var(--counter-accent-contrast);
}

.training-records {
  flex: 0 0 auto;
  min-width: 104px;
  color: var(--counter-muted);
}

.figure-button {
  position: relative;
  height: 330px;
  border: none;
  border-radius: 36px;
  background:
    radial-gradient(circle at top, rgba(255, 255, 255, 0.22), transparent 45%),
    linear-gradient(180deg, var(--counter-card-strong), rgba(255, 255, 255, 0.02));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.1);
  cursor: pointer;
  overflow: hidden;
  touch-action: manipulation;
  user-select: none;
  -webkit-user-select: none;
  -webkit-tap-highlight-color: transparent;
  transition: transform 0.14s ease, box-shadow 0.14s ease;
}

.figure-button:hover {
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.1),
    0 20px 40px rgba(0, 0, 0, 0.08);
}

.figure-button:active,
.figure-button.bump {
  transform: scale(0.985);
}

.figure-orbit {
  position: absolute;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.orbit-a {
  inset: 22px;
}

.orbit-b {
  inset: 44px;
  border-style: dashed;
  opacity: 0.6;
}

.step-float {
  position: absolute;
  right: 26px;
  top: 22px;
  padding: 8px 12px;
  border-radius: 999px;
  background: var(--counter-accent-soft);
  color: var(--counter-accent);
  font-size: 12px;
  font-weight: 800;
}

.figure-shell {
  position: absolute;
  inset: 68px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: var(--counter-shell);
  box-shadow:
    inset 0 10px 22px rgba(255, 255, 255, 0.12),
    inset 0 -12px 22px rgba(0, 0, 0, 0.16),
    0 0 0 14px var(--counter-shell-soft),
    0 0 0 28px rgba(255, 255, 255, 0.03),
    0 0 38px var(--counter-ring);
}

.shell-mokugyo {
  border-radius: 48% 52% 42% 42% / 56% 56% 42% 42%;
}

.shell-bell {
  inset: 62px 82px 70px;
  border-radius: 42% 42% 30% 30% / 34% 34% 56% 56%;
}

.shell-drum {
  inset: 72px 70px;
  border-radius: 32px;
}

.shell-chime {
  border-radius: 32% 68% 42% 58% / 44% 36% 64% 56%;
}

.figure-mark {
  font-size: clamp(34px, 6vw, 56px);
  font-weight: 900;
  letter-spacing: 0.08em;
}

.figure-name {
  font-size: 24px;
  font-weight: 800;
}

.figure-hit {
  color: var(--counter-muted);
  font-size: 13px;
}

.tap-hints {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 18px;
  color: var(--counter-muted);
  font-size: 12px;
}

.week-heatmap {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 8px;
  margin-top: 18px;
}

.heat-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  aspect-ratio: 1;
  border-radius: 18px;
  background: linear-gradient(180deg, var(--counter-accent), rgba(255, 255, 255, 0.55));
  color: #fff;
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.heat-cell.today {
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.7);
}

.heat-cell.future {
  opacity: 0.16 !important;
}

.heat-day {
  font-size: 11px;
  font-weight: 700;
}

.heat-count {
  font-size: 18px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.mini-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 18px;
}

.mini-stat {
  padding: 14px;
  border-radius: 18px;
  background: var(--counter-card-strong);
}

.mini-stat span {
  display: block;
  color: var(--counter-muted);
  font-size: 12px;
}

.mini-stat strong {
  display: block;
  margin-top: 8px;
  font-size: 15px;
  line-height: 1.4;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 18px;
}

.toolbar :deep(.el-button) {
  color: var(--counter-muted);
}

.toolbar :deep(.el-button:hover) {
  color: var(--counter-text);
}

.toolbar :deep(.el-button--primary) {
  color: var(--counter-accent);
}

.goal-dialog-content {
  text-align: center;
  color: var(--el-text-color-regular);
}

.goal-dialog-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 88px;
  height: 88px;
  margin-bottom: 14px;
  border-radius: 28px;
  background: var(--counter-accent-soft);
  color: var(--counter-accent);
  font-size: 34px;
  font-weight: 900;
}

.panel-overlay {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.34);
  backdrop-filter: blur(8px);
}

.panel-card {
  width: min(720px, 100%);
  max-height: min(88vh, 860px);
  overflow: auto;
  padding: 22px;
  border-radius: 26px;
  background: var(--counter-card-strong);
  border: 1px solid var(--counter-border);
  box-shadow: var(--counter-shadow);
}

.panel-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.panel-card-head h3 {
  margin: 0;
  font-size: 20px;
}

.panel-close {
  border: none;
  background: transparent;
  color: var(--counter-muted);
  font-size: 13px;
  cursor: pointer;
}

.settings-form :deep(.el-form-item__label) {
  color: var(--counter-muted);
}

.settings-form :deep(.el-input-number),
.settings-form :deep(.el-slider) {
  width: 100%;
}

.field-note {
  display: block;
  margin-top: 8px;
  color: var(--counter-subtle);
  font-size: 12px;
}

.switch-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.switch-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 180px;
  padding: 12px 14px;
  border-radius: 16px;
  background: var(--counter-card);
}

.setting-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.setting-pill {
  padding: 10px 14px;
  border: 1px solid var(--counter-border);
  border-radius: 999px;
  background: transparent;
  color: var(--counter-muted);
  cursor: pointer;
}

.setting-pill.active {
  background: var(--counter-accent-soft);
  border-color: var(--counter-accent);
  color: var(--counter-accent);
}

.danger-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.calendar-nav {
  display: flex;
  align-items: center;
  gap: 12px;
}

.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 8px;
}

.calendar-head {
  text-align: center;
  color: var(--counter-muted);
  font-size: 12px;
  font-weight: 700;
}

.calendar-cell {
  min-height: 72px;
  padding: 10px 8px;
  border-radius: 16px;
  background: var(--counter-card);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
}

.calendar-cell.dim {
  opacity: 0.18;
}

.calendar-cell.filled {
  background: var(--counter-accent-soft);
}

.calendar-cell.hit {
  box-shadow: inset 0 0 0 1px var(--counter-success);
}

.calendar-cell.today {
  box-shadow: inset 0 0 0 2px var(--counter-accent);
}

.calendar-day {
  font-weight: 700;
}

.calendar-count {
  color: var(--counter-accent);
  font-size: 12px;
  font-weight: 800;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.history-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border-radius: 18px;
  background: var(--counter-card);
}

.history-date {
  display: block;
  font-size: 15px;
}

.history-day {
  margin: 4px 0 0;
  color: var(--counter-muted);
  font-size: 12px;
}

.history-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.history-count {
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.history-speed {
  color: var(--counter-muted);
  font-size: 12px;
}

/* ── 训练会话：开始面板 / 总结面板 / 敲击时间轴（方案 §D2⑥） ── */
.start-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.session-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 46vh;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.session-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid var(--counter-border);
  border-radius: 16px;
  background: var(--counter-card-strong);
  cursor: pointer;
}

.session-item.active {
  border-color: var(--counter-accent);
}

.session-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.session-range {
  font-size: 14px;
  font-variant-numeric: tabular-nums;
}

.session-item-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
  color: var(--counter-muted);
  font-size: 12px;
}

.session-timeline {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 16px;
}

.session-timeline-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 6px;
  font-size: 13px;
  font-weight: 800;
}

.session-timeline-note {
  color: var(--counter-muted);
  font-size: 11px;
  font-weight: 500;
}

.timeline-list {
  display: flex;
  flex-direction: column;
  max-height: 40vh;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.timeline-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  border-bottom: 1px solid var(--counter-border);
  font-size: 13px;
}

.timeline-time {
  color: var(--counter-muted);
  font-variant-numeric: tabular-nums;
}

.timeline-delta {
  min-width: 46px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.timeline-delta.minus {
  color: #f59e0b;
}

.timeline-kind {
  flex: 1 1 auto;
  color: var(--counter-muted);
  font-size: 12px;
}

.timeline-delete {
  min-width: 48px;
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid var(--counter-border);
  border-radius: 10px;
  background: transparent;
  color: #f59e0b;
  font-size: 12px;
  cursor: pointer;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
}

.empty-state {
  padding: 42px 16px;
  text-align: center;
  color: var(--counter-muted);
}

/* 方案 §A1：断点表唯一口径 —— 1024 / 767.98 / 420 + 横屏（文件末尾）。
   767.98 而非 768：外壳 isMobile 判定是 width < 768（App.vue:464），
   写成 768 会在「恰好 768px」出现 CSS 按移动端渲染、JS 不套 .mobile-main 的裂缝。 */
@media (max-width: 1024px) {
  .stats-grid,
  .preset-grid,
  .work-grid,
  .mini-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767.98px) {
  .counter-app {
    /* 71 = 56(.mobile-header 固定高, App.vue:567-580) + 15(.mobile-main padding-top, App.vue:1044-1047)。
       不再叠加 safe-area-inset-top：56px 固定 header 自身已经覆盖顶部刘海区，再叠一次会白吃最多 47px。 */
    --ct-chrome-top: 71px;

    padding: 8px max(8px, env(safe-area-inset-right))
      calc(14px + env(safe-area-inset-bottom))
      max(8px, env(safe-area-inset-left));

    /* 移动端父容器高度不定（App.vue:1152-1160 把 .main-content 改成 height:auto），
       百分比 min-height 会按 CSS 2.1 §10.7 解析成 0 → 改用显式算式。 */
    min-height: calc(100vh - var(--ct-chrome-top)); /* 兜底：不支持 dvh 的旧 WebView */
    min-height: calc(100dvh - var(--ct-chrome-top));

    /* ⚠️ 本区块【不声明 overflow】：继承基类的 hidden（方案 §A2(5)）。
       删掉 v1 那对「可滚 + 主动截断滚动链」的组合 —— 正是它造成了内层容器阻断滚动链的病灶
       （L2 #46b 要求本文件不再出现滚动链截断属性）。 */
  }

  .counter-shell {
    gap: 8px;
  }

  .hero-card,
  .preset-grid {
    display: none;
  }

  .hero-card,
  .preset-panel,
  .tap-panel,
  .insight-panel,
  .panel-card {
    padding: 14px;
    border-radius: 18px;
  }

  .hero-top,
  .panel-head,
  .goal-meta,
  .panel-card-head {
    flex-direction: column;
    align-items: flex-start;
  }

  .hero-pills {
    justify-content: flex-start;
  }

  .work-grid,
  .mini-stats {
    grid-template-columns: 1fr;
  }

  .tap-panel {
    /* 方案 §A3 首屏预算：100dvh − 外壳上偏移(71) − .counter-app 上下 padding(8+14) − 底部安全区。
       原 `calc(100dvh - 26px - …)` 没减掉外壳 71px 偏移 → 面板底边被推出首屏。 */
    min-height: calc(100dvh - var(--ct-chrome-top) - 22px - env(safe-area-inset-bottom));
    display: flex;
    flex-direction: column;
    padding: 14px 12px 12px;
    border-radius: 22px;
  }

  .work-grid {
    gap: 8px;
  }

  .mobile-focus-strip {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 12px;
    margin-top: 0;
    padding: 2px 0 0;
  }

  .tap-panel > .panel-head {
    display: none;
  }

  .mobile-focus-date {
    display: block;
    margin-bottom: 2px;
    color: var(--counter-subtle);
    font-size: 11px;
    letter-spacing: 0;
  }

  .mobile-focus-count strong.bump {
    animation: count-bump 0.16s ease;
  }

  .mobile-focus-label {
    display: block;
    color: var(--counter-muted);
    font-size: 11px;
    letter-spacing: 0;
  }

  .mobile-focus-count strong {
    display: block;
    margin-top: 4px;
    font-size: 48px;
    line-height: 1;
    font-weight: 900;
    letter-spacing: 0;
    font-variant-numeric: tabular-nums;
  }

  .mobile-focus-meta {
    display: flex;
    flex-direction: column;
    gap: 8px;
    text-align: right;
  }

  .mobile-focus-meta span {
    display: inline-flex;
    align-items: center;
    justify-content: flex-end;
    min-height: 30px;
    padding: 0 10px;
    border-radius: 999px;
    background: var(--counter-card-strong);
    color: var(--counter-muted);
    font-size: 11px;
    white-space: nowrap;
  }

  .step-strip {
    gap: 8px;
    margin-top: 12px;
    flex-wrap: nowrap;
    overflow-x: auto;
    padding: 0 0 2px;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }

  .step-strip::-webkit-scrollbar {
    display: none;
  }

  .step-chip {
    flex: 0 0 auto;
    min-width: 56px;
    min-height: 40px;
    padding: 9px 12px;
    font-size: 13px;
  }

  .tap-stage {
    position: relative;
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
    /* 200px 是敲击面下限：极矮视口下主按钮仍是一整块面而不是细条（方案 §A4） */
    min-height: 200px;
    margin-top: 10px;
  }

  .side-round {
    position: absolute;
    bottom: 14px;
    left: 14px;
    z-index: 3;
    width: 56px !important;
    height: 56px !important;
    box-shadow: 0 12px 28px rgba(0, 0, 0, 0.16);
    backdrop-filter: blur(12px);
  }

  .side-plus {
    right: 14px;
    left: auto;
  }

  .figure-button {
    /* 吃掉 .tap-stage 的剩余高度；min-height: 0 是关键 —— 原来的 360/330px 固定下限
       会在空间不足时把面板【撑破】而不是收缩，把主按钮推出首屏（方案 §A3）。 */
    flex: 1 1 auto;
    width: 100%;
    height: auto;
    min-height: 0;
    border-radius: 26px;
  }

  .figure-shell {
    inset: 52px;
  }

  .shell-bell {
    inset: 48px 56px 54px;
  }

  .shell-drum {
    inset: 58px 48px;
  }

  .step-float {
    top: 16px;
    right: 16px;
    min-height: 30px;
    padding: 7px 10px;
    font-size: 11px;
  }

  .figure-mark {
    font-size: 42px;
  }

  .figure-name {
    font-size: 21px;
  }

  .figure-hit {
    font-size: 12px;
  }

  .orbit-a {
    inset: 18px;
  }

  .orbit-b {
    inset: 34px;
  }

  .tap-hints {
    display: none;
  }

  .mobile-main-actions {
    display: flex;
    justify-content: space-between;
    gap: 6px;
    margin-top: 10px;
    overflow-x: auto;
    padding: 8px;
    border-radius: 16px;
    background: var(--counter-card-strong);
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }

  /* 训练入口在移动端必须紧凑 —— §A3 的 228px 首屏预算里没有它这一行，
     压到 44 + 8 = 52px 才能让 320×568 的敲击面仍 ≥ 200px 下限。 */
  .training-entry {
    gap: 8px;
    margin-top: 8px;
  }

  .training-start,
  .training-records {
    min-height: 44px;
    font-size: 13px;
  }

  .training-records {
    min-width: 88px;
  }

  .audio-notice {
    margin-top: 8px;
  }

  .session-list {
    max-height: 40vh;
  }

  .timeline-list {
    max-height: 34vh;
  }

  .timeline-row {
    min-height: 48px;
  }

  .timeline-delete {
    min-height: 40px;
  }

  .mobile-main-actions::-webkit-scrollbar {
    display: none;
  }

  .mobile-main-actions :deep(.el-button) {
    flex: 1 0 86px;
    min-height: 38px;
    margin-left: 0;
    color: var(--counter-muted);
  }

  .mobile-main-actions :deep(.el-button--primary) {
    color: var(--counter-accent);
  }

  .insight-panel {
    padding: 12px;
    border-radius: 18px;
  }

  .insight-panel .panel-head {
    flex-direction: row;
    align-items: center;
  }

  .insight-panel .panel-head p {
    display: none;
  }

  .insight-panel .panel-head h2 {
    font-size: 15px;
  }

  .insight-panel .panel-head-meta {
    padding: 5px 9px;
  }

  .panel-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .panel-card {
    width: 100%;
    max-height: min(88dvh, 760px);
    border-radius: 24px 24px 0 0;
    padding: 12px 16px calc(18px + env(safe-area-inset-bottom));
    /* 原滚动链截断属性已删除（方案 L2 #46b 要求该属性在本文件无命中）。
       代价：把面板内列表滚到底后继续拖拽会带动背后文档层滚动 —— 影响可忽略，
       因为面板是 fixed 全屏遮罩、背后内容本来就看不见。 */
  }

  .panel-card::before {
    content: '';
    display: block;
    width: 42px;
    height: 5px;
    margin: 0 auto 14px;
    border-radius: 999px;
    background: var(--counter-border);
  }

  .week-heatmap {
    gap: 5px;
    margin-top: 10px;
  }

  .heat-cell {
    border-radius: 13px;
    gap: 4px;
    min-height: 44px;
  }

  .heat-count {
    font-size: 14px;
  }

  .mini-stats {
    gap: 8px;
    margin-top: 10px;
  }

  .mini-stat {
    padding: 10px 12px;
    border-radius: 14px;
  }

  .mini-stat strong {
    margin-top: 5px;
    font-size: 14px;
  }

  .toolbar {
    flex-wrap: nowrap;
    overflow-x: auto;
    margin-top: 10px;
    padding: 4px 0 2px;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }

  .toolbar::-webkit-scrollbar {
    display: none;
  }

  .toolbar :deep(.el-button) {
    flex: 0 0 auto;
    min-height: 36px;
    padding-left: 4px;
    padding-right: 4px;
  }

  .calendar-cell {
    min-height: 58px;
    border-radius: 12px;
  }
}

@media (max-width: 420px) {
  .counter-app {
    /* 方案 §A2：不能写死 6px，否则横向安全区（刘海横屏 / 圆角屏）失效 */
    padding-left: max(8px, env(safe-area-inset-left));
    padding-right: max(8px, env(safe-area-inset-right));
  }

  .tap-panel {
    padding: 12px 10px 10px;
  }

  .mobile-focus-count strong {
    font-size: 42px;
  }

  .mobile-focus-meta {
    gap: 6px;
  }

  .mobile-focus-meta span {
    min-height: 28px;
    padding: 0 8px;
  }

  /* 原 `.figure-button { min-height: 330px }` 已删除（方案 §A3）：
     固定下限会在空间不足时把面板撑破，改由 767.98 区块的 min-height: 0 统一覆盖。 */

  .figure-shell {
    inset: 44px;
  }

  .shell-bell {
    inset: 44px 48px 50px;
  }

  .shell-drum {
    inset: 54px 42px;
  }

  .side-round {
    bottom: 12px;
    left: 12px;
    width: 52px !important;
    height: 52px !important;
  }

  .side-plus {
    right: 12px;
    left: auto;
  }

  .mobile-main-actions :deep(.el-button) {
    flex-basis: 78px;
  }
}

/* 方案 §A6 横屏：放在文件末尾且【不嵌套】在移动断点内 ——
   844×390 这类机型宽度 > 768（走桌面分支），667×390 这类宽度 ≤ 768（走移动分支），两者都要命中。
   验收放宽为「主按钮 + 加减号 + 结束训练按钮可达（允许滚动一屏内）」，不要求首屏完整可见。 */
@media (orientation: landscape) and (max-height: 480px) {
  .work-grid {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
    gap: 8px;
  }

  .tap-panel {
    /* 横屏由宽度而非高度承载，复位移动端的首屏算式 */
    min-height: auto;
  }

  .step-strip,
  .mobile-focus-meta {
    display: none;
  }
}
</style>
