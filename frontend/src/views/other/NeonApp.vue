<template>
  <div class="neon-app">
    <!-- Neon 使用独立的容器 -->
    <div id="neon-root"></div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { createRoot } from 'react-dom/client'
import React from 'react'
import { StrictMode } from 'react'

// 导入 neon 样式和 i18n
import '../neon/index.css'
import '../neon/locales/i18n'

// 导入 AppRouter
import { AppRouter } from '../../neon/router'

let reactRoot = null

onMounted(() => {
  // 使用独立的容器 ID，避免与 Vue 根容器冲突
  const container = document.getElementById('neon-root')
  if (container) {
    reactRoot = createRoot(container)
    reactRoot.render(
      React.createElement(StrictMode, null,
        React.createElement(AppRouter)
      )
    )
    console.log('Neon app mounted')
  } else {
    console.error('Neon root container not found')
  }
})

onUnmounted(() => {
  if (reactRoot) {
    reactRoot.unmount()
  }
})
</script>

<style>
.neon-app {
  width: 100%;
  height: 100vh;
  margin: -16px;
  background: #0a0a0a;
}
</style>
