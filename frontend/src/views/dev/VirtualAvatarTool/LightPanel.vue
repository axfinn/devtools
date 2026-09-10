<!--
  LightPanel.vue —— 光照面板(P2)
  控制项:
    - 平行光强度(滑块 0-2)
    - 环境光强度(滑块 0-1.5)
    - 阴影开关
-->
<template>
  <div class="light-panel">
    <el-form label-position="top" size="small">
      <el-form-item label="平行光强度">
        <el-slider v-model="dirIntensity" :min="0" :max="2" :step="0.05" @change="apply" />
      </el-form-item>
      <el-form-item label="环境光强度">
        <el-slider v-model="ambIntensity" :min="0" :max="1.5" :step="0.05" @change="apply" />
      </el-form-item>
      <el-form-item label="阴影">
        <el-switch v-model="shadowOn" @change="apply" />
      </el-form-item>
      <el-form-item label="补光">
        <el-switch v-model="fillOn" @change="apply" />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'

const props = defineProps({
  sceneApi: { type: Object, default: null },
})
const emit = defineEmits(['change'])

const dirIntensity = ref(0.7)
const ambIntensity = ref(0.55)
const shadowOn = ref(true)
const fillOn = ref(true)

let dir = null, amb = null, fill = null, mesh = null

watch(() => props.sceneApi, (api) => {
  if (!api) return
  for (const c of api.scene.children) {
    if (c.isDirectionalLight && c === api.scene.children.find((x) => x.isDirectionalLight)) dir = c
    if (c.isAmbientLight) amb = c
  }
  // 第二个 dir 是补光
  const dirs = api.scene.children.filter((c) => c.isDirectionalLight)
  dir = dirs[0]
  fill = dirs[1]
  mesh = api.mesh
  apply()
}, { immediate: true })

function apply() {
  if (dir) dir.intensity = dirIntensity.value
  if (amb) amb.intensity = ambIntensity.value
  if (fill) {
    fill.visible = fillOn.value
    fill.intensity = fillOn.value ? 0.35 : 0
  }
  if (dir) {
    dir.castShadow = shadowOn.value
    dir.shadow.mapSize.width = shadowOn.value ? 1024 : 0
    dir.shadow.mapSize.height = shadowOn.value ? 1024 : 0
  }
  if (mesh) mesh.castShadow = mesh.receiveShadow = shadowOn.value
  emit('change', { dir: dirIntensity.value, amb: ambIntensity.value, shadow: shadowOn.value, fill: fillOn.value })
}
</script>

<style scoped>
.light-panel {
  padding: 4px;
}
.light-panel :deep(.el-form-item) {
  margin-bottom: 12px;
}
.light-panel :deep(.el-form-item__label) {
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  letter-spacing: 0.3px;
  padding-bottom: 2px;
}
</style>
