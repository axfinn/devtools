<template>
  <div class="tool-container ai-chat-page">
    <!-- Password gate -->
    <div v-if="!authenticated" class="auth-gate">
      <div class="auth-card">
        <div class="auth-icon">🔒</div>
        <h3>AI Chat 需要密码访问</h3>
        <p>请输入访问密码以使用 AI 对话功能</p>
        <el-form @submit.prevent="login" class="auth-form">
          <el-input
            v-model="passwordInput"
            type="password"
            show-password
            placeholder="请输入访问密码"
            size="large"
            @keyup.enter="login"
          />
          <el-button type="primary" size="large" :loading="loggingIn" @click="login" style="width:100%;margin-top:12px">
            进入
          </el-button>
        </el-form>
      </div>
    </div>

    <template v-else>
    <div class="tool-header">
      <div class="header-row">
        <div>
          <h2>AI Chat</h2>
          <p>多模型智能对话，支持图片/语音/音乐生成。安装 <a href="#" @click.prevent="downloadExtension">AskIt 浏览器扩展</a> 获得更强大的页面分析能力。</p>
        </div>
        <el-button type="primary" plain size="small" @click="downloadExtension">
          <el-icon><Download /></el-icon>
          下载 AskIt 扩展
        </el-button>
      </div>
    </div>

    <div class="chat-layout">
      <!-- Sidebar: conversations -->
      <div class="chat-sidebar">
        <el-button type="primary" class="new-chat-btn" @click="newConversation">
          <el-icon><Plus /></el-icon>
          新对话
        </el-button>
        <div class="conversation-list">
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conversation-item"
            :class="{ active: conv.id === currentConvId }"
            @click="switchConversation(conv.id)"
          >
            <span class="conv-title">{{ conv.title || '新对话' }}</span>
            <el-button text size="small" class="conv-delete" @click.stop="deleteConversation(conv.id)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </div>

      <!-- Main chat area -->
      <div class="chat-main">
        <!-- Settings bar -->
        <div class="chat-settings">
          <el-select v-model="selectedModel" placeholder="选择模型" size="small" class="model-select">
            <el-option-group label="MiniMax">
              <el-option value="MiniMax-M2.7" label="MiniMax-M2.7" />
              <el-option value="MiniMax-M2.5" label="MiniMax-M2.5" />
            </el-option-group>
            <el-option-group label="DeepSeek">
              <el-option value="deepseek-chat" label="DeepSeek Chat" />
              <el-option value="deepseek-reasoner" label="DeepSeek Reasoner" />
            </el-option-group>
            <el-option-group label="DashScope">
              <el-option value="qwen3.5-plus" label="Qwen 3.5 Plus" />
            </el-option-group>
          </el-select>
          <el-popover trigger="click" width="300">
            <template #reference>
              <el-button text size="small">
                <el-icon><Setting /></el-icon>
              </el-button>
            </template>
            <div class="settings-popover">
              <div class="setting-item">
                <label>系统提示词</label>
                <el-input v-model="systemPrompt" type="textarea" :rows="3" placeholder="可选：设置 AI 角色" />
              </div>
              <div class="setting-item">
                <label>温度 {{ temperature.toFixed(1) }}</label>
                <el-slider v-model="temperature" :min="0" :max="1" :step="0.1" />
              </div>
            </div>
          </el-popover>
          <el-button text size="small" @click="shareChat" :disabled="messages.length === 0" title="分享对话">
            📤 分享
          </el-button>
        </div>

        <!-- Messages -->
        <div class="messages-container" ref="messagesRef">
          <div v-if="messages.length === 0" class="empty-state">
            <div class="empty-icon">💬</div>
            <p>开始一段新对话，或选择下方工具使用多媒体功能</p>
          </div>
          <div v-for="msg in messages" :key="msg.id" class="message" :class="msg.role">
            <div class="message-avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
            <div class="message-bubble">
              <img v-if="msg.imageUrl" :src="msg.imageUrl" class="msg-user-image" />
              <div class="message-content" v-html="renderMarkdown(msg.content)"></div>
            </div>
          </div>
          <div v-if="streaming" class="message assistant">
            <div class="message-avatar">🤖</div>
            <div class="message-bubble">
              <div class="message-content">
                <span v-html="renderMarkdown(streamContent)"></span>
                <span class="cursor-blink">▊</span>
              </div>
            </div>
          </div>
          <div v-if="featureLoading" class="message assistant">
            <div class="message-avatar">🤖</div>
            <div class="message-bubble">
              <div class="message-content feature-loading">
                <span class="loading-spinner"></span>
                <span>{{ featureLoadingText }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Feature toolbar -->
        <div class="feature-toolbar">
          <button
            v-for="feat in FEATURES"
            :key="feat.id"
            class="feature-btn"
            :class="{ active: activeFeature === feat.id }"
            @click="toggleFeature(feat.id)"
            :title="feat.tip"
          >
            {{ feat.icon }} {{ feat.label }}
          </button>
          <span class="toolbar-divider"></span>
          <button class="feature-btn" @click="triggerImageUpload" title="上传图片">
            📎 图片
          </button>
          <input ref="fileInputRef" type="file" accept="image/*" style="display:none" @change="handleFileSelect" />
        </div>

        <!-- Image preview -->
        <div v-if="pendingImage" class="image-preview-area">
          <img :src="pendingImage" class="preview-thumb" />
          <el-button text size="small" @click="pendingImage = ''">✕ 移除</el-button>
        </div>

        <!-- Input -->
        <div class="chat-input-area">
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="2"
            :autosize="{ minRows: 2, maxRows: 6 }"
            :placeholder="inputPlaceholder"
            @keydown="handleKeydown"
            @paste="handlePaste"
            :disabled="streaming || featureLoading"
          />
          <div class="input-actions">
            <el-button v-if="!streaming && !featureLoading" type="primary" :disabled="!canSend" @click="sendMessage">
              发送
            </el-button>
            <el-button v-else-if="streaming" type="danger" @click="stopStreaming">
              停止
            </el-button>
            <el-button v-else type="info" disabled>
              处理中...
            </el-button>
          </div>
        </div>
      </div>
    </div>
    </template>
  </div>
</template>
<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Download, Plus, Delete, Setting } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'
import { API_BASE } from '../../api.js'

const md = new MarkdownIt({ html: true, linkify: true, breaks: true })

const FEATURES = [
  { id: 'image-gen', icon: '🎨', label: '绘画', tip: '输入描述生成图片' },
  { id: 'vision', icon: '👁️', label: '识图', tip: '上传图片让 AI 分析' },
  { id: 'tts', icon: '🔊', label: '朗读', tip: '输入文本生成语音' },
  { id: 'music', icon: '🎵', label: '作曲', tip: '描述风格生成音乐' },
  { id: 'music-cover', icon: '🎤', label: '翻唱', tip: '输入音频URL生成翻唱' },
]

const FEATURE_PLACEHOLDERS = {
  '': '输入消息... (Enter 发送, Shift+Enter 换行)',
  'image-gen': '描述你想生成的图片，如：一只在星空下奔跑的猫...',
  'vision': '输入问题，如：这张图片里有什么？（请先上传图片）',
  'tts': '输入要朗读的文本...',
  'music': '描述音乐风格，如：轻快的电子舞曲，带有钢琴旋律...',
  'music-cover': '第一行输入音频URL，第二行输入风格描述（可选）',
}

const AUTH_KEY = 'ai-chat-access-token'
const authenticated = ref(false)
const passwordInput = ref('')
const loggingIn = ref(false)

const selectedModel = ref('MiniMax-M2.7')
const systemPrompt = ref('')
const temperature = ref(0.7)
const inputText = ref('')
const messages = ref([])
const streaming = ref(false)
const streamContent = ref('')
const messagesRef = ref(null)
const currentConvId = ref('')
const conversations = ref([])
const activeFeature = ref('')
const pendingImage = ref('')
const featureLoading = ref(false)
const featureLoadingText = ref('')
const fileInputRef = ref(null)
let abortController = null

const inputPlaceholder = computed(() => FEATURE_PLACEHOLDERS[activeFeature.value] || FEATURE_PLACEHOLDERS[''])
const canSend = computed(() => {
  if (activeFeature.value === 'vision') return !!(inputText.value.trim() || pendingImage.value)
  return !!inputText.value.trim()
})

function mediaHeaders() {
  const pw = localStorage.getItem(AUTH_KEY) || ''
  return { 'Content-Type': 'application/json', 'X-Super-Admin-Password': pw }
}

function collectMediaUrls(value, bucket) {
  if (!value) return
  if (typeof value === 'string') {
    if (value.startsWith('http://') || value.startsWith('https://')) bucket.push(value)
    return
  }
  if (Array.isArray(value)) { value.forEach(v => collectMediaUrls(v, bucket)); return }
  if (typeof value === 'object') Object.values(value).forEach(v => collectMediaUrls(v, bucket))
}

function extractTaskUrls(task) {
  const urls = Array.isArray(task.result_urls) ? [...task.result_urls] : []
  collectMediaUrls(task.primary_asset, urls)
  collectMediaUrls(task.result, urls)
  return Array.from(new Set(urls))
}

// 异步媒体模型（图片/音乐/视频）需轮询任务直到完成
async function submitAndPollMediaTask(body, { interval = 4000, timeout = 480000 } = {}) {
  const resp = await fetch(`${API_BASE}/api/minimax/token-plan/v1/generations`, {
    method: 'POST',
    headers: mediaHeaders(),
    body: JSON.stringify(body)
  })
  const data = await resp.json().catch(() => ({}))
  if (!resp.ok) throw new Error(data.base_resp?.status_msg || data.error || '任务提交失败')

  // 同步返回（部分模型直接给出 URL）
  const inlineUrls = extractTaskUrls(data)
  if (inlineUrls.length > 0) return { urls: inlineUrls, taskId: data.task_id || '' }

  const taskId = data.task_id
  if (!taskId) throw new Error('未返回任务ID')

  const deadline = Date.now() + timeout
  while (Date.now() < deadline) {
    await new Promise(r => setTimeout(r, interval))
    const tResp = await fetch(`${API_BASE}/api/minimax/token-plan/tasks/${encodeURIComponent(taskId)}`, {
      headers: mediaHeaders()
    })
    const tData = await tResp.json().catch(() => ({}))
    if (!tResp.ok) continue
    if (tData.status === 'succeeded') {
      const urls = extractTaskUrls(tData)
      if (urls.length === 0) throw new Error('任务完成但未返回媒体URL')
      return { urls, taskId }
    }
    if (tData.status === 'failed') throw new Error(tData.error || '任务执行失败')
  }
  throw new Error('任务超时未完成')
}

// MiniMax 音频是带签名/无后缀的 CDN 链接，直接放进 <audio src> 常因过期/跨域/octet-stream 无法播放。
// 经后端下载代理拉成 blob（强制 audio/mpeg、同源、无过期）后再播放。
async function fetchTaskAudioBlobUrl(taskId, fallbackUrl) {
  if (taskId) {
    try {
      const resp = await fetch(`${API_BASE}/api/minimax/token-plan/tasks/${encodeURIComponent(taskId)}/download`, {
        headers: mediaHeaders()
      })
      if (resp.ok) {
        const blob = await resp.blob()
        return URL.createObjectURL(blob)
      }
    } catch {}
  }
  return fallbackUrl
}

onMounted(() => {
  const saved = localStorage.getItem(AUTH_KEY)
  if (saved) {
    authenticated.value = true
    initChat()
  }
})

async function login() {
  if (!passwordInput.value.trim()) { ElMessage.warning('请输入密码'); return }
  loggingIn.value = true
  try {
    const resp = await fetch(`${API_BASE}/api/ai-chat/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: passwordInput.value })
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok || data.error) throw new Error(data.error || '密码错误')
    localStorage.setItem(AUTH_KEY, passwordInput.value)
    authenticated.value = true
    ElMessage.success('验证通过')
    initChat()
  } catch (err) {
    ElMessage.error(err.message || '验证失败')
  } finally { loggingIn.value = false }
}

function initChat() {
  loadConversations()
  if (conversations.value.length === 0) { newConversation() }
  else { currentConvId.value = conversations.value[0].id; loadMessages() }
}

function generateId() { return Date.now().toString(36) + Math.random().toString(36).slice(2, 8) }
function loadConversations() { const raw = localStorage.getItem('ai-chat-conversations'); conversations.value = raw ? JSON.parse(raw) : [] }
function saveConversations() { localStorage.setItem('ai-chat-conversations', JSON.stringify(conversations.value)) }
function loadMessages() { const raw = localStorage.getItem(`ai-chat-messages-${currentConvId.value}`); messages.value = raw ? JSON.parse(raw) : [] }
function saveMessages() { localStorage.setItem(`ai-chat-messages-${currentConvId.value}`, JSON.stringify(messages.value)) }

function newConversation() {
  const id = generateId()
  conversations.value.unshift({ id, title: '新对话', createdAt: Date.now() })
  currentConvId.value = id
  messages.value = []
  saveConversations()
}
function switchConversation(id) { currentConvId.value = id; loadMessages() }
function deleteConversation(id) {
  conversations.value = conversations.value.filter(c => c.id !== id)
  localStorage.removeItem(`ai-chat-messages-${id}`)
  saveConversations()
  if (currentConvId.value === id) {
    if (conversations.value.length > 0) { currentConvId.value = conversations.value[0].id; loadMessages() }
    else { newConversation() }
  }
}

function renderMarkdown(text) { if (!text) return ''; return md.render(text) }
function scrollToBottom() { nextTick(() => { if (messagesRef.value) messagesRef.value.scrollTop = messagesRef.value.scrollHeight }) }
function handleKeydown(e) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage() } }

function toggleFeature(id) { activeFeature.value = activeFeature.value === id ? '' : id }
function triggerImageUpload() { fileInputRef.value?.click() }

function handleFileSelect(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => { pendingImage.value = reader.result; activeFeature.value = 'vision' }
  reader.readAsDataURL(file)
  e.target.value = ''
}

function handlePaste(e) {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      const reader = new FileReader()
      reader.onload = () => { pendingImage.value = reader.result; activeFeature.value = 'vision' }
      reader.readAsDataURL(file)
      return
    }
  }
}

function updateConvTitle(text) {
  const conv = conversations.value.find(c => c.id === currentConvId.value)
  if (conv && conv.title === '新对话') { conv.title = text.slice(0, 30); saveConversations() }
}
async function sendMessage() {
  const text = inputText.value.trim()
  if (!canSend.value || streaming.value || featureLoading.value) return

  if (activeFeature.value && activeFeature.value !== '') {
    await handleFeature(activeFeature.value, text)
    return
  }

  messages.value.push({ id: generateId(), role: 'user', content: text })
  inputText.value = ''
  updateConvTitle(text)
  scrollToBottom()

  streaming.value = true
  streamContent.value = ''
  abortController = new AbortController()

  const apiMessages = []
  if (systemPrompt.value.trim()) apiMessages.push({ role: 'system', content: systemPrompt.value.trim() })
  for (const msg of messages.value) apiMessages.push({ role: msg.role, content: msg.content })

  try {
    const response = await fetch(`${API_BASE}/api/internal/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ model: selectedModel.value, messages: apiMessages, temperature: temperature.value, stream: true }),
      signal: abortController.signal,
    })
    if (!response.ok) { const err = await response.json().catch(() => ({})); throw new Error(err.error || `HTTP ${response.status}`) }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6)
        if (data === '[DONE]') break
        if (data.startsWith(':')) continue
        try {
          const chunk = JSON.parse(data)
          const delta = chunk.choices?.[0]?.delta?.content
          if (delta) { streamContent.value += delta; scrollToBottom() }
        } catch {}
      }
    }
    if (streamContent.value) messages.value.push({ id: generateId(), role: 'assistant', content: streamContent.value })
  } catch (err) {
    if (err.name !== 'AbortError') {
      ElMessage.error(`请求失败: ${err.message}`)
      if (streamContent.value) messages.value.push({ id: generateId(), role: 'assistant', content: streamContent.value })
    }
  } finally {
    streaming.value = false
    streamContent.value = ''
    saveMessages()
    scrollToBottom()
  }
}

function stopStreaming() { if (abortController) { abortController.abort(); abortController = null } }

async function handleFeature(feature, text) {
  inputText.value = ''
  const userContent = text || (feature === 'vision' ? '请分析这张图片' : '')
  const userMsg = { id: generateId(), role: 'user', content: userContent }
  if (feature === 'vision' && pendingImage.value) userMsg.imageUrl = pendingImage.value
  messages.value.push(userMsg)
  updateConvTitle(userContent)
  scrollToBottom()

  featureLoading.value = true
  try {
    switch (feature) {
      case 'image-gen': await handleImageGen(text); break
      case 'vision': await handleVision(text, pendingImage.value); break
      case 'tts': await handleTTS(text); break
      case 'music': await handleMusic(text); break
      case 'music-cover': await handleMusicCover(text); break
    }
  } catch (err) {
    messages.value.push({ id: generateId(), role: 'assistant', content: `❌ 错误: ${err.message}` })
  } finally {
    featureLoading.value = false
    featureLoadingText.value = ''
    pendingImage.value = ''
    saveMessages()
    scrollToBottom()
  }
}

async function handleImageGen(prompt) {
  featureLoadingText.value = '🎨 正在生成图片...'
  const { urls } = await submitAndPollMediaTask(
    { model: 'image-01', prompt, aspect_ratio: '1:1', n: 1, prompt_optimizer: true, response_format: 'url' },
    { interval: 3000, timeout: 300000 }
  )
  messages.value.push({ id: generateId(), role: 'assistant', content: `![生成的图片](${urls[0]})` })
}

async function handleVision(prompt, imageDataUrl) {
  if (!imageDataUrl) throw new Error('请先上传图片')
  featureLoadingText.value = '👁️ 正在分析图片...'
  const resp = await fetch(`${API_BASE}/api/image-understanding/qwen-vision`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ images: [imageDataUrl], prompt: prompt || '请描述图片内容' })
  })
  const data = await resp.json().catch(() => ({}))
  if (!resp.ok) throw new Error(data.error || `图片分析失败 (HTTP ${resp.status})`)
  messages.value.push({ id: generateId(), role: 'assistant', content: data.text || '无法分析图片' })
}

async function handleTTS(text) {
  if (!text) throw new Error('请输入要朗读的文本')
  featureLoadingText.value = '🔊 正在合成语音...'
  const resp = await fetch(`${API_BASE}/api/minimax/tts/v1/generations`, {
    method: 'POST',
    headers: mediaHeaders(),
    body: JSON.stringify({ model: 'speech-2.8-hd', text, voice: 'shanghai', speed: 1.0, audio_format: 'mp3' })
  })
  if (!resp.ok) { const d = await resp.json().catch(() => ({})); throw new Error(d.error || '语音合成失败') }
  const blob = await resp.blob()
  const blobUrl = URL.createObjectURL(blob)
  messages.value.push({ id: generateId(), role: 'assistant', content: `<audio controls src="${blobUrl}"></audio>` })
}
async function handleMusic(prompt) {
  if (!prompt) throw new Error('请描述音乐风格')
  featureLoadingText.value = '🎵 正在生成歌词...'
  const lyricsResp = await fetch(`${API_BASE}/api/minimax/music/v1/lyrics_generation`, {
    method: 'POST',
    headers: mediaHeaders(),
    body: JSON.stringify({ mode: 'write_full_song', prompt })
  })
  const lyricsData = await lyricsResp.json()
  if (!lyricsResp.ok || lyricsData.base_resp?.status_code) throw new Error(lyricsData.base_resp?.status_msg || lyricsData.error || '歌词生成失败')
  const lyrics = lyricsData.lyrics || lyricsData.data?.lyrics || ''
  const title = lyricsData.song_title || lyricsData.data?.title || '未命名'

  featureLoadingText.value = '🎵 正在创作音乐...'
  const { urls, taskId } = await submitAndPollMediaTask({
    model: 'music-2.6', prompt, lyrics,
    audio_setting: { format: 'mp3', sample_rate: 44100, bitrate: 256000 },
    output_format: 'url'
  }, { interval: 4000, timeout: 480000 })
  const audioSrc = await fetchTaskAudioBlobUrl(taskId, urls[0])
  messages.value.push({ id: generateId(), role: 'assistant', content: `**🎵 ${title}**\n\n${lyrics}\n\n<audio controls src="${audioSrc}"></audio>` })
}

async function handleMusicCover(text) {
  const lines = text.split('\n').filter(l => l.trim())
  const audioUrl = lines[0]?.trim()
  const prompt = lines.slice(1).join(' ').trim() || 'Mandopop, warm vocal'
  if (!audioUrl || !audioUrl.startsWith('http')) throw new Error('第一行请输入有效的音频URL')

  featureLoadingText.value = '🎤 正在提取音频特征...'
  const preResp = await fetch(`${API_BASE}/api/minimax/music/v1/cover_preprocess`, {
    method: 'POST',
    headers: mediaHeaders(),
    body: JSON.stringify({ model: 'music-cover', audio_url: audioUrl })
  })
  const preData = await preResp.json()
  if (!preResp.ok || preData.base_resp?.status_code) throw new Error(preData.base_resp?.status_msg || preData.error || '音频预处理失败')
  const coverFeatureId = preData.cover_feature_id || preData.data?.cover_feature_id
  const formattedLyrics = preData.formatted_lyrics || preData.data?.formatted_lyrics || ''
  if (!coverFeatureId) throw new Error('未返回特征ID')

  featureLoadingText.value = '🎤 正在生成翻唱...'
  const { urls, taskId } = await submitAndPollMediaTask({
    model: 'music-cover', cover_feature_id: coverFeatureId,
    lyrics: formattedLyrics, prompt,
    audio_setting: { format: 'mp3', sample_rate: 44100, bitrate: 256000 },
    output_format: 'url'
  }, { interval: 4000, timeout: 480000 })
  const audioSrc = await fetchTaskAudioBlobUrl(taskId, urls[0])
  messages.value.push({ id: generateId(), role: 'assistant', content: `**🎤 翻唱生成完成**\n\n<audio controls src="${audioSrc}"></audio>` })
}

function downloadExtension() { window.open(`${API_BASE}/api/askit/extension`, '_blank') }

async function shareChat() {
  if (messages.value.length === 0) { ElMessage.warning('没有对话记录可分享'); return }
  try {
    const conv = conversations.value.find(c => c.id === currentConvId.value)
    const title = conv?.title || 'AI Chat 对话'
    const payload = JSON.stringify({
      messages: messages.value.map(m => ({ role: m.role, content: m.content, imageUrl: m.imageUrl })),
      title,
      exportedAt: new Date().toISOString(),
    })
    const resp = await fetch(`${API_BASE}/api/paste`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: payload, title, language: 'askit-chat', expires_in: 72 })
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) throw new Error(data.error || `分享失败 (HTTP ${resp.status})`)
    const shareUrl = `${location.origin}/chat/${data.id}`
    await navigator.clipboard.writeText(shareUrl)
    ElMessage.success(`分享链接已复制: ${shareUrl}`)
  } catch (err) {
    ElMessage.error(`分享失败: ${err.message}`)
  }
}

watch(messages, () => { if (currentConvId.value) saveMessages() }, { deep: true })
</script>
<style scoped>
.auth-gate { display: flex; align-items: center; justify-content: center; min-height: 60vh; }
.auth-card { text-align: center; padding: 40px; border-radius: 16px; background: var(--bg-card, #fff); box-shadow: 0 4px 24px rgba(0,0,0,0.06); max-width: 360px; width: 100%; }
.auth-icon { font-size: 48px; margin-bottom: 16px; }
.auth-card h3 { margin: 0 0 8px; font-size: 18px; color: var(--text-primary, #1f2937); }
.auth-card p { margin: 0 0 20px; font-size: 13px; color: var(--text-secondary, #6b7280); }
.auth-form { max-width: 280px; margin: 0 auto; }

.ai-chat-page { height: calc(100vh - 60px); display: flex; flex-direction: column; }
.tool-header { padding: 16px 24px; border-bottom: 1px solid var(--border-light, #eee); }
.tool-header h2 { margin: 0 0 4px; font-size: 18px; }
.tool-header p { margin: 0; font-size: 13px; color: #666; }
.tool-header a { color: #409eff; text-decoration: none; }
.header-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; }

.chat-layout { flex: 1; display: flex; overflow: hidden; }
.chat-sidebar { width: 220px; border-right: 1px solid var(--border-light, #eee); display: flex; flex-direction: column; padding: 12px; background: var(--bg-sidebar, #fafafa); }
.new-chat-btn { width: 100%; margin-bottom: 12px; }
.conversation-list { flex: 1; overflow-y: auto; }
.conversation-item { padding: 8px 12px; border-radius: 6px; cursor: pointer; display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; font-size: 13px; color: #333; }
.conversation-item:hover { background: #f0f0f0; }
.conversation-item.active { background: #e8f0fe; color: #1a73e8; }
.conv-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.conv-delete { opacity: 0; }
.conversation-item:hover .conv-delete { opacity: 1; }

.chat-main { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.chat-settings { padding: 8px 16px; border-bottom: 1px solid var(--border-light, #eee); display: flex; align-items: center; gap: 8px; }
.model-select { width: 200px; }
.settings-popover .setting-item { margin-bottom: 12px; }
.settings-popover label { display: block; font-size: 12px; color: #666; margin-bottom: 4px; }

.messages-container { flex: 1; overflow-y: auto; padding: 16px 24px; }
.empty-state { text-align: center; padding: 60px 0; color: #999; }
.empty-icon { font-size: 48px; margin-bottom: 12px; }

.message { display: flex; gap: 12px; margin-bottom: 16px; max-width: 85%; }
.message.user { margin-left: auto; flex-direction: row-reverse; }
.message-avatar { width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0; background: #f5f5f5; }
.message-bubble { display: flex; flex-direction: column; min-width: 0; }
.message-content { padding: 10px 14px; border-radius: 12px; font-size: 14px; line-height: 1.6; word-break: break-word; }
.message.user .message-content { background: #e8f0fe; color: #1a1a1a; }
.message.assistant .message-content { background: #f5f5f5; color: #1a1a1a; }
.msg-user-image { max-width: 200px; max-height: 150px; border-radius: 8px; margin-bottom: 6px; }
.message-content :deep(pre) { background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 8px; overflow-x: auto; font-size: 13px; }
.message-content :deep(code) { font-family: 'Fira Code', monospace; }
.message-content :deep(p) { margin: 0 0 8px; }
.message-content :deep(p:last-child) { margin: 0; }
.message-content :deep(img) { max-width: 100%; max-height: 400px; border-radius: 8px; margin: 8px 0; display: block; }
.message-content :deep(audio) { width: 100%; margin: 8px 0; display: block; }

.feature-loading { display: flex; align-items: center; gap: 8px; color: #666; }
.loading-spinner { width: 14px; height: 14px; border: 2px solid #ddd; border-top-color: #409eff; border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.cursor-blink { animation: blink 1s infinite; }
@keyframes blink { 0%, 50% { opacity: 1; } 51%, 100% { opacity: 0; } }

.feature-toolbar { display: flex; align-items: center; gap: 4px; padding: 6px 24px; flex-wrap: wrap; }
.feature-btn { padding: 4px 10px; border-radius: 14px; font-size: 12px; cursor: pointer; border: 1px solid #e0e0e0; background: #fff; transition: all 0.15s; white-space: nowrap; }
.feature-btn:hover { background: #f0f7ff; border-color: #b3d8ff; }
.feature-btn.active { background: #409eff; color: #fff; border-color: #409eff; }
.toolbar-divider { width: 1px; height: 20px; background: #e0e0e0; margin: 0 4px; }

.image-preview-area { padding: 4px 24px; display: flex; align-items: center; gap: 8px; }
.preview-thumb { max-height: 60px; border-radius: 6px; border: 1px solid #eee; }

.chat-input-area { padding: 12px 24px 16px; display: flex; gap: 12px; align-items: flex-end; }
.chat-input-area .el-textarea { flex: 1; }
.input-actions { flex-shrink: 0; }

@media (max-width: 768px) {
  .chat-sidebar { display: none; }
  .chat-layout { flex-direction: column; }
  .feature-toolbar { padding: 6px 12px; }
}
</style>
