<!--
  AvatarCanvas.vue —— Three.js 画布容器
  职责:
    1. 探测 WebGL2 可用性(降级路径 D)
    2. 调用已有 scene.js 的 createAvatarScene(canvas)
    3. onUnmount 时 dispose 释放 GPU 资源
    4. 把 scene 句柄 emit 给上层(Stage / Inspector 用)
  P0 范围:不接 GLTFLoader(只渲染默认骨骼人体),不接模型替换。
-->
<template>
  <div class="avatar-canvas-wrap">
    <canvas v-if="webglOk" ref="canvasRef" class="avatar-canvas" />
    <div v-else class="avatar-canvas-fallback">
      <div class="fallback-icon">⚠</div>
      <div class="fallback-title">当前环境不支持 WebGL2</div>
      <div class="fallback-desc">
        请用 Chrome / Edge / Safari 16+ 打开本页面。
        <br />3D 调试面板无法渲染,但左/右侧栏功能仍可用。
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { createAvatarScene } from './scene.js'

const emit = defineEmits(['scene-ready', 'scene-error'])

const canvasRef = ref(null)
const webglOk = ref(true)
let sceneApi = null

// 探测 WebGL2 —— 失败则不挂 canvas,显示降级占位(降级路径 D)
function probeWebGL2() {
  try {
    const c = document.createElement('canvas')
    return !!c.getContext('webgl2')
  } catch (_) {
    return false
  }
}

onMounted(() => {
  if (!probeWebGL2()) {
    webglOk.value = false
    emit('scene-error', { reason: 'webgl2-unsupported' })
    return
  }
  if (!canvasRef.value) return
  try {
    sceneApi = createAvatarScene(canvasRef.value)
    emit('scene-ready', sceneApi)
  } catch (err) {
    webglOk.value = false
    console.error('[AvatarCanvas] createAvatarScene failed:', err)
    emit('scene-error', { reason: 'scene-init-failed', error: err })
  }
})

onUnmounted(() => {
  if (sceneApi && typeof sceneApi.dispose === 'function') {
    try { sceneApi.dispose() } catch (_) { /* noop */ }
  }
  sceneApi = null
})
</script>

<style scoped>
.avatar-canvas-wrap {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--avatar-bg-stage, #0A0D14);
  overflow: hidden;
}
.avatar-canvas {
  display: block;
  width: 100%;
  height: 100%;
  outline: none;
}
.avatar-canvas-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 32px;
  color: var(--avatar-text-secondary, #9098AB);
  text-align: center;
  gap: 12px;
}
.fallback-icon {
  font-size: 48px;
  color: var(--avatar-warning, #F5B547);
}
.fallback-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--avatar-text-primary, #E6E9EF);
}
.fallback-desc {
  font-size: 13px;
  line-height: 1.6;
  max-width: 360px;
}
</style>