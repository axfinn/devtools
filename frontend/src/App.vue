<template>
  <el-container class="app-container" :class="{ 'fullscreen-mode': hideSidebar }">
    <!-- 移动端头部 -->
    <el-header v-if="isMobile && !hideSidebar" class="mobile-header">
      <div class="mobile-header-left">
        <el-icon :size="24" class="menu-trigger" @click="showDrawer = true"><Menu /></el-icon>
        <span class="mobile-title">DevTools</span>
      </div>
      <div class="mobile-header-right">
        <el-tooltip :content="themeModeName" placement="bottom">
          <el-icon :size="20" class="theme-toggle" @click="toggleTheme">
            <component :is="themeIcon" />
          </el-icon>
        </el-tooltip>
        <span class="current-tool">{{ currentTitle }}</span>
      </div>
    </el-header>

    <!-- 移动端抽屉菜单 -->
    <el-drawer
      v-if="isMobile && !hideSidebar"
      v-model="showDrawer"
      direction="ltr"
      size="70%"
      :show-close="false"
      class="mobile-drawer"
    >
      <template #header>
        <div class="drawer-header">
          <el-icon :size="24" color="#409eff"><Tools /></el-icon>
          <span class="drawer-title">DevTools</span>
        </div>
      </template>
      <el-menu
        :default-active="$route.path"
        router
        class="drawer-menu"
        @select="showDrawer = false"
      >
        <el-menu-item
          v-for="route in menuRoutes"
          :key="route.path"
          :index="route.path"
        >
          <el-icon><component :is="route.meta.icon" /></el-icon>
          <span>{{ route.meta.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-drawer>

    <!-- PC端侧边栏 -->
    <el-aside v-if="!isMobile && !hideSidebar" :width="isCollapse ? '64px' : '220px'" class="sidebar">
      <div class="sidebar-header">
        <div class="logo" @click="isCollapse = !isCollapse">
          <el-icon :size="24"><Tools /></el-icon>
          <span v-show="!isCollapse" class="logo-text">DevTools</span>
        </div>
        <el-tooltip :content="themeModeName" placement="bottom">
          <el-icon :size="18" class="theme-toggle-pc" @click.stop="toggleTheme">
            <component :is="themeIcon" />
          </el-icon>
        </el-tooltip>
      </div>
      <!-- 搜索框 -->
      <div v-if="!isCollapse" class="sidebar-search">
        <el-input
          v-model="sidebarSearch"
          placeholder="搜索工具..."
          clearable
          size="small"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      <!-- 分类菜单 -->
      <div class="sidebar-content" v-if="!isCollapse">
        <div
          v-for="(routes, category) in groupedMenuRoutes"
          :key="category"
          class="sidebar-group"
        >
          <div class="sidebar-group-title" @click="toggleCategory(category)">
            <el-icon>
              <component :is="expandedCategories[category] ? ArrowDown : ArrowRight" />
            </el-icon>
            <span>{{ categories[category]?.name || '其他' }}</span>
          </div>
          <div v-show="expandedCategories[category]" class="sidebar-group-items">
            <div
              v-for="route in routes"
              :key="route.path"
              class="sidebar-item"
              :class="{ active: $route.path === route.path }"
              @click="$router.push(route.path)"
            >
              <el-icon><component :is="route.meta.icon" /></el-icon>
              <span>{{ route.meta.title }}</span>
            </div>
          </div>
        </div>
      </div>
      <!-- 折叠模式显示简单菜单 -->
      <el-menu
        v-else
        :default-active="$route.path"
        :collapse="true"
        router
        class="sidebar-menu"
      >
        <el-menu-item
          v-for="route in filteredMenuRoutes"
          :key="route.path"
          :index="route.path"
        >
          <el-tooltip :content="route.meta.title" placement="right">
            <el-icon><component :is="route.meta.icon" /></el-icon>
          </el-tooltip>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主内容区 -->
    <el-main class="main-content" :class="{ 'mobile-main': isMobile }">
      <router-view v-slot="{ Component }">
        <keep-alive :exclude="['NeonApp']">
          <component :is="Component" />
        </keep-alive>
      </router-view>

      <!-- 页面底部 Footer -->
      <div v-if="!hideSidebar" class="page-footer">
        <div class="footer-content">
          <a class="footer-link" href="https://github.com/axfinn/devtools" target="_blank">
            <svg class="github-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
            </svg>
            <span>GitHub</span>
          </a>
          <div class="footer-divider">•</div>
          <div class="footer-link" @click="showDonateDialog = true">
            <el-icon><Coffee /></el-icon>
            <span>支持项目</span>
          </div>
        </div>
      </div>
    </el-main>

    <!-- 捐赠对话框 -->
    <el-dialog
      v-model="showDonateDialog"
      title="支持项目"
      width="400px"
      class="donate-dialog"
      :append-to-body="true"
    >
      <div class="donate-content">
        <p class="donate-text">如果这个项目对你有帮助，欢迎请作者喝杯咖啡</p>
        <div class="qr-codes">
          <div class="qr-item">
            <img src="/alipay.jpeg" alt="支付宝" />
            <span>支付宝</span>
          </div>
          <div class="qr-item">
            <img src="/wxpay.jpeg" alt="微信支付" />
            <span>微信支付</span>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 全局媒体播放器栏 (悬浮，跨页面持久) -->
    <Teleport to="body">
      <Transition name="pb-slide">
        <div v-if="playerState.currentIndex >= 0" class="player-bar">
          <!-- Progress thin line at top -->
          <div class="pb-progress-track" @click="seekBar">
            <div class="pb-progress-fill" :style="{ width: progressPercent + '%' }" />
          </div>

          <div class="pb-inner">
            <!-- Track info -->
            <div class="pb-info" @click="goToGallery">
              <div class="pb-visual">
                <div class="pb-waveform">
                  <span v-for="i in 4" :key="i" class="pb-bar" :class="{ active: playerState.isPlaying }" :style="{ animationDelay: `${i * 0.15}s` }" />
                </div>
              </div>
              <div class="pb-meta">
                <span class="pb-title">{{ currentTrack?.title || '未命名' }}</span>
                <span class="pb-type">{{ currentTrack?.result_type === 'audio' ? '音乐' : '视频' }}</span>
              </div>
            </div>

            <!-- Transport controls -->
            <div class="pb-controls">
              <button class="pb-btn" :class="{ active: playerState.shuffleMode }" title="随机播放" @click="player.toggleShuffle()">
                <el-icon :size="16"><Rank /></el-icon>
              </button>
              <button class="pb-btn" title="上一首" @click="player.skipPrev()">
                <el-icon :size="20"><ArrowLeftBold /></el-icon>
              </button>
              <button class="pb-btn pb-btn-play" title="播放/暂停" @click="player.togglePlay()">
                <el-icon :size="22"><component :is="playerState.isPlaying ? 'VideoPause' : 'VideoPlay'" /></el-icon>
              </button>
              <button class="pb-btn" title="下一首" @click="player.skipNext()">
                <el-icon :size="20"><ArrowRightBold /></el-icon>
              </button>
              <button
                class="pb-btn"
                :class="{ active: playerState.loopMode !== 'off' }"
                title="循环模式"
                @click="cycleLoop"
              >
                <el-icon :size="16"><RefreshRight /></el-icon>
                <span v-if="playerState.loopMode === 'one'" class="pb-loop-badge">1</span>
              </button>
            </div>

            <!-- Time + close -->
            <div class="pb-right">
              <span class="pb-time">{{ fmtTime(playerState.currentTime) }} / {{ fmtTime(playerState.duration) }}</span>
              <button class="pb-btn pb-btn-close" title="停止播放" @click="player.stop()">
                <el-icon :size="16"><Close /></el-icon>
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Menu, Coffee, Sunny, Moon, Monitor, Search, ArrowDown, ArrowRight, Tools, VideoPlay, VideoPause, ArrowLeftBold, ArrowRightBold, RefreshRight, Rank, Close } from '@element-plus/icons-vue'
import { useTheme } from './composables/useTheme'
import { useMediaPlayer } from './composables/useMediaPlayer'

const router = useRouter()
const route = useRoute()
const isCollapse = ref(false)
const showDrawer = ref(false)
const isMobile = ref(false)
const showDonateDialog = ref(false)
const hiddenRoutes = ref([])

// Global media player
const player = useMediaPlayer()
const playerState = player.state
const currentTrack = computed(() => player.currentTrack())
const progressPercent = computed(() => {
  if (!playerState.duration) return 0
  return Math.min((playerState.currentTime / playerState.duration) * 100, 100)
})

function cycleLoop() {
  const modes = ['off', 'all', 'one']
  const idx = modes.indexOf(playerState.loopMode)
  player.setLoopMode(modes[(idx + 1) % 3])
}

function seekBar(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  const pct = (e.clientX - rect.left) / rect.width
  player.seek(pct * playerState.duration)
}

function goToGallery() {
  if (route.path !== '/gallery') {
    router.push('/gallery')
  }
}

function fmtTime(s) {
  return player.fmtTime(s)
}

// 启动时拉取隐藏路由配置
fetch('/api/console/settings').then(r => r.json()).then(d => {
  hiddenRoutes.value = d.hidden_routes || []
}).catch(() => {})

// 主题管理
const { themeMode, currentTheme, setThemeMode, toggleTheme } = useTheme()

// 是否隐藏侧边栏（全屏模式，或分享预览页）
const hideSidebar = computed(() => route.meta?.hideSidebar === true || !!route.query.share)

// 当前页面标题
const currentTitle = computed(() => {
  const currentRoute = menuRoutes.value.find(r => r.path === route.path)
  return currentRoute?.meta?.title || 'DevTools'
})

import { categories } from './router'

const menuRoutes = computed(() => {
  return router.options.routes.filter(route =>
    route.meta && route.meta.title && !route.path.includes(':') && route.meta.category && route.meta.category !== 'home' &&
    (route.path === '/console' || !hiddenRoutes.value.includes(route.path))
  )
})

// 侧边栏搜索（带防抖）
const sidebarSearch = ref('')
const debouncedSearch = ref('')
let searchTimer = null
watch(sidebarSearch, (val) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { debouncedSearch.value = val }, 300)
})
const filteredMenuRoutes = computed(() => {
  if (!debouncedSearch.value) return menuRoutes.value
  const keyword = debouncedSearch.value.toLowerCase()
  return menuRoutes.value.filter(route =>
    route.meta?.title?.toLowerCase().includes(keyword) ||
    route.meta?.description?.toLowerCase().includes(keyword)
  )
})

// 分类菜单
const groupedMenuRoutes = computed(() => {
  const groups = {}
  filteredMenuRoutes.value.forEach(route => {
    const category = route.meta?.category || 'other'
    if (!groups[category]) {
      groups[category] = []
    }
    groups[category].push(route)
  })
  return groups
})

// 分类展开状态
const expandedCategories = ref(Object.keys(categories).reduce((acc, key) => {
  acc[key] = true
  return acc
}, {}))

const toggleCategory = (category) => {
  expandedCategories.value[category] = !expandedCategories.value[category]
}

// 检测屏幕宽度
const checkMobile = () => {
  const width = window.innerWidth
  isMobile.value = width < 768
  if (!isMobile.value) {
    showDrawer.value = false
  }
  // 中等屏幕自动折叠侧边栏
  if (width >= 768 && width < 1024) {
    isCollapse.value = true
  }
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)

  // 初始化主题
  const { init } = useTheme()
  const cleanup = init()

  // 组件卸载时清理监听器
  onUnmounted(() => {
    window.removeEventListener('resize', checkMobile)
    if (cleanup) cleanup()
    clearTimeout(searchTimer)
  })
})

// 获取主题图标
const themeIcon = computed(() => {
  if (themeMode.value === 'auto') {
    return Monitor
  }
  return themeMode.value === 'dark' ? Moon : Sunny
})

// 获取主题模式名称
const themeModeName = computed(() => {
  const modeText = {
    auto: '自动 (跟随系统)',
    light: '浅色模式',
    dark: '深色模式'
  }
  return modeText[themeMode.value]
})
</script>

<style scoped>
.app-container {
  height: 100vh;
  height: 100dvh;
  background-color: var(--bg-base);
  overflow: hidden;
  transition: background-color 0.3s;
}

.app-container.fullscreen-mode {
  display: block;
}

.app-container.fullscreen-mode .main-content {
  height: 100vh;
  height: 100dvh;
  padding: 0;
}

/* 移动端头部 */
.mobile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: var(--bg-primary);
  border-bottom: 1px solid var(--border-base);
  padding: 0 15px;
  height: 56px;
  position: fixed; /* 改为固定定位 */
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  transition: background-color 0.3s, border-color 0.3s;
}

.mobile-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.menu-trigger {
  color: #409eff;
  cursor: pointer;
  padding: 8px;
}

.mobile-title {
  font-size: 18px;
  font-weight: bold;
  color: #409eff;
}

.mobile-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-secondary);
  font-size: 14px;
}

.theme-toggle {
  color: var(--text-secondary);
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  transition: all 0.3s;
  display: flex;
  align-items: center;
}

.theme-toggle:hover {
  color: var(--color-primary);
  background: rgba(64, 158, 255, 0.1);
}

.current-tool {
  background: var(--bg-secondary);
  padding: 4px 12px;
  border-radius: 12px;
  transition: background-color 0.3s;
}

/* 移动端抽屉 */
.drawer-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
}

.drawer-title {
  font-size: 20px;
  font-weight: bold;
  color: #409eff;
}

.drawer-menu {
  border-right: none;
  background-color: var(--bg-primary);
}

.drawer-menu .el-menu-item {
  height: 50px;
  line-height: 50px;
  font-size: 16px;
  color: var(--text-secondary);
}

.drawer-menu .el-menu-item.is-active {
  color: var(--color-primary);
}

/* PC端侧边栏 */
.sidebar {
  background-color: var(--bg-primary);
  transition: width 0.3s, background-color 0.3s;
  overflow: hidden;
  flex-shrink: 0;
  z-index: 10;
}

.sidebar-header {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  border-bottom: 1px solid var(--border-base);
  transition: border-color 0.3s;
  gap: 8px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #409eff;
  cursor: pointer;
  flex: 1;
  min-width: 0;
}

.logo-text {
  font-size: 18px;
  font-weight: bold;
}

.theme-toggle-pc {
  color: var(--text-secondary);
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  transition: all 0.3s;
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.theme-toggle-pc:hover {
  color: var(--color-primary);
  background: rgba(64, 158, 255, 0.1);
}

/* 侧边栏搜索 */
.sidebar-search {
  padding: 12px;
  border-bottom: 1px solid var(--border-base);
}

/* 侧边栏内容 */
.sidebar-content {
  height: calc(100vh - 100px);
  height: calc(100dvh - 100px);
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-group {
  padding: 8px 0;
}

.sidebar-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: color 0.2s;
}

.sidebar-group-title:hover {
  color: var(--text-primary);
}

.sidebar-group-items {
  padding: 0 8px;
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px 10px 28px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 14px;
  transition: all 0.2s;
}

.sidebar-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.sidebar-item.active {
  background: rgba(64, 158, 255, 0.1);
  color: var(--color-primary);
}

.sidebar-menu {
  border-right: none;
  height: calc(100vh - 60px);
  height: calc(100dvh - 60px);
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 200px;
}

.sidebar {
  position: relative;
}

/* Element Plus Menu 主题覆盖 */
:global(.sidebar-menu.el-menu) {
  background-color: var(--bg-primary);
}

:global(.sidebar-menu .el-menu-item) {
  color: var(--text-secondary);
}

:global(.sidebar-menu .el-menu-item.is-active) {
  color: var(--color-primary);
}

/* 捐赠对话框 */
.donate-content {
  text-align: center;
}

.donate-text {
  color: var(--text-secondary);
  margin-bottom: 20px;
  font-size: 14px;
}

.qr-codes {
  display: flex;
  justify-content: center;
  gap: 30px;
}

.qr-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.qr-item img {
  width: 150px;
  height: 150px;
  border-radius: 8px;
  object-fit: contain;
}

.qr-item span {
  color: var(--text-secondary);
  font-size: 14px;
}

/* 主内容区 */
.main-content {
  background-color: var(--bg-base);
  padding: 20px;
  color: var(--text-primary);
  height: 100vh;
  height: 100dvh;
  overflow-y: auto;
  flex: 1;
  min-width: 0;
  transition: background-color 0.3s, color 0.3s;
}

.mobile-main {
  padding: 15px;
  padding-top: calc(56px + 15px); /* 头部高度 + 内边距 */
  padding-bottom: 100px; /* 为底部留出足够空间 */
  height: auto; /* 改为自动高度 */
  min-height: calc(100vh - 56px); /* 最小高度减去头部 */
  min-height: calc(100dvh - 56px);
}

/* 页面底部 Footer */
.page-footer {
  margin-top: 40px;
  padding: 20px 0;
  border-top: 1px solid var(--border-base);
  transition: border-color 0.3s;
}

.footer-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
  color: var(--text-tertiary);
  font-size: 14px;
  transition: color 0.3s;
  flex-wrap: wrap;
}

.footer-link {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-tertiary);
  text-decoration: none;
  cursor: pointer;
  transition: color 0.2s;
}

.footer-link:hover {
  color: var(--color-primary);
}

.footer-link .github-icon {
  width: 16px;
  height: 16px;
}

.footer-link .el-icon {
  font-size: 16px;
}

.footer-divider {
  color: var(--border-dark);
  transition: color 0.3s;
}

/* 移动端 Footer 适配 */
@media (max-width: 768px) {
  .page-footer {
    margin-top: 30px;
    padding: 16px 0 20px;
  }

  .footer-content {
    font-size: 12px;
    gap: 10px;
    padding: 0 15px;
  }

  .footer-link .github-icon {
    width: 14px;
    height: 14px;
  }

  .footer-link .el-icon {
    font-size: 14px;
  }
}

@media (max-width: 480px) {
  .page-footer {
    margin-top: 20px;
    padding: 10px 0 14px;
  }

  .footer-content {
    font-size: 11px;
    gap: 8px;
  }

  .footer-link span {
    display: none;
  }

  .footer-divider {
    display: none;
  }
}

/* 中等屏幕适配 */
@media (min-width: 768px) and (max-width: 1024px) {
  .sidebar-menu {
    height: calc(100vh - 60px);
    height: calc(100dvh - 60px);
  }
}

/* 移动端全局样式调整 */
@media (max-width: 768px) {
  .app-container {
    flex-direction: column;
    overflow: visible; /* 允许滚动 */
  }

  .main-content {
    height: auto; /* 移动端不限制高度 */
    overflow-y: visible; /* 允许自然滚动 */
  }
}
</style>

<style>
/* ===== 全局媒体播放器栏 (持久化悬浮) ===== */
.player-bar {
  position: fixed;
  bottom: 0; left: 0; right: 0;
  z-index: 10000;
  background: rgba(15, 23, 42, 0.96);
  backdrop-filter: blur(24px) saturate(1.5);
  border-top: 1px solid rgba(56, 189, 248, 0.25);
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.5);
  user-select: none;
}

.pb-progress-track {
  position: absolute;
  top: -3px; left: 0; right: 0;
  height: 3px;
  background: rgba(255,255,255,0.08);
  cursor: pointer;
  transition: height 0.15s;
}
.pb-progress-track:hover { height: 6px; top: -6px; }
.pb-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #38bdf8, #6366f1, #a78bfa);
  border-radius: 0 2px 2px 0;
  transition: width 0.15s linear;
}

.pb-inner {
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 16px;
  gap: 14px;
}

.pb-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
  cursor: pointer;
}
.pb-visual {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 22px;
  flex-shrink: 0;
}
.pb-waveform { display: flex; align-items: flex-end; gap: 3px; height: 22px; }
.pb-bar {
  width: 4px;
  border-radius: 2px;
  background: rgba(56, 189, 248, 0.4);
  height: 6px;
  transition: background 0.2s;
}
.pb-bar.active {
  animation: pbWave 0.7s ease-in-out infinite alternate;
  background: linear-gradient(180deg, #38bdf8, #6366f1);
}
@keyframes pbWave {
  0% { height: 6px; }
  100% { height: 22px; }
}
.pb-meta {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.pb-title {
  font-size: 13px;
  font-weight: 600;
  color: #f1f5f9;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pb-type {
  font-size: 11px;
  color: #38bdf8;
}

.pb-controls {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.pb-btn {
  width: 36px; height: 36px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
  position: relative;
}
.pb-btn:hover { background: rgba(148, 163, 184, 0.12); color: #e2e8f0; }
.pb-btn.active { color: #38bdf8; }
.pb-btn-play {
  width: 42px; height: 42px;
  background: #fff;
  color: #0f172a;
}
.pb-btn-play:hover { background: #e2e8f0; color: #0f172a; transform: scale(1.05); }
.pb-btn-close:hover { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
.pb-loop-badge {
  position: absolute;
  top: 2px; right: 2px;
  font-size: 9px;
  font-weight: 700;
  color: #38bdf8;
}

.pb-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.pb-time {
  font-size: 11px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
  min-width: 80px;
  text-align: right;
}

/* Player bar slide transition */
.pb-slide-enter-active { transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1); }
.pb-slide-leave-active { transition: all 0.25s ease-in; }
.pb-slide-enter-from,
.pb-slide-leave-to { transform: translateY(100%); opacity: 0; }

/* Push up player bar on mobile for safe area */
@supports (padding-bottom: env(safe-area-inset-bottom)) {
  .player-bar { padding-bottom: env(safe-area-inset-bottom); }
}

@media (max-width: 768px) {
  .pb-inner { padding: 0 10px; gap: 8px; height: 52px; }
  .pb-time { display: none; }
  .pb-title { font-size: 12px; }
  .pb-btn { width: 32px; height: 32px; }
  .pb-btn-play { width: 38px; height: 38px; }
  .pb-visual { display: none; }
}

/* 全局移动端样式 */
@media (max-width: 768px) {
  /* 抽屉样式 */
  .mobile-drawer .el-drawer__header {
    margin-bottom: 0;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-base);
  }

  .mobile-drawer .el-drawer__body {
    padding: 0;
    background-color: var(--bg-primary);
  }

  .mobile-drawer .el-drawer {
    background-color: var(--bg-primary);
  }

  /* 捐赠对话框移动端适配 */
  .donate-dialog {
    width: 90% !important;
  }

  .qr-codes {
    flex-direction: column;
    gap: 20px;
  }

  .qr-item img {
    width: 120px;
    height: 120px;
  }

  /* 工具页面通用适配 */
  .tool-container {
    gap: 12px !important;
  }

  .tool-header {
    flex-direction: column;
    align-items: flex-start !important;
  }

  .tool-header h2 {
    font-size: 20px !important;
  }

  .tool-header .actions {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .tool-header .actions .el-button {
    flex: 1;
    min-width: 80px;
  }

  /* 编辑器容器适配 */
  .editor-container {
    grid-template-columns: 1fr !important;
    min-height: auto !important;
  }

  .editor-panel {
    min-height: 250px !important;
  }

  .code-editor {
    min-height: 200px !important;
    font-size: 14px !important;
  }

  /* 选项行适配 */
  .options-row {
    flex-direction: column;
    align-items: stretch !important;
    gap: 12px !important;
  }

  .option-item {
    width: 100%;
    justify-content: space-between;
  }

  /* 结果区域适配 */
  .result-card {
    padding: 20px !important;
  }

  .url-display {
    font-size: 12px !important;
  }

  .qr-code {
    width: 120px !important;
    height: 120px !important;
  }

  /* 提示区域适配 */
  .tips-section {
    padding: 15px !important;
  }

  /* 表格适配 */
  .el-table {
    font-size: 12px;
  }

  /* 输入框适配 */
  .el-input,
  .el-select {
    width: 100% !important;
  }

  /* 按钮适配 */
  .el-button--large {
    padding: 12px 20px !important;
    font-size: 14px !important;
  }

  /* 对话框适配 */
  .el-dialog {
    width: 90% !important;
    margin: 5vh auto !important;
  }

  /* 标签适配 */
  .feature-hints {
    flex-wrap: wrap;
  }

  .feature-hints .el-tag {
    font-size: 11px !important;
  }
}

/* 小屏幕特殊处理 */
@media (max-width: 480px) {
  .mobile-header {
    padding: 0 10px;
  }

  .mobile-main {
    padding: 10px;
    padding-top: calc(56px + 10px); /* 头部高度 + 内边距 */
    padding-bottom: 90px; /* 底部留白 */
  }

  .tool-header .actions .el-button {
    font-size: 12px;
    padding: 8px 12px;
  }

  .tool-header .actions .el-button .el-icon {
    display: none;
  }

  .code-editor {
    font-size: 13px !important;
  }

  .result-info {
    flex-direction: column;
    gap: 8px !important;
  }
}
</style>
