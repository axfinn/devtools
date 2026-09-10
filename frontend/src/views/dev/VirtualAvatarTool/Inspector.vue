<!--
  Inspector.vue —— 右侧检查器(P2)
  P0 是空壳;P2 接入:
    - 骨骼树列表(可点击选中)
    - BoneTransform(选中后编辑数值)
    - 姿态关键帧列表(占位,本期简化)
    - 最近活动日志(沿用)
-->
<template>
  <aside class="inspector">
    <!-- 骨骼列表 -->
    <section class="inspector-section">
      <h4 class="inspector-title">骨骼</h4>
      <div v-if="!boneNames.length" class="inspector-empty">
        <span>场景尚未就绪</span>
      </div>
      <div v-else class="bone-tree">
        <div
          v-for="b in boneNames"
          :key="b"
          class="bone-row"
          :class="{ active: selectedBoneName === b }"
          @click="$emit('select-bone', b)"
        >
          <span class="bone-name">{{ b }}</span>
        </div>
      </div>
    </section>

    <!-- 选中骨骼的变换编辑 -->
    <section v-if="selectedBone" class="inspector-section">
      <h4 class="inspector-title">变换</h4>
      <BoneTransform
        :bone="selectedBone"
        :world-pos="selectedBoneWorldPos"
        @change="onBoneChange"
        @reset="onBoneReset"
      />
    </section>

    <!-- 姿态预设快照 -->
    <section class="inspector-section">
      <h4 class="inspector-title">姿态关键帧</h4>
      <div class="keyframe-list">
        <div
          v-for="kf in posePresets"
          :key="kf"
          class="kf-row"
          @click="$emit('apply-pose', kf)"
        >
          <span class="kf-dot"></span>
          <span class="kf-name">{{ kf }}</span>
        </div>
        <div v-if="!posePresets.length" class="inspector-empty">无</div>
      </div>
    </section>

    <!-- 最近活动 -->
    <section class="inspector-section">
      <h4 class="inspector-title">最近活动</h4>
      <ul class="inspector-log">
        <li v-for="(entry, i) in log" :key="i">{{ entry }}</li>
        <li v-if="!log.length" class="inspector-log-empty">暂无</li>
      </ul>
    </section>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import BoneTransform from './BoneTransform.vue'

const props = defineProps({
  boneNames: { type: Array, default: () => [] },
  byName: { type: Object, default: () => ({}) },
  selectedBoneName: { type: String, default: '' },
  posePresets: { type: Array, default: () => [] },
  log: { type: Array, default: () => [] },
})

const emit = defineEmits(['select-bone', 'apply-pose', 'bone-change', 'bone-reset'])

const selectedBone = computed(() => {
  if (!props.selectedBoneName) return null
  return props.byName[props.selectedBoneName] || null
})

const selectedBoneWorldPos = computed(() => {
  const b = selectedBone.value
  if (!b) return [0, 0, 0]
  b.updateWorldMatrix(true, false)
  const m = b.matrixWorld.elements
  return [m[12], m[13], m[14]]
})

function onBoneChange(payload) { emit('bone-change', payload) }
function onBoneReset(payload) { emit('bone-reset', payload) }
</script>

<style scoped>
.inspector {
  background: var(--bg-primary, #1e1e1e);
  border-left: 1px solid var(--border-base, #404040);
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
  overflow-y: auto;
  height: 100%;
}
.inspector-section {
  background: var(--bg-secondary, #252525);
  border: 1px solid var(--border-light, #333);
  border-radius: 10px;
  padding: 10px;
}
.inspector-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary, #909399);
  letter-spacing: 0.5px;
  text-transform: uppercase;
  margin-bottom: 8px;
}
.inspector-empty {
  font-size: 12px;
  color: var(--text-tertiary, #909399);
  text-align: center;
  padding: 12px 0;
}
.bone-tree {
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.bone-row {
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-secondary, #a0a0a0);
  cursor: pointer;
  transition: background-color 100ms;
}
.bone-row:hover {
  background: var(--bg-hover, #333);
  color: var(--text-primary, #e0e0e0);
}
.bone-row.active {
  background: rgba(64, 158, 255, 0.12);
  color: var(--color-primary, #409eff);
}
.bone-name {
  display: block;
}
.keyframe-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.kf-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  color: var(--text-secondary, #a0a0a0);
  transition: background-color 100ms;
}
.kf-row:hover {
  background: var(--bg-hover, #333);
  color: var(--text-primary, #e0e0e0);
}
.kf-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary, #409eff);
  flex-shrink: 0;
}
.kf-name {
  font-family: 'JetBrains Mono', monospace;
}
.inspector-log {
  list-style: none;
  margin: 0;
  padding: 0;
  font-size: 11px;
  color: var(--text-secondary, #a0a0a0);
  font-family: 'JetBrains Mono', monospace;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 160px;
  overflow-y: auto;
}
.inspector-log li {
  padding: 3px 0;
  border-bottom: 1px dashed var(--border-light, #333);
  word-break: break-all;
}
.inspector-log-empty {
  color: var(--text-tertiary, #909399);
  font-style: italic;
  border: none !important;
}
</style>
