<template>
  <el-card class="speech-card">
    <template #header>
      <div class="speech-header">
        <div>
          <div class="speech-title">MiniMax Speech</div>
          <div class="speech-subtitle">同步/异步 TTS、文件管理、音色设计与复刻</div>
        </div>
        <div class="speech-actions">
          <el-input
            v-model="debugApiKey"
            type="password"
            show-password
            placeholder="用于语音调试的 API Key（可选，优先于超管密码）"
            style="width: 320px;"
          />
          <el-tag :type="debugApiKey ? 'success' : superAdminPassword ? 'warning' : 'info'">
            {{ debugApiKey ? '当前使用 API Key 调试' : superAdminPassword ? '当前使用超管调试' : '尚未提供调试凭证' }}
          </el-tag>
          <el-button @click="loadDocs">查看文档</el-button>
        </div>
      </div>
    </template>

    <el-alert type="info" :closable="false" show-icon>
      <template #title>
        语音网关优先复用当前页面签发的 API Key；未填写调试 Key 时，会回退到超级管理员密码进行页面调试。
      </template>
    </el-alert>

    <el-tabs v-model="activeTab" class="speech-tabs">
      <el-tab-pane label="同步 TTS" name="sync">
        <div class="speech-grid two-col">
          <div>
            <el-form label-position="top">
              <el-form-item label="模型">
                <el-select v-model="syncForm.model" style="width: 220px;">
                  <el-option v-for="model in speechModels" :key="model" :label="model" :value="model" />
                </el-select>
              </el-form-item>
              <el-form-item label="音色">
                <el-select v-model="syncForm.voiceId" filterable allow-create default-first-option placeholder="输入或选择 voice_id" style="width: 320px;">
                  <el-option v-for="voice in voiceOptions" :key="voice.voice_id" :label="voiceLabel(voice)" :value="voice.voice_id" />
                </el-select>
              </el-form-item>
              <el-form-item label="文本">
                <el-input v-model="syncForm.text" type="textarea" :rows="5" placeholder="请输入要合成的文本" />
              </el-form-item>
              <div class="speech-inline-row">
                <el-form-item label="语速">
                  <el-slider v-model="syncForm.speed" :min="0.5" :max="2" :step="0.1" show-stops style="width: 220px;" />
                </el-form-item>
                <el-form-item label="格式">
                  <el-select v-model="syncForm.format" style="width: 120px;">
                    <el-option label="mp3" value="mp3" />
                    <el-option label="wav" value="wav" />
                    <el-option label="flac" value="flac" />
                    <el-option label="pcm" value="pcm" />
                  </el-select>
                </el-form-item>
                <el-form-item label="采样率">
                  <el-select v-model="syncForm.sampleRate" style="width: 140px;">
                    <el-option :label="16000" :value="16000" />
                    <el-option :label="22050" :value="22050" />
                    <el-option :label="32000" :value="32000" />
                    <el-option :label="44100" :value="44100" />
                  </el-select>
                </el-form-item>
              </div>
              <div class="speech-inline-row actions-row">
                <el-button type="primary" :loading="syncing" @click="synthesizeSync">合成语音</el-button>
                <el-button @click="loadVoices">刷新音色</el-button>
              </div>
            </el-form>
          </div>
          <div>
            <div v-if="syncAudioUrl" class="audio-result">
              <div class="result-title">音频预览</div>
              <audio :src="syncAudioUrl" controls style="width: 100%;" />
              <el-button type="primary" size="small" :loading="sharingSync" @click="shareSyncResult" style="margin-top: 10px;">保存分享</el-button>
            </div>
            <div class="result-title">最近响应</div>
            <pre class="speech-json">{{ syncResponsePretty }}</pre>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="异步 TTS" name="async">
        <div class="speech-grid two-col">
          <div>
            <el-form label-position="top">
              <el-form-item label="模型">
                <el-select v-model="asyncForm.model" style="width: 220px;">
                  <el-option v-for="model in speechModels" :key="model" :label="model" :value="model" />
                </el-select>
              </el-form-item>
              <el-form-item label="音色">
                <el-select v-model="asyncForm.voiceId" filterable allow-create default-first-option placeholder="输入或选择 voice_id" style="width: 320px;">
                  <el-option v-for="voice in voiceOptions" :key="voice.voice_id" :label="voiceLabel(voice)" :value="voice.voice_id" />
                </el-select>
              </el-form-item>
              <el-form-item label="输入方式">
                <el-radio-group v-model="asyncForm.mode">
                  <el-radio value="text">直接文本</el-radio>
                  <el-radio value="file">文本文件 file_id</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item v-if="asyncForm.mode === 'text'" label="长文本内容">
                <el-input v-model="asyncForm.text" type="textarea" :rows="6" placeholder="适合较长文本，提交后返回 task_id 再轮询" />
              </el-form-item>
              <el-form-item v-else label="文本文件 file_id">
                <el-select v-model="asyncForm.textFileId" filterable allow-create default-first-option placeholder="优先从已上传文件中选择" style="width: 320px;">
                  <el-option v-for="file in textFiles" :key="file.file_id" :label="fileLabel(file)" :value="file.file_id" />
                </el-select>
              </el-form-item>
              <div class="speech-inline-row actions-row">
                <el-button type="primary" :loading="creatingTask" @click="createAsyncTask">创建任务</el-button>
                <el-button :loading="pollingTask" @click="pollCurrentTask">轮询当前任务</el-button>
                <el-button @click="loadTasks">刷新任务</el-button>
              </div>
            </el-form>
          </div>
          <div>
            <div v-if="asyncAudioUrl" class="audio-result">
              <div class="result-title">任务音频</div>
              <audio :src="asyncAudioUrl" controls style="width: 100%;" />
              <el-button type="primary" size="small" :loading="sharingAsync" @click="shareAsyncResult" style="margin-top: 10px;">保存分享</el-button>
            </div>
            <div class="result-title">当前任务</div>
            <pre class="speech-json">{{ currentTaskPretty }}</pre>
          </div>
        </div>
        <el-table :data="tasks" v-loading="loadingTasks" stripe size="small" max-height="320" style="margin-top: 18px;">
          <el-table-column prop="task_id" label="任务 ID" min-width="180">
            <template #default="{ row }">
              <el-button text type="primary" @click="openTask(row)">{{ row.task_id }}</el-button>
            </template>
          </el-table-column>
          <el-table-column prop="model" label="模型" width="140" />
          <el-table-column prop="status" label="状态" width="120" />
          <el-table-column prop="output_file_id" label="输出 file_id" min-width="180" />
          <el-table-column label="操作" width="180">
            <template #default="{ row }">
              <el-button text @click="openTask(row)">查看</el-button>
              <el-button v-if="row.output_file_id" text type="success" @click="previewTaskFile(row.output_file_id)">预览音频</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="文件管理" name="files">
        <div class="speech-grid two-col">
          <div>
            <el-form label-position="top">
              <el-form-item label="上传用途">
                <el-select v-model="filePurpose" style="width: 220px;">
                  <el-option label="voice_clone" value="voice_clone" />
                  <el-option label="prompt_audio" value="prompt_audio" />
                  <el-option label="t2a_async_input" value="t2a_async_input" />
                </el-select>
              </el-form-item>
              <el-form-item label="文件">
                <input type="file" @change="onFileSelected" />
              </el-form-item>
              <div class="speech-inline-row actions-row">
                <el-button type="primary" :loading="uploadingFile" @click="uploadSpeechFile">上传文件</el-button>
                <el-button @click="loadFiles">刷新文件列表</el-button>
              </div>
            </el-form>
          </div>
          <div>
            <div v-if="filePreviewUrl" class="audio-result">
              <div class="result-title">文件预览</div>
              <audio :src="filePreviewUrl" controls style="width: 100%;" />
            </div>
            <div class="result-title">最近文件响应</div>
            <pre class="speech-json">{{ fileResponsePretty }}</pre>
          </div>
        </div>
        <el-table :data="files" v-loading="loadingFiles" stripe size="small" max-height="320" style="margin-top: 18px;">
          <el-table-column prop="file_id" label="file_id" min-width="180" />
          <el-table-column prop="purpose" label="用途" width="150" />
          <el-table-column prop="file_name" label="文件名" min-width="160" />
          <el-table-column prop="bytes" label="大小" width="120">
            <template #default="{ row }">{{ formatBytes(row.bytes || row.size || 0) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220">
            <template #default="{ row }">
              <el-button text type="primary" @click="previewSpeechFile(row.file_id)">预览</el-button>
              <el-button text @click="copyText(row.file_id)">复制 ID</el-button>
              <el-button text type="danger" @click="deleteSpeechFile(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="音色管理" name="voices">
        <div class="speech-grid two-col">
          <div>
            <el-card shadow="never" class="speech-inner-card">
              <template #header>
                <div class="inner-card-header">
                  <span>音色设计</span>
                  <el-button text @click="loadVoices">刷新音色</el-button>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="voice_id">
                  <el-input v-model="designForm.voiceId" placeholder="例如 custom-designer-001" />
                </el-form-item>
                <el-form-item label="设计描述 prompt">
                  <el-input v-model="designForm.prompt" type="textarea" :rows="4" placeholder="描述你想要的音色风格" />
                </el-form-item>
                <el-form-item label="试听文本 preview_text">
                  <el-input v-model="designForm.previewText" type="textarea" :rows="3" placeholder="用于生成试听的文本" />
                </el-form-item>
                <el-button type="primary" :loading="designingVoice" @click="designVoice">提交设计</el-button>
              </el-form>
            </el-card>

            <el-card shadow="never" class="speech-inner-card" style="margin-top: 16px;">
              <template #header>
                <span>音色复刻</span>
              </template>
              <el-form label-position="top">
                <el-form-item label="voice_id">
                  <el-input v-model="cloneForm.voiceId" placeholder="例如 custom-clone-001" />
                </el-form-item>
                <el-form-item label="源音频 file_id">
                  <el-select v-model="cloneForm.fileId" filterable allow-create default-first-option placeholder="优先选择 voice_clone 类型文件" style="width: 100%;">
                    <el-option v-for="file in cloneableFiles" :key="file.file_id" :label="fileLabel(file)" :value="file.file_id" />
                  </el-select>
                </el-form-item>
                <el-form-item label="参考 prompt_audio（可选）">
                  <el-select v-model="cloneForm.promptAudio" filterable clearable allow-create default-first-option placeholder="可选：上传 prompt_audio 文件后选择" style="width: 100%;">
                    <el-option v-for="file in promptAudioFiles" :key="file.file_id" :label="fileLabel(file)" :value="file.file_id" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-checkbox v-model="cloneForm.needNoiseReduction">启用降噪</el-checkbox>
                </el-form-item>
                <el-button type="primary" :loading="cloningVoice" @click="cloneVoice">提交复刻</el-button>
              </el-form>
            </el-card>
          </div>

          <div>
            <div class="result-title">最近音色响应</div>
            <pre class="speech-json">{{ voiceResponsePretty }}</pre>
            <el-table :data="voiceOptions" v-loading="loadingVoices" stripe size="small" max-height="420" style="margin-top: 16px;">
              <el-table-column prop="voice_id" label="voice_id" min-width="160" />
              <el-table-column prop="name" label="名称" min-width="140" />
              <el-table-column prop="voice_type" label="类型" width="120" />
              <el-table-column label="操作" width="220">
                <template #default="{ row }">
                  <el-button text @click="useVoice(row)">用于合成</el-button>
                  <el-button text @click="copyText(row.voice_id)">复制 ID</el-button>
                  <el-button v-if="canDeleteVoice(row)" text type="danger" @click="deleteVoice(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="docsVisible" width="960px" title="MiniMax Speech 文档">
      <div v-if="speechDocs">
        <el-alert :title="speechDocs.summary" type="info" :closable="false" />
        <el-descriptions :column="1" border style="margin-top: 16px;">
          <el-descriptions-item label="Base URL">{{ speechDocs.base_url }}</el-descriptions-item>
          <el-descriptions-item label="Upstream">{{ speechDocs.upstream }}</el-descriptions-item>
          <el-descriptions-item label="鉴权">{{ speechDocs.auth?.api_key }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="speechDocs.routes || []" stripe size="small" style="margin-top: 16px;">
          <el-table-column prop="method" label="Method" width="120" />
          <el-table-column prop="path" label="Path" min-width="260" />
          <el-table-column prop="description" label="Description" min-width="240" />
        </el-table>
        <pre class="speech-json" style="margin-top: 16px;">{{ JSON.stringify(speechDocs.examples || {}, null, 2) }}</pre>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps({
  superAdminPassword: {
    type: String,
    default: ''
  },
  prefillApiKey: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['share'])

const activeTab = ref('sync')
const debugApiKey = ref('')
const docsVisible = ref(false)
const speechDocs = ref(null)
const superAdminPassword = computed(() => props.superAdminPassword || '')

const speechModels = [
  'speech-01-hd',
  'speech-01-turbo',
  'speech-02-hd',
  'speech-02-turbo',
  'speech-2.6-hd',
  'speech-2.6-turbo',
  'speech-2.8-hd',
  'speech-2.8-turbo'
]

const fileListPurposes = ['voice_clone', 'prompt_audio', 't2a_async_input']
const activeCredential = computed(() => debugApiKey.value.trim() || props.superAdminPassword.trim())

const voiceOptions = ref([])
const files = ref([])
const tasks = ref([])
const loadingVoices = ref(false)
const loadingFiles = ref(false)
const loadingTasks = ref(false)

const syncing = ref(false)
const creatingTask = ref(false)
const pollingTask = ref(false)
const uploadingFile = ref(false)
const designingVoice = ref(false)
const cloningVoice = ref(false)
const sharingSync = ref(false)
const sharingAsync = ref(false)

const syncResponse = ref(null)
const syncAudioUrl = ref('')
const currentTask = ref(null)
const asyncAudioUrl = ref('')
const fileResponse = ref(null)
const filePreviewUrl = ref('')
const voiceResponse = ref(null)

const syncForm = reactive({
  model: 'speech-2.8-hd',
  voiceId: 'male-qn-qingse',
  text: '你好，这是 DevTools MiniMax 语音网关的同步测试。',
  speed: 1,
  format: 'mp3',
  sampleRate: 32000
})

const asyncForm = reactive({
  model: 'speech-2.8-hd',
  voiceId: 'male-qn-qingse',
  mode: 'text',
  text: '这是一个异步长文本语音任务示例。你可以把这里替换为更长的播报内容。',
  textFileId: ''
})

const filePurpose = ref('voice_clone')
const selectedFile = ref(null)

const designForm = reactive({
  voiceId: 'custom-designer-001',
  prompt: '一位温柔、理性、清晰的中文讲解女声，适合教程和知识播报。',
  previewText: '你好，欢迎使用 DevTools MiniMax 语音网关。'
})

const cloneForm = reactive({
  voiceId: 'custom-clone-001',
  fileId: '',
  promptAudio: '',
  needNoiseReduction: true
})

const syncResponsePretty = computed(() => pretty(syncResponse.value))
const currentTaskPretty = computed(() => pretty(currentTask.value))
const fileResponsePretty = computed(() => pretty(fileResponse.value))
const voiceResponsePretty = computed(() => pretty(voiceResponse.value))

const cloneableFiles = computed(() => files.value.filter(file => (file.purpose || '').includes('voice_clone')))
const promptAudioFiles = computed(() => files.value.filter(file => (file.purpose || '').includes('prompt_audio')))
const textFiles = computed(() => files.value.filter(file => (file.purpose || '').includes('t2a_async_input')))

watch(() => props.prefillApiKey, (value) => {
  if (!debugApiKey.value && value) {
    debugApiKey.value = value
  }
}, { immediate: true })

watch(activeCredential, (value, previousValue) => {
  if (!value || value === previousValue) return
  void hydrateSpeechResources()
}, { immediate: true })

onBeforeUnmount(() => {
  resetAudio('sync')
  resetAudio('async')
  resetAudio('file')
})

function hasAnyCredential() {
  return Boolean(debugApiKey.value.trim() || props.superAdminPassword.trim())
}

function requireCredential() {
  if (!hasAnyCredential()) {
    ElMessage.error('请先填写调试 API Key，或在父页面输入超级管理员密码')
    return false
  }
  return true
}

function jsonHeaders() {
  if (debugApiKey.value.trim()) {
    return {
      Authorization: `Bearer ${debugApiKey.value.trim()}`,
      'Content-Type': 'application/json'
    }
  }
  if (props.superAdminPassword.trim()) {
    return {
      'X-Super-Admin-Password': props.superAdminPassword.trim(),
      'Content-Type': 'application/json'
    }
  }
  return null
}

function authHeaders() {
  if (debugApiKey.value.trim()) {
    return {
      Authorization: `Bearer ${debugApiKey.value.trim()}`
    }
  }
  if (props.superAdminPassword.trim()) {
    return {
      'X-Super-Admin-Password': props.superAdminPassword.trim()
    }
  }
  return null
}

async function hydrateSpeechResources() {
  await Promise.allSettled([loadVoices(), loadFiles(), loadTasks()])
}

async function loadDocs() {
  const res = await fetch('/api/minimax/speech/docs')
  const data = await res.json()
  speechDocs.value = data
  docsVisible.value = true
}

async function loadVoices() {
  if (!requireCredential()) return
  loadingVoices.value = true
  try {
    const res = await fetch('/api/minimax/speech/v1/get_voice', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify({ voice_type: 'all' })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '加载音色失败')
    voiceResponse.value = data
    voiceOptions.value = normalizeVoices(data)
  } catch (err) {
    ElMessage.error(err.message || '加载音色失败')
  } finally {
    loadingVoices.value = false
  }
}

async function loadFiles() {
  if (!requireCredential()) return
  loadingFiles.value = true
  try {
    const responses = await Promise.all(fileListPurposes.map(async (purpose) => {
      const res = await fetch(`/api/minimax/speech/v1/files/list?purpose=${encodeURIComponent(purpose)}`, {
        headers: authHeaders()
      })
      const data = await safeJson(res)
      if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || `加载 ${purpose} 文件失败`)
      return { purpose, data }
    }))
    const mergedFiles = mergeByFileID(responses.flatMap(({ data }) => normalizeFiles(data)))
    fileResponse.value = {
      base_resp: { status_code: 0, status_msg: 'merged' },
      purposes: responses.map(item => item.purpose),
      files: mergedFiles,
      raw_responses: responses
    }
    files.value = mergedFiles
  } catch (err) {
    ElMessage.error(err.message || '加载文件失败')
  } finally {
    loadingFiles.value = false
  }
}

async function loadTasks() {
  if (!requireCredential()) return
  loadingTasks.value = true
  try {
    const res = await fetch('/api/minimax/speech/tasks?limit=50', {
      headers: authHeaders()
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载任务失败')
    tasks.value = data.tasks || []
  } catch (err) {
    ElMessage.error(err.message || '加载任务失败')
  } finally {
    loadingTasks.value = false
  }
}

async function synthesizeSync() {
  if (!requireCredential()) return
  if (!syncForm.text.trim()) {
    ElMessage.error('请输入要合成的文本')
    return
  }
  syncing.value = true
  resetAudio('sync')
  try {
    const payload = {
      model: syncForm.model,
      text: syncForm.text.trim(),
      voice_setting: {
        voice_id: syncForm.voiceId,
        speed: syncForm.speed
      },
      audio_setting: {
        format: syncForm.format,
        sample_rate: syncForm.sampleRate
      }
    }
    const res = await fetch('/api/minimax/speech/v1/t2a_v2', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await res.json()
    syncResponse.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '合成失败')
    await setAudioFromSpeechResponse(data, 'sync')
    ElMessage.success('同步语音合成成功')
  } catch (err) {
    ElMessage.error(err.message || '同步语音合成失败')
  } finally {
    syncing.value = false
  }
}

async function createAsyncTask() {
  if (!requireCredential()) return
  if (asyncForm.mode === 'text' && !asyncForm.text.trim()) {
    ElMessage.error('请输入异步任务文本')
    return
  }
  if (asyncForm.mode === 'file' && !asyncForm.textFileId.trim()) {
    ElMessage.error('请选择文本文件 file_id')
    return
  }
  creatingTask.value = true
  try {
    const payload = {
      model: asyncForm.model,
      voice_id: asyncForm.voiceId,
      voice_setting: {
        voice_id: asyncForm.voiceId
      }
    }
    if (asyncForm.mode === 'text') {
      payload.text = asyncForm.text.trim()
    } else {
      payload.text_file_id = asyncForm.textFileId.trim()
    }
    const res = await fetch('/api/minimax/speech/v1/t2a_async_v2', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await res.json()
    currentTask.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '创建任务失败')
    const taskId = extractTaskId(data)
    if (taskId) {
      ElMessage.success(`任务已创建: ${taskId}`)
      await loadTasks()
    } else {
      ElMessage.success('任务已创建')
    }
  } catch (err) {
    ElMessage.error(err.message || '创建任务失败')
  } finally {
    creatingTask.value = false
  }
}

async function pollCurrentTask() {
  if (!requireCredential()) return
  const taskId = extractTaskId(currentTask.value)
  if (!taskId) {
    ElMessage.error('当前没有可轮询的任务')
    return
  }
  pollingTask.value = true
  try {
    const res = await fetch(`/api/minimax/speech/v1/query/t2a_async_query_v2?task_id=${encodeURIComponent(taskId)}`, {
      headers: authHeaders()
    })
    const data = await res.json()
    currentTask.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '轮询任务失败')
    const fileId = extractFileId(data)
    if (fileId) {
      await previewTaskFile(fileId)
    }
    await loadTasks()
  } catch (err) {
    ElMessage.error(err.message || '轮询任务失败')
  } finally {
    pollingTask.value = false
  }
}

async function openTask(row) {
  if (!requireCredential()) return
  try {
    const res = await fetch(`/api/minimax/speech/tasks/${encodeURIComponent(row.task_id)}`, {
      headers: authHeaders()
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载任务详情失败')
    currentTask.value = data.raw_response || data.task || data
    if (data.task?.output_file_id) {
      await previewTaskFile(data.task.output_file_id)
    }
  } catch (err) {
    ElMessage.error(err.message || '加载任务详情失败')
  }
}

async function uploadSpeechFile() {
  if (!requireCredential()) return
  if (!selectedFile.value) {
    ElMessage.error('请先选择文件')
    return
  }
  uploadingFile.value = true
  try {
    const formData = new FormData()
    formData.append('purpose', filePurpose.value)
    formData.append('file', selectedFile.value)
    const res = await fetch('/api/minimax/speech/v1/files/upload', {
      method: 'POST',
      headers: authHeaders(),
      body: formData
    })
    const data = await res.json()
    fileResponse.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '上传文件失败')
    const fileId = extractFileId(data)
    if (fileId) {
      cloneForm.fileId = cloneForm.fileId || fileId
      if (filePurpose.value === 't2a_async_input') {
        asyncForm.textFileId = asyncForm.textFileId || fileId
      }
    }
    ElMessage.success('文件上传成功')
    selectedFile.value = null
    await loadFiles()
  } catch (err) {
    ElMessage.error(err.message || '上传文件失败')
  } finally {
    uploadingFile.value = false
  }
}

async function previewSpeechFile(fileId) {
  if (!requireCredential()) return
  resetAudio('file')
  try {
    const res = await fetch(`/api/minimax/speech/v1/files/retrieve_content?file_id=${encodeURIComponent(fileId)}`, {
      headers: authHeaders()
    })
    if (!res.ok) {
      const data = await safeJson(res)
      throw new Error(data.error || '读取文件失败')
    }
    const blob = await res.blob()
    filePreviewUrl.value = URL.createObjectURL(blob)
  } catch (err) {
    ElMessage.error(err.message || '读取文件失败')
  }
}

async function previewTaskFile(fileId) {
  if (!requireCredential()) return
  resetAudio('async')
  try {
    const res = await fetch(`/api/minimax/speech/v1/files/retrieve_content?file_id=${encodeURIComponent(fileId)}`, {
      headers: authHeaders()
    })
    if (!res.ok) {
      const data = await safeJson(res)
      throw new Error(data.error || '读取任务音频失败')
    }
    const blob = await res.blob()
    asyncAudioUrl.value = URL.createObjectURL(blob)
  } catch (err) {
    ElMessage.error(err.message || '读取任务音频失败')
  }
}

async function deleteSpeechFile(row) {
  if (!requireCredential()) return
  try {
    await ElMessageBox.confirm(`确认删除文件 ${row.file_id} ?`, '提示', { type: 'warning' })
    const res = await fetch('/api/minimax/speech/v1/files/delete', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify({ file_id: row.file_id })
    })
    const data = await res.json()
    fileResponse.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '删除文件失败')
    ElMessage.success('文件已删除')
    await loadFiles()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error(err.message || '删除文件失败')
    }
  }
}

async function designVoice() {
  if (!requireCredential()) return
  if (!designForm.voiceId.trim() || !designForm.prompt.trim() || !designForm.previewText.trim()) {
    ElMessage.error('请完整填写音色设计参数')
    return
  }
  designingVoice.value = true
  try {
    const res = await fetch('/api/minimax/speech/v1/voice_design', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify({
        voice_id: designForm.voiceId.trim(),
        prompt: designForm.prompt.trim(),
        preview_text: designForm.previewText.trim()
      })
    })
    const data = await res.json()
    voiceResponse.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '音色设计失败')
    ElMessage.success('音色设计请求已提交')
    await loadVoices()
  } catch (err) {
    ElMessage.error(err.message || '音色设计失败')
  } finally {
    designingVoice.value = false
  }
}

async function cloneVoice() {
  if (!requireCredential()) return
  if (!cloneForm.voiceId.trim() || !cloneForm.fileId.trim()) {
    ElMessage.error('请填写 voice_id 并选择源音频 file_id')
    return
  }
  cloningVoice.value = true
  try {
    const payload = {
      voice_id: cloneForm.voiceId.trim(),
      file_id: cloneForm.fileId.trim(),
      need_noise_reduction: cloneForm.needNoiseReduction
    }
    if (cloneForm.promptAudio.trim()) {
      payload.prompt_audio = cloneForm.promptAudio.trim()
    }
    const res = await fetch('/api/minimax/speech/v1/voice_clone', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await res.json()
    voiceResponse.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '音色复刻失败')
    ElMessage.success('音色复刻请求已提交')
    await loadVoices()
  } catch (err) {
    ElMessage.error(err.message || '音色复刻失败')
  } finally {
    cloningVoice.value = false
  }
}

async function deleteVoice(row) {
  if (!requireCredential()) return
  try {
    await ElMessageBox.confirm(`确认删除音色 ${row.voice_id} ?`, '提示', { type: 'warning' })
    const res = await fetch('/api/minimax/speech/v1/delete_voice', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify({
        voice_id: row.voice_id,
        voice_type: deleteVoiceType(row)
      })
    })
    const data = await res.json()
    voiceResponse.value = data
    if (!res.ok) throw new Error(data.error || data.base_resp?.status_msg || '删除音色失败')
    ElMessage.success('音色已删除')
    await loadVoices()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error(err.message || '删除音色失败')
    }
  }
}

function useVoice(row) {
  syncForm.voiceId = row.voice_id
  asyncForm.voiceId = row.voice_id
  activeTab.value = 'sync'
  ElMessage.success(`已选中音色 ${row.voice_id}`)
}

function canDeleteVoice(row) {
  const type = String(row.voice_type || row.type || '').toLowerCase()
  return ['voice_cloning', 'voice_generation', 'custom', 'user', 'clone', 'generated'].includes(type)
}

function deleteVoiceType(row) {
  return String(row.voice_type || row.type || '').toLowerCase() === 'voice_generation'
    ? 'voice_generation'
    : 'voice_cloning'
}

function speechDataRoot(payload) {
  return payload?.data || payload || {}
}

function normalizeVoices(payload) {
  const data = speechDataRoot(payload)
  const groups = [
    ['system_voice', 'system'],
    ['system_voice_list', 'system'],
    ['voice_cloning', 'voice_cloning'],
    ['voice_generation', 'voice_generation'],
    ['custom_voice_list', 'custom'],
    ['voice_list', 'voice'],
    ['voices', 'voice'],
    ['items', 'voice']
  ]
  const result = []
  const seen = new Set()
  groups.forEach(([key, fallbackType]) => {
    const items = Array.isArray(data[key]) ? data[key] : []
    items.forEach(item => {
      const voiceId = item.voice_id || item.id
      if (!voiceId || seen.has(voiceId)) return
      seen.add(voiceId)
      result.push({
        voice_id: voiceId,
        name: item.voice_name || item.name || item.display_name || voiceId,
        voice_type: item.voice_type || item.type || fallbackType,
        preview_text: item.preview_text || item.preview || '',
        raw: item
      })
    })
  })
  const singleVoice = data.voice || data
  if (result.length === 0 && singleVoice?.voice_id) {
    result.push({
      voice_id: singleVoice.voice_id,
      name: singleVoice.voice_name || singleVoice.name || singleVoice.voice_id,
      voice_type: singleVoice.voice_type || 'voice',
      raw: singleVoice
    })
  }
  return result
}

function normalizeFiles(payload) {
  const data = speechDataRoot(payload)
  const groups = ['file_list', 'files', 'items']
  const result = []
  const seen = new Set()
  groups.forEach((key) => {
    const items = Array.isArray(data[key]) ? data[key] : []
    items.forEach(item => {
      const fileId = item.file_id || item.id
      if (!fileId || seen.has(fileId)) return
      seen.add(fileId)
      result.push({
        file_id: fileId,
        purpose: item.purpose || item.file_type || '',
        file_name: item.file_name || item.filename || item.name || '',
        bytes: item.bytes || item.size || 0,
        raw: item
      })
    })
  })
  const singleFile = data.file || data
  if (result.length === 0 && singleFile?.file_id) {
    result.push({
      file_id: singleFile.file_id,
      purpose: singleFile.purpose || '',
      file_name: singleFile.file_name || singleFile.filename || '',
      bytes: singleFile.bytes || singleFile.size || 0,
      raw: singleFile
    })
  }
  return result
}

function mergeByFileID(list) {
  const seen = new Set()
  return list.filter((item) => {
    if (!item?.file_id || seen.has(item.file_id)) return false
    seen.add(item.file_id)
    return true
  })
}

async function setAudioFromSpeechResponse(payload, target) {
  const data = payload?.data || payload || {}
  const audioHex = data.audio_file || data.audio || payload?.audio_file || payload?.audio
  const fileId = extractFileId(payload)
  if (typeof audioHex === 'string' && audioHex.trim()) {
    const blob = hexToBlob(audioHex, guessAudioMime(syncForm.format))
    setAudioUrl(target, URL.createObjectURL(blob))
    return
  }
  if (fileId) {
    if (target === 'sync') {
      await previewSpeechFile(fileId)
      syncAudioUrl.value = filePreviewUrl.value
      filePreviewUrl.value = ''
    }
    return
  }
  const audioUrl = data.audio_url || payload?.audio_url
  if (audioUrl) {
    setAudioUrl(target, audioUrl)
  }
}

function setAudioUrl(target, url) {
  if (target === 'sync') {
    resetAudio('sync')
    syncAudioUrl.value = url
  } else if (target === 'async') {
    resetAudio('async')
    asyncAudioUrl.value = url
  } else if (target === 'file') {
    resetAudio('file')
    filePreviewUrl.value = url
  }
}

function resetAudio(target) {
  if (target === 'sync' && syncAudioUrl.value) {
    revokeBlobUrl(syncAudioUrl.value)
    syncAudioUrl.value = ''
  }
  if (target === 'async' && asyncAudioUrl.value) {
    revokeBlobUrl(asyncAudioUrl.value)
    asyncAudioUrl.value = ''
  }
  if (target === 'file' && filePreviewUrl.value) {
    revokeBlobUrl(filePreviewUrl.value)
    filePreviewUrl.value = ''
  }
}

function revokeBlobUrl(url) {
  if (String(url || '').startsWith('blob:')) {
    URL.revokeObjectURL(url)
  }
}

function extractTaskId(payload) {
  return payload?.data?.task_id || payload?.task_id || payload?.task?.task_id || ''
}

function extractFileId(payload) {
  return payload?.data?.file?.file_id || payload?.file?.file_id || payload?.data?.file_id || payload?.data?.audio_file_id || payload?.file_id || payload?.audio_file_id || payload?.task?.output_file_id || ''
}

function pretty(value) {
  return JSON.stringify(value || {}, null, 2)
}

function fileLabel(file) {
  return `${file.file_name || file.file_id} (${file.file_id})`
}

function voiceLabel(voice) {
  const parts = [voice.name]
  if (voice.voice_type) parts.push(voice.voice_type)
  parts.push(voice.voice_id)
  return parts.join(' / ')
}

function formatBytes(bytes) {
  const size = Number(bytes || 0)
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(2)} MB`
}

function onFileSelected(event) {
  selectedFile.value = event.target.files?.[0] || null
}

async function safeJson(res) {
  try {
    return await res.json()
  } catch {
    return {}
  }
}

function guessAudioMime(format) {
  switch (format) {
    case 'wav':
      return 'audio/wav'
    case 'flac':
      return 'audio/flac'
    case 'pcm':
      return 'audio/pcm'
    default:
      return 'audio/mpeg'
  }
}

function hexToBlob(hex, mime) {
  const normalized = String(hex || '').replace(/^0x/, '').replace(/\s+/g, '')
  const pairs = normalized.match(/.{1,2}/g) || []
  const bytes = new Uint8Array(pairs.map(item => parseInt(item, 16)))
  return new Blob([bytes], { type: mime })
}

async function blobUrlToDataUrl(blobUrl) {
  if (!blobUrl || !String(blobUrl).startsWith('blob:')) return null
  try {
    const response = await fetch(blobUrl)
    const blob = await response.blob()
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(reader.result)
      reader.onerror = reject
      reader.readAsDataURL(blob)
    })
  } catch { return null }
}

async function shareSyncResult() {
  if (!syncAudioUrl.value || !hasAnyCredential()) return
  sharingSync.value = true
  try {
    const dataUrl = await blobUrlToDataUrl(syncAudioUrl.value)
    const draft = {
      sourceLabel: '语音合成',
      title: `TTS · ${syncForm.text.trim().slice(0, 36)}`,
      summary: `${syncForm.model} 同步合成 · 音色 ${syncForm.voiceId}`,
      resultType: 'speech',
      model: syncForm.model,
      payload: { text: syncForm.text, voice_id: syncForm.voiceId, speed: syncForm.speed, raw: syncResponse.value },
      assets: dataUrl ? [{ data_url: dataUrl, kind: 'audio', filename: `tts-${syncForm.model}.mp3`, content_type: guessAudioMime(syncForm.format) }] : []
    }
    emit('share', draft)
  } catch (err) {
    ElMessage.error(err.message || '保存分享失败')
  } finally {
    sharingSync.value = false
  }
}

async function shareAsyncResult() {
  if (!asyncAudioUrl.value || !hasAnyCredential()) return
  sharingAsync.value = true
  try {
    const dataUrl = await blobUrlToDataUrl(asyncAudioUrl.value)
    const text = asyncForm.mode === 'text' ? asyncForm.text : asyncForm.textFileId
    const draft = {
      sourceLabel: '语音合成',
      title: `TTS · ${String(text || '').trim().slice(0, 36)}`,
      summary: `${asyncForm.model} 异步合成 · 音色 ${asyncForm.voiceId}`,
      resultType: 'speech',
      model: asyncForm.model,
      payload: { text: text, voice_id: asyncForm.voiceId, mode: asyncForm.mode, raw: currentTask.value },
      assets: dataUrl ? [{ data_url: dataUrl, kind: 'audio', filename: `tts-${asyncForm.model}.mp3`, content_type: 'audio/mpeg' }] : []
    }
    emit('share', draft)
  } catch (err) {
    ElMessage.error(err.message || '保存分享失败')
  } finally {
    sharingAsync.value = false
  }
}

async function copyText(text) {
  await navigator.clipboard.writeText(text)
  ElMessage.success('已复制')
}
</script>

<style scoped>
.speech-card {
  border-radius: 22px;
}

.speech-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.speech-title {
  font-size: 18px;
  font-weight: 700;
  color: #1f2937;
}

.speech-subtitle {
  margin-top: 4px;
  color: #6b7280;
  font-size: 13px;
}

.speech-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.speech-tabs {
  margin-top: 18px;
}

.speech-grid {
  display: grid;
  gap: 18px;
}

.speech-grid.two-col {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.speech-inline-row {
  display: flex;
  gap: 14px;
  align-items: flex-end;
  flex-wrap: wrap;
}

.actions-row {
  margin-top: 8px;
}

.speech-json {
  margin: 0;
  padding: 14px;
  border-radius: 16px;
  background: #0f172a;
  color: #dbeafe;
  min-height: 160px;
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.result-title {
  margin: 0 0 10px;
  font-weight: 600;
  color: #1f2937;
}

.audio-result {
  margin-bottom: 14px;
  padding: 14px;
  border-radius: 16px;
  background: linear-gradient(135deg, #fff7ed 0%, #eff6ff 100%);
  border: 1px solid #fed7aa;
}

.speech-inner-card {
  border-radius: 18px;
}

.inner-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

@media (max-width: 960px) {
  .speech-header {
    flex-direction: column;
  }

  .speech-actions {
    justify-content: flex-start;
  }

  .speech-grid.two-col {
    grid-template-columns: 1fr;
  }
}
</style>
