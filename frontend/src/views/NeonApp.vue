<template>
  <div class="neon-app" ref="containerRef">
    <div id="neon-root"></div>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { createRoot } from 'react-dom/client'
import React from 'react'
import { StrictMode } from 'react'

// 导入 React 应用
import App from '../neon/App'

// 导入 neon 样式和 i18n
import '../neon/index.css'
import '../neon/locales/i18n'

// 导入路由器
import { AppRouter } from '../neon/router'

const containerRef = ref(null)
let reactRoot = null

onMounted(() => {
  const container = document.getElementById('neon-root')
  if (container) {
    try {
      reactRoot = createRoot(container)
      reactRoot.render(
        React.createElement(StrictMode, null,
          React.createElement(AppRouter)
        )
      )
    } catch (e) {
      console.error('Failed to render Neon app:', e)
    }
  }
})

onBeforeUnmount(() => {
  if (reactRoot) {
    try {
      reactRoot.unmount()
    } catch (e) {
      console.error('Failed to unmount Neon app:', e)
    }
  }
})
</script>

<style scoped>
.neon-app {
  width: 100%;
  height: 100%;
  margin: -16px;
  overflow: hidden;
  background: #0a0a0a;
}

.neon-app :deep(#neon-root) {
  width: 100%;
  height: 100%;
}

.neon-app :deep(.h-screen) {
  height: 100% !important;
}

.neon-app :deep(html), .neon-app :deep(body) {
  height: 100%;
  margin: 0;
  padding: 0;
}
</style>
