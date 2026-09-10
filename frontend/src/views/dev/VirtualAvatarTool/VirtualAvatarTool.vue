<!--
  VirtualAvatarTool.vue —— 顶层壳(P2 完成版)
  布局:
    TopBar(标题 + 快捷操作 + 主题切换)
    ├─ LeftRail(场景/资产/光照/后处理图标 + 对应面板)
    ├─ Stage(主舞台 + CameraPeek + Timeline + ExportDialog)
    └─ Inspector(选中骨骼/姿态关键帧)—— P2 实装
  视觉:从 P0 的强制 dark-only 改为 P2 的"跟随全局主题"——html.dark / html.light 自动切换;
  Avatar 模块内部的 --avatar-* 改为 --color-* 全局命名,被所有模块消费。
-->
<template>
  <div class="avatar-dark tool-root">
    <!-- 顶部:标题 + 快捷操作 + 主题切换 -->
    <header class="tool-topbar">
      <div class="topbar-left">
        <span class="topbar-icon">⌬</span>
        <span class="topbar-title">Virtual Avatar Studio</span>
        <span class="topbar-subtitle">/dev/avatar</span>
      </div>
      <div class="topbar-right">
        <el-button size="small" @click="importDrawerOpen = true">
          <el-icon><Upload /></el-icon>
          导入模型
        </el-button>
        <el-button size="small" @click="cameraOpen = !cameraOpen">
          <el-icon><VideoCamera /></el-icon>
          摄像头调试
        </el-button>
        <el-button size="small" type="primary" @click="exportOpen = true">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
        <!-- 全局主题切换:独立按钮,影响整站 -->
        <el-tooltip :content="`主题:${themeLabel}`" placement="bottom">
          <el-button size="small" plain circle @click="onToggleTheme">
            <el-icon><component :is="themeIcon" /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </header>

    <div class="tool-body">
      <!-- 左侧图标栏 + 面板 -->
      <div class="tool-left">
        <aside class="tool-rail">
          <div class="rail-item" :class="{ active: activeRail === 'scene' }" @click="activeRail = 'scene'" title="场景">
            <el-icon :size="20"><Picture /></el-icon>
          </div>
          <div class="rail-item" :class="{ active: activeRail === 'assets' }" @click="activeRail = 'assets'" title="资产">
            <el-icon :size="20"><FolderOpened /></el-icon>
            <span v-if="libCount > 0" class="rail-badge">{{ libCount }}</span>
          </div>
          <div class="rail-item" :class="{ active: activeRail === 'light' }" @click="activeRail = 'light'" title="光照">
            <el-icon :size="20"><Sunny /></el-icon>
          </div>
          <div class="rail-item" :class="{ active: activeRail === 'postfx' }" @click="activeRail = 'postfx'" title="后处理">
            <el-icon :size="20"><MagicStick /></el-icon>
          </div>
        </aside>
        <div class="rail-panel">
          <ScenePanel v-if="activeRail === 'scene'" :scene-api="sceneApi" />
          <AssetsPanel
            v-else-if="activeRail === 'assets'"
            :active-id="selectedModelId"
            @select="onModelSelect"
            @register-me="onRegisterMe"
          />
          <LightPanel v-else-if="activeRail === 'light'" :scene-api="sceneApi" />
          <PostFXPanel v-else-if="activeRail === 'postfx'" :scene-api="sceneApi" />
        </div>
      </div>

      <!-- 主舞台 -->
      <main class="tool-main">
        <Stage @scene-ready="onSceneReady" @pose-change="onPoseChange">
          <template #camera-peek>
            <CameraPeek v-if="cameraOpen" @state-change="onCameraState" />
          </template>
        </Stage>
      </main>

      <!-- 右侧 Inspector(P2 实装) -->
      <Inspector
        :bone-names="boneNames"
        :by-name="byName"
        :selected-bone-name="selectedBoneName"
        :pose-presets="posePresets"
        :log="recentLog"
        @select-bone="onSelectBone"
        @apply-pose="onPickPose"
        @bone-change="onBoneChange"
        @bone-reset="onBoneReset"
      />
    </div>

    <!-- 底部 Timeline(P2 实装) -->
    <Timeline
      :clips="clips"
      :active-clip-id="activeClipId"
      @select="onClipSelect"
      @play="onClipPlay"
      @share="onClipShare"
      @delete="onClipDelete"
      @clear="onClipsClear"
      @rec-toggle="onRecToggle"
    />

    <!-- 导出 dialog(P2) -->
    <ExportDialog v-model="exportOpen" :scene-api="sceneApi" @exported="onExported" />

    <!-- 导入 drawer(P2) -->
    <ImportDrawer v-model="importDrawerOpen" @imported="onImported" />

    <!-- 错误提示 -->
    <el-alert
      v-if="initError"
      class="tool-alert"
      :title="initError"
      type="warning"
      :closable="false"
      show-icon
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Upload, VideoCamera, Download, Picture, FolderOpened, Sunny, MagicStick, Sunny as Sun, Moon, Monitor } from '@element-plus/icons-vue'
import Stage from './Stage.vue'
import CameraPeek from './CameraPeek.vue'
import Timeline from './Timeline.vue'
import ExportDialog from './ExportDialog.vue'
import ImportDrawer from './ImportDrawer.vue'
import Inspector from './Inspector.vue'
import ScenePanel from './ScenePanel.vue'
import AssetsPanel from './AssetsPanel.vue'
import LightPanel from './LightPanel.vue'
import PostFXPanel from './PostFXPanel.vue'
import { useAssets } from './composable/useAssets.js'
import { useTheme } from '@/composables/useTheme.js'

const activeRail = ref('scene')
const initError = ref('')
const recentLog = ref([])
const assets = useAssets()
const libCount = ref(0)
const sceneApi = ref(null)
const boneNames = ref([])
const byName = ref({})
const posePresets = ref([])

const selectedBoneName = ref('')
const selectedModelId = ref('')
const clips = ref([])
const activeClipId = ref('')

const cameraOpen = ref(false)
const exportOpen = ref(false)
const importDrawerOpen = ref(false)
const cameraState = ref('idle')

// 主题:沿用全局 useTheme
const { themeMode, setThemeMode } = useTheme()
const themeLabel = computed(() => ({ auto: '自动', light: '浅色', dark: '深色' }[themeMode.value] || '自动'))
const themeIcon = computed(() => {
  if (themeMode.value === 'auto') return Monitor
  return themeMode.value === 'dark' ? Moon : Sun
})
function onToggleTheme() {
  const modes = ['auto', 'light', 'dark']
  const idx = modes.indexOf(themeMode.value)
  setThemeMode(modes[(idx + 1) % 3])
}

function onSceneReady(api) {
  sceneApi.value = api
  boneNames.value = api.boneNames || []
  byName.value = api.byName || {}
  posePresets.value = api.poses || []
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 场景就绪 · ${api.boneNames.length} 根骨骼 · ${api.poses.length} 种姿态`)
  // 启用"显示骨骼"自动选中第一根
  if (api.boneNames?.length && !selectedBoneName.value) {
    selectedBoneName.value = api.boneNames[0]
  }
}

function onPoseChange(name) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 切换姿态 → ${name}`)
}

function onPickPose(name) {
  if (!sceneApi.value || typeof sceneApi.value.applyPose !== 'function') return
  if (sceneApi.value.applyPose(name)) {
    recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 应用姿态 ${name}`)
  }
}

function onSelectBone(name) {
  selectedBoneName.value = name
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 选中骨骼 → ${name}`)
}

function onBoneChange(payload) {
  if (!sceneApi.value || !payload) return
  const bone = byName.value[payload.name]
  if (!bone) return
  if (payload.position) bone.position.set(payload.position.x, payload.position.y, payload.position.z)
  if (payload.rotation) bone.rotation.set(payload.rotation.x, payload.rotation.y, payload.rotation.z)
  if (payload.scale) bone.scale.set(payload.scale.x, payload.scale.y, payload.scale.z)
  if (sceneApi.value.skeleton) sceneApi.value.skeleton.update()
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] ${payload.name}.${payload.kind} 编辑`)
}

function onBoneReset(payload) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] ${payload.name} 重置`)
}

function onCameraState(s) {
  cameraState.value = s
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 摄像头 → ${s}`)
}

function onClipSelect(clip) {
  activeClipId.value = clip.id || clip.title
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 选中片段 → ${clip.title || clip.id}`)
}

function onClipPlay(clip) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 播放片段 → ${clip.title || clip.id}`)
}

function onClipShare(clip) {
  // 调后端创建 share link
  assets.createShareLink('clip', clip.id, { expiresInDays: 30 })
    .then((r) => {
      ElMessage.success(`分享码: ${r.code}`)
      recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 分享片段 → ${r.share_url}`)
    })
    .catch((e) => ElMessage.error('分享失败:' + e.message))
}

function onClipDelete(clip) {
  clips.value = clips.value.filter((c) => c !== clip)
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 删除片段 → ${clip.title || clip.id}`)
}

function onClipsClear() {
  clips.value = []
  activeClipId.value = ''
}

function onRecToggle(on) {
  if (on) {
    ElMessage.info('开始录制(本期只占位,真实录制进 P3 联调)')
  } else {
    // 录制结束 → 自动加一个 clip
    const dur = 3
    const startSec = clips.value.reduce((acc, c) => Math.max(acc, (c.startSec || 0) + (c.durationSec || 0)), 0)
    const clip = {
      id: 'local-' + Date.now().toString(36),
      title: '本地录制 #' + (clips.value.length + 1),
      durationSec: dur,
      frameCount: dur * 30,
      fps: 30,
      startSec,
      createdAt: new Date().toISOString(),
    }
    clips.value = [...clips.value, clip]
    recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 新片段 → ${clip.title}`)
  }
}

function onModelSelect(m) {
  selectedModelId.value = m.id
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 选中模型 → ${m.title || m.id}`)
  // TODO: GLTFLoader 加载并替换场景(留给 P3 联调;P2 占位只显示日志)
}

function onRegisterMe(m) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 加入我的资产 → ${m.title || m.id}`)
}

function onExported(payload) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 导出 ${payload.format} → ${payload.filename}`)
}

function onImported(payload) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 上传模型 → ${payload.id}`)
  assets.refreshModels({ limit: 50 })
}

// P1:挂载时拉一次 /api/avatar/models,把库总数显示在左侧 rail 的资产图标旁;
onMounted(async () => {
  const data = await assets.refreshModels({ limit: 1 })
  if (data) {
    libCount.value = data.count ?? (assets.models.value?.length || 0)
    recentLog.value.unshift(
      `[${new Date().toLocaleTimeString()}] 资产库就绪 · ${assets.models.value.length} 个模型(P1 /api/avatar/models)`
    )
  } else if (assets.lastError.value) {
    initError.value = `资产库离线: ${assets.lastError.value}(检查后端是否启动)`
  }
})
</script>

<style scoped>
.tool-root {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh;
  width: 100%;
  margin: -20px;
  /* 抵消 App.vue 里 .main-content 的 padding */
}
.tool-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border-light);
  flex-shrink: 0;
}
.topbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.topbar-icon {
  font-size: 22px;
  color: var(--color-accent, #7C5CFF);
}
.topbar-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.3px;
}
.topbar-subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: 'JetBrains Mono', monospace;
}
.topbar-right {
  display: flex;
  gap: 8px;
}

.tool-body {
  display: grid;
  grid-template-columns: 56px 280px 1fr 300px;
  flex: 1;
  min-height: 0;
}

.tool-left {
  display: flex;
  border-right: 1px solid var(--border-light);
  background: var(--bg-primary);
}
.tool-rail {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 0;
  align-items: center;
  flex-shrink: 0;
  width: 56px;
  border-right: 1px solid var(--border-light);
}
.rail-item {
  position: relative;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background-color 120ms, color 120ms;
}
.rail-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.rail-item.active {
  background: var(--color-accent-soft, rgba(124, 92, 255, 0.13));
  color: var(--color-accent, #7C5CFF);
}
.rail-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--color-accent, #7C5CFF);
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  text-align: center;
  font-family: 'JetBrains Mono', monospace;
  pointer-events: none;
}

.rail-panel {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
  min-width: 0;
}

.tool-main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.tool-alert {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  width: 480px;
  max-width: 90vw;
}

@media (max-width: 1280px) {
  .tool-body {
    grid-template-columns: 56px 240px 1fr 280px;
  }
}

@media (max-width: 1024px) {
  .tool-body {
    grid-template-columns: 56px 1fr;
  }
  .rail-panel, :deep(.tool-inspector) {
    display: none;
  }
}
</style>

<style>
/* 局部样式不 scoped —— 把 .avatar-dark 的主题变量注入全局可见 */
@import './theme.css';
</style>
