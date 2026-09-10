<!--
  PostFXPanel.vue —— 后处理 / 渲染设置(P2)
  控制项:
    - 抗锯齿(MSAA)
    - 色调(暖色 / 冷色 / 中性)
    - 背景(纯色 / 透明)
-->
<template>
  <div class="postfx-panel">
    <el-form label-position="top" size="small">
      <el-form-item label="抗锯齿">
        <el-switch v-model="antialias" @change="applyAA" />
      </el-form-item>
      <el-form-item label="色调">
        <el-radio-group v-model="tone" @change="applyTone">
          <el-radio-button value="warm">暖</el-radio-button>
          <el-radio-button value="cool">冷</el-radio-button>
          <el-radio-button value="neutral">中性</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="背景">
        <el-radio-group v-model="bgMode" @change="applyBg">
          <el-radio-button value="solid">纯色</el-radio-button>
          <el-radio-button value="transparent">透明</el-radio-button>
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

const antialias = ref(true)
const tone = ref('neutral')
const bgMode = ref('solid')

const TONE_BG = {
  warm: '#1a1410',
  cool: '#0a1020',
  neutral: '#0A0D14',
}

watch(() => props.sceneApi, (api) => {
  if (!api) return
  applyBg()
}, { immediate: true })

function applyAA() {
  if (!props.sceneApi) return
  // three.js antialias 在 renderer 构造时就定了;运行时只能改 pixelRatio
  // 这里仅 emit,不实际重建 renderer(避免大动作)
  emit('change', { antialias: antialias.value, tone: tone.value, bg: bgMode.value })
}

function applyTone() {
  if (!props.sceneApi) return
  const scene = props.sceneApi.scene
  // 只在 bgMode === solid 时生效
  if (bgMode.value === 'solid') {
    import('three').then((THREE) => {
      scene.background = new THREE.Color(TONE_BG[tone.value] || '#0A0D14')
    })
  }
  emit('change', { antialias: antialias.value, tone: tone.value, bg: bgMode.value })
}

function applyBg() {
  if (!props.sceneApi) return
  const scene = props.sceneApi.scene
  const renderer = props.sceneApi.renderer
  if (bgMode.value === 'transparent') {
    import('three').then((THREE) => {
      scene.background = null
      renderer.setClearColor(0x000000, 0)
    })
  } else {
    import('three').then((THREE) => {
      scene.background = new THREE.Color(TONE_BG[tone.value] || '#0A0D14')
      renderer.setClearColor(0x000000, 1)
    })
  }
  emit('change', { antialias: antialias.value, tone: tone.value, bg: bgMode.value })
}
</script>

<style scoped>
.postfx-panel {
  padding: 4px;
}
.postfx-panel :deep(.el-form-item) {
  margin-bottom: 12px;
}
.postfx-panel :deep(.el-form-item__label) {
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  letter-spacing: 0.3px;
  padding-bottom: 2px;
}
</style>
