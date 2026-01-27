<template>
  <el-container class="app-container" :class="{ 'fullscreen-mode': hideSidebar }">
    <!-- 移动端头部 -->
    <el-header v-if="isMobile && !hideSidebar" class="mobile-header">
      <div class="mobile-header-left">
        <el-icon :size="24" class="menu-trigger" @click="showDrawer = true"><Menu /></el-icon>
        <span class="mobile-title">DevTools</span>
      </div>
      <div class="mobile-header-right">
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
        background-color="#1e1e1e"
        text-color="#a0a0a0"
        active-text-color="#409eff"
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
    <el-aside v-if="!isMobile && !hideSidebar" :width="isCollapse ? '64px' : '200px'" class="sidebar">
      <div class="logo" @click="isCollapse = !isCollapse">
        <el-icon :size="24"><Tools /></el-icon>
        <span v-show="!isCollapse" class="logo-text">DevTools</span>
      </div>
      <el-menu
        :default-active="$route.path"
        :collapse="isCollapse"
        router
        class="sidebar-menu"
        background-color="#1e1e1e"
        text-color="#a0a0a0"
        active-text-color="#409eff"
      >
        <el-menu-item
          v-for="route in menuRoutes"
          :key="route.path"
          :index="route.path"
        >
          <el-icon><component :is="route.meta.icon" /></el-icon>
          <template #title>{{ route.meta.title }}</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主内容区 -->
    <el-main class="main-content" :class="{ 'mobile-main': isMobile }">
      <router-view v-slot="{ Component }">
        <keep-alive>
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
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Menu, Coffee } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const isCollapse = ref(false)
const showDrawer = ref(false)
const isMobile = ref(false)
const showDonateDialog = ref(false)

// 是否隐藏侧边栏（全屏模式）
const hideSidebar = computed(() => route.meta?.hideSidebar === true)

// 当前页面标题
const currentTitle = computed(() => {
  const currentRoute = menuRoutes.value.find(r => r.path === route.path)
  return currentRoute?.meta?.title || 'DevTools'
})

const menuRoutes = computed(() => {
  return router.options.routes.filter(route =>
    route.meta && route.meta.title && !route.path.includes(':')
  )
})

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
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
.app-container {
  height: 100vh;
  background-color: #121212;
  overflow: hidden;
}

.app-container.fullscreen-mode {
  display: block;
}

.app-container.fullscreen-mode .main-content {
  height: 100vh;
  padding: 0;
}

/* 移动端头部 */
.mobile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #1e1e1e;
  border-bottom: 1px solid #333;
  padding: 0 15px;
  height: 56px;
  position: sticky;
  top: 0;
  z-index: 100;
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
  color: #a0a0a0;
  font-size: 14px;
}

.current-tool {
  background: #333;
  padding: 4px 12px;
  border-radius: 12px;
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
}

.drawer-menu .el-menu-item {
  height: 50px;
  line-height: 50px;
  font-size: 16px;
}

/* PC端侧边栏 */
.sidebar {
  background-color: #1e1e1e;
  transition: width 0.3s;
  overflow: hidden;
  flex-shrink: 0;
  z-index: 10;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #409eff;
  cursor: pointer;
  border-bottom: 1px solid #333;
}

.logo-text {
  font-size: 18px;
  font-weight: bold;
}

.sidebar-menu {
  border-right: none;
  height: calc(100vh - 60px);
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 200px;
}

.sidebar {
  position: relative;
}

/* 捐赠对话框 */
.donate-content {
  text-align: center;
}

.donate-text {
  color: #666;
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
  color: #666;
  font-size: 14px;
}

/* 主内容区 */
.main-content {
  background-color: #121212;
  padding: 20px;
  color: #e0e0e0;
  height: 100vh;
  overflow-y: auto;
  flex: 1;
  min-width: 0;
}

.mobile-main {
  padding: 15px;
  padding-bottom: 80px; /* 为底部留出空间 */
  height: calc(100vh - 56px); /* 减去移动端头部高度 */
}

/* 页面底部 Footer */
.page-footer {
  margin-top: 40px;
  padding: 20px 0;
  border-top: 1px solid #333;
}

.footer-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
  color: #808080;
  font-size: 14px;
}

.footer-link {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #808080;
  text-decoration: none;
  cursor: pointer;
  transition: color 0.2s;
}

.footer-link:hover {
  color: #409eff;
}

.footer-link .github-icon {
  width: 16px;
  height: 16px;
}

.footer-link .el-icon {
  font-size: 16px;
}

.footer-divider {
  color: #404040;
}

/* 中等屏幕适配 */
@media (min-width: 768px) and (max-width: 1024px) {
  .sidebar-menu {
    height: calc(100vh - 60px);
  }
}

/* 移动端全局样式调整 */
@media (max-width: 768px) {
  .app-container {
    flex-direction: column;
    overflow: hidden;
  }
}
</style>

<style>
/* 全局移动端样式 */
@media (max-width: 768px) {
  /* 抽屉样式 */
  .mobile-drawer .el-drawer__header {
    margin-bottom: 0;
    padding: 16px 20px;
    border-bottom: 1px solid #333;
  }

  .mobile-drawer .el-drawer__body {
    padding: 0;
    background-color: #1e1e1e;
  }

  .mobile-drawer .el-drawer {
    background-color: #1e1e1e;
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
