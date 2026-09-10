<!--
  ClipCard.vue —— 姿态片段卡片(P2)
  每张卡 = 一段录制(姿态名 + 时长 + 帧数 + 缩略图色块)。
  点击展开:显示帧详情 + 删除按钮 + 跳到模型按钮。
  缩略图占位 = 渐变色 + pose 文字;真实截图后期接 canvas.toDataURL。
-->
<template>
  <el-card class="clip-card" :class="{ expanded }" shadow="hover">
    <div class="card-row" @click="toggle">
      <div class="thumb" :style="thumbStyle">
        <span class="thumb-label">{{ clip.title || clip.name || '片段' }}</span>
      </div>
      <div class="card-meta">
        <div class="meta-title">{{ clip.title || clip.name || '未命名' }}</div>
        <div class="meta-sub">
          <span>{{ fmtDuration(clip.durationSec) }}</span>
          <span class="dot">·</span>
          <span>{{ clip.frameCount || 0 }} 帧</span>
          <span v-if="clip.fps" class="dot">·</span>
          <span v-if="clip.fps">{{ clip.fps }} fps</span>
        </div>
      </div>
      <div class="card-actions">
        <el-tag v-if="clip.hasPassword" type="warning" size="small">🔒</el-tag>
        <el-button size="small" plain @click.stop="$emit('play', clip)">
          <el-icon><VideoPlay /></el-icon>
        </el-button>
        <el-button size="small" plain @click.stop="$emit('share', clip)">
          <el-icon><Share /></el-icon>
        </el-button>
        <el-button size="small" type="danger" plain @click.stop="$emit('delete', clip)">
          <el-icon><Delete /></el-icon>
        </el-button>
        <el-icon class="expand-icon"><ArrowDown v-if="!expanded" /><ArrowUp v-else /></el-icon>
      </div>
    </div>

    <transition name="el-fade-in">
      <div v-if="expanded" class="card-expand">
        <el-descriptions :column="2" size="small" border>
          <el-descriptions-item label="ID">{{ clip.id || '—' }}</el-descriptions-item>
          <el-descriptions-item label="模型">{{ clip.modelId || '—' }}</el-descriptions-item>
          <el-descriptions-item label="帧数">{{ clip.frameCount || 0 }}</el-descriptions-item>
          <el-descriptions-item label="时长">{{ fmtDuration(clip.durationSec) }}</el-descriptions-item>
          <el-descriptions-item label="创建">{{ fmtDate(clip.createdAt) }}</el-descriptions-item>
          <el-descriptions-item label="过期">{{ fmtDate(clip.expiresAt) }}</el-descriptions-item>
        </el-descriptions>
        <slot name="expand" :clip="clip" />
      </div>
    </transition>
  </el-card>
</template>

<script setup>
import { computed, ref } from 'vue'
import { VideoPlay, Share, Delete, ArrowDown, ArrowUp } from '@element-plus/icons-vue'

const props = defineProps({
  clip: { type: Object, required: true },
  defaultExpanded: { type: Boolean, default: false },
})
const emit = defineEmits(['play', 'share', 'delete', 'toggle'])

const expanded = ref(props.defaultExpanded)

function toggle() {
  expanded.value = !expanded.value
  emit('toggle', expanded.value)
}

const thumbStyle = computed(() => {
  // 用 hash 把 title → hue;同 title 永远是同颜色,好看
  const title = props.clip.title || props.clip.name || 'x'
  let h = 0
  for (let i = 0; i < title.length; i++) h = (h * 31 + title.charCodeAt(i)) & 0xffff
  const hue = h % 360
  return {
    background: `linear-gradient(135deg, hsl(${hue} 60% 35%), hsl(${(hue + 60) % 360} 70% 25%))`,
  }
})

function fmtDuration(sec) {
  if (!sec || sec < 0) return '0.0s'
  if (sec < 60) return sec.toFixed(1) + 's'
  const m = Math.floor(sec / 60)
  const s = (sec % 60).toFixed(1)
  return `${m}m${s}s`
}

function fmtDate(s) {
  if (!s) return '—'
  try {
    const d = new Date(s)
    if (isNaN(d.getTime())) return String(s)
    return d.toLocaleString()
  } catch (_) { return String(s) }
}
</script>

<style scoped>
.clip-card {
  margin-bottom: 8px;
  border-radius: 10px;
}
.clip-card :deep(.el-card__body) {
  padding: 10px;
}
.card-row {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}
.thumb {
  width: 56px;
  height: 42px;
  border-radius: 6px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 9px;
  font-weight: 600;
  text-shadow: 0 1px 2px rgba(0,0,0,0.4);
}
.thumb-label {
  text-align: center;
  padding: 0 4px;
  line-height: 1.2;
}
.card-meta {
  flex: 1;
  min-width: 0;
}
.meta-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary, #e0e0e0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.meta-sub {
  display: flex;
  gap: 4px;
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  font-family: 'JetBrains Mono', monospace;
  margin-top: 2px;
}
.dot {
  opacity: 0.5;
}
.card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.expand-icon {
  color: var(--text-tertiary, #909399);
  transition: transform 200ms;
}
.card-expand {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-light, #333);
}
.expanded .expand-icon {
  transform: rotate(180deg);
}
</style>
