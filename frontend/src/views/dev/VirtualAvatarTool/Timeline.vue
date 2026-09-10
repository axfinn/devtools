<!--
  Timeline.vue —— 姿态片段时间轴(P2)
  横向滚动条;每段 clip 在条上是一块;点击展开 ClipCard。
  支持:
    - 录制按钮(startRec/stopRec,本期先做 UI,真实录制数据进 P3 联调)
    - 当前时间游标(可拖)
    - 时间刻度(秒)
  clips 是 props(由父组件从后端拉);emit play/share/delete 由父组件处理。
-->
<template>
  <div class="timeline">
    <!-- 顶部工具条 -->
    <div class="timeline-toolbar">
      <div class="toolbar-left">
        <el-button-group>
          <el-button :type="recording ? 'danger' : 'primary'" size="small" @click="onToggleRec">
            <el-icon><VideoCameraFilled v-if="!recording" /><VideoPause v-else /></el-icon>
            {{ recording ? '停止录制' : '录制' }}
          </el-button>
          <el-button size="small" :disabled="!clips.length" @click="onClear">清空</el-button>
        </el-button-group>
        <span class="cursor-time">{{ fmtTime(cursorTime) }} / {{ fmtTime(totalDuration) }}</span>
      </div>
      <div class="toolbar-right">
        <el-button-group>
          <el-button size="small" @click="zoom = Math.max(0.5, zoom - 0.25)">−</el-button>
          <el-button size="small" @click="zoom = Math.min(4, zoom + 0.25)">+</el-button>
        </el-button-group>
        <span class="zoom-label">{{ Math.round(zoom * 100) }}%</span>
      </div>
    </div>

    <!-- 时间刻度 -->
    <div class="timeline-ruler" :style="{ width: `${zoom * baseWidth}px` }">
      <div
        v-for="t in tickMarks"
        :key="t"
        class="tick"
        :style="{ left: `${t * pxPerSec}px` }"
      >
        <span class="tick-label">{{ fmtTick(t) }}</span>
      </div>
    </div>

    <!-- 轨道 -->
    <div class="timeline-track" :style="{ width: `${zoom * baseWidth}px` }" ref="trackRef" @click="onTrackClick">
      <div v-if="!clips.length" class="track-empty">还没有片段 — 点上方"录制"开始</div>
      <div
        v-for="(clip, i) in clips"
        :key="clip.id || i"
        class="clip-block"
        :class="{ active: activeClipId === (clip.id || i) }"
        :style="{
          left: `${(clip.startSec || 0) * pxPerSec}px`,
          width: `${Math.max(8, (clip.durationSec || 1) * pxPerSec)}px`,
          background: clipColor(clip),
        }"
        :title="clip.title || clip.name"
        @click.stop="$emit('select', clip)"
      >
        <span class="clip-label">{{ clip.title || clip.name || '片段' }}</span>
      </div>

      <!-- 游标 -->
      <div class="cursor" :style="{ left: `${cursorTime * pxPerSec}px` }" @mousedown.stop="onCursorDrag"></div>
    </div>

    <!-- 下方:展开的 ClipCard 列表 -->
    <div v-if="expandedClips.length" class="timeline-cards">
      <ClipCard
        v-for="clip in expandedClips"
        :key="clip.id || clip.title"
        :clip="clip"
        :default-expanded="true"
        @play="(c) => $emit('play', c)"
        @share="(c) => $emit('share', c)"
        @delete="(c) => $emit('delete', c)"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { VideoCameraFilled, VideoPause } from '@element-plus/icons-vue'
import ClipCard from './ClipCard.vue'

const props = defineProps({
  clips: { type: Array, default: () => [] },
  activeClipId: { type: [String, Number], default: '' },
})
const emit = defineEmits(['select', 'play', 'share', 'delete', 'clear', 'rec-toggle'])

const zoom = ref(1)
const baseWidth = 600
const pxPerSec = computed(() => 30 * zoom.value)

const recording = ref(false)
const cursorTime = ref(0)

const totalDuration = computed(() => {
  if (!props.clips.length) return 10
  return Math.max(10, ...props.clips.map((c) => (c.startSec || 0) + (c.durationSec || 1)))
})

const tickMarks = computed(() => {
  const t = totalDuration.value
  const step = t > 60 ? 5 : 1
  const out = []
  for (let s = 0; s <= t; s += step) out.push(s)
  return out
})

const expandedClips = computed(() => {
  // 只显示 activeClipId 对应的那张,简化交互
  if (!props.activeClipId) return []
  return props.clips.filter((c) => (c.id || c.title) === props.activeClipId).slice(0, 1)
})

function fmtTime(sec) {
  if (!sec || sec < 0) return '0:00.0'
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${s.toFixed(1).padStart(4, '0')}`
}
function fmtTick(sec) {
  return sec % 60 === 0 ? `${sec}s` : ''
}

function clipColor(clip) {
  // 用 title hash → hue
  const t = clip.title || clip.name || 'x'
  let h = 0
  for (let i = 0; i < t.length; i++) h = (h * 31 + t.charCodeAt(i)) & 0xffff
  const hue = h % 360
  return `linear-gradient(135deg, hsl(${hue} 60% 45%), hsl(${(hue + 60) % 360} 65% 35%))`
}

function onToggleRec() {
  recording.value = !recording.value
  emit('rec-toggle', recording.value)
}

function onClear() {
  emit('clear')
}

function onTrackClick(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  const x = e.clientX - rect.left + e.currentTarget.scrollLeft
  cursorTime.value = Math.max(0, x / pxPerSec.value)
}

function onCursorDrag(e) {
  const startX = e.clientX
  const startT = cursorTime.value
  const track = trackRef.value
  function move(ev) {
    const dx = ev.clientX - startX
    cursorTime.value = Math.max(0, startT + dx / pxPerSec.value)
  }
  function up() {
    document.removeEventListener('mousemove', move)
    document.removeEventListener('mouseup', up)
  }
  document.addEventListener('mousemove', move)
  document.addEventListener('mouseup', up)
}

const trackRef = ref(null)
</script>

<style scoped>
.timeline {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 12px 8px;
  background: var(--bg-primary, #1e1e1e);
  border-top: 1px solid var(--border-base, #404040);
}
.timeline-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.cursor-time {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-secondary, #a0a0a0);
  margin-left: 8px;
}
.zoom-label {
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  font-family: 'JetBrains Mono', monospace;
}
.timeline-ruler {
  position: relative;
  height: 16px;
  overflow: hidden;
  background: var(--bg-secondary, #252525);
  border-radius: 4px;
}
.tick {
  position: absolute;
  top: 0;
  bottom: 0;
  border-left: 1px solid var(--border-light, #333);
}
.tick-label {
  font-size: 9px;
  color: var(--text-tertiary, #909399);
  padding-left: 2px;
  line-height: 16px;
  font-family: 'JetBrains Mono', monospace;
}
.timeline-track {
  position: relative;
  height: 36px;
  overflow-x: auto;
  overflow-y: hidden;
  background: var(--bg-secondary, #252525);
  border-radius: 4px;
  cursor: pointer;
}
.track-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: var(--text-tertiary, #909399);
}
.clip-block {
  position: absolute;
  top: 4px;
  bottom: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  padding: 0 6px;
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 80ms, box-shadow 100ms;
  overflow: hidden;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(0,0,0,0.4);
}
.clip-block:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}
.clip-block.active {
  outline: 2px solid var(--color-primary, #409eff);
  outline-offset: 1px;
}
.clip-label {
  pointer-events: none;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cursor {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--color-primary, #409eff);
  pointer-events: auto;
  cursor: ew-resize;
  z-index: 2;
}
.cursor::before {
  content: '';
  position: absolute;
  top: -2px;
  left: -5px;
  width: 12px;
  height: 6px;
  background: var(--color-primary, #409eff);
  border-radius: 2px;
}
.timeline-cards {
  margin-top: 6px;
  max-height: 180px;
  overflow-y: auto;
}
</style>
