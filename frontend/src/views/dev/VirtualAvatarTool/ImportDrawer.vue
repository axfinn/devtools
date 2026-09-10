<!--
  ImportDrawer.vue —— 导入抽屉(P2)
  两种入口:
    - 拖拽(整个抽屉 div 监听 dragover/drop)
    - 文件选择器(<input type="file">)
  接受 .glb / .gltf;选完文件 emit imported,父组件负责调 /api/avatar/models 上传。
-->
<template>
  <el-drawer
    :model-value="modelValue"
    @update:model-value="(v) => $emit('update:modelValue', v)"
    title="导入模型"
    direction="rtl"
    size="420px"
  >
    <div
      class="import-drop"
      :class="{ dragging }"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="onDrop"
      @click="triggerFilePicker"
    >
      <div class="drop-icon">📥</div>
      <div class="drop-title">把 .glb / .gltf 拖到这里</div>
      <div class="drop-sub">或者<span class="drop-link">点击选择文件</span></div>
      <input
        ref="fileInputRef"
        type="file"
        accept=".glb,.gltf"
        style="display:none"
        @change="onFileChange"
      />
    </div>

    <el-divider>已选文件</el-divider>

    <div v-if="!pendingFile" class="empty">尚未选择文件</div>
    <div v-else class="file-card">
      <div class="file-icon">📦</div>
      <div class="file-meta">
        <strong>{{ pendingFile.name }}</strong>
        <small>{{ fmtSize(pendingFile.size) }} · {{ ext(pendingFile.name) }}</small>
      </div>
      <el-input
        v-model="title"
        placeholder="模型标题(可选)"
        size="small"
        clearable
      />
      <el-button
        type="primary"
        size="small"
        :loading="uploading"
        :disabled="uploading"
        @click="onUpload"
      >
        上传
      </el-button>
    </div>
  </el-drawer>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { uploadModel } from './composable/useAssets.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'imported'])

const dragging = ref(false)
const pendingFile = ref(null)
const title = ref('')
const uploading = ref(false)
const fileInputRef = ref(null)

function triggerFilePicker() {
  fileInputRef.value?.click()
}

function onFileChange(e) {
  const f = e.target.files?.[0]
  if (f) acceptFile(f)
  e.target.value = ''
}

function onDrop(e) {
  dragging.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f) acceptFile(f)
}

function acceptFile(f) {
  const lower = f.name.toLowerCase()
  if (!lower.endsWith('.glb') && !lower.endsWith('.gltf')) {
    ElMessage.warning('只支持 .glb / .gltf 文件')
    return
  }
  pendingFile.value = f
  if (!title.value) title.value = f.name.replace(/\.[^.]+$/, '')
}

async function onUpload() {
  if (!pendingFile.value) return
  uploading.value = true
  try {
    const r = await uploadModel(pendingFile.value, { title: title.value, format: ext(pendingFile.value) })
    ElMessage.success(`上传成功: ${r.id}`)
    emit('imported', r)
    pendingFile.value = null
    title.value = ''
    emit('update:modelValue', false)
  } catch (e) {
    ElMessage.error('上传失败:' + (e.message || String(e)))
  } finally {
    uploading.value = false
  }
}

function fmtSize(b) {
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`
  return `${(b / 1024 / 1024).toFixed(1)} MB`
}

function ext(name) {
  const m = name.toLowerCase().match(/\.([^.]+)$/)
  return m ? m[1] : 'glb'
}
</script>

<style scoped>
.import-drop {
  border: 2px dashed var(--border-base, #404040);
  border-radius: 12px;
  padding: 40px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 200ms;
  background: var(--bg-secondary, #252525);
}
.import-drop:hover,
.import-drop.dragging {
  border-color: var(--color-primary, #409eff);
  background: rgba(64, 158, 255, 0.05);
}
.drop-icon {
  font-size: 40px;
  margin-bottom: 12px;
}
.drop-title {
  font-size: 14px;
  color: var(--text-primary, #e0e0e0);
  font-weight: 500;
  margin-bottom: 4px;
}
.drop-sub {
  font-size: 12px;
  color: var(--text-tertiary, #909399);
}
.drop-link {
  color: var(--color-primary, #409eff);
  margin-left: 4px;
  font-weight: 500;
}
.empty {
  font-size: 12px;
  color: var(--text-tertiary, #909399);
  text-align: center;
  padding: 24px 0;
}
.file-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: var(--bg-secondary, #252525);
  border: 1px solid var(--border-light, #333);
  border-radius: 8px;
}
.file-card > .file-icon {
  font-size: 24px;
  align-self: center;
}
.file-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  text-align: center;
}
.file-meta strong {
  font-size: 13px;
  color: var(--text-primary, #e0e0e0);
  word-break: break-all;
}
.file-meta small {
  font-size: 11px;
  color: var(--text-tertiary, #909399);
  font-family: 'JetBrains Mono', monospace;
}
</style>
