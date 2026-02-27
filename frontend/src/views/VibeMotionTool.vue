<template>
  <div class="tool-container vibe-tool">
    <div class="vibe-container">
      <iframe
        ref="iframeRef"
        src="/neon/index.html"
        frameborder="0"
        class="vibe-iframe"
        sandbox="allow-forms allow-modals allow-popups allow-presentation allow-same-origin allow-scripts"
        allow="accelerometer; camera; encrypted-media; gyroscope; microphone; autoplay; fullscreen"
        @load="onIframeLoad"
      ></iframe>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const iframeRef = ref(null)

const onIframeLoad = () => {
  console.log('Neon Vibe Motion loaded')
}

onMounted(() => {
  // 确保 iframe 正确加载
  if (iframeRef.value) {
    iframeRef.value.addEventListener('load', () => {
      console.log('iframe loaded successfully')
    })
  }
})
</script>

<style scoped>
.vibe-tool {
  display: flex;
  flex-direction: column;
  height: 100%;
  margin: -16px;
}

.vibe-container {
  flex: 1;
  overflow: hidden;
  background: #0a0a0a;
}

.vibe-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #0a0a0a;
}
</style>
