<!--
  CameraPeek.vue —— 摄像头小窗(P2)
  通过 slot camera-peek 嵌入 Stage 右上角,默认 240x180;
  顶部用 el-alert 把四态文案差异展示出来:
    - granted    :绿色 info   "已授权 · 画面实时同步"
    - requesting :蓝色 info   "等待浏览器权限框…"
    - denied     :红色 warning "权限被拒绝" + [再试] 按钮
    - unsupported:橙色 warning "浏览器不支持" + [了解详情] link
    - no_device  :橙色 warning "未检测到摄像头" + [再试] 按钮
    - error      :红色 error  "摄像头错误: <msg>"
  点击 alert 右上的 [×] 或 [再试] 触发 retry。
-->
<template>
  <div class="camera-peek" :class="`state-${state}`">
    <el-alert
      :type="alertType"
      :title="alertTitle"
      :description="description"
      show-icon
      :closable="false"
      class="camera-alert"
    >
      <template #default>
        <div class="alert-row">
          <span class="alert-emoji">{{ emoji }}</span>
          <div class="alert-text">
            <div class="alert-title">{{ alertTitle }}</div>
            <div class="alert-desc">{{ description }}</div>
          </div>
          <el-button v-if="canRetry" size="small" type="primary" plain @click="onRetry">
            再试
          </el-button>
          <el-button v-if="state === CameraState.UNSUPPORTED" size="small" plain @click="showHelp = true">
            了解详情
          </el-button>
        </div>
      </template>
    </el-alert>

    <div class="camera-frame">
      <video ref="videoRef" class="camera-video" autoplay muted playsinline />
      <div v-if="state !== CameraState.GRANTED" class="camera-placeholder">
        <span class="placeholder-emoji">{{ emoji }}</span>
        <span class="placeholder-text">{{ placeholderText }}</span>
      </div>
    </div>

    <div class="camera-actions">
      <el-button v-if="state === CameraState.IDLE" size="small" type="primary" @click="onRequest">
        <el-icon><VideoCamera /></el-icon> 启动摄像头
      </el-button>
      <el-button v-else-if="state === CameraState.GRANTED" size="small" plain @click="onStop">
        <el-icon><VideoPause /></el-icon> 关闭
      </el-button>
      <!-- 动捕按钮:摄像头授权后才能启用 -->
      <el-button
        v-if="state === CameraState.GRANTED && sceneApi"
        :type="mocapOn ? 'danger' : 'success'"
        size="small"
        plain
        :loading="mocapState === MocapState.LOADING"
        @click="onToggleMocap"
      >
        <el-icon><Aim v-if="!mocapOn" /><VideoPause v-else /></el-icon>
        {{ mocapOn ? '关闭动捕' : '动捕试一下' }}
      </el-button>
      <el-tooltip v-if="mocapOn && mocapFps" :content="`动捕中 · ${mocapFps} fps`" placement="top">
        <span class="mocap-pill">{{ mocapFps }} fps</span>
      </el-tooltip>
    </div>
    <el-alert
      v-if="mocapState === MocapState.ERROR && mocapError"
      class="mocap-alert"
      type="error"
      :title="`动捕失败:${mocapError.message || String(mocapError)}`"
      :closable="false"
      show-icon
    />

    <el-drawer v-model="showHelp" title="为什么摄像头无法启动?" direction="rtl" size="320px">
      <div class="help-content">
        <h4>可能原因</h4>
        <ul>
          <li>当前页面是 <b>非 HTTPS / 非 localhost</b>(浏览器拒绝暴露摄像头)</li>
          <li>浏览器版本过低(getUserMedia 需要 Chrome 53+ / Safari 11+)</li>
          <li>系统隐私设置中禁用了摄像头</li>
        </ul>
        <h4>建议</h4>
        <ol>
          <li>用 <code>https://</code> 或 <code>http://localhost</code> 访问</li>
          <li>升级浏览器到最新版本</li>
          <li>在系统设置里授权浏览器使用摄像头</li>
        </ol>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Aim, VideoCamera, VideoPause } from '@element-plus/icons-vue'
import { useCamera, stateEmoji } from './composable/useCamera.js'
import { CameraState } from './cameraProbe.js'
import { createMocap, MocapState } from './mocap.js'

const props = defineProps({
  autoStart: { type: Boolean, default: false },
  sceneApi: { type: Object, default: null },
})

const emit = defineEmits(['state-change', 'mocap-change'])

const cam = useCamera({ facingMode: 'user', audio: false })
const videoRef = ref(null)
const showHelp = ref(false)
let mocap = null
const mocapOn = ref(false)
const mocapState = ref(MocapState.IDLE)
const mocapError = ref(null)
const mocapFps = ref(0)

const state = computed(() => cam.state.value)
const error = computed(() => cam.error.value)
const emoji = computed(() => stateEmoji(state.value))

const alertType = computed(() => {
  switch (state.value) {
    case CameraState.GRANTED: return 'success'
    case CameraState.REQUESTING: return 'info'
    case CameraState.DENIED: return 'error'
    case CameraState.UNSUPPORTED:
    case CameraState.NO_DEVICE: return 'warning'
    case CameraState.ERROR: return 'error'
    default: return 'info'
  }
})

const alertTitle = computed(() => {
  switch (state.value) {
    case CameraState.IDLE: return '摄像头未启动'
    case CameraState.REQUESTING: return '等待浏览器权限框…'
    case CameraState.GRANTED: return '已授权'
    case CameraState.DENIED: return '权限被拒绝'
    case CameraState.NO_DEVICE: return '未检测到摄像头'
    case CameraState.UNSUPPORTED: return '浏览器不支持'
    case CameraState.ERROR: return '摄像头错误'
    default: return state.value
  }
})

const description = computed(() => {
  if (state.value === CameraState.ERROR && error.value) {
    return error.value.message || String(error.value)
  }
  return cam.describe()
})

const placeholderText = computed(() => {
  if (state.value === CameraState.IDLE) return '点击下方按钮启动'
  if (state.value === CameraState.REQUESTING) return '等待权限授权…'
  return ''
})

const canRetry = computed(() => {
  return [
    CameraState.DENIED,
    CameraState.NO_DEVICE,
    CameraState.ERROR,
  ].includes(state.value)
})

function onRequest() { cam.request() }
function onStop() { cam.stop() }
function onRetry() { cam.retry() }

async function onToggleMocap() {
  if (!props.sceneApi) {
    ElMessage.warning('场景还没就绪,稍后再试')
    return
  }
  if (mocapOn.value) {
    // 关闭 → 停掉循环,恢复当前姿态
    if (mocap) mocap.stop()
    props.sceneApi.resetMocap?.()
    mocapOn.value = false
    mocapFps.value = 0
    emit('mocap-change', false)
    return
  }
  if (!videoRef.value || state.value !== CameraState.GRANTED) {
    ElMessage.warning('请先启动摄像头并允许权限')
    return
  }
  // 启动
  mocap = createMocap({ sceneApi: props.sceneApi })
  const unsub = mocap.subscribe(({ state, error, fps }) => {
    mocapState.value = state
    mocapError.value = error
    if (typeof fps === 'number') mocapFps.value = fps
  })
  try {
    await mocap.start(videoRef.value)
    mocapOn.value = true
    emit('mocap-change', true)
    ElMessage.success('动捕已开启 · 站立于摄像头前看效果')
  } catch (e) {
    mocapOn.value = false
    ElMessage.error('动捕启动失败:' + (e.message || String(e)))
    unsub()
    mocap?.dispose()
    mocap = null
  }
}

watch(state, (s) => emit('state-change', s))
// 摄像头被外部关闭时 → 自动停动捕
watch(state, (s) => {
  if (s !== CameraState.GRANTED && mocapOn.value) onToggleMocap()
})

onMounted(async () => {
  if (videoRef.value) cam.attach(videoRef.value)
  if (props.autoStart) await cam.request()
})

onBeforeUnmount(async () => {
  if (mocap) {
    try { await mocap.dispose() } catch (_) {}
    mocap = null
  }
})
</script>

<style scoped>
.camera-peek {
  width: 240px;
  background: var(--bg-primary, #1e1e1e);
  border: 1px solid var(--border-base, #404040);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}
.camera-alert :deep(.el-alert__content) {
  padding: 0;
}
.alert-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.alert-emoji {
  font-size: 18px;
  flex-shrink: 0;
}
.alert-text {
  flex: 1;
  min-width: 0;
}
.alert-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary, #e0e0e0);
  line-height: 1.3;
}
.alert-desc {
  font-size: 11px;
  color: var(--text-secondary, #a0a0a0);
  margin-top: 2px;
  line-height: 1.4;
}
.camera-frame {
  position: relative;
  width: 100%;
  aspect-ratio: 4 / 3;
  background: #000;
}
.camera-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transform: scaleX(-1); /* 镜像:自拍视角 */
}
.camera-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: rgba(0, 0, 0, 0.5);
  color: var(--text-tertiary, #909399);
  font-size: 11px;
}
.placeholder-emoji {
  font-size: 28px;
  opacity: 0.7;
}
.camera-actions {
  display: flex;
  padding: 8px;
  border-top: 1px solid var(--border-base, #404040);
}
.camera-actions .el-button {
  flex: 1;
}
.help-content {
  padding: 0 8px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-regular, #606266);
}
.help-content h4 {
  margin: 16px 0 8px;
  color: var(--text-primary, #303133);
}
.help-content ul, .help-content ol {
  padding-left: 20px;
}
.help-content code {
  background: var(--bg-secondary, #f5f5f5);
  padding: 1px 6px;
  border-radius: 4px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}
.mocap-pill {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  margin-left: 8px;
  font-size: 10px;
  font-family: 'JetBrains Mono', monospace;
  background: var(--color-success-soft, rgba(103, 194, 58, 0.15));
  color: var(--color-success, #67c23a);
  border-radius: 10px;
}
.mocap-alert {
  margin: 6px 8px 8px;
}
</style>
