<!--
  ScenePanel.vue —— 场景设置面板(P2)
  控制项:
    - 网格开关
    - 坐标轴开关
    - 环境预设(深色 / 浅色 / 渐变)
  props.sceneApi 由父组件传 scene.js 的返回值;props 默认值允许 IDE 预览。
-->
<template>
  <div class="scene-panel">
    <el-form label-position="top" size="small">
      <!-- 形象示例:4 个内置 visual preset,点一下换画布里的人物造型 -->
      <el-form-item v-if="visualPresets.length" label="示例形象">
        <div class="preset-grid">
          <button
            v-for="p in visualPresets"
            :key="p.name"
            class="preset-card"
            :class="{ active: currentPreset === p.name }"
            :style="{ '--accent': p.accent }"
            :title="p.desc"
            @click="onPickPreset(p.name)"
          >
            <span class="preset-emoji">{{ presetEmoji(p.name) }}</span>
            <span class="preset-label">{{ labelFor(p) }}</span>
          </button>
        </div>
        <small class="preset-hint">点击切换 — 直接在画布看到不同造型</small>
      </el-form-item>
      <el-form-item label="网格">
        <el-switch v-model="gridOn" @change="apply" />
      </el-form-item>
      <el-form-item label="坐标轴">
        <el-switch v-model="axesOn" @change="apply" />
      </el-form-item>
      <el-form-item label="环境预设">
        <el-radio-group v-model="envPreset" @change="applyEnv">
          <el-radio-button value="dark">深色</el-radio-button>
          <el-radio-button value="light">浅色</el-radio-button>
          <el-radio-button value="gradient">渐变</el-radio-button>
        </el-radio-group>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'

const props = defineProps({
  sceneApi: { type: Object, default: null },
})
const emit = defineEmits(['change'])

const gridOn = ref(true)
const axesOn = ref(false)
const envPreset = ref('dark')

const ENV_COLORS = {
  dark: '#0A0D14',
  light: '#eef2f7',
  gradient: null, // 用 gradient texture
}

// 示例形象:从 sceneApi 拉出来
const visualPresets = computed(() => props.sceneApi?.visualPresets || [])
const currentPreset = computed(() => props.sceneApi?.currentVisualPreset || 'human')

function labelFor(p) {
  return p.label || p.name
}
function presetEmoji(name) {
  // 用 emoji 给每个 preset 一个直观图标,降低"这是什么"的认知负担
  return ({ human: '🧍', robot: '🤖', sphere: '🟣', voxel: '🟩' })[name] || '✨'
}
function onPickPreset(name) {
  if (!props.sceneApi) return
  if (props.sceneApi.applyVisualPreset?.(name)) {
    emit('change', { kind: 'visualPreset', name })
  }
}

watch(() => props.sceneApi, (api) => {
  if (!api) return
  apply()
}, { immediate: true })

function apply() {
  if (!props.sceneApi) return
  // 网格 / 坐标轴:scene.js 没暴露 grid/axes 句柄,我们用 scene.children 找
  let grid = null
  let axes = null
  for (const c of props.sceneApi.scene.children) {
    if (c.isGridHelper) grid = c
    else if (c.isAxesHelper) axes = c
  }
  if (grid) grid.visible = gridOn.value
  if (axes) axes.visible = axesOn.value
  emit('change', { grid: gridOn.value, axes: axesOn.value, env: envPreset.value })
}

function applyEnv(p) {
  if (!props.sceneApi) return
  const scene = props.sceneApi.scene
  const v = p || envPreset.value
  if (v === 'gradient') {
    // 渐变:用一个 CanvasTexture 当背景
    if (scene.background && scene.background.isTexture) {
      // 已经有了就不重建
    } else {
      const c = document.createElement('canvas')
      c.width = 256; c.height = 256
      const ctx = c.getContext('2d')
      const grad = ctx.createLinearGradient(0, 0, 0, 256)
      grad.addColorStop(0, '#1a1f3a')
      grad.addColorStop(1, '#0a0d14')
      ctx.fillStyle = grad
      ctx.fillRect(0, 0, 256, 256)
      // 动态 import three 避免循环依赖
      import('three').then((THREE) => {
        scene.background = new THREE.CanvasTexture(c)
      })
    }
  } else {
    import('three').then((THREE) => {
      scene.background = new THREE.Color(ENV_COLORS[v] || '#0A0D14')
    })
  }
  emit('change', { grid: gridOn.value, axes: axesOn.value, env: envPreset.value })
}
</script>

<style scoped>
.scene-panel {
  padding: 4px;
}
.scene-panel :deep(.el-form-item) {
  margin-bottom: 12px;
}
.scene-panel :deep(.el-form-item__label) {
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  letter-spacing: 0.3px;
  padding-bottom: 2px;
}

/* 示例形象卡片 — 2x2 网格,直观点击切换 */
.preset-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
  width: 100%;
}
.preset-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 8px 4px;
  border: 1px solid var(--border-light, #333);
  border-radius: 6px;
  background: var(--bg-secondary, #252525);
  color: var(--text-secondary, #a0a0a0);
  cursor: pointer;
  font-size: 11px;
  transition: all 120ms;
}
.preset-card:hover {
  border-color: var(--accent, #409eff);
  color: var(--text-primary, #e0e0e0);
}
.preset-card.active {
  border-color: var(--accent, #409eff);
  background: color-mix(in srgb, var(--accent, #409eff) 14%, var(--bg-secondary, #252525));
  color: var(--text-primary, #e0e0e0);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent, #409eff) 25%, transparent);
}
.preset-emoji {
  font-size: 22px;
  line-height: 1;
}
.preset-label {
  font-size: 11px;
  white-space: nowrap;
}
.preset-hint {
  display: block;
  margin-top: 4px;
  font-size: 10px;
  color: var(--text-tertiary, #909399);
  line-height: 1.4;
}
</style>
