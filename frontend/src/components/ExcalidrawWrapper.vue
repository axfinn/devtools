<template>
  <div ref="containerRef" class="excalidraw-container"></div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { Excalidraw, exportToBlob, exportToSvg, serializeAsJSON } from '@excalidraw/excalidraw'
import '@excalidraw/excalidraw/index.css'

const props = defineProps({
  initialData: {
    type: Object,
    default: null
  },
  viewModeEnabled: {
    type: Boolean,
    default: false
  },
  zenModeEnabled: {
    type: Boolean,
    default: false
  },
  theme: {
    type: String,
    default: 'dark'
  },
  langCode: {
    type: String,
    default: 'zh-CN'
  }
})

const emit = defineEmits(['change', 'ready'])

const containerRef = ref(null)
let reactRoot = null
let excalidrawAPI = null

const renderExcalidraw = () => {
  if (!containerRef.value) return

  const excalidrawProps = {
    initialData: props.initialData,
    viewModeEnabled: props.viewModeEnabled,
    zenModeEnabled: props.zenModeEnabled,
    theme: props.theme,
    langCode: props.langCode,
    excalidrawAPI: (api) => {
      excalidrawAPI = api
      emit('ready', api)
    },
    onChange: (elements, appState, files) => {
      emit('change', { elements, appState, files })
    },
    UIOptions: {
      canvasActions: {
        loadScene: true,
        export: { saveFileToDisk: true },
        saveAsImage: true
      }
    }
  }

  const element = React.createElement(Excalidraw, excalidrawProps)

  if (!reactRoot) {
    reactRoot = createRoot(containerRef.value)
  }
  reactRoot.render(element)
}

onMounted(() => {
  renderExcalidraw()
})

onUnmounted(() => {
  if (reactRoot) {
    reactRoot.unmount()
    reactRoot = null
  }
  excalidrawAPI = null
})

watch(() => props.initialData, () => {
  if (excalidrawAPI && props.initialData) {
    excalidrawAPI.updateScene(props.initialData)
  }
}, { deep: true })

watch(() => props.viewModeEnabled, (val) => {
  if (excalidrawAPI) {
    excalidrawAPI.updateScene({ appState: { viewModeEnabled: val } })
  }
})

watch(() => props.theme, (val) => {
  if (excalidrawAPI) {
    excalidrawAPI.updateScene({ appState: { theme: val } })
  }
})

// Expose methods to parent
const getSceneData = () => {
  if (!excalidrawAPI) return null
  const elements = excalidrawAPI.getSceneElements()
  const appState = excalidrawAPI.getAppState()
  const files = excalidrawAPI.getFiles()
  return { elements, appState, files }
}

const getSceneJSON = () => {
  if (!excalidrawAPI) return null
  const elements = excalidrawAPI.getSceneElements()
  const appState = excalidrawAPI.getAppState()
  const files = excalidrawAPI.getFiles()
  return serializeAsJSON(elements, appState, files, 'local')
}

const loadScene = (data) => {
  if (!excalidrawAPI) return
  excalidrawAPI.updateScene(data)
  if (data.files) {
    excalidrawAPI.addFiles(Object.values(data.files))
  }
}

const resetScene = () => {
  if (!excalidrawAPI) return
  excalidrawAPI.resetScene()
}

const exportToPng = async (options = {}) => {
  if (!excalidrawAPI) return null
  const elements = excalidrawAPI.getSceneElements()
  const appState = excalidrawAPI.getAppState()
  const files = excalidrawAPI.getFiles()

  return await exportToBlob({
    elements,
    appState: {
      ...appState,
      exportWithDarkMode: props.theme === 'dark',
      ...options.appState
    },
    files,
    mimeType: 'image/png',
    ...options
  })
}

const exportToSvgString = async (options = {}) => {
  if (!excalidrawAPI) return null
  const elements = excalidrawAPI.getSceneElements()
  const appState = excalidrawAPI.getAppState()
  const files = excalidrawAPI.getFiles()

  const svg = await exportToSvg({
    elements,
    appState: {
      ...appState,
      exportWithDarkMode: props.theme === 'dark',
      ...options.appState
    },
    files,
    ...options
  })

  return svg.outerHTML
}

const getAPI = () => excalidrawAPI

defineExpose({
  getSceneData,
  getSceneJSON,
  loadScene,
  resetScene,
  exportToPng,
  exportToSvgString,
  getAPI
})
</script>

<style scoped>
.excalidraw-container {
  width: 100%;
  height: 100%;
  min-height: 500px;
}

.excalidraw-container :deep(.excalidraw) {
  width: 100%;
  height: 100%;
}

.excalidraw-container :deep(.excalidraw .App-menu_top) {
  z-index: 10;
}
</style>
