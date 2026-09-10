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
import { ref, watch } from 'vue'

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
</style>
