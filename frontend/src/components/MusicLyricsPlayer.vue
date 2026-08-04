<script>
// LRC 解析与同步算法:同时给组件运行时和 vitest 单测用,保持纯函数。
// 与 Vue 单文件组件并存:vue-loader 会把 <script> 当成普通模块,导出对测试可见。

const LRC_TAG = /^\[(\d{1,2}):(\d{1,2})(?:[.:](\d{1,3}))?\]\s*/
const LRC_META = /^\[(ti|ar|al|by|offset|length):.*\]$/i
// 句末标点 — 当原文没有 \n 且也没有 LRC 时间戳时,按这些标点把长串切成行
const SENTENCE_BREAK = /[。！？!?；;]+|\n+/g

export function parseLyrics(text) {
  if (!text || typeof text !== 'string') return []
  const out = []
  const raw = text.split(/\r?\n/)
  for (const line of raw) {
    const trimmed = line.trim()
    if (!trimmed) continue
    if (LRC_META.test(trimmed)) continue

    let rest = trimmed
    const times = []
    while (true) {
      const m = rest.match(LRC_TAG)
      if (!m) break
      const min = parseInt(m[1], 10)
      const sec = parseInt(m[2], 10)
      let ms = 0
      if (m[3]) {
        const padded = m[3].padEnd(3, '0').slice(0, 3)
        ms = parseInt(padded, 10)
      }
      times.push(min * 60 + sec + ms / 1000)
      rest = rest.slice(m[0].length)
    }
    const body = rest.trim()
    if (!body) continue
    if (times.length > 0) {
      for (const t of times) out.push({ time: t, text: body })
    } else {
      out.push({ time: null, text: body })
    }
  }

  // 如果整篇没有换行、也没有 LRC 标签,试按句末标点切;支持一些歌词被打成单行的情况
  if (out.length <= 1 && text && !text.includes('\n') && !LRC_TAG.test(text.trim())) {
    const sentences = text
      .split(SENTENCE_BREAK)
      .map(s => s.trim())
      .filter(Boolean)
    if (sentences.length > 1) {
      return sentences.map(text => ({ time: null, text }))
    }
  }

  const hasAnyTime = out.some(l => l.time !== null)
  if (hasAnyTime) {
    out.sort((a, b) => {
      if (a.time === null) return 1
      if (b.time === null) return -1
      return a.time - b.time
    })
  }
  return out
}

export function detectMode(lines) {
  if (!lines || lines.length === 0) return 'linear'
  const withTime = lines.filter(l => l.time !== null).length
  return withTime / lines.length >= 0.3 ? 'lrc' : 'linear'
}

// 找当前行:LRC 模式用真实时间戳,线性模式按 currentTime / duration 等分。
// 返回 -1 表示"当前时间还没到第一行"(例:纯 LRC 还没到第一句)。
export function findActiveIndex(lines, currentTime, duration, mode) {
  if (!lines || lines.length === 0) return -1
  if (mode === 'lrc') {
    const timed = lines.filter(l => l.time !== null)
    if (timed.length === 0) {
      return Math.min(Math.max(0, Math.floor((currentTime / Math.max(duration, 1)) * lines.length)), lines.length - 1)
    }
    if (currentTime < timed[0].time) return -1
    // 最后一个时间戳之后:停在最后一行
    if (currentTime >= timed[timed.length - 1].time) {
      return lines.lastIndexOf(timed[timed.length - 1])
    }
    let idx = -1
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].time !== null && lines[i].time <= currentTime) idx = i
    }
    return idx
  }
  // linear
  if (!duration || duration <= 0) return 0
  if (currentTime >= duration) return lines.length - 1
  if (currentTime <= 0) return 0
  return Math.min(Math.floor((currentTime / duration) * lines.length), lines.length - 1)
}

// 给每行算一个"点击跳到这里"的秒数。
// LRC 模式:用真实时间戳;线性模式:把整首歌按时长平均分给每行。
export function computeSeekTime(line, index, total, duration, mode) {
  if (mode === 'lrc' && line.time !== null) return line.time
  if (mode === 'linear' && duration > 0 && total > 0) {
    return (index / total) * duration
  }
  return 0
}

export function formatLyricsTime(s) {
  if (!s || !isFinite(s) || s < 0) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}
</script>

<script setup>
import { ref, computed, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { VideoPlay, VideoPause } from '@element-plus/icons-vue'

const props = defineProps({
  audioUrl: { type: String, required: true },
  lyrics: { type: String, default: '' },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
})

const audioEl = ref(null)
const isPlaying = ref(false)
const currentTime = ref(0)
const duration = ref(0)

const lines = computed(() => parseLyrics(props.lyrics))
const mode = computed(() => detectMode(lines.value))
const progressPercent = computed(() => {
  if (!duration.value) return 0
  return Math.min(100, Math.max(0, (currentTime.value / duration.value) * 100))
})

function onLoadedMetadata() {
  if (audioEl.value) duration.value = audioEl.value.duration || 0
}
function onTimeUpdate() {
  if (audioEl.value) currentTime.value = audioEl.value.currentTime || 0
}
function onPlay() { isPlaying.value = true }
function onPause() { isPlaying.value = false }
function onEnded() { isPlaying.value = false }

async function togglePlay() {
  const el = audioEl.value
  if (!el) return
  if (el.paused) {
    try {
      await el.play()
    } catch (err) {
      ElMessage.error('音频播放失败:' + (err?.message || err))
    }
  } else {
    el.pause()
  }
}

function seek(seconds) {
  const el = audioEl.value
  if (!el) return
  el.currentTime = Math.max(0, Math.min(seconds, duration.value || seconds))
}

function onProgressClick(e) {
  if (!duration.value) return
  const rect = e.currentTarget.getBoundingClientRect()
  const ratio = (e.clientX - rect.left) / rect.width
  seek(ratio * duration.value)
}

function seekToLine(idx) {
  if (idx < 0 || idx >= lines.value.length) return
  const line = lines.value[idx]
  const target = computeSeekTime(line, idx, lines.value.length, duration.value, mode.value)
  seek(target)
}

function downloadLyrics() {
  if (!props.lyrics) return
  const blob = new Blob([props.lyrics], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${props.title || 'lyrics'}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

onBeforeUnmount(() => {
  // 离开页面时把音频停掉,免得在背景继续播放
  if (audioEl.value) {
    try { audioEl.value.pause() } catch {}
  }
})
</script>

<template>
  <el-card class="music-lyrics-player" shadow="never">
    <template #header>
      <div class="mlp-head">
        <div class="mlp-titles">
          <span class="mlp-eyebrow">🎵 MiniMax Music</span>
          <h3 class="mlp-title">{{ title || '未命名歌曲' }}</h3>
          <span v-if="subtitle" class="mlp-subtitle">{{ subtitle }}</span>
        </div>
        <el-button text :disabled="!lyrics" @click="downloadLyrics">下载歌词</el-button>
      </div>
    </template>

    <div class="mlp-controls">
      <button class="mlp-play" :aria-label="isPlaying ? 'pause' : 'play'" type="button" @click="togglePlay">
        <el-icon :size="22">
          <VideoPause v-if="isPlaying" />
          <VideoPlay v-else />
        </el-icon>
      </button>
      <div class="mlp-progress" @click="onProgressClick">
        <div class="mlp-progress-fill" :style="{ width: progressPercent + '%' }"></div>
      </div>
      <div class="mlp-time">
        <span>{{ formatLyricsTime(currentTime) }}</span>
        <span class="mlp-time-sep">/</span>
        <span>{{ formatLyricsTime(duration) }}</span>
      </div>
    </div>

    <div class="mlp-lyrics">
      <div
        v-for="(line, idx) in lines"
        :key="idx"
        class="mlp-lyrics-line"
        @click="seekToLine(idx)"
      >
        <span v-if="line.time !== null && mode === 'lrc'" class="mlp-lyrics-time">
          {{ formatLyricsTime(line.time) }}
        </span>
        <span class="mlp-lyrics-text">{{ line.text }}</span>
      </div>
      <div v-if="lines.length === 0" class="mlp-lyrics-empty">这个分享没有附带歌词文本</div>
      <p v-else class="mlp-lyrics-hint">点击任意行跳转 · 当前进度看上方时间</p>
    </div>

    <audio
      ref="audioEl"
      :src="audioUrl"
      preload="metadata"
      @loadedmetadata="onLoadedMetadata"
      @timeupdate="onTimeUpdate"
      @ended="onEnded"
      @play="onPlay"
      @pause="onPause"
    />
  </el-card>
</template>

<style scoped>
.music-lyrics-player {
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.94);
  border: 1px solid rgba(15, 23, 42, 0.08);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.06);
}

.mlp-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.mlp-titles {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.mlp-eyebrow {
  font-size: 12px;
  letter-spacing: 0.18em;
  color: #0e7490;
  text-transform: uppercase;
}

.mlp-title {
  margin: 0;
  font-size: 22px;
  line-height: 1.25;
  color: #0f172a;
  word-break: break-word;
}

.mlp-subtitle {
  font-size: 13px;
  color: #64748b;
}

.mlp-controls {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 4px 0 14px;
}

.mlp-play {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: linear-gradient(135deg, #0e7490, #1e40af);
  color: #fff;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.12s ease, box-shadow 0.12s ease;
  box-shadow: 0 8px 20px rgba(14, 116, 144, 0.25);
}

.mlp-play:hover {
  transform: scale(1.04);
  box-shadow: 0 10px 24px rgba(14, 116, 144, 0.32);
}

.mlp-play:active {
  transform: scale(0.97);
}

.mlp-progress {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.08);
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.mlp-progress-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, #0e7490, #f97316);
  transition: width 0.12s linear;
}

.mlp-time {
  font-variant-numeric: tabular-nums;
  color: #475569;
  font-size: 13px;
  min-width: 92px;
  text-align: right;
}

.mlp-time-sep {
  margin: 0 4px;
  color: #94a3b8;
}

.mlp-lyrics {
  margin-top: 4px;
  max-height: 360px;
  overflow-y: auto;
  padding: 14px 18px;
  border-radius: 18px;
  background: linear-gradient(180deg, #f8fafc 0%, #eef6ff 100%);
  scrollbar-width: thin;
}

.mlp-lyrics-line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 8px 0;
  font-size: 16px;
  line-height: 1.6;
  color: #334155;
  cursor: pointer;
  transition: color 0.15s ease, background-color 0.15s ease;
  border-radius: 8px;
  user-select: none;
}

.mlp-lyrics-line:hover {
  color: #0e7490;
  background-color: rgba(14, 116, 144, 0.06);
}

.mlp-lyrics-time {
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  color: #94a3b8;
  min-width: 40px;
}

.mlp-lyrics-text {
  flex: 1;
  word-break: break-word;
}

.mlp-lyrics-empty {
  padding: 24px 0;
  text-align: center;
  color: #94a3b8;
}

.mlp-lyrics-hint {
  margin: 8px 0 0;
  padding-top: 10px;
  border-top: 1px dashed rgba(15, 23, 42, 0.08);
  font-size: 12px;
  color: #94a3b8;
  text-align: center;
}

@media (max-width: 640px) {
  .mlp-title { font-size: 18px; }
  .mlp-lyrics { max-height: 280px; padding: 10px 14px; }
  .mlp-lyrics-line { font-size: 14px; padding: 6px 0; }
  .mlp-time { min-width: 78px; font-size: 12px; }
  .mlp-play { width: 40px; height: 40px; }
}
</style>