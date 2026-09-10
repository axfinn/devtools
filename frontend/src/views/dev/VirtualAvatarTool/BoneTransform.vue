<!--
  BoneTransform.vue —— 选中骨骼后的变换编辑器(P2)
  三组数值输入(位置 / 旋转 / 缩放),支持:
    - 直接输入数字(Enter 或 blur 应用)
    - +/- 按钮微调 0.1(0.01 + Shift)
    - 重置按钮(回 T-pose 默认)
  数值变化 emit 给父组件,父组件写回 three.js Bone。
-->
<template>
  <div class="bone-transform">
    <div class="bt-header">
      <span class="bt-title">骨骼</span>
      <span class="bt-name">{{ bone?.name || '—' }}</span>
      <el-button size="small" plain @click="onReset" :disabled="!bone">重置</el-button>
    </div>

    <div v-if="!bone" class="bt-empty">
      <el-icon><Aim /></el-icon>
      <span>点击画布中的骨骼以查看 / 编辑</span>
    </div>

    <template v-else>
      <!-- 位置 -->
      <div class="bt-group">
        <label class="bt-label">位置 (x,y,z)</label>
        <div class="bt-row">
          <el-input-number
            v-for="axis in AXES"
            :key="`p-${axis}`"
            v-model="local.position[axis]"
            :precision="3"
            :step="0.1"
            size="small"
            controls-position="right"
            @change="emitChange('position')"
          />
        </div>
      </div>

      <!-- 旋转 -->
      <div class="bt-group">
        <label class="bt-label">旋转 (rx,ry,rz,rad)</label>
        <div class="bt-row">
          <el-input-number
            v-for="axis in AXES"
            :key="`r-${axis}`"
            v-model="local.rotation[axis]"
            :precision="3"
            :step="0.1"
            :min="-6.283"
            :max="6.283"
            size="small"
            controls-position="right"
            @change="emitChange('rotation')"
          />
        </div>
      </div>

      <!-- 缩放 -->
      <div class="bt-group">
        <label class="bt-label">缩放 (x,y,z)</label>
        <div class="bt-row">
          <el-input-number
            v-for="axis in AXES"
            :key="`s-${axis}`"
            v-model="local.scale[axis]"
            :precision="3"
            :step="0.1"
            :min="0.01"
            :max="10"
            size="small"
            controls-position="right"
            @change="emitChange('scale')"
          />
        </div>
      </div>

      <div class="bt-meta">
        <span class="meta-pill">parent: <b>{{ bone.parent?.isBone ? bone.parent.name : '—' }}</b></span>
        <span class="meta-pill">世界位置: <b>{{ fmtVec(worldPos) }}</b></span>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, reactive, watch } from 'vue'
import { Aim } from '@element-plus/icons-vue'

const props = defineProps({
  bone: { type: Object, default: null },
  worldPos: { type: Array, default: () => [0, 0, 0] },
})

const emit = defineEmits(['change', 'reset'])

const AXES = ['x', 'y', 'z']

const local = reactive({
  position: { x: 0, y: 0, z: 0 },
  rotation: { x: 0, y: 0, z: 0 },
  scale: { x: 1, y: 1, z: 1 },
})

watch(() => props.bone, (b) => {
  if (!b) return
  local.position = { x: round(b.position.x), y: round(b.position.y), z: round(b.position.z) }
  local.rotation = { x: round(b.rotation.x), y: round(b.rotation.y), z: round(b.rotation.z) }
  local.scale = { x: round(b.scale.x), y: round(b.scale.y), z: round(b.scale.z) }
}, { immediate: true })

function round(v) {
  return Math.round((v + Number.EPSILON) * 1000) / 1000
}

function emitChange(kind) {
  if (!props.bone) return
  emit('change', {
    name: props.bone.name,
    kind,
    position: { ...local.position },
    rotation: { ...local.rotation },
    scale: { ...local.scale },
  })
}

function onReset() {
  if (!props.bone) return
  // 重置到 T-pose:全部旋转归零,位置回到 BONE_DEFS 默认值很贵,这里只重置旋转
  props.bone.rotation.set(0, 0, 0)
  props.bone.scale.set(1, 1, 1)
  emit('reset', { name: props.bone.name })
  // 同步 local(等下一帧 watcher 触发)
  local.rotation = { x: 0, y: 0, z: 0 }
  local.scale = { x: 1, y: 1, z: 1 }
}

function fmtVec(v) {
  if (!v || v.length < 3) return '—'
  return `(${v[0]?.toFixed(2)}, ${v[1]?.toFixed(2)}, ${v[2]?.toFixed(2)})`
}
</script>

<style scoped>
.bone-transform {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.bt-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-light, #ebeef5);
}
.bt-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary, #909399);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}
.bt-name {
  flex: 1;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary, #409eff);
  font-family: 'JetBrains Mono', monospace;
}
.bt-empty {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
  font-size: 12px;
  color: var(--text-tertiary, #909399);
}
.bt-empty .el-icon {
  font-size: 18px;
}
.bt-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.bt-label {
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  letter-spacing: 0.3px;
}
.bt-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px;
}
.bt-row :deep(.el-input-number) {
  width: 100%;
}
.bt-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}
.meta-pill {
  font-size: 10px;
  padding: 2px 6px;
  background: var(--bg-secondary, #f5f5f5);
  color: var(--text-secondary, #666);
  border-radius: 4px;
  font-family: 'JetBrains Mono', monospace;
}
</style>
