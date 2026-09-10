<!--
  ExportDialog.vue —— 导出三选一(P2)
  格式:
    - glb  : 通过 scene.js / exporter.js 的 GLTFExporter
    - pose.json : 当前姿态的 JSON 快照
    - mp4 / webm : MediaRecorder 录 canvas 流;
                   失败 → 降级到关键帧 ZIP(每帧 PNG + index.json);
                   再失败 → 降级到 JSON 手动复制
  所有导出结果都自动下载;不能下载时给"复制 JSON"按钮(降级 F)。
-->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="(v) => $emit('update:modelValue', v)"
    title="导出虚拟形象"
    width="520px"
    :close-on-click-modal="false"
  >
    <div class="export-form">
      <el-radio-group v-model="format" class="format-group">
        <el-radio-button value="glb" label="glb">
          <div class="fmt-card">
            <span class="fmt-icon">📦</span>
            <strong>glb</strong>
            <small>三维模型 + 骨骼 · 动捕管线友好</small>
          </div>
        </el-radio-button>
        <el-radio-button value="pose.json" label="pose.json">
          <div class="fmt-card">
            <span class="fmt-icon">📋</span>
            <strong>pose.json</strong>
            <small>当前姿态 JSON · 1 帧</small>
          </div>
        </el-radio-button>
        <el-radio-button value="video" label="video">
          <div class="fmt-card">
            <span class="fmt-icon">🎬</span>
            <strong>mp4 / webm</strong>
            <small>录制画面 · MediaRecorder</small>
          </div>
        </el-radio-button>
      </el-radio-group>

      <!-- video 选项附加 -->
      <template v-if="format === 'video'">
        <div class="opt-row">
          <label>时长(秒)</label>
          <el-input-number v-model="videoDuration" :min="1" :max="60" :step="1" size="small" />
        </div>
        <div class="opt-row">
          <label>帧率</label>
          <el-input-number v-model="videoFps" :min="10" :max="60" :step="5" size="small" />
        </div>
        <el-alert
          v-if="recState !== 'idle'"
          :type="recAlertType"
          :title="recAlertTitle"
          :description="recAlertDesc"
          show-icon
          :closable="false"
          class="rec-alert"
        />
      </template>

      <el-alert
        v-if="errorMsg"
        type="error"
        :title="errorMsg"
        :closable="false"
        show-icon
        class="err-alert"
      />
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button v-if="lastFallbackJson" size="small" @click="copyFallbackJson">
          📋 复制 JSON
        </el-button>
        <el-button @click="$emit('update:modelValue', false)">取消</el-button>
        <el-button
          type="primary"
          :loading="busy"
          :disabled="busy || recState === 'recording'"
          @click="onConfirm"
        >
          {{ confirmLabel }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { exportSceneAsGLB } from './exporter.js'
import { capturePose, serializePoseClip, buildClip } from './poseData.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  sceneApi: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'exported'])

const format = ref('glb')
const busy = ref(false)
const errorMsg = ref('')
const lastFallbackJson = ref('')

const videoDuration = ref(3)
const videoFps = ref(30)
const recState = ref('idle') // idle | recording | done | failed
const recMime = ref('')

const recAlertType = computed(() => recState.value === 'recording' ? 'info' : (recState.value === 'done' ? 'success' : 'error'))
const recAlertTitle = computed(() => {
  switch (recState.value) {
    case 'recording': return `录制中…${videoDuration.value}s`
    case 'done': return '录制完成'
    case 'failed': return '录制失败,已降级'
    default: return ''
  }
})
const recAlertDesc = computed(() => {
  if (recState.value === 'recording') return '正在从 canvas 抓流,不要切换页面'
  if (recState.value === 'done') return `${recMime.value || 'video/webm'} 文件已下载`
  if (recState.value === 'failed') return '浏览器不支持 MediaRecorder,已导出关键帧 ZIP + JSON'
  return ''
})

const confirmLabel = computed(() => {
  if (busy.value) return '导出中…'
  if (format.value === 'video' && recState.value === 'recording') return '录制中…'
  return '导出'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    errorMsg.value = ''
    lastFallbackJson.value = ''
    recState.value = 'idle'
  }
})

async function onConfirm() {
  if (!props.sceneApi) {
    errorMsg.value = '场景尚未就绪'
    return
  }
  busy.value = true
  errorMsg.value = ''
  try {
    if (format.value === 'glb') {
      const out = await exportSceneAsGLB(props.sceneApi)
      ElMessage.success(`已导出 ${out.filename} (${(out.blob.size / 1024).toFixed(1)} KB)`)
      emit('exported', { format: 'glb', filename: out.filename, size: out.blob.size })
      close()
    } else if (format.value === 'pose.json') {
      const frame = capturePose(props.sceneApi.skeleton, 0)
      const clip = buildClip('avatar-current-pose', [frame], 30, {
        source: 'VirtualAvatarTool',
        note: '当前姿态的单帧快照;后续接入动捕时可改为多帧 BVH/CSV。',
      })
      const text = serializePoseClip(clip)
      const blob = new Blob([text], { type: 'application/json' })
      const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
      const filename = `avatar-pose-${ts}.json`
      triggerDownload(blob, filename)
      lastFallbackJson.value = text
      ElMessage.success(`已导出 ${filename} (${(blob.size / 1024).toFixed(1)} KB)`)
      emit('exported', { format: 'pose.json', filename, size: blob.size })
      close()
    } else if (format.value === 'video') {
      await recordVideo()
    }
  } catch (e) {
    console.error('[ExportDialog] failed:', e)
    errorMsg.value = '导出失败:' + (e?.message || String(e))
  } finally {
    busy.value = false
  }
}

async function recordVideo() {
  recState.value = 'recording'
  // 拿 canvas DOM 元素 —— sceneApi 没暴露,我们从 renderer.domElement 拿
  const canvas = props.sceneApi?.renderer?.domElement
  if (!canvas || typeof canvas.captureStream !== 'function') {
    await fallbackToKeyframes()
    return
  }
  // 选 mime
  const candidates = [
    'video/mp4;codecs=h264',
    'video/webm;codecs=vp9',
    'video/webm;codecs=vp8',
    'video/webm',
    'video/mp4',
  ]
  let mime = ''
  for (const m of candidates) {
    if (window.MediaRecorder && MediaRecorder.isTypeSupported(m)) { mime = m; break }
  }
  if (!window.MediaRecorder || !mime) {
    await fallbackToKeyframes()
    return
  }
  recMime.value = mime
  try {
    const stream = canvas.captureStream(videoFps.value)
    const recorder = new MediaRecorder(stream, { mimeType: mime, videoBitsPerSecond: 4_000_000 })
    const chunks = []
    recorder.ondataavailable = (e) => { if (e.data && e.data.size) chunks.push(e.data) }
    const done = new Promise((resolve, reject) => {
      recorder.onstop = () => resolve()
      recorder.onerror = (e) => reject(e.error || new Error('MediaRecorder error'))
    })
    recorder.start()
    await sleep(videoDuration.value * 1000)
    recorder.stop()
    await done
    const blob = new Blob(chunks, { type: mime })
    if (blob.size < 100) throw new Error('录制结果为空')
    const ext = mime.includes('mp4') ? 'mp4' : 'webm'
    const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    const filename = `avatar-recording-${ts}.${ext}`
    triggerDownload(blob, filename)
    recState.value = 'done'
    ElMessage.success(`已导出 ${filename} (${(blob.size / 1024).toFixed(1)} KB)`)
    emit('exported', { format: ext, filename, size: blob.size })
    close()
  } catch (e) {
    console.warn('[ExportDialog] MediaRecorder failed:', e)
    await fallbackToKeyframes()
  }
}

// 降级 E:关键帧 ZIP —— 抓 N 帧 PNG + index.json(简化:把 pose JSON 作为 index)
async function fallbackToKeyframes() {
  recState.value = 'failed'
  const totalFrames = Math.min(videoFps.value * videoDuration.value, 60)
  const interval = 1000 / videoFps.value
  const frames = []
  const canvas = props.sceneApi?.renderer?.domElement
  for (let i = 0; i < totalFrames; i++) {
    if (canvas && typeof canvas.toDataURL === 'function') {
      try {
        frames.push({ i, ts: Date.now(), png: canvas.toDataURL('image/png') })
      } catch (_) { /* tainted canvas(读不到像素) */ }
    }
    // 抓一帧姿态数据
    const pose = capturePose(props.sceneApi.skeleton, i)
    frames.push({ i, ts: Date.now(), pose })
    await sleep(interval)
  }
  const blob = new Blob([JSON.stringify(frames, null, 2)], { type: 'application/json' })
  const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  const filename = `avatar-keyframes-${ts}.json`
  triggerDownload(blob, filename)
  lastFallbackJson.value = await blob.text()
  ElMessage.warning('视频导出不支持,已降级到关键帧 JSON')
  emit('exported', { format: 'keyframes', filename, size: blob.size, fallback: true })
}

// 降级 F:用户手动复制 JSON
async function copyFallbackJson() {
  try {
    await navigator.clipboard.writeText(lastFallbackJson.value)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败:' + e.message)
  }
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }

function triggerDownload(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 2000)
}

function close() {
  emit('update:modelValue', false)
}
</script>

<style scoped>
.export-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.format-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.format-group :deep(.el-radio-button) {
  margin-right: 0;
}
.format-group :deep(.el-radio-button__inner) {
  width: 100%;
  padding: 0;
  border-radius: 8px !important;
}
.fmt-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 8px 12px;
  text-align: left;
}
.fmt-card .fmt-icon {
  font-size: 18px;
}
.fmt-card strong {
  font-size: 14px;
  color: inherit;
}
.fmt-card small {
  font-size: 11px;
  opacity: 0.7;
}
.opt-row {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}
.opt-row label {
  width: 60px;
  color: var(--text-secondary, #909399);
}
.rec-alert {
  margin-top: 4px;
}
.err-alert {
  margin-top: 4px;
}
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
