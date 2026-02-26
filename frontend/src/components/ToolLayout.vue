<template>
  <div class="tool-layout" :class="{ 'is-fullscreen': fullscreen }">
    <!-- 工具头部 -->
    <div class="tool-layout-header" v-if="title">
      <h2>{{ title }}</h2>
      <div class="tool-layout-actions">
        <slot name="actions" />
      </div>
    </div>

    <!-- 主要内容 -->
    <div class="tool-layout-content">
      <slot />
    </div>

    <!-- 底部额外内容 -->
    <div class="tool-layout-footer" v-if="$slots.footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  title: {
    type: String,
    default: ''
  }
})

const fullscreen = ref(false)

const toggleFullscreen = () => {
  fullscreen.value = !fullscreen.value
}

defineExpose({
  fullscreen,
  toggleFullscreen
})
</script>

<style scoped>
.tool-layout {
  display: flex;
  flex-direction: column;
  min-height: 500px;
}

.tool-layout.is-fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  background: var(--bg-base);
  padding: 20px;
}

.tool-layout-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding-bottom: 15px;
  flex-shrink: 0;
}

.tool-layout-header h2 {
  margin: 0;
  color: var(--text-primary);
}

.tool-layout-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.tool-layout-content {
  flex: 1;
  min-height: 0;
}

.tool-layout-footer {
  margin-top: 15px;
  flex-shrink: 0;
}
</style>
