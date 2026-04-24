<template>
  <div class="minimax-studio">
    <section class="studio-hero">
      <div class="hero-copy">
        <p class="eyebrow">MINIMAX STUDIO</p>
        <h2>把文本、语音、视频、音乐、图像理解放到一个工作台里。</h2>
        <p class="hero-desc">
          页面直接复用项目现有的 MiniMax 配置与网关能力。文本生成、Speech、Hailuo、music-2.5 / 2.6、music-cover、lyrics_generation、image-01 都能从这里发起。
        </p>
        <div class="hero-tips">
          <span>支持 `AI Gateway API Key`</span>
          <span>也支持 `超级管理员密码` 直接调试</span>
          <span>`coding-plan-vlm / search` 走图像理解能力入口</span>
        </div>
      </div>

      <el-card class="credential-card" shadow="never">
        <template #header>
          <div class="panel-head">
            <span>调试凭证</span>
            <el-button text @click="clearCredentials">清空</el-button>
          </div>
        </template>
        <el-input
          v-model="superAdminPassword"
          type="password"
          show-password
          placeholder="AI Gateway 超管密码"
          class="credential-input"
          @keyup.enter="saveCredentials"
        />
        <el-input
          v-model="apiKey"
          type="password"
          show-password
          placeholder="可选：AI Gateway API Key"
          class="credential-input"
          @keyup.enter="saveCredentials"
        />
        <div class="credential-actions">
          <el-button type="primary" @click="saveCredentials">保存到会话</el-button>
          <el-button @click="openOfficialDocs">官方文档</el-button>
        </div>
        <el-alert
          :title="hasCredential ? credentialHint : '未填写凭证时，文本/媒体/音乐调用会被禁用'"
          :type="hasCredential ? 'success' : 'warning'"
          :closable="false"
          show-icon
        />
      </el-card>
    </section>

    <section class="capability-grid">
      <button
        v-for="item in capabilityCards"
        :key="item.name"
        class="capability-card"
        :class="`tone-${item.tone}`"
        @click="jumpToTab(item.tab)"
      >
        <div class="capability-top">
          <span class="capability-name">{{ item.name }}</span>
          <span class="capability-badge">{{ item.badge }}</span>
        </div>
        <strong>{{ item.model }}</strong>
        <p>{{ item.desc }}</p>
      </button>
    </section>

    <el-tabs v-model="activeTab" class="studio-tabs">
      <el-tab-pane label="文本生成" name="text">
        <section class="panel-grid">
          <el-card class="panel-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>MiniMax Text</span>
                <el-tag size="small">{{ textForm.model }}</el-tag>
              </div>
            </template>

            <el-form label-position="top">
              <el-form-item label="模型">
                <el-select v-model="textForm.model" class="w-full">
                  <el-option v-for="model in textModels" :key="model" :label="model" :value="model" />
                </el-select>
              </el-form-item>
              <el-form-item label="System Prompt">
                <el-input v-model="textForm.system" type="textarea" :rows="3" placeholder="例如：你是一位严谨的中文助手。" />
              </el-form-item>
              <el-form-item label="User Prompt">
                <el-input v-model="textForm.prompt" type="textarea" :rows="8" placeholder="输入你的问题、需求或任务。" />
              </el-form-item>
              <div class="inline-fields">
                <el-form-item label="Temperature">
                  <el-slider v-model="textForm.temperature" :min="0" :max="1.5" :step="0.1" show-stops />
                </el-form-item>
                <el-form-item label="Max Tokens">
                  <el-input-number v-model="textForm.maxTokens" :min="128" :max="8192" class="w-full" />
                </el-form-item>
              </div>
              <div class="panel-actions">
                <el-button type="primary" :loading="textLoading" @click="generateText">开始生成</el-button>
                <el-button @click="resetText">清空</el-button>
              </div>
            </el-form>
          </el-card>

          <el-card class="panel-card result-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>响应</span>
                <el-button text :disabled="!textResult" @click="copyText(textResult)">复制文本</el-button>
              </div>
            </template>
            <div v-if="textResult" class="result-text">{{ textResult }}</div>
            <div v-else class="empty-block">还没有文本结果。</div>
            <pre class="result-json">{{ textRawPretty }}</pre>
          </el-card>
        </section>
      </el-tab-pane>

      <el-tab-pane label="语音 / 音色" name="speech">
        <AIGatewaySpeechPanel :super-admin-password="superAdminPassword" :prefill-api-key="apiKey" />
      </el-tab-pane>

      <el-tab-pane label="媒体生成" name="media">
        <section class="panel-grid">
          <el-card class="panel-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>媒体任务</span>
                <el-button text @click="loadMediaTasks">刷新任务</el-button>
              </div>
            </template>

            <el-form label-position="top">
              <el-form-item label="模型">
                <el-select v-model="mediaForm.model" class="w-full">
                  <el-option
                    v-for="item in mediaModels"
                    :key="item.value"
                    :label="`${item.label} · ${item.value}`"
                    :value="item.value"
                  />
                </el-select>
                <div class="field-hint">{{ selectedMediaModel?.hint }}</div>
              </el-form-item>

              <el-form-item label="Prompt">
                <el-input v-model="mediaForm.prompt" type="textarea" :rows="5" placeholder="描述你要生成的画面、音乐风格或主题。" />
              </el-form-item>

              <div v-if="isMusicModel" class="inline-fields">
                <el-form-item label="时长（秒）">
                  <el-input-number v-model="mediaForm.duration" :min="6" :max="300" class="w-full" />
                </el-form-item>
                <el-form-item label="cover_feature_id">
                  <el-input v-model="mediaForm.coverFeatureId" placeholder="music-cover 时必填" />
                </el-form-item>
              </div>

              <el-form-item v-if="isMusicModel" label="歌词（可选）">
                <el-input v-model="mediaForm.lyrics" type="textarea" :rows="4" placeholder="可以从下方 lyrics_generation 一键带入。" />
              </el-form-item>

              <div v-if="isVideoModel" class="inline-fields">
                <el-form-item label="时长（秒）">
                  <el-input-number v-model="mediaForm.duration" :min="6" :max="10" class="w-full" />
                </el-form-item>
                <el-form-item label="分辨率">
                  <el-select v-model="mediaForm.resolution" class="w-full">
                    <el-option label="768P" value="768P" />
                    <el-option label="1080P" value="1080P" />
                  </el-select>
                </el-form-item>
              </div>

              <div v-if="isImageModel" class="inline-fields">
                <el-form-item label="尺寸">
                  <el-select v-model="mediaForm.size" class="w-full">
                    <el-option label="1024x1024" value="1024x1024" />
                    <el-option label="1344x768" value="1344x768" />
                    <el-option label="768x1344" value="768x1344" />
                  </el-select>
                </el-form-item>
                <el-form-item label="张数">
                  <el-input-number v-model="mediaForm.count" :min="1" :max="4" class="w-full" />
                </el-form-item>
              </div>

              <el-form-item label="高级参数 JSON">
                <el-input
                  v-model="mediaParametersText"
                  type="textarea"
                  :rows="5"
                  placeholder='{"style":"cinematic","camera_movement":"push_in"}'
                />
              </el-form-item>

              <div class="panel-actions between">
                <el-switch v-model="mediaForm.autoPoll" active-text="自动轮询" />
                <div class="inline-actions">
                  <el-button type="primary" :loading="mediaSubmitting" @click="submitMedia">提交任务</el-button>
                  <el-button @click="resetMediaForm">重置</el-button>
                </div>
              </div>
            </el-form>
          </el-card>

          <el-card class="panel-card result-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>当前任务</span>
                <div class="inline-actions">
                  <el-button text :disabled="!currentMediaTask" @click="pollCurrentMediaTask">轮询</el-button>
                  <el-button text :disabled="!currentMediaTask" @click="downloadCurrentMedia">代理下载</el-button>
                </div>
              </div>
            </template>

            <div v-if="currentMediaTask" class="task-summary">
              <div class="task-meta">
                <el-tag :type="taskTagType(currentMediaTask.status)">{{ currentMediaTask.status }}</el-tag>
                <span>{{ currentMediaTask.model }}</span>
                <span>{{ formatTime(currentMediaTask.created_at) }}</span>
              </div>
              <p class="task-error" v-if="currentMediaTask.error">{{ currentMediaTask.error }}</p>
              <div v-if="mediaAssets.length" class="asset-grid">
                <div v-for="(asset, index) in mediaAssets" :key="asset.url + index" class="asset-card">
                  <img v-if="asset.type === 'image'" :src="asset.url" alt="asset" />
                  <video v-else-if="asset.type === 'video'" :src="asset.url" controls />
                  <audio v-else :src="asset.url" controls />
                  <div class="asset-url">{{ asset.url }}</div>
                </div>
              </div>
              <div v-else class="empty-block">任务还没有可预览产物。</div>
            </div>
            <div v-else class="empty-block">还没有选中任务。</div>

            <pre class="result-json">{{ mediaTaskPretty }}</pre>
          </el-card>
        </section>

        <el-card class="panel-card task-table-card" shadow="never">
          <template #header>
            <div class="panel-head">
              <span>任务列表</span>
              <span class="field-hint">只展示当前凭证能看到的任务</span>
            </div>
          </template>

          <el-table :data="mediaTasks" v-loading="mediaTasksLoading" stripe size="small" max-height="420">
            <el-table-column prop="task_id" label="任务 ID" min-width="170">
              <template #default="{ row }">
                <el-button text type="primary" @click="openMediaTask(row.task_id)">{{ row.task_id }}</el-button>
              </template>
            </el-table-column>
            <el-table-column prop="model" label="模型" min-width="150" />
            <el-table-column prop="status" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="taskTagType(row.status)">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="结果" min-width="220">
              <template #default="{ row }">
                <span class="result-links">{{ (row.result_urls || []).slice(0, 2).join(' | ') || '-' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="音乐工作流" name="music">
        <section class="panel-grid">
          <el-card class="panel-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>lyrics_generation</span>
                <el-button text :disabled="!lyricsResultText" @click="applyLyricsToMedia">带入音乐任务</el-button>
              </div>
            </template>
            <el-form label-position="top">
              <el-form-item label="模式">
                <el-select v-model="lyricsForm.mode" class="w-full">
                  <el-option label="write_full_song" value="write_full_song" />
                  <el-option label="continue_song" value="continue_song" />
                </el-select>
              </el-form-item>
              <el-form-item label="Prompt">
                <el-input v-model="lyricsForm.prompt" type="textarea" :rows="5" placeholder="例如：写一首温柔的城市夜晚情歌。" />
              </el-form-item>
              <el-form-item label="高级参数 JSON">
                <el-input v-model="lyricsForm.jsonText" type="textarea" :rows="4" placeholder='{"language":"zh","genre":"mandopop"}' />
              </el-form-item>
              <div class="panel-actions">
                <el-button type="primary" :loading="lyricsLoading" @click="generateLyrics">生成歌词</el-button>
              </div>
            </el-form>
          </el-card>

          <el-card class="panel-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>music-cover 前处理</span>
                <el-button text :disabled="!coverFeatureId" @click="applyCoverToMedia">带入翻唱任务</el-button>
              </div>
            </template>
            <el-form label-position="top">
              <el-form-item label="参考音频 URL">
                <el-input v-model="coverForm.audioUrl" placeholder="输入公网可访问的音频 URL" />
              </el-form-item>
              <el-form-item label="高级参数 JSON">
                <el-input v-model="coverForm.jsonText" type="textarea" :rows="4" placeholder='{"model":"music-cover"}' />
              </el-form-item>
              <div class="panel-actions">
                <el-button type="primary" :loading="coverLoading" @click="preprocessCover">生成 cover_feature_id</el-button>
              </div>
            </el-form>
          </el-card>
        </section>

        <section class="panel-grid">
          <el-card class="panel-card result-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>歌词结果</span>
                <el-button text :disabled="!lyricsResultText" @click="copyText(lyricsResultText)">复制</el-button>
              </div>
            </template>
            <div v-if="lyricsResultText" class="result-text">{{ lyricsResultText }}</div>
            <div v-else class="empty-block">还没有歌词结果。</div>
            <pre class="result-json">{{ lyricsRawPretty }}</pre>
          </el-card>

          <el-card class="panel-card result-card" shadow="never">
            <template #header>
              <div class="panel-head">
                <span>翻唱前处理结果</span>
                <el-button text :disabled="!coverFeatureId" @click="copyText(coverFeatureId)">复制 feature_id</el-button>
              </div>
            </template>
            <div v-if="coverFeatureId" class="result-key">{{ coverFeatureId }}</div>
            <div v-else class="empty-block">还没有 cover_feature_id。</div>
            <pre class="result-json">{{ coverRawPretty }}</pre>
          </el-card>
        </section>
      </el-tab-pane>

      <el-tab-pane label="图像理解 / Coding Plan" name="coding">
        <el-alert
          title="coding-plan-vlm / coding-plan-search 入口"
          type="info"
          :closable="false"
          show-icon
          class="coding-alert"
        >
          <template #default>
            这里直接嵌入现有的 MiniMax MCP 图像理解能力。它就是当前项目里最接近 `coding-plan-vlm / search` 的使用入口。
          </template>
        </el-alert>
        <ImageUnderstandingTool class="coding-embed" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import AIGatewaySpeechPanel from '../components/AIGatewaySpeechPanel.vue'
import ImageUnderstandingTool from './ImageUnderstandingTool.vue'

const OFFICIAL_DOCS_URL = 'https://platform.minimaxi.com/docs/api-reference/api-overview'
const SUPER_ADMIN_KEY = 'minimax_studio_super_admin_password'
const API_KEY_STORAGE = 'minimax_studio_api_key'

const activeTab = ref('text')
const superAdminPassword = ref(sessionStorage.getItem(SUPER_ADMIN_KEY) || '')
const apiKey = ref(sessionStorage.getItem(API_KEY_STORAGE) || '')

const textLoading = ref(false)
const textRaw = ref(null)
const textResult = ref('')
const textForm = ref({
  model: 'MiniMax-M2.5',
  system: '你是一个严谨、直接、中文优先的智能助手。',
  prompt: '',
  temperature: 0.4,
  maxTokens: 2048
})

const mediaSubmitting = ref(false)
const mediaTasksLoading = ref(false)
const mediaTasks = ref([])
const currentMediaTask = ref(null)
const mediaPollTimer = ref(null)
const mediaParametersText = ref('{\n  "style": "cinematic"\n}')
const mediaForm = ref({
  model: 'MiniMax-Hailuo-2.3-Fast',
  prompt: '',
  duration: 6,
  resolution: '768P',
  size: '1024x1024',
  count: 1,
  coverFeatureId: '',
  lyrics: '',
  autoPoll: true
})

const lyricsLoading = ref(false)
const lyricsRaw = ref(null)
const lyricsForm = ref({
  mode: 'write_full_song',
  prompt: '',
  jsonText: '{\n  "language": "zh"\n}'
})

const coverLoading = ref(false)
const coverRaw = ref(null)
const coverForm = ref({
  audioUrl: '',
  jsonText: '{\n  "model": "music-cover"\n}'
})

const textModels = [
  'MiniMax-M2.5',
  'MiniMax-M2.5-highspeed',
  'MiniMax-M2.1',
  'MiniMax-M2.1-highspeed',
  'MiniMax-M2',
  'MiniMax-M2.7'
]

const mediaModels = [
  { value: 'MiniMax-Hailuo-2.3-Fast', label: 'Hailuo Fast', hint: '适合 768P / 6 秒视频。', tone: 'video' },
  { value: 'MiniMax-Hailuo-2.3', label: 'Hailuo 2.3', hint: '标准 Hailuo 视频生成。', tone: 'video' },
  { value: 'MiniMax-Hailuo-02', label: 'Hailuo 02', hint: '兼容旧版 Hailuo 视频链路。', tone: 'video' },
  { value: 'T2V-01-Director', label: 'T2V Director', hint: '导演风格的视频生成。', tone: 'video' },
  { value: 'T2V-01', label: 'T2V 01', hint: '基础视频生成。', tone: 'video' },
  { value: 'music-2.5', label: 'Music 2.5', hint: '音乐生成，适合常规乐曲草稿。', tone: 'music' },
  { value: 'music-2.6', label: 'Music 2.6', hint: '更适合新版音乐额度。', tone: 'music' },
  { value: 'music-cover', label: 'Music Cover', hint: '翻唱任务，需先拿到 cover_feature_id。', tone: 'music' },
  { value: 'image-01', label: 'Image 01', hint: '图像生成。', tone: 'image' },
  { value: 'image-01-live', label: 'Image 01 Live', hint: '图像生成 live 版。', tone: 'image' }
]

const capabilityCards = [
  { name: '文本生成', badge: 'Direct', model: 'MiniMax-M2.x', desc: '直接调 Anthropic 兼容文本接口。', tab: 'text', tone: 'text' },
  { name: 'Text to Speech HD', badge: 'Direct', model: 'speech-2.8-hd', desc: '官方 Speech / TTS 调试入口。', tab: 'speech', tone: 'speech' },
  { name: 'Hailuo-2.3-Fast-768P 6s', badge: 'Direct', model: 'MiniMax-Hailuo-2.3-Fast', desc: '快速视频链路。', tab: 'media', tone: 'video' },
  { name: 'Hailuo-2.3-768P 6s', badge: 'Direct', model: 'MiniMax-Hailuo-2.3', desc: '标准视频链路。', tab: 'media', tone: 'video' },
  { name: 'music-2.5', badge: 'Direct', model: 'music-2.5', desc: '基础音乐生成。', tab: 'media', tone: 'music' },
  { name: 'music-2.6', badge: 'Direct', model: 'music-2.6', desc: '新版音乐额度入口。', tab: 'media', tone: 'music' },
  { name: 'music-cover', badge: 'Workflow', model: 'music-cover', desc: '支持前处理 + 生成。', tab: 'music', tone: 'music' },
  { name: 'lyrics_generation', badge: 'Workflow', model: 'lyrics_generation', desc: '先写歌词，再带入音乐生成。', tab: 'music', tone: 'music' },
  { name: 'image-01', badge: 'Direct', model: 'image-01', desc: '图像生成任务入口。', tab: 'media', tone: 'image' },
  { name: 'coding-plan-vlm', badge: 'Integrated', model: 'MiniMax MCP', desc: '在图像理解页内使用。', tab: 'coding', tone: 'coding' },
  { name: 'coding-plan-search', badge: 'Integrated', model: 'MiniMax MCP', desc: '在图像理解页内使用。', tab: 'coding', tone: 'coding' }
]

const hasCredential = computed(() => Boolean(apiKey.value.trim() || superAdminPassword.value.trim()))
const credentialHint = computed(() => apiKey.value.trim() ? '当前优先使用 API Key 调试。' : '当前使用超级管理员密码调试。')
const selectedMediaModel = computed(() => mediaModels.find(item => item.value === mediaForm.value.model) || null)
const isMusicModel = computed(() => mediaForm.value.model.startsWith('music-'))
const isVideoModel = computed(() => mediaForm.value.model.startsWith('MiniMax-Hailuo-') || mediaForm.value.model.startsWith('T2V-'))
const isImageModel = computed(() => mediaForm.value.model.startsWith('image-'))
const textRawPretty = computed(() => pretty(textRaw.value))
const mediaTaskPretty = computed(() => pretty(currentMediaTask.value))
const lyricsRawPretty = computed(() => pretty(lyricsRaw.value))
const coverRawPretty = computed(() => pretty(coverRaw.value))
const lyricsResultText = computed(() => extractLyricsText(lyricsRaw.value))
const coverFeatureId = computed(() => extractCoverFeatureId(coverRaw.value))
const mediaAssets = computed(() => extractTaskAssets(currentMediaTask.value))

watch(() => currentMediaTask.value?.status, (status) => {
  if (!status || !mediaForm.value.autoPoll) return
  if (status === 'pending' || status === 'running') {
    scheduleMediaPoll()
  } else {
    clearMediaPoll()
  }
})

watch(activeTab, (tab) => {
  if (tab !== 'media') {
    clearMediaPoll()
  } else if (currentMediaTask.value && mediaForm.value.autoPoll) {
    scheduleMediaPoll()
  }
})

onMounted(() => {
  if (hasCredential.value) {
    void loadMediaTasks()
  }
})

onBeforeUnmount(() => {
  clearMediaPoll()
})

function saveCredentials() {
  sessionStorage.setItem(SUPER_ADMIN_KEY, superAdminPassword.value)
  sessionStorage.setItem(API_KEY_STORAGE, apiKey.value)
  ElMessage.success('MiniMax Studio 凭证已保存到当前会话')
  if (hasCredential.value) {
    void loadMediaTasks()
  }
}

function clearCredentials() {
  superAdminPassword.value = ''
  apiKey.value = ''
  sessionStorage.removeItem(SUPER_ADMIN_KEY)
  sessionStorage.removeItem(API_KEY_STORAGE)
  ElMessage.success('已清空当前会话凭证')
}

function requireCredential() {
  if (!hasCredential.value) {
    ElMessage.error('请先填写 AI Gateway API Key 或超级管理员密码')
    return false
  }
  return true
}

function authHeaders() {
  if (apiKey.value.trim()) {
    return { Authorization: `Bearer ${apiKey.value.trim()}` }
  }
  if (superAdminPassword.value.trim()) {
    return { 'X-Super-Admin-Password': superAdminPassword.value.trim() }
  }
  return null
}

function jsonHeaders(extra = {}) {
  const headers = authHeaders()
  if (!headers) return null
  return {
    ...headers,
    'Content-Type': 'application/json',
    ...extra
  }
}

async function generateText() {
  if (!requireCredential()) return
  if (!textForm.value.prompt.trim()) {
    ElMessage.error('请输入用户提示词')
    return
  }
  textLoading.value = true
  try {
    const messages = []
    if (textForm.value.system.trim()) {
      messages.push({ role: 'system', content: textForm.value.system.trim() })
    }
    messages.push({ role: 'user', content: textForm.value.prompt.trim() })
    const res = await fetch('/api/minimax/anthropic/v1/messages', {
      method: 'POST',
      headers: jsonHeaders({ 'Anthropic-Version': '2023-06-01' }),
      body: JSON.stringify({
        model: textForm.value.model,
        max_tokens: textForm.value.maxTokens,
        temperature: textForm.value.temperature,
        messages
      })
    })
    const data = await res.json()
    textRaw.value = data
    if (!res.ok) throw new Error(data.error || '文本生成失败')
    textResult.value = extractAnthropicText(data)
    if (!textResult.value) {
      ElMessage.warning('请求成功，但未提取到文本内容')
      return
    }
    ElMessage.success('文本生成成功')
  } catch (err) {
    ElMessage.error(err.message || '文本生成失败')
  } finally {
    textLoading.value = false
  }
}

function resetText() {
  textForm.value.prompt = ''
  textRaw.value = null
  textResult.value = ''
}

async function submitMedia() {
  if (!requireCredential()) return
  if (!mediaForm.value.prompt.trim()) {
    ElMessage.error('请输入 Prompt')
    return
  }
  mediaSubmitting.value = true
  try {
    const payload = {
      model: mediaForm.value.model,
      prompt: mediaForm.value.prompt.trim()
    }
    if (isVideoModel.value) {
      payload.duration = mediaForm.value.duration
      payload.resolution = mediaForm.value.resolution
    }
    if (isImageModel.value) {
      payload.size = mediaForm.value.size
      payload.count = mediaForm.value.count
    }
    if (isMusicModel.value) {
      payload.duration = mediaForm.value.duration
      if (mediaForm.value.lyrics.trim()) {
        payload.lyrics = mediaForm.value.lyrics.trim()
      }
      if (mediaForm.value.coverFeatureId.trim()) {
        payload.cover_feature_id = mediaForm.value.coverFeatureId.trim()
      }
    }
    Object.assign(payload, parseOptionalJSON(mediaParametersText.value, '高级参数 JSON'))

    const res = await fetch('/api/minimax/token-plan/v1/generations', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await safeJson(res)
    if (!res.ok) throw new Error(data.error || '媒体任务提交失败')
    currentMediaTask.value = data
    if (data.task_id) {
      ElMessage.success(`任务已提交: ${data.task_id}`)
      await openMediaTask(data.task_id)
      await loadMediaTasks()
    } else {
      ElMessage.success('请求已完成')
    }
  } catch (err) {
    ElMessage.error(err.message || '媒体任务提交失败')
  } finally {
    mediaSubmitting.value = false
  }
}

async function loadMediaTasks() {
  if (!requireCredential()) return
  mediaTasksLoading.value = true
  try {
    const res = await fetch('/api/minimax/token-plan/tasks?limit=30', {
      headers: authHeaders()
    })
    const data = await safeJson(res)
    if (!res.ok) throw new Error(data.error || '加载任务失败')
    mediaTasks.value = data.tasks || []
  } catch (err) {
    ElMessage.error(err.message || '加载任务失败')
  } finally {
    mediaTasksLoading.value = false
  }
}

async function openMediaTask(taskId) {
  if (!requireCredential()) return
  try {
    const res = await fetch(`/api/minimax/token-plan/tasks/${encodeURIComponent(taskId)}`, {
      headers: authHeaders()
    })
    const data = await safeJson(res)
    if (!res.ok) throw new Error(data.error || '读取任务详情失败')
    currentMediaTask.value = data
  } catch (err) {
    ElMessage.error(err.message || '读取任务详情失败')
  }
}

async function pollCurrentMediaTask() {
  const taskId = currentMediaTask.value?.task_id
  if (!taskId) {
    ElMessage.error('当前没有任务')
    return
  }
  await openMediaTask(taskId)
  await loadMediaTasks()
}

function scheduleMediaPoll() {
  clearMediaPoll()
  mediaPollTimer.value = window.setTimeout(async () => {
    if (activeTab.value !== 'media' || !currentMediaTask.value?.task_id) return
    await pollCurrentMediaTask()
  }, 6000)
}

function clearMediaPoll() {
  if (mediaPollTimer.value) {
    window.clearTimeout(mediaPollTimer.value)
    mediaPollTimer.value = null
  }
}

async function downloadCurrentMedia() {
  if (!requireCredential()) return
  const taskId = currentMediaTask.value?.task_id
  if (!taskId) {
    ElMessage.error('当前没有可下载任务')
    return
  }
  try {
    const res = await fetch(`/api/minimax/token-plan/tasks/${encodeURIComponent(taskId)}/download`, {
      headers: authHeaders()
    })
    if (!res.ok) {
      const data = await safeJson(res)
      throw new Error(data.error || '下载失败')
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${taskId}.${guessExtension(blob.type)}`
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
    ElMessage.success('已开始下载')
  } catch (err) {
    ElMessage.error(err.message || '下载失败')
  }
}

function resetMediaForm() {
  mediaForm.value.prompt = ''
  mediaForm.value.lyrics = ''
  mediaForm.value.coverFeatureId = ''
  mediaParametersText.value = '{\n  "style": "cinematic"\n}'
}

async function generateLyrics() {
  if (!requireCredential()) return
  if (!lyricsForm.value.prompt.trim()) {
    ElMessage.error('请输入歌词生成 Prompt')
    return
  }
  lyricsLoading.value = true
  try {
    const payload = {
      mode: lyricsForm.value.mode,
      prompt: lyricsForm.value.prompt.trim(),
      ...parseOptionalJSON(lyricsForm.value.jsonText, '歌词高级参数 JSON')
    }
    const res = await fetch('/api/minimax/music/v1/lyrics_generation', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await safeJson(res)
    lyricsRaw.value = data
    if (!res.ok) throw new Error(data.error || '歌词生成失败')
    ElMessage.success('歌词生成成功')
  } catch (err) {
    ElMessage.error(err.message || '歌词生成失败')
  } finally {
    lyricsLoading.value = false
  }
}

async function preprocessCover() {
  if (!requireCredential()) return
  if (!coverForm.value.audioUrl.trim()) {
    ElMessage.error('请输入参考音频 URL')
    return
  }
  coverLoading.value = true
  try {
    const payload = {
      audio_url: coverForm.value.audioUrl.trim(),
      ...parseOptionalJSON(coverForm.value.jsonText, '翻唱高级参数 JSON')
    }
    const res = await fetch('/api/minimax/music/v1/cover_preprocess', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await safeJson(res)
    coverRaw.value = data
    if (!res.ok) throw new Error(data.error || '翻唱前处理失败')
    if (!extractCoverFeatureId(data)) {
      ElMessage.warning('请求成功，但没有提取到 cover_feature_id')
      return
    }
    ElMessage.success('cover_feature_id 已生成')
  } catch (err) {
    ElMessage.error(err.message || '翻唱前处理失败')
  } finally {
    coverLoading.value = false
  }
}

function applyLyricsToMedia() {
  const lyrics = lyricsResultText.value
  if (!lyrics) {
    ElMessage.error('当前没有可带入的歌词')
    return
  }
  mediaForm.value.model = 'music-2.6'
  mediaForm.value.lyrics = lyrics
  activeTab.value = 'media'
  ElMessage.success('歌词已带入媒体任务')
}

function applyCoverToMedia() {
  const featureId = coverFeatureId.value
  if (!featureId) {
    ElMessage.error('当前没有可带入的 cover_feature_id')
    return
  }
  mediaForm.value.model = 'music-cover'
  mediaForm.value.coverFeatureId = featureId
  activeTab.value = 'media'
  ElMessage.success('cover_feature_id 已带入翻唱任务')
}

function jumpToTab(tab) {
  activeTab.value = tab
}

function openOfficialDocs() {
  window.open(OFFICIAL_DOCS_URL, '_blank', 'noopener,noreferrer')
}

function copyText(text) {
  navigator.clipboard.writeText(text || '')
    .then(() => ElMessage.success('已复制'))
    .catch(() => ElMessage.error('复制失败'))
}

function pretty(value) {
  return value ? JSON.stringify(value, null, 2) : ''
}

function parseOptionalJSON(text, label) {
  const raw = String(text || '').trim()
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch (err) {
    throw new Error(`${label} 不是合法 JSON`)
  }
}

function extractAnthropicText(payload) {
  const content = Array.isArray(payload?.content) ? payload.content : []
  const text = content
    .filter(item => item?.type === 'text' && item?.text)
    .map(item => item.text)
    .join('\n')
  if (text) return text
  const thinking = content
    .filter(item => item?.type === 'thinking' && item?.thinking)
    .map(item => item.thinking)
    .join('\n')
  return thinking || payload?.content || ''
}

function extractLyricsText(payload) {
  if (!payload) return ''
  const data = payload.data || payload
  return data.lyrics || data.lyric || data.full_lyrics || data.text || ''
}

function extractCoverFeatureId(payload) {
  if (!payload) return ''
  const data = payload.data || payload
  return data.cover_feature_id || data.feature_id || ''
}

function extractTaskAssets(task) {
  if (!task) return []
  const urls = Array.isArray(task.result_urls) ? [...task.result_urls] : []
  collectUrls(task.result, urls)
  return Array.from(new Set(urls)).map((url) => ({
    url,
    type: inferAssetType(url, task.content_type, task.model)
  }))
}

function collectUrls(value, bucket) {
  if (!value) return
  if (typeof value === 'string') {
    if (value.startsWith('http://') || value.startsWith('https://')) {
      bucket.push(value)
    }
    return
  }
  if (Array.isArray(value)) {
    value.forEach(item => collectUrls(item, bucket))
    return
  }
  if (typeof value === 'object') {
    Object.values(value).forEach(item => collectUrls(item, bucket))
  }
}

function inferAssetType(url, contentType, model) {
  const text = `${contentType || ''} ${model || ''} ${url || ''}`.toLowerCase()
  if (text.includes('video') || text.endsWith('.mp4')) return 'video'
  if (text.includes('audio') || text.endsWith('.mp3') || text.endsWith('.wav')) return 'audio'
  return 'image'
}

function guessExtension(contentType) {
  if (contentType.includes('video')) return 'mp4'
  if (contentType.includes('wav')) return 'wav'
  if (contentType.includes('audio')) return 'mp3'
  if (contentType.includes('png')) return 'png'
  if (contentType.includes('jpeg')) return 'jpg'
  return 'bin'
}

function taskTagType(status) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

function formatTime(value) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

async function safeJson(res) {
  try {
    return await res.json()
  } catch {
    return {}
  }
}
</script>

<style scoped>
.minimax-studio {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px;
  min-height: 100%;
  background:
    radial-gradient(circle at top left, rgba(15, 118, 110, 0.14), transparent 28%),
    radial-gradient(circle at top right, rgba(37, 99, 235, 0.12), transparent 26%),
    linear-gradient(180deg, #f5fbfa 0%, #f8fafc 55%, #eef6ff 100%);
}

.studio-hero {
  display: grid;
  grid-template-columns: 1.4fr minmax(320px, 420px);
  gap: 20px;
  align-items: stretch;
}

.hero-copy,
.credential-card,
.panel-card,
.capability-card {
  border: 1px solid rgba(15, 23, 42, 0.08);
  box-shadow: 0 18px 50px rgba(15, 23, 42, 0.06);
}

.hero-copy {
  padding: 28px;
  border-radius: 28px;
  background: linear-gradient(135deg, rgba(6, 95, 70, 0.95), rgba(30, 64, 175, 0.92));
  color: #f8fafc;
}

.eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  opacity: 0.78;
}

.hero-copy h2 {
  margin: 0 0 14px;
  font-size: 30px;
  line-height: 1.2;
}

.hero-desc {
  margin: 0;
  max-width: 720px;
  color: rgba(248, 250, 252, 0.9);
  font-size: 15px;
  line-height: 1.8;
}

.hero-tips {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
}

.hero-tips span {
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.14);
  font-size: 12px;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.credential-input + .credential-input {
  margin-top: 12px;
}

.credential-actions {
  display: flex;
  gap: 10px;
  margin: 16px 0;
}

.capability-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.capability-card {
  padding: 16px;
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.88);
  text-align: left;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.capability-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.1);
}

.capability-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.capability-name {
  font-size: 13px;
  color: #0f172a;
}

.capability-badge {
  padding: 4px 8px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.08);
  font-size: 11px;
  color: #334155;
}

.capability-card strong {
  display: block;
  margin-bottom: 8px;
  font-size: 16px;
  color: #0f172a;
}

.capability-card p,
.field-hint,
.result-links,
.asset-url {
  color: #64748b;
  font-size: 12px;
  line-height: 1.6;
}

.tone-text { border-top: 3px solid #2563eb; }
.tone-speech { border-top: 3px solid #0f766e; }
.tone-video { border-top: 3px solid #ea580c; }
.tone-music { border-top: 3px solid #ca8a04; }
.tone-image { border-top: 3px solid #7c3aed; }
.tone-coding { border-top: 3px solid #475569; }

.studio-tabs :deep(.el-tabs__header) {
  margin-bottom: 18px;
}

.panel-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.panel-card {
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.9);
}

.result-card {
  min-height: 320px;
}

.inline-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.panel-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.panel-actions.between {
  justify-content: space-between;
}

.inline-actions {
  display: flex;
  gap: 8px;
}

.result-text,
.result-key {
  white-space: pre-wrap;
  line-height: 1.8;
  color: #0f172a;
  background: #f8fafc;
  border-radius: 18px;
  padding: 16px;
  min-height: 88px;
}

.result-key {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  word-break: break-all;
}

.result-json {
  margin: 14px 0 0;
  padding: 14px;
  border-radius: 18px;
  background: #0f172a;
  color: #dbeafe;
  font-size: 12px;
  line-height: 1.6;
  overflow: auto;
  min-height: 140px;
}

.empty-block {
  padding: 18px;
  border-radius: 18px;
  background: #f8fafc;
  color: #64748b;
  text-align: center;
}

.task-summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  color: #334155;
  font-size: 13px;
}

.task-error {
  margin: 0;
  color: #b91c1c;
  font-size: 13px;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.asset-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border-radius: 18px;
  background: #f8fafc;
  overflow: hidden;
}

.asset-card img,
.asset-card video,
.asset-card audio {
  width: 100%;
  border-radius: 14px;
  background: #e2e8f0;
}

.asset-card img,
.asset-card video {
  min-height: 180px;
  object-fit: cover;
}

.coding-alert {
  margin-bottom: 16px;
}

.coding-embed :deep(.tool-container) {
  padding: 0;
}

.task-table-card {
  margin-top: 18px;
}

@media (max-width: 1120px) {
  .studio-hero,
  .panel-grid {
    grid-template-columns: 1fr;
  }

  .capability-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .minimax-studio {
    padding: 14px;
    gap: 14px;
  }

  .hero-copy {
    padding: 22px;
  }

  .hero-copy h2 {
    font-size: 24px;
  }

  .capability-grid,
  .inline-fields,
  .asset-grid {
    grid-template-columns: 1fr;
  }

  .credential-actions,
  .panel-actions,
  .inline-actions {
    flex-wrap: wrap;
  }
}
</style>
