<template>
  <div class="tool-container vibe-tool">
    <div class="tool-header">
      <h2><el-icon><VideoCamera /></el-icon> AI 动效</h2>
      <div class="header-actions">
        <el-button @click="refreshFrame">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
    </div>
    <div class="vibe-container">
      <iframe
        ref="iframeRef"
        src="/neon/index.html"
        frameborder="0"
        class="vibe-iframe"
        @load="onIframeLoad"
      ></iframe>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { VideoCamera, Refresh } from '@element-plus/icons-vue'

const iframeRef = ref(null)

const onIframeLoad = () => {
  console.log('Neon Vibe Motion loaded')
}

const refreshFrame = () => {
  if (iframeRef.value) {
    iframeRef.value.src = iframeRef.value.src
  }
}
</script>

<style scoped>
.vibe-tool {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color);
  flex-shrink: 0;
}

.tool-header h2 {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.vibe-container {
  flex: 1;
  overflow: hidden;
  background: #000;
}

.vibe-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #0a0a0a;
}
</style>
