<template>
  <div class="neon-app" ref="containerRef">
    <div id="neon-root"></div>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { createRoot } from 'react-dom/client'
import React from 'react'

// 导入 React 应用
import App from '../neon/App'

const containerRef = ref(null)
let reactRoot = null

onMounted(() => {
  const container = document.getElementById('neon-root')
  if (container) {
    try {
      reactRoot = createRoot(container)
      reactRoot.render(React.createElement(App))
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
</style>
