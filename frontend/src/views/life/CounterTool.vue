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
                <i :style="{ background: scene.swatches[0] }"></i>
                <i :style="{ background: scene.swatches[1] }"></i>
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

          <div class="tap-stage">
            <el-button
              class="side-round"
              :icon="Minus"
              circle
              size="large"
              :disabled="todayCount <= 0"
              @click="decrement"
            />

            <button
              type="button"
              class="figure-button"
              :class="{ bump: isBumping }"
              @pointerdown="primeAudio"
              @click="increment"
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
              @click="increment"
            />
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
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
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

const STORAGE_KEY = 'counter_v4'
const LEGACY_KEY = 'counter_v3'
const MAX_HISTORY_DAYS = 365
const SPEED_WINDOW_MS = 15000
const stepOptions = [1, 3, 5, 10, 20]
const weekHeaders = ['一', '二', '三', '四', '五', '六', '日']

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

let goalFired = false
let tapEvents = []
let autoTimer = null
let bumpTimer = null
let clockTimer = null
let audioContext = null

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

  checkDayRollover()
  refreshSpeed()
  persistAll()
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

  archiveDay(currentDate.value, todayCount.value, bestSpeed.value)
  currentDate.value = todayKey
  todayCount.value = 0
  speed.value = 0
  bestSpeed.value = 0
  lastActionDelta.value = 0
  tapEvents = []
  goalFired = false
}

function bumpFigure() {
  isBumping.value = true
  window.clearTimeout(bumpTimer)
  bumpTimer = window.setTimeout(() => {
    isBumping.value = false
  }, 150)
}

function increment() {
  applyDelta(step.value, true)
}

function decrement() {
  applyDelta(-step.value, false)
}

function applyDelta(delta, shouldFeedback) {
  checkDayRollover()
  const nextValue = Math.max(0, todayCount.value + delta)
  const actualDelta = nextValue - todayCount.value

  if (!actualDelta) return

  todayCount.value = nextValue
  lastActionDelta.value = actualDelta

  if (actualDelta > 0 && shouldFeedback) {
    registerHit(actualDelta)
  }

  if (dailyGoal.value > 0 && todayCount.value >= dailyGoal.value && !goalFired) {
    goalFired = true
    nextTick(() => {
      goalReached.value = true
    })
  }
}

function undoLast() {
  if (!lastActionDelta.value) return

  todayCount.value = Math.max(0, todayCount.value - lastActionDelta.value)
  lastActionDelta.value = 0
  goalFired = dailyGoal.value > 0 && todayCount.value >= dailyGoal.value
  ElMessage.success('已撤销上一步')
}

function registerHit(delta) {
  bumpFigure()
  playFigureSound()
  vibrate()
  tapEvents.push({ time: Date.now(), delta })
  refreshSpeed()
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
    increment()
  }, autoInterval.value)
}

function stopAuto() {
  if (autoTimer) {
    window.clearInterval(autoTimer)
    autoTimer = null
  }
}

function primeAudio() {
  if (!soundOn.value) return
  try {
    getAudioContext()
  } catch (_) {
    // ignore
  }
}

function getAudioContext() {
  if (!audioContext) {
    audioContext = new (window.AudioContext || window.webkitAudioContext)()
  }

  if (audioContext.state === 'suspended') {
    audioContext.resume()
  }

  return audioContext
}

function playTone(ctx, config) {
  const oscillator = ctx.createOscillator()
  const gainNode = ctx.createGain()
  const start = ctx.currentTime + (config.delay || 0)

  oscillator.type = config.type || 'sine'
  oscillator.frequency.setValueAtTime(config.frequency, start)

  if (config.endFrequency && config.endFrequency !== config.frequency) {
    oscillator.frequency.exponentialRampToValueAtTime(config.endFrequency, start + config.duration)
  }

  gainNode.gain.setValueAtTime(Math.max(config.gain, 0.0001), start)
  gainNode.gain.exponentialRampToValueAtTime(0.001, start + config.duration)

  if (config.filterType) {
    const filter = ctx.createBiquadFilter()
    filter.type = config.filterType
    filter.frequency.value = config.filterFrequency || config.frequency
    filter.Q.value = config.filterQ || 1
    oscillator.connect(filter)
    filter.connect(gainNode)
  } else {
    oscillator.connect(gainNode)
  }

  gainNode.connect(ctx.destination)
  oscillator.start(start)
  oscillator.stop(start + config.duration)
}

function playFigureSound() {
  if (!soundOn.value || volume.value <= 0) return

  try {
    const ctx = getAudioContext()
    const level = volume.value / 100

    switch (activeFigure.value.sound) {
      case 'mokugyo':
        playTone(ctx, {
          type: 'triangle',
          frequency: 720,
          endFrequency: 430,
          duration: 0.14,
          gain: level * 0.52,
          filterType: 'bandpass',
          filterFrequency: 860,
          filterQ: 4.6
        })
        playTone(ctx, {
          type: 'sine',
          frequency: 240,
          endFrequency: 190,
          duration: 0.18,
          gain: level * 0.18,
          delay: 0.01
        })
        break
      case 'bell':
        playTone(ctx, {
          type: 'sine',
          frequency: 1180,
          endFrequency: 960,
          duration: 0.34,
          gain: level * 0.36
        })
        playTone(ctx, {
          type: 'triangle',
          frequency: 860,
          endFrequency: 660,
          duration: 0.4,
          gain: level * 0.22,
          delay: 0.01
        })
        playTone(ctx, {
          type: 'sine',
          frequency: 1450,
          endFrequency: 1180,
          duration: 0.28,
          gain: level * 0.11,
          delay: 0.02
        })
        break
      case 'drum':
        playTone(ctx, {
          type: 'sine',
          frequency: 200,
          endFrequency: 84,
          duration: 0.22,
          gain: level * 0.6
        })
        playTone(ctx, {
          type: 'triangle',
          frequency: 540,
          endFrequency: 260,
          duration: 0.08,
          gain: level * 0.16,
          delay: 0.004
        })
        break
      case 'chime':
        playTone(ctx, {
          type: 'sine',
          frequency: 1320,
          endFrequency: 1080,
          duration: 0.24,
          gain: level * 0.28
        })
        playTone(ctx, {
          type: 'triangle',
          frequency: 1640,
          endFrequency: 1360,
          duration: 0.22,
          gain: level * 0.16,
          delay: 0.01
        })
        playTone(ctx, {
          type: 'sine',
          frequency: 980,
          endFrequency: 860,
          duration: 0.18,
          gain: level * 0.1,
          delay: 0.02
        })
        break
      default:
        break
    }
  } catch (_) {
    // ignore audio failures
  }
}

function vibrate() {
  if (!vibeOn.value || !navigator.vibrate) return

  try {
    navigator.vibrate(activeFigure.value.vibration)
  } catch (_) {
    // ignore vibration failures
  }
}

function onKeyDown(event) {
  const target = event.target
  if (target && typeof target.closest === 'function' && target.closest('input, textarea, [contenteditable="true"]')) {
    return
  }

  if (event.code === 'Space' || event.code === 'ArrowUp' || event.code === 'Equal') {
    event.preventDefault()
    increment()
    return
  }

  if (event.code === 'ArrowDown' || event.code === 'Minus' || event.code === 'Backspace') {
    event.preventDefault()
    decrement()
    return
  }

  if (event.code === 'KeyZ' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    undoLast()
  }
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
  ElMessageBox.confirm('重置今日计数、节奏和撤销状态？', '确认', {
    type: 'warning',
    confirmButtonText: '确认',
    cancelButtonText: '取消'
  }).then(() => {
    stopAuto()
    autoMode.value = false
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
  ElMessageBox.confirm('清空全部历史记录，但保留今天的计数？', '确认', {
    type: 'warning',
    confirmButtonText: '确认',
    cancelButtonText: '取消'
  }).then(() => {
    dailyRecords.value = []
    saveRecords()
    ElMessage.success('历史记录已清空')
  }).catch(() => {})
}

function resetAll() {
  ElMessageBox.confirm('清除今日计数和所有历史数据？此操作不可恢复。', '确认', {
    type: 'warning',
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    confirmButtonClass: 'el-button--danger'
  }).then(() => {
    stopAuto()
    autoMode.value = false
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
    ElMessage.success('计数器数据已清除')
  }).catch(() => {})
}

function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') {
    persistAll()
  }
}

function handlePageHide() {
  persistAll()
}

onMounted(() => {
  loadAll()
  clockTimer = window.setInterval(() => {
    checkDayRollover()
    refreshSpeed()
  }, 1000)
  document.addEventListener('keydown', onKeyDown)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('beforeunload', handlePageHide)
  window.addEventListener('pagehide', handlePageHide)
})

onUnmounted(() => {
  stopAuto()
  window.clearTimeout(bumpTimer)
  window.clearInterval(clockTimer)
  document.removeEventListener('keydown', onKeyDown)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('beforeunload', handlePageHide)
  window.removeEventListener('pagehide', handlePageHide)
  persistAll()
})
</script>

<style scoped>
.counter-app {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  padding: 24px 16px 40px;
  background: var(--counter-bg);
  color: var(--counter-text);
  overflow: hidden;
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
  transition: all 0.18s ease;
}

.step-chip.active {
  background: var(--counter-accent);
  border-color: var(--counter-accent);
  color: var(--counter-accent-contrast);
}

.tap-stage {
  display: grid;
  grid-template-columns: 68px minmax(0, 1fr) 68px;
  align-items: center;
  gap: 14px;
  margin-top: 22px;
}

.side-round {
  width: 56px !important;
  height: 56px !important;
  background: var(--counter-card-strong);
  border-color: var(--counter-border) !important;
  color: var(--counter-text) !important;
}

.side-plus {
  background: var(--counter-accent) !important;
  border-color: var(--counter-accent) !important;
  color: var(--counter-accent-contrast) !important;
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

.empty-state {
  padding: 42px 16px;
  text-align: center;
  color: var(--counter-muted);
}

@media (max-width: 980px) {
  .stats-grid,
  .preset-grid,
  .work-grid,
  .mini-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .counter-app {
    padding: 10px 10px 24px;
  }

  .counter-shell {
    gap: 12px;
  }

  .hero-card,
  .preset-panel,
  .tap-panel,
  .insight-panel,
  .panel-card {
    padding: 16px;
    border-radius: 20px;
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

  .hero-card,
  .preset-grid {
    display: none;
  }

  .work-grid,
  .mini-stats {
    grid-template-columns: 1fr;
  }

  .tap-panel {
    padding-bottom: 10px;
    gap: 0;
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
    margin-bottom: 4px;
    color: var(--counter-subtle);
    font-size: 11px;
    letter-spacing: 0.04em;
  }

  .mobile-focus-count strong.bump {
    animation: count-bump 0.16s ease;
  }

  .mobile-focus-label {
    display: block;
    color: var(--counter-muted);
    font-size: 11px;
    letter-spacing: 0.05em;
  }

  .mobile-focus-count strong {
    display: block;
    margin-top: 6px;
    font-size: clamp(38px, 10vw, 52px);
    line-height: 1;
    font-weight: 900;
    letter-spacing: -0.04em;
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
  }

  .step-strip {
    gap: 8px;
    margin-top: 10px;
    flex-wrap: nowrap;
    overflow-x: auto;
    padding-bottom: 2px;
    scrollbar-width: none;
  }

  .step-strip::-webkit-scrollbar {
    display: none;
  }

  .step-chip {
    flex: 0 0 auto;
    min-width: 52px;
    padding: 9px 12px;
    font-size: 13px;
  }

  .tap-stage {
    grid-template-columns: 50px minmax(0, 1fr) 50px;
    gap: 10px;
    margin-top: 10px;
  }

  .side-round {
    width: 48px !important;
    height: 48px !important;
  }

  .figure-button {
    height: min(58vh, 460px);
    min-height: 336px;
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
    gap: 10px;
    margin-top: 12px;
    overflow-x: auto;
    padding: 8px 10px 2px;
    border-radius: 18px;
    background: var(--counter-card-strong);
    scrollbar-width: none;
  }

  .mobile-main-actions::-webkit-scrollbar {
    display: none;
  }

  .mobile-main-actions :deep(.el-button) {
    flex: 0 0 auto;
    color: var(--counter-muted);
  }

  .mobile-main-actions :deep(.el-button--primary) {
    color: var(--counter-accent);
  }

  .panel-head p {
    display: none;
  }

  .insight-panel {
    display: none;
  }

  .panel-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .panel-card {
    width: 100%;
    max-height: 84vh;
    border-radius: 24px 24px 0 0;
    padding: 12px 16px 24px;
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
    gap: 6px;
    margin-top: 12px;
  }

  .heat-count {
    font-size: 15px;
  }

  .mini-stats {
    gap: 8px;
    margin-top: 12px;
  }

  .mini-stat {
    padding: 12px;
    border-radius: 16px;
  }

  .toolbar {
    flex-wrap: nowrap;
    overflow-x: auto;
    margin-top: 12px;
    padding-bottom: 2px;
    scrollbar-width: none;
  }

  .toolbar::-webkit-scrollbar {
    display: none;
  }

  .toolbar :deep(.el-button) {
    flex: 0 0 auto;
    padding-left: 2px;
    padding-right: 2px;
  }

  .calendar-cell {
    min-height: 58px;
    border-radius: 12px;
  }
}

@media (max-width: 420px) {
  .mobile-focus-strip {
    align-items: flex-start;
    flex-direction: column;
  }

  .figure-button {
    height: min(52vh, 380px);
    min-height: 300px;
  }

  .figure-shell {
    inset: 48px;
  }

  .shell-bell {
    inset: 46px 50px 50px;
  }

  .shell-drum {
    inset: 56px 44px;
  }
}
</style>
