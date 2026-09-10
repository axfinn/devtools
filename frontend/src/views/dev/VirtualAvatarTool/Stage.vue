<!--
  Stage.vue —— 主舞台(P0 范围)
  构成:
    - 中央:AvatarCanvas(包 scene.js)
    - 顶部小窗:CameraPeek(P2 留 slot)
    - 录制 HUD:P2 留 slot
    - 底部:Timeline(P2 留 slot,目前给一行 P0 占位说明)
    - 右上:姿态预设选择(T-pose / Wave / Salute / Crouch / Walk-L)
  设计意图:把已有 scene.js 包起来,用户进入就能看到可拖拽、可切姿态的 demo 人体。
-->
<template>
  <div class="stage-wrap">
    <!-- 顶栏:姿态预设(用 scene.js 已经导出的 POSE_PRESETS) -->
    <div class="stage-topbar">
      <span class="stage-topbar-label">姿态预设</span>
      <el-button-group>
        <el-button
          v-for="pose in poses"
          :key="pose"
          size="small"
          :type="activePose === pose ? 'primary' : 'default'"
          @click="onPickPose(pose)"
        >
          {{ pose }}
        </el-button>
      </el-button-group>
      <span class="stage-topbar-skeleton">
        <el-checkbox v-model="showSkeleton" @change="onToggleSkeleton">
          显示骨骼
        </el-checkbox>
      </span>
      <span class="stage-topbar-status">
        <span class="status-dot" :class="statusClass"></span>
        {{ statusText }}
      </span>
    </div>

    <!-- 主舞台 -->
    <div class="stage-main">
      <AvatarCanvas @scene-ready="onSceneReady" @scene-error="onSceneError" />
      <!-- 摄像头小窗(P2 接入) -->
      <div class="stage-camera-slot">
        <slot name="camera-peek" />
      </div>
      <!-- 录制 HUD(P2 接入) -->
      <div class="stage-recording-slot">
        <slot name="recording-hud" />
      </div>
    </div>

    <!-- 时间轴(P2 接入) -->
    <div class="stage-timeline">
      <slot name="timeline">
        <div class="timeline-placeholder">
          <span>📼 时间轴 —— P2 接入:姿态片段 / 模型快照 / 录制帧</span>
        </div>
      </slot>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import AvatarCanvas from './AvatarCanvas.vue'

const emit = defineEmits(['scene-ready', 'scene-error', 'pose-change'])

const poses = ref(['T-pose', 'Wave', 'Salute', 'Crouch', 'Walk-L'])
const activePose = ref('T-pose')
const showSkeleton = ref(false)
const status = ref('loading') // loading | ready | error
const statusText = ref('加载中…')

let sceneApi = null

function statusClass() {
  return {
    'status-loading': status.value === 'loading',
    'status-ready': status.value === 'ready',
    'status-error': status.value === 'error',
  }
}

function onSceneReady(api) {
  sceneApi = api
  status.value = 'ready'
  statusText.value = `已就绪 · ${api.boneNames.length} 根骨骼 · ${api.poses.length} 种姿态`
  emit('scene-ready', api)
}

function onSceneError(payload) {
  status.value = 'error'
  if (payload?.reason === 'webgl2-unsupported') {
    statusText.value = 'WebGL2 不支持'
  } else {
    statusText.value = '场景初始化失败'
  }
  emit('scene-error', payload)
}

function onPickPose(name) {
  if (!sceneApi) return
  if (sceneApi.applyPose(name)) {
    activePose.value = name
    emit('pose-change', name)
  }
}

function onToggleSkeleton(v) {
  if (!sceneApi) return
  sceneApi.setSkeletonVisible(v)
}
</script>

<style scoped>
.stage-wrap {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background: var(--avatar-bg-stage, #0A0D14);
  color: var(--avatar-text-primary, #E6E9EF);
}
.stage-topbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--avatar-border-subtle, #232938);
  background: var(--avatar-bg-panel, #11151F);
  flex-shrink: 0;
  flex-wrap: wrap;
}
.stage-topbar-label {
  font-size: 12px;
  color: var(--avatar-text-tertiary, #5C6479);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}
.stage-topbar-skeleton {
  margin-left: auto;
  display: flex;
  align-items: center;
}
.stage-topbar-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--avatar-text-secondary, #9098AB);
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
}
.status-loading { background: var(--avatar-warning, #F5B547); animation: pulse 1s infinite; }
.status-ready { background: var(--avatar-success, #4ADE80); }
.status-error { background: var(--avatar-danger, #F47174); }
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.stage-main {
  flex: 1;
  position: relative;
  min-height: 0;
}
.stage-camera-slot {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 5;
}
.stage-recording-slot {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 5;
}

.stage-timeline {
  flex-shrink: 0;
  height: 88px;
  border-top: 1px solid var(--avatar-border-subtle, #232938);
  background: var(--avatar-bg-panel, #11151F);
  display: flex;
  align-items: center;
  justify-content: center;
}
.timeline-placeholder {
  font-size: 12px;
  color: var(--avatar-text-tertiary, #5C6479);
  letter-spacing: 0.5px;
}
</style>