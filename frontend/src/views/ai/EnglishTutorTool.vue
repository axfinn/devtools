<template>
  <div class="tool-container english-tutor-page">
    <section class="page-header">
      <div>
        <h2>AI 英语学习</h2>
        <p>公开可用的英语学习页：翻译、拼读、音标、例句、纠错和对话练习。</p>
      </div>
      <div class="header-actions">
        <el-tag type="success" effect="plain">服务端受限代理</el-tag>
        <el-button :icon="Refresh" :loading="loadingMeta" @click="loadMeta">刷新限制</el-button>
      </div>
    </section>

    <div class="workspace-grid">
      <section class="left-panel">
        <el-card class="panel-card">
          <template #header>
            <div class="card-header">
              <span>学习输入</span>
              <el-tag size="small" type="info">{{ currentMode.label }}</el-tag>
            </div>
          </template>

          <el-form label-position="top">
            <el-form-item label="学习模式">
              <el-segmented v-model="mode" :options="modeOptions" class="mode-segmented" />
            </el-form-item>

            <div class="settings-row">
              <el-form-item label="目标语言">
                <el-select v-model="targetLanguage" @change="saveSettings">
                  <el-option label="中文" value="中文" />
                  <el-option label="英文" value="英文" />
                  <el-option label="日文" value="日文" />
                  <el-option label="韩文" value="韩文" />
                </el-select>
              </el-form-item>
              <el-form-item label="水平">
                <el-select v-model="level" @change="saveSettings">
                  <el-option label="入门" value="入门" />
                  <el-option label="初级" value="初级" />
                  <el-option label="中级" value="中级" />
                  <el-option label="高级" value="高级" />
                </el-select>
              </el-form-item>
            </div>

            <el-form-item label="文本、单词或句子">
              <el-input
                v-model="inputText"
                type="textarea"
                :rows="9"
                :maxlength="limits.max_text_chars"
                show-word-limit
                resize="vertical"
                placeholder="输入英文单词、句子、段落，或输入中文让 AI 翻译成英文..."
                @keydown.meta.enter.prevent="runTutor"
                @keydown.ctrl.enter.prevent="runTutor"
              />
            </el-form-item>

            <el-form-item v-if="mode === 'api'" label="自定义要求">
              <el-input
                v-model="customInstruction"
                type="textarea"
                :rows="4"
                :maxlength="limits.max_custom_instruction_chars"
                show-word-limit
                resize="vertical"
                placeholder="例如：翻译成商务邮件英文，并解释关键词拼读。仅限英语学习相关要求。"
              />
            </el-form-item>

            <div class="action-row">
              <el-button type="primary" :icon="MagicStick" :loading="loading" @click="runTutor">
                AI 处理
              </el-button>
              <el-button :icon="Microphone" :loading="speaking" @click="speak(inputText)">
                朗读输入
              </el-button>
              <el-button :icon="Delete" @click="clearAll">清空</el-button>
            </div>
          </el-form>
        </el-card>

        <el-card class="panel-card">
          <template #header>
            <div class="card-header">
              <span>快捷模板</span>
              <el-button text size="small" @click="fillSample">示例</el-button>
            </div>
          </template>
          <div class="template-grid">
            <button
              v-for="item in quickTemplates"
              :key="item.title"
              class="template-button"
              type="button"
              @click="applyTemplate(item)"
            >
              <span>{{ item.title }}</span>
              <small>{{ item.text }}</small>
            </button>
          </div>
        </el-card>
      </section>

      <section class="right-panel">
        <el-card class="panel-card result-card">
          <template #header>
            <div class="card-header">
              <span>学习结果</span>
              <div class="card-actions">
                <el-button text size="small" :icon="DocumentCopy" @click="copyResult">复制</el-button>
                <el-button text size="small" :icon="Microphone" :loading="speaking" @click="speak(resultSpeechText)">
                  朗读
                </el-button>
              </div>
            </div>
          </template>

          <el-empty v-if="!loading && !result && !rawResult" description="输入内容后开始学习" />
          <div v-if="loading" class="loading-state">
            <el-skeleton :rows="8" animated />
          </div>

          <div v-if="result" class="result-sections">
            <section v-if="result.translation || result.polished_translation" class="result-section primary-result">
              <div class="section-title">
                <el-icon><Switch /></el-icon>
                <span>翻译</span>
              </div>
              <p v-if="result.translation" class="translation-text">{{ result.translation }}</p>
              <p v-if="result.polished_translation" class="muted-text">{{ result.polished_translation }}</p>
            </section>

            <section v-if="result.pronunciation" class="result-section">
              <div class="section-title">
                <el-icon><Headset /></el-icon>
                <span>音标与拼读</span>
              </div>
              <div class="pronunciation-grid">
                <div class="pron-card">
                  <span>IPA</span>
                  <strong>{{ result.pronunciation.ipa || '-' }}</strong>
                </div>
                <div class="pron-card">
                  <span>音节</span>
                  <strong>{{ result.pronunciation.syllables || '-' }}</strong>
                </div>
                <div class="pron-card">
                  <span>重音</span>
                  <strong>{{ result.pronunciation.stress || '-' }}</strong>
                </div>
              </div>
              <div v-if="asList(result.pronunciation.phonics).length" class="chip-list">
                <el-tag v-for="item in asList(result.pronunciation.phonics)" :key="item" effect="plain">
                  {{ item }}
                </el-tag>
              </div>
              <p v-if="result.pronunciation.tip" class="tip-text">{{ result.pronunciation.tip }}</p>
            </section>

            <section v-if="asList(result.key_points).length" class="result-section">
              <div class="section-title">
                <el-icon><Reading /></el-icon>
                <span>重点理解</span>
              </div>
              <ul class="clean-list">
                <li v-for="item in asList(result.key_points)" :key="item">{{ item }}</li>
              </ul>
            </section>

            <section v-if="asList(result.vocabulary).length" class="result-section">
              <div class="section-title">
                <el-icon><Collection /></el-icon>
                <span>词汇拆解</span>
              </div>
              <div class="vocab-list">
                <div v-for="item in asList(result.vocabulary)" :key="vocabKey(item)" class="vocab-item">
                  <strong>{{ item.word || item }}</strong>
                  <span>{{ item.meaning || item.explain || '' }}</span>
                  <small>{{ item.example || '' }}</small>
                </div>
              </div>
            </section>

            <section v-if="asList(result.examples).length" class="result-section">
              <div class="section-title">
                <el-icon><ChatLineSquare /></el-icon>
                <span>例句</span>
              </div>
              <div class="example-list">
                <div v-for="item in asList(result.examples)" :key="exampleKey(item)" class="example-item">
                  <p>{{ item.english || item }}</p>
                  <span>{{ item.chinese || item.translation || '' }}</span>
                </div>
              </div>
            </section>

            <section v-if="result.correction" class="result-section">
              <div class="section-title">
                <el-icon><EditPen /></el-icon>
                <span>纠错建议</span>
              </div>
              <div class="correction-grid">
                <div v-if="result.correction.original">
                  <span class="field-label">原文</span>
                  <p>{{ result.correction.original }}</p>
                </div>
                <div v-if="result.correction.corrected">
                  <span class="field-label">修正</span>
                  <p>{{ result.correction.corrected }}</p>
                </div>
              </div>
              <ul v-if="asList(result.correction.notes).length" class="clean-list">
                <li v-for="item in asList(result.correction.notes)" :key="item">{{ item }}</li>
              </ul>
            </section>

            <section v-if="asList(result.practice).length" class="result-section">
              <div class="section-title">
                <el-icon><QuestionFilled /></el-icon>
                <span>练习</span>
              </div>
              <div class="practice-list">
                <div v-for="(item, index) in asList(result.practice)" :key="index" class="practice-item">
                  <strong>{{ index + 1 }}. {{ item.question || item }}</strong>
                  <span v-if="item.answer">参考：{{ item.answer }}</span>
                </div>
              </div>
            </section>
          </div>

          <pre v-else-if="rawResult" class="raw-result">{{ rawResult }}</pre>
        </el-card>

        <el-card class="panel-card api-card">
          <template #header>
            <div class="card-header">
              <span>公开接口</span>
              <el-button text size="small" :icon="DocumentCopy" @click="copyCurl">复制 cURL</el-button>
            </div>
          </template>
          <el-tabs v-model="apiTab">
            <el-tab-pane label="请求体" name="body">
              <pre class="code-box">{{ requestPreview }}</pre>
            </el-tab-pane>
            <el-tab-pane label="cURL" name="curl">
              <pre class="code-box">{{ curlPreview }}</pre>
            </el-tab-pane>
            <el-tab-pane label="限制" name="limits">
              <el-descriptions :column="1" border>
                <el-descriptions-item label="接口">POST /api/english-tutor</el-descriptions-item>
                <el-descriptions-item label="单次文本">{{ limits.max_text_chars }} 字符</el-descriptions-item>
                <el-descriptions-item label="自定义要求">{{ limits.max_custom_instruction_chars }} 字符</el-descriptions-item>
                <el-descriptions-item label="频控">{{ limits.rate_limit_per_minute }}/分钟/IP，{{ limits.rate_limit_per_hour }}/小时/IP</el-descriptions-item>
                <el-descriptions-item label="说明">后端固定学习提示词和模型，不暴露 AI Gateway Key，不支持通用 AI 调用。</el-descriptions-item>
              </el-descriptions>
            </el-tab-pane>
            <el-tab-pane label="历史" name="history">
              <div v-if="history.length" class="history-list">
                <button
                  v-for="item in history"
                  :key="item.id"
                  class="history-item"
                  type="button"
                  @click="restoreHistory(item)"
                >
                  <span>{{ item.modeLabel }}</span>
                  <small>{{ item.text }}</small>
                </button>
              </div>
              <el-empty v-else description="暂无历史" />
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </section>
    </div>

    <audio ref="audioPlayer" :src="audioUrl" controls class="audio-player" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ChatLineSquare,
  Collection,
  Delete,
  DocumentCopy,
  EditPen,
  Headset,
  MagicStick,
  Microphone,
  QuestionFilled,
  Reading,
  Refresh,
  Switch
} from '@element-plus/icons-vue'

const API_BASE = '/api/english-tutor'
const STORAGE_KEY = 'english_tutor_settings'
const HISTORY_KEY = 'english_tutor_history'

const mode = ref('translate')
const targetLanguage = ref('中文')
const level = ref('初级')
const inputText = ref('')
const customInstruction = ref('')
const loadingMeta = ref(false)
const loading = ref(false)
const result = ref(null)
const rawResult = ref('')
const history = ref([])
const apiTab = ref('body')
const speaking = ref(false)
const audioUrl = ref('')
const audioPlayer = ref(null)
const limits = ref({
  max_text_chars: 2000,
  max_custom_instruction_chars: 240,
  rate_limit_per_minute: 8,
  rate_limit_per_hour: 50
})

const modes = [
  { value: 'translate', label: '翻译' },
  { value: 'pronounce', label: '拼读' },
  { value: 'explain', label: '精讲' },
  { value: 'correct', label: '纠错' },
  { value: 'dialogue', label: '对话' },
  { value: 'api', label: '接口请求' }
]

const modeOptions = modes.map(item => ({ label: item.label, value: item.value }))
const currentMode = computed(() => modes.find(item => item.value === mode.value) || modes[0])

const quickTemplates = [
  { title: '单词拼读', mode: 'pronounce', text: 'comfortable', instruction: '' },
  { title: '句子翻译', mode: 'translate', text: '我想把这个接口接入到现有系统里。', instruction: '' },
  { title: '表达纠错', mode: 'correct', text: 'I very like learn English by AI tool.', instruction: '' },
  { title: '面试对话', mode: 'dialogue', text: 'Introduce a backend project that uses Go and Vue.', instruction: '' }
]

const requestPayload = computed(() => ({
  mode: mode.value,
  text: inputText.value.trim() || 'comfortable',
  target_language: targetLanguage.value,
  level: level.value,
  custom_instruction: mode.value === 'api' ? customInstruction.value.trim() : undefined
}))

const requestPreview = computed(() => JSON.stringify(requestPayload.value, null, 2))
const curlPreview = computed(() => `curl -X POST ${window.location.origin}${API_BASE} \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify(requestPayload.value, null, 2).replace(/'/g, "'\\''")}'`)

const resultSpeechText = computed(() => {
  if (result.value?.correction?.corrected) return result.value.correction.corrected
  if (result.value?.examples?.[0]?.english) return result.value.examples[0].english
  if (result.value?.translation) return result.value.translation
  return inputText.value
})

function loadSettings() {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    targetLanguage.value = saved.targetLanguage || '中文'
    level.value = saved.level || '初级'
  } catch {
    targetLanguage.value = '中文'
  }

  try {
    history.value = JSON.parse(localStorage.getItem(HISTORY_KEY) || '[]')
  } catch {
    history.value = []
  }
}

function saveSettings() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({
    targetLanguage: targetLanguage.value,
    level: level.value
  }))
}

async function loadMeta() {
  loadingMeta.value = true
  try {
    const res = await fetch(`${API_BASE}/meta`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载限制失败')
    limits.value = { ...limits.value, ...(data.limits || {}) }
  } catch (err) {
    ElMessage.warning(err.message || '限制信息不可用，已使用默认限制')
  } finally {
    loadingMeta.value = false
  }
}

async function runTutor() {
  if (!inputText.value.trim()) {
    ElMessage.error('请输入要学习的内容')
    return
  }

  saveSettings()
  loading.value = true
  result.value = null
  rawResult.value = ''

  try {
    const res = await fetch(API_BASE, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(requestPayload.value)
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || data.message || '英语学习接口请求失败')

    const parsed = parseJsonContent(data.content || '')
    if (parsed) {
      result.value = normalizeResult(parsed)
    } else {
      rawResult.value = data.content || JSON.stringify(data, null, 2)
    }
    pushHistory()
  } catch (err) {
    ElMessage.error(err.message || '处理失败')
  } finally {
    loading.value = false
  }
}

function parseJsonContent(content) {
  if (!content) return null
  const cleaned = content
    .replace(/^```json\s*/i, '')
    .replace(/^```\s*/i, '')
    .replace(/```$/i, '')
    .trim()
  try {
    return JSON.parse(cleaned)
  } catch {
    const match = cleaned.match(/\{[\s\S]*\}/)
    if (!match) return null
    try {
      return JSON.parse(match[0])
    } catch {
      return null
    }
  }
}

function normalizeResult(value) {
  return {
    translation: value.translation || value.translated_text || '',
    polished_translation: value.polished_translation || value.natural_expression || '',
    pronunciation: value.pronunciation || {},
    key_points: value.key_points || value.explanations || value.notes || [],
    vocabulary: value.vocabulary || value.words || [],
    examples: value.examples || [],
    correction: value.correction || null,
    practice: value.practice || value.exercises || []
  }
}

function asList(value) {
  if (!value) return []
  return Array.isArray(value) ? value : [value]
}

function vocabKey(item) {
  return typeof item === 'string' ? item : `${item.word || ''}-${item.meaning || ''}-${item.example || ''}`
}

function exampleKey(item) {
  return typeof item === 'string' ? item : `${item.english || ''}-${item.chinese || ''}`
}

function pushHistory() {
  const record = {
    id: Date.now(),
    mode: mode.value,
    modeLabel: currentMode.value.label,
    text: inputText.value.trim().slice(0, 120),
    fullText: inputText.value,
    customInstruction: customInstruction.value
  }
  history.value = [record, ...history.value.filter(item => item.fullText !== record.fullText)].slice(0, 12)
  localStorage.setItem(HISTORY_KEY, JSON.stringify(history.value))
}

function restoreHistory(item) {
  mode.value = item.mode
  inputText.value = item.fullText
  customInstruction.value = item.customInstruction || ''
  apiTab.value = 'body'
}

function applyTemplate(item) {
  mode.value = item.mode
  inputText.value = item.text
  customInstruction.value = item.instruction || ''
}

function fillSample() {
  inputText.value = 'Although the architecture looks complicated, each service has a clear responsibility.'
  mode.value = 'explain'
}

function clearAll() {
  inputText.value = ''
  customInstruction.value = ''
  result.value = null
  rawResult.value = ''
  audioUrl.value = ''
}

async function copyText(text, message = '已复制') {
  if (!text) {
    ElMessage.warning('没有可复制的内容')
    return
  }
  await navigator.clipboard.writeText(text)
  ElMessage.success(message)
}

function copyResult() {
  const text = result.value ? JSON.stringify(result.value, null, 2) : rawResult.value
  copyText(text)
}

function copyCurl() {
  copyText(curlPreview.value, 'cURL 已复制')
}

async function speak(text) {
  const content = (text || '').trim()
  if (!content) {
    ElMessage.warning('没有可朗读的内容')
    return
  }

  speaking.value = true
  try {
    const res = await fetch('/api/edge-tts/tts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: content.slice(0, 500),
        voice: 'en-US-JennyNeural'
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'TTS 合成失败')
    audioUrl.value = `/api/chat/uploads/${data.filename}`
    setTimeout(() => {
      audioPlayer.value?.play?.()
    }, 80)
  } catch (err) {
    if ('speechSynthesis' in window) {
      const utterance = new SpeechSynthesisUtterance(content.slice(0, 500))
      utterance.lang = 'en-US'
      window.speechSynthesis.cancel()
      window.speechSynthesis.speak(utterance)
      ElMessage.info('Edge TTS 不可用，已使用浏览器朗读')
    } else {
      ElMessage.error(err.message || '朗读失败')
    }
  } finally {
    speaking.value = false
  }
}

onMounted(() => {
  loadSettings()
  loadMeta()
})
</script>

<style scoped>
.english-tutor-page {
  max-width: 1480px;
  margin: 0 auto;
  padding: 18px;
}

.page-header,
.card-header,
.header-actions,
.action-row,
.card-actions {
  display: flex;
  align-items: center;
}

.page-header {
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: 8px;
}

.page-header h2 {
  margin: 0;
  font-size: 24px;
  color: var(--text-primary);
}

.page-header p {
  margin: 6px 0 0;
  color: var(--text-secondary);
}

.header-actions {
  gap: 10px;
  flex-wrap: wrap;
}

.workspace-grid {
  display: grid;
  grid-template-columns: minmax(360px, 0.9fr) minmax(520px, 1.25fr);
  gap: 16px;
}

.left-panel,
.right-panel,
.result-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-card {
  border-radius: 8px;
}

.card-header {
  justify-content: space-between;
  gap: 12px;
}

.card-actions {
  gap: 4px;
}

.mode-segmented {
  width: 100%;
}

:deep(.mode-segmented .el-segmented__group) {
  width: 100%;
}

.settings-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.action-row {
  gap: 10px;
  flex-wrap: wrap;
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.template-button,
.history-item {
  text-align: left;
  border: 1px solid var(--border-light);
  background: var(--bg-primary);
  color: var(--text-primary);
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
}

.template-button:hover,
.history-item:hover {
  border-color: var(--color-primary);
  background: var(--bg-hover);
}

.template-button span,
.history-item span {
  display: block;
  font-weight: 600;
  margin-bottom: 4px;
}

.template-button small,
.history-item small {
  display: block;
  color: var(--text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-card {
  min-height: 520px;
}

.loading-state {
  padding: 8px 0;
}

.result-section {
  border: 1px solid var(--border-light);
  border-radius: 8px;
  padding: 14px;
  background: var(--bg-primary);
}

.primary-result {
  background: var(--bg-active);
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  font-weight: 700;
  color: var(--text-primary);
}

.translation-text {
  margin: 0;
  font-size: 18px;
  line-height: 1.7;
  color: var(--text-primary);
}

.muted-text,
.tip-text {
  margin: 8px 0 0;
  color: var(--text-secondary);
  line-height: 1.6;
}

.pronunciation-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 10px;
}

.pron-card {
  border: 1px solid var(--border-lighter);
  border-radius: 8px;
  padding: 10px;
  background: var(--bg-tertiary);
}

.pron-card span,
.field-label {
  display: block;
  color: var(--text-secondary);
  font-size: 12px;
  margin-bottom: 4px;
}

.pron-card strong {
  color: var(--text-primary);
  word-break: break-word;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.clean-list {
  margin: 0;
  padding-left: 20px;
  color: var(--text-primary);
  line-height: 1.7;
}

.vocab-list,
.example-list,
.practice-list {
  display: grid;
  gap: 10px;
}

.vocab-item,
.example-item,
.practice-item {
  border: 1px solid var(--border-lighter);
  border-radius: 8px;
  padding: 10px;
  background: var(--bg-tertiary);
}

.vocab-item strong,
.example-item p,
.practice-item strong {
  display: block;
  margin: 0 0 6px;
  color: var(--text-primary);
}

.vocab-item span,
.example-item span,
.practice-item span {
  display: block;
  color: var(--text-secondary);
  line-height: 1.5;
}

.vocab-item small {
  display: block;
  margin-top: 4px;
  color: var(--text-tertiary);
  line-height: 1.5;
}

.correction-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 10px;
}

.correction-grid p {
  margin: 0;
  line-height: 1.6;
}

.code-box,
.raw-result {
  margin: 0;
  padding: 14px;
  border-radius: 8px;
  background: var(--code-bg);
  color: var(--code-text);
  overflow: auto;
  max-height: 360px;
  font-size: 12px;
  line-height: 1.6;
}

.history-list {
  display: grid;
  gap: 8px;
  max-height: 320px;
  overflow: auto;
}

.audio-player {
  width: 100%;
}

@media (max-width: 1100px) {
  .workspace-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .english-tutor-page {
    padding: 10px;
  }

  .header-actions,
  .action-row {
    align-items: stretch;
    flex-direction: column;
  }

  .page-header {
    padding: 14px;
  }

  .page-header h2 {
    font-size: 21px;
  }

  .page-header p {
    font-size: 13px;
    line-height: 1.5;
  }

  .action-row .el-button {
    width: 100%;
    margin-left: 0;
  }

  :deep(.mode-segmented) {
    --el-segmented-item-selected-color: var(--color-primary);
    padding: 4px;
  }

  :deep(.mode-segmented .el-segmented__group) {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 4px;
  }

  :deep(.mode-segmented .el-segmented__item) {
    min-height: 40px;
    padding: 0 6px;
  }

  :deep(.el-card__body) {
    padding: 14px;
  }

  .settings-row,
  .template-grid,
  .pronunciation-grid,
  .correction-grid {
    grid-template-columns: 1fr;
  }
}
</style>
