<!--
  VirtualAvatarTool.vue —— 顶层壳(P0)
  布局:
    TopBar(标题 + 快捷操作)
    ├─ LeftRail(场景/资产/光照/后处理图标)—— P0 给空壳,P2 填面板
    ├─ Stage(主舞台 + Timeline)
    └─ Inspector(选中骨骼/姿态关键帧)—— P0 给空壳
  视觉:Dark Studio + Neon Accent,强制 dark-only,
       主题色集中在 .avatar-dark 的 --avatar-accent 变量。
  P0 不实现:模型导入/导出/摄像头/录制/Timeline/Inspector 面板 —— 留到 P2。
-->
<template>
  <div class="avatar-dark tool-root">
    <header class="tool-topbar">
      <div class="topbar-left">
        <span class="topbar-icon">⌬</span>
        <span class="topbar-title">Virtual Avatar Studio</span>
        <span class="topbar-subtitle">/dev/avatar</span>
      </div>
      <div class="topbar-right">
        <el-button size="small" disabled>
          <el-icon><Upload /></el-icon>
          导入模型
        </el-button>
        <el-button size="small" disabled>
          <el-icon><VideoCamera /></el-icon>
          摄像头调试
        </el-button>
        <el-button size="small" type="primary" disabled>
          <el-icon><Download /></el-icon>
          导出
        </el-button>
      </div>
    </header>

    <div class="tool-body">
      <!-- 左侧图标栏(P0 空壳,P2 接入场景/资产/光照/后处理面板) -->
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

      <!-- 主舞台 -->
      <main class="tool-main">
        <Stage @scene-ready="onSceneReady" @pose-change="onPoseChange" />
      </main>

      <!-- 右侧 Inspector(P0 空壳,P2 接入选中骨骼 / 关键帧面板) -->
      <aside class="tool-inspector">
        <div class="inspector-section">
          <h4 class="inspector-title">骨骼</h4>
          <div class="inspector-empty">
            <el-icon :size="24"><Aim /></el-icon>
            <span>P2 接入:选中骨骼后在此调位置 / 旋转 / 缩放</span>
          </div>
        </div>
        <div class="inspector-section">
          <h4 class="inspector-title">姿态关键帧</h4>
          <div class="inspector-empty">
            <el-icon :size="24"><Clock /></el-icon>
            <span>P2 接入:关键帧列表 / 时间码 / 插值曲线</span>
          </div>
        </div>
        <div class="inspector-section">
          <h4 class="inspector-title">最近活动</h4>
          <ul class="inspector-log">
            <li v-for="(entry, i) in recentLog" :key="i">{{ entry }}</li>
            <li v-if="!recentLog.length" class="inspector-log-empty">暂无</li>
          </ul>
        </div>
      </aside>
    </div>

    <!-- 加载完成前的微小提示 -->
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
import { onMounted, ref } from 'vue'
import Stage from './Stage.vue'
import { useAssets } from './composable/useAssets.js'

const activeRail = ref('scene')
const initError = ref('')
const recentLog = ref([])
const assets = useAssets()
const libCount = ref(0)

function onSceneReady(api) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 场景就绪 · ${api.boneNames.length} 根骨骼`)
}

function onPoseChange(name) {
  recentLog.value.unshift(`[${new Date().toLocaleTimeString()}] 切换姿态 → ${name}`)
}

// P1:挂载时拉一次 /api/avatar/models,把库总数显示在左侧 rail 的资产图标旁;
// recentLog 追加一行便于阿修 review 时肉眼确认 e2e 联通。
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
  height: calc(100vh - 0px);
  height: calc(100dvh - 0px);
  width: 100%;
  margin: -20px;
  /* 抵消 App.vue 里 .main-content 的 padding */
}
.tool-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: var(--avatar-bg-panel, #11151F);
  border-bottom: 1px solid var(--avatar-border-subtle, #232938);
  flex-shrink: 0;
}
.topbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.topbar-icon {
  font-size: 22px;
  color: var(--avatar-accent, #7C5CFF);
}
.topbar-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--avatar-text-primary, #E6E9EF);
  letter-spacing: 0.3px;
}
.topbar-subtitle {
  font-size: 12px;
  color: var(--avatar-text-tertiary, #5C6479);
  font-family: 'JetBrains Mono', monospace;
}
.topbar-right {
  display: flex;
  gap: 8px;
}

.tool-body {
  display: grid;
  grid-template-columns: 56px 1fr 280px;
  flex: 1;
  min-height: 0;
}

.tool-rail {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 0;
  background: var(--avatar-bg-panel, #11151F);
  border-right: 1px solid var(--avatar-border-subtle, #232938);
  align-items: center;
}
.rail-item {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--avatar-text-secondary, #9098AB);
  cursor: pointer;
  transition: background-color 120ms, color 120ms;
}
.rail-item:hover {
  background: var(--avatar-bg-elevated, #181D2B);
  color: var(--avatar-text-primary, #E6E9EF);
}
.rail-item.active {
  background: var(--avatar-accent-soft, rgba(124, 92, 255, 0.13));
  color: var(--avatar-accent, #7C5CFF);
}
.rail-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--avatar-accent, #7C5CFF);
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  text-align: center;
  font-family: 'JetBrains Mono', monospace;
  pointer-events: none;
}
.rail-item {
  position: relative;
}

.tool-main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.tool-inspector {
  background: var(--avatar-bg-panel, #11151F);
  border-left: 1px solid var(--avatar-border-subtle, #232938);
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  overflow-y: auto;
}
.inspector-section {
  background: var(--avatar-bg-elevated, #181D2B);
  border: 1px solid var(--avatar-border-subtle, #232938);
  border-radius: var(--avatar-radius-card, 12px);
  padding: 14px;
}
.inspector-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--avatar-text-tertiary, #5C6479);
  letter-spacing: 0.6px;
  text-transform: uppercase;
  margin-bottom: 10px;
}
.inspector-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 14px 0;
  color: var(--avatar-text-tertiary, #5C6479);
  font-size: 12px;
  text-align: center;
  line-height: 1.6;
}
.inspector-log {
  list-style: none;
  margin: 0;
  padding: 0;
  font-size: 12px;
  color: var(--avatar-text-secondary, #9098AB);
  font-family: 'JetBrains Mono', monospace;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 200px;
  overflow-y: auto;
}
.inspector-log li {
  padding: 4px 0;
  border-bottom: 1px solid var(--avatar-border-subtle, #232938);
}
.inspector-log-empty {
  color: var(--avatar-text-tertiary, #5C6479);
  font-style: italic;
  border: none !important;
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

@media (max-width: 1024px) {
  .tool-body {
    grid-template-columns: 56px 1fr;
  }
  .tool-inspector {
    display: none;
  }
}
</style>

<style>
/* 局部样式不 scoped —— 把 .avatar-dark 的主题变量注入全局可见 */
@import './theme.css';
</style>