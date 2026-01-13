<template>
  <el-container class="app-container">
    <!-- 移动端头部 -->
    <el-header v-if="isMobile" class="mobile-header">
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
      v-if="isMobile"
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
    <el-aside v-if="!isMobile" :width="isCollapse ? '64px' : '200px'" class="sidebar">
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
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Menu } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const isCollapse = ref(false)
const showDrawer = ref(false)
const isMobile = ref(false)

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
  isMobile.value = window.innerWidth < 768
  if (!isMobile.value) {
    showDrawer.value = false
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
  min-height: 100vh;
  background-color: #121212;
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

/* 主内容区 */
.main-content {
  background-color: #121212;
  padding: 20px;
  color: #e0e0e0;
}

.mobile-main {
  padding: 15px;
  padding-bottom: 80px; /* 为底部留出空间 */
}

/* 移动端全局样式调整 */
@media (max-width: 768px) {
  .app-container {
    flex-direction: column;
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
