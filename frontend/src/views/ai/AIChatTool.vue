<template>
  <div class="tool-container chatplus-page">
    <!-- ═══════════ 密码门 ═══════════ -->
    <div v-if="!auth.isAuthenticated.value" class="auth-gate">
      <div class="auth-card">
        <div class="auth-logo">✦</div>
        <h3>AI Chat</h3>
        <p>局域网 Ollama + MiniMax 双引擎对话</p>
        <el-form @submit.prevent="auth.login" class="auth-form">
          <el-input
            v-model="auth.passwordInput.value"
            type="password"
            show-password
            placeholder="请输入访问密码"
            size="large"
            @keyup.enter="auth.login"
          />
          <el-button type="primary" size="large" :loading="auth.loggingIn.value" class="auth-btn" @click="auth.login">
            进入
          </el-button>
        </el-form>
      </div>
    </div>

    <!-- ═══════════ 主界面 ═══════════ -->
    <div v-else class="chatplus-shell">
      <!-- 移动端抽屉遮罩 -->
      <div v-if="sidebarOpen && isMobileViewport" class="cp-backdrop" @click="sidebarOpen = false"></div>
      <!-- 侧栏：会话列表 -->
      <aside class="cp-sidebar" :class="{ open: sidebarOpen }">
        <button class="cp-new-chat" @click="newConversation">
          <span class="ic">＋</span>
          <span v-if="!sidebarCollapsed">新对话</span>
        </button>
        <div class="cp-conv-list">
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="cp-conv-item"
            :class="{ active: conv.id === currentConvId }"
            :title="conv.title"
            @click="switchConversation(conv.id)"
          >
            <span class="conv-title">{{ conv.title || '新对话' }}</span>
            <button class="conv-del" title="删除" @click.stop="deleteConversation(conv.id)">
              <el-icon><Delete /></el-icon>
            </button>
          </div>
          <div v-if="conversations.length === 0" class="cp-conv-empty">暂无会话</div>
        </div>
        <div class="cp-sidebar-footer">
          <button class="cp-side-btn" @click="toggleSidebar" title="关闭侧栏">
            <span class="ic">☰</span>
            <span>关闭侧栏</span>
          </button>
          <button class="cp-side-btn" @click="auth.logout">
            <span class="ic">⎋</span>
            <span>退出登录</span>
          </button>
        </div>
      </aside>

      <!-- 主区域 -->
      <main class="cp-main">
        <!-- 顶栏 -->
        <header class="cp-topbar">
          <button v-if="!sidebarOpen" class="cp-icon-btn" @click="toggleSidebar" title="打开侧栏">☰</button>
          <el-select
            v-model="settings.model"
            placeholder="选择模型"
            size="default"
            class="cp-model-select"
            :loading="modelsLoading"
          >
            <el-option-group v-for="g in modelGroups" :key="g.label" :label="g.label">
              <el-option
                v-for="m in g.items"
                :key="m.value"
                :value="m.value"
                :label="m.label"
              >
                <span class="model-option-label">{{ m.label }}</span>
                <span v-if="m.tag" class="model-option-tag">{{ m.tag }}</span>
              </el-option>
            </el-option-group>
          </el-select>
          <span v-if="engineStatus" class="engine-badge" :class="engineStatus.type">{{ engineStatus.text }}</span>
          <div class="topbar-spacer"></div>
          <el-popover trigger="click" width="320" placement="bottom-end">
            <template #reference>
              <button class="cp-icon-btn" title="对话设置">⚙</button>
            </template>
            <div class="cp-settings-pop">
              <div class="setting-item">
                <label>系统提示词（仅对当前设置生效）</label>
                <el-input v-model="settings.systemPrompt" type="textarea" :rows="4" placeholder="可选：为 AI 设定角色，例如「你是一个严谨的技术助手」" />
              </div>
              <div class="setting-item">
                <label>随机性 Temperature：{{ settings.temperature.toFixed(1) }}</label>
                <el-slider v-model="settings.temperature" :min="0" :max="1.5" :step="0.1" />
              </div>
            </div>
          </el-popover>
        </header>

        <!-- 消息区 -->
        <div class="cp-messages" ref="messagesRef" @scroll="onScroll">
          <!-- 欢迎屏 -->
          <div v-if="messages.length === 0 && !streaming" class="cp-welcome">
            <div class="welcome-logo">✦</div>
            <h2>有什么可以帮你？</h2>
            <p class="welcome-sub">本地 Ollama 模型与 MiniMax 云端模型随时切换，对话仅保存在你的浏览器本地</p>
            <div class="welcome-chips">
              <button v-for="s in SUGGESTIONS" :key="s" class="chip" @click="quickAsk(s)">{{ s }}</button>
            </div>
          </div>

          <!-- 消息列表 -->
          <template v-for="msg in displayMessages" :key="msg.id">
            <div class="cp-msg" :class="msg.role">
              <div class="msg-avatar" :class="msg.role">{{ msg.role === 'user' ? '你' : '✦' }}</div>
              <div class="msg-body">
                <!-- 思考过程（可折叠） -->
                <details v-if="msg.reasoning" class="msg-reasoning" :open="msg.streaming && !msg.content">
                  <summary>思考过程</summary>
                  <div class="reasoning-content">{{ msg.reasoning }}</div>
                </details>
                <div class="msg-content" :class="{ streaming: msg.streaming }" v-html="renderMarkdown(msg.content)"></div>
                <!-- 操作行 -->
                <div v-if="!msg.streaming" class="msg-actions">
                  <button v-if="msg.role === 'assistant'" class="action-btn" title="复制回复" @click="copyMessage(msg)">
                    <el-icon><CopyDocument /></el-icon>
                  </button>
                  <button
                    v-if="msg.role === 'assistant' && msg === lastAssistant && !streaming"
                    class="action-btn"
                    title="重新生成"
                    @click="regenerate"
                  >
                    <el-icon><RefreshRight /></el-icon>
                  </button>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- 输入区 -->
        <footer class="cp-composer">
          <div v-if="errorMsg" class="cp-error">
            <el-icon><WarningFilled /></el-icon>
            <span>{{ errorMsg }}</span>
            <button class="err-close" @click="errorMsg = ''">✕</button>
          </div>
          <div class="composer-box" :class="{ focused: composerFocused }">
            <el-input
              v-model="userInput"
              type="textarea"
              :autosize="{ minRows: 1, maxRows: 8 }"
              placeholder="输入消息…（Enter 发送，Shift + Enter 换行）"
              resize="none"
              @focus="composerFocused = true"
              @blur="composerFocused = false"
              @keydown="onKeydown"
              class="composer-input"
            />
            <button v-if="streaming" class="send-btn stop" title="停止生成" @click="stopGeneration">
              <span class="stop-square"></span>
            </button>
            <button v-else class="send-btn" :disabled="!userInput.trim() || modelsLoading" title="发送" @click="sendMessage">
              <el-icon><Promotion /></el-icon>
            </button>
          </div>
          <div class="composer-hint">内容由 AI 生成，请仔细甄别 · MiniMax 密钥保存在服务端，不会暴露给浏览器</div>
        </footer>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted, onActivated, onDeactivated, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, CopyDocument, RefreshRight, Promotion, WarningFilled } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'
import hljs from '../../utils/highlight'
import 'highlight.js/styles/github-dark.css'
import texmath from 'markdown-it-texmath'
import katex from 'katex'
import 'katex/dist/katex.min.css'
import { API_BASE } from '../../api'
import { useAdminAuth } from '../../composables/useAdminAuth'

// ══════════════ 常量 ══════════════
const CONV_LIST_KEY = 'chat-plus-conv-list'
const MSGS_KEY_PREFIX = 'chat-plus-msgs-'
const SETTINGS_KEY = 'chat-plus-settings'

const SUGGESTIONS = [
  '用通俗的语言解释一下 Transformer 的注意力机制',
  '帮我写一个 Go 语言的并发安全 LRU 缓存',
  '把这段话润色得更专业：……',
  '给我一周减脂期的三餐搭配建议',
]

// ══════════════ 密码门 ══════════════
const auth = useAdminAuth({
  storageKey: 'chat-plus-access-token',
  verifyEndpoint: '/api/chat-plus/verify',
  credentialMode: 'body',
  headerName: 'X-Chat-Password',
  trustStored: true,
})

// ══════════════ Markdown 渲染 ══════════════
const md = new MarkdownIt({
  html: false, // 防 XSS：模型输出也可能被注入 HTML
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    const langLabel = lang || 'text'
    const encoded = encodeURIComponent(str)
    let body
    if (lang && hljs.getLanguage(lang)) {
      try {
        body = hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
      } catch { body = '' }
    }
    if (!body) body = md.utils.escapeHtml(str)
    return `<div class="code-block"><div class="code-head"><span class="code-lang">${md.utils.escapeHtml(langLabel)}</span>` +
      `<button class="code-copy" data-code="${encoded}">复制</button></div>` +
      `<pre class="hljs"><code>${body}</code></pre></div>`
  },
})
md.use(texmath, { engine: katex, delimiters: ['dollars', 'brackets'], katexOptions: { throwOnError: false } })

function renderMarkdown(content) {
  if (!content) return ''
  return md.render(content)
}

// 代码块复制：事件委托（v-html 内容无法绑定局部事件）
function handleContentClick(e) {
  const btn = e.target.closest?.('.code-copy')
  if (!btn) return
  const code = decodeURIComponent(btn.dataset.code || '')
  navigator.clipboard.writeText(code).then(() => {
    btn.textContent = '已复制'
    setTimeout(() => { btn.textContent = '复制' }, 1500)
  })
}

// ══════════════ 会话与消息存储 ══════════════
const conversations = ref([])   // [{id,title,createdAt,updatedAt}]
const currentConvId = ref('')
const messages = ref([])        // 当前会话消息 [{id,role,content,reasoning,model,ts}]
const settings = ref({ model: '', systemPrompt: '', temperature: 0.7 })
const userInput = ref('')
const composerFocused = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
const sidebarOpen = ref(true)
const isMobileViewport = ref(false)
const modelsLoading = ref(false)
const modelInfo = ref({ minimax: { available: false, models: [] }, ollama: { available: false, models: [], error: '' } })

function uid() { return Date.now().toString(36) + Math.random().toString(36).slice(2, 8) }

function loadConversations() {
  try { conversations.value = JSON.parse(localStorage.getItem(CONV_LIST_KEY) || '[]') } catch { conversations.value = [] }
}
function saveConversations() {
  localStorage.setItem(CONV_LIST_KEY, JSON.stringify(conversations.value))
}
function loadMessages(convId) {
  try { messages.value = JSON.parse(localStorage.getItem(MSGS_KEY_PREFIX + convId) || '[]') } catch { messages.value = [] }
}
function saveMessages() {
  if (currentConvId.value) localStorage.setItem(MSGS_KEY_PREFIX + currentConvId.value, JSON.stringify(messages.value))
}
function loadSettings() {
  try {
    const s = JSON.parse(localStorage.getItem(SETTINGS_KEY) || '{}')
    settings.value = {
      model: s.model || '',
      systemPrompt: s.systemPrompt || '',
      temperature: typeof s.temperature === 'number' ? s.temperature : 0.7,
    }
  } catch { /* 保持默认 */ }
}
function saveSettings() {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings.value))
}

// ══════════════ 模型列表 ══════════════
const modelGroups = computed(() => {
  const groups = []
  const ol = modelInfo.value.ollama
  if (ol.available && ol.models.length) {
    groups.push({
      label: `Ollama 本地（${ol.models.length}）`,
      items: ol.models.map(m => ({
        value: m.id,
        label: m.name,
        tag: m.parameter_size || (m.size_gb ? `${m.size_gb.toFixed(1)}GB` : ''),
      })),
    })
  }
  const mm = modelInfo.value.minimax
  if (mm.available) {
    groups.push({
      label: 'MiniMax 云端',
      items: mm.models.map(m => ({ value: m.id, label: m.label, tag: m.description })),
    })
  }
  return groups
})

const currentModelLabel = computed(() => {
  for (const g of modelGroups.value) {
    const found = g.items.find(m => m.value === settings.value.model)
    if (found) return found.label
  }
  return settings.value.model || '选择模型'
})

const engineStatus = computed(() => {
  const { minimax, ollama } = modelInfo.value
  const parts = []
  if (ollama.available) parts.push(`Ollama ${ollama.models.length}`)
  if (minimax.available) parts.push('MiniMax')
  if (parts.length === 0) return { type: 'off', text: ollama.error ? '引擎不可用' : '加载中…' }
  return { type: 'on', text: parts.join(' · ') }
})

async function loadModels() {
  modelsLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/models`, { headers: auth.authHeader() })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${res.status}`)
    }
    modelInfo.value = await res.json()
    // 默认模型：优先 MiniMax-M3，其次第一个可用模型
    if (!settings.value.model || !modelGroups.value.some(g => g.items.some(m => m.value === settings.value.model))) {
      const first = modelGroups.value.flatMap(g => g.items)[0]
      settings.value.model = first ? first.value : ''
    }
    if (!modelInfo.value.ollama.available && modelInfo.value.ollama.error) {
      errorMsg.value = `Ollama（${modelInfo.value.ollama.base_url}）暂不可用：${modelInfo.value.ollama.error}`
    }
  } catch (e) {
    errorMsg.value = `获取模型列表失败：${e.message}`
  } finally {
    modelsLoading.value = false
  }
}

// ══════════════ 会话管理 ══════════════
function newConversation() {
  switchConversation('__draft__')
}
function switchConversation(id) {
  if (isMobileViewport.value) sidebarOpen.value = false // 移动端选完会话自动收起抽屉
  if (id === '__draft__') {
    currentConvId.value = ''
    messages.value = []
    return
  }
  currentConvId.value = id
  loadMessages(id)
}
function deleteConversation(id) {
  conversations.value = conversations.value.filter(c => c.id !== id)
  localStorage.removeItem(MSGS_KEY_PREFIX + id)
  saveConversations()
  if (currentConvId.value === id) switchConversation('__draft__')
}
function ensureConversation(firstUserText) {
  if (currentConvId.value) return
  const conv = {
    id: uid(),
    title: (firstUserText || '新对话').slice(0, 30),
    createdAt: Date.now(),
    updatedAt: Date.now(),
  }
  conversations.value.unshift(conv)
  currentConvId.value = conv.id
  saveConversations()
}
function touchConversation() {
  const conv = conversations.value.find(c => c.id === currentConvId.value)
  if (conv) {
    conv.updatedAt = Date.now()
    saveConversations()
  }
}

// ══════════════ 发送 / 流式接收 ══════════════
const messagesRef = ref(null)
let stickToBottom = true
let abortController = null

const displayMessages = computed(() => {
  const list = [...messages.value]
  if (streaming.value && streamMsg.value) list.push(streamMsg.value)
  return list
})
const lastAssistant = computed(() => [...messages.value].reverse().find(m => m.role === 'assistant'))
const streamMsg = ref(null)

function scrollToBottom(force = false) {
  if (!stickToBottom && !force) return
  nextTick(() => {
    const el = messagesRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}
function onScroll() {
  const el = messagesRef.value
  if (!el) return
  stickToBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

function buildPayloadMessages() {
  const msgs = []
  if (settings.value.systemPrompt.trim()) {
    msgs.push({ role: 'system', content: settings.value.systemPrompt.trim() })
  }
  for (const m of messages.value) {
    if (m.content && m.content.trim()) msgs.push({ role: m.role, content: m.content })
  }
  return msgs
}

function quickAsk(text) {
  userInput.value = text
  sendMessage()
}

function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    sendMessage()
  }
}

async function sendMessage() {
  const text = userInput.value.trim()
  if (!text || streaming.value) return
  if (!settings.value.model) {
    ElMessage.warning('请先选择模型')
    return
  }
  errorMsg.value = ''
  userInput.value = ''
  ensureConversation(text)
  messages.value.push({ id: uid(), role: 'user', content: text, ts: Date.now() })
  saveMessages()
  scrollToBottom(true)
  await doStream()
}

async function regenerate() {
  if (streaming.value) return
  // 去掉最后一条 assistant 消息后重发
  const idx = messages.value.map(m => m.role).lastIndexOf('assistant')
  if (idx >= 0) messages.value.splice(idx, 1)
  saveMessages()
  await doStream()
}

async function doStream() {
  streaming.value = true
  stickToBottom = true
  streamMsg.value = { id: uid(), role: 'assistant', content: '', reasoning: '', model: settings.value.model, ts: Date.now(), streaming: true }
  scrollToBottom(true)

  abortController = new AbortController()
  let stopped = false
  abortController.signal.addEventListener('abort', () => { stopped = true })

  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
      body: JSON.stringify({
        model: settings.value.model,
        messages: buildPayloadMessages(),
        temperature: settings.value.temperature,
        stream: true,
      }),
      signal: abortController.signal,
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${res.status}`)
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buf = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      const lines = buf.split('\n')
      buf = lines.pop()
      for (const raw of lines) {
        const line = raw.trim()
        if (!line.startsWith('data:')) continue        // 心跳注释行 / 空行跳过
        const data = line.slice(5).trim()
        if (data === '[DONE]') continue
        let chunk
        try { chunk = JSON.parse(data) } catch { continue }
        const delta = chunk?.choices?.[0]?.delta || {}
        const reasoningPiece = delta.reasoning || delta.reasoning_content || ''
        if (reasoningPiece) streamMsg.value.reasoning += reasoningPiece
        if (delta.content) {
          streamMsg.value.content += delta.content
          scrollToBottom()
        } else if (reasoningPiece) {
          scrollToBottom()
        }
      }
    }
  } catch (e) {
    if (!stopped) {
      errorMsg.value = `请求失败：${e.message}`
    }
  } finally {
    const sm = streamMsg.value
    const hasContent = sm && (sm.content || sm.reasoning)
    if (sm && hasContent) {
      messages.value.push({ id: sm.id, role: 'assistant', content: sm.content, reasoning: sm.reasoning, model: sm.model, ts: Date.now() })
      saveMessages()
      touchConversation()
    }
    streamMsg.value = null
    streaming.value = false
    abortController = null
    scrollToBottom()
  }
}

function stopGeneration() {
  abortController?.abort()
}

function copyMessage(msg) {
  navigator.clipboard.writeText(msg.content).then(() => ElMessage.success('已复制'))
}

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

// ══════════════ 生命周期 ══════════════
let mqMobile = null
function handleViewportChange(e) {
  isMobileViewport.value = e.matches
  if (e.matches) sidebarOpen.value = false // 转竖屏/小屏时默认收起抽屉
}

// keep-alive 下切页时 body class 必须跟随激活状态，否则会锁死其他页面的滚动
function activateChatShell() {
  document.body.classList.add('chatplus-active')
  document.body.classList.toggle('chatplus-gate', !auth.isAuthenticated.value)
}
function deactivateChatShell() {
  document.body.classList.remove('chatplus-active')
  document.body.classList.remove('chatplus-gate')
}

onMounted(async () => {
  document.addEventListener('click', handleContentClick)
  activateChatShell()
  mqMobile = window.matchMedia('(max-width: 768px)')
  isMobileViewport.value = mqMobile.matches
  sidebarOpen.value = !isMobileViewport.value
  if (mqMobile.addEventListener) mqMobile.addEventListener('change', handleViewportChange)
  loadSettings()
  loadConversations()
  await auth.tryStored() // 成功后由上面的 watch 触发 loadModels，避免重复请求
})

onUnmounted(() => {
  document.removeEventListener('click', handleContentClick)
  deactivateChatShell()
  if (mqMobile && mqMobile.removeEventListener) mqMobile.removeEventListener('change', handleViewportChange)
})
// 未登录时隐藏全局宠物，避免宠物窗口压在密码卡片/输入框上；
// 登录成功（含首次输入密码）后必须再拉一次模型列表，否则下拉为空导致无法发送
watch(
  () => auth.isAuthenticated.value,
  (authed) => {
    document.body.classList.toggle('chatplus-gate', !authed)
    if (authed) loadModels()
  },
  { immediate: true }
)

onDeactivated(() => {
  deactivateChatShell()
  if (mqMobile && mqMobile.removeEventListener) mqMobile.removeEventListener('change', handleViewportChange)
})
onActivated(() => {
  activateChatShell()
  if (mqMobile) {
    isMobileViewport.value = mqMobile.matches
    if (mqMobile.addEventListener) mqMobile.addEventListener('change', handleViewportChange)
  }
  // 回到本页时若模型列表为空（如首次登录时接口失败），补拉一次
  if (auth.isAuthenticated.value && modelGroups.value.length === 0) loadModels()
})
</script>

<style scoped>
.chatplus-page {
  padding: 0 !important;
  height: calc(100dvh - 40px); /* 桌面端：main-content 上下各 20px 内边距 */
  min-height: 0;
  gap: 0;
  overflow: hidden;
  box-sizing: border-box;
}

/* ═══════════ 密码门 ═══════════ */
.auth-gate {
  display: flex; align-items: center; justify-content: center;
  height: 100%; min-height: 420px;
}
.auth-card {
  text-align: center; padding: 44px 40px; border-radius: 20px;
  background: var(--bg-primary); border: 1px solid var(--border-light);
  box-shadow: 0 8px 32px var(--shadow-light); max-width: 380px; width: 90%;
}
.auth-logo {
  width: 56px; height: 56px; margin: 0 auto 16px; border-radius: 16px;
  display: flex; align-items: center; justify-content: center;
  font-size: 26px; color: #fff;
  background: linear-gradient(135deg, #409eff, #7c5cff);
}
.auth-card h3 { margin: 0 0 6px; font-size: 20px; color: var(--text-primary); }
.auth-card p { margin: 0 0 22px; font-size: 13px; color: var(--text-secondary); }
.auth-btn { width: 100%; margin-top: 14px; border-radius: 10px; }

/* ═══════════ 布局 ═══════════ */
.chatplus-shell {
  display: flex; height: 100%; overflow: hidden;
}

/* 侧栏 */
.cp-sidebar {
  width: 250px; flex-shrink: 0;
  display: flex; flex-direction: column;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-light);
  transition: width 0.2s ease, transform 0.25s ease;
  overflow: hidden;
}
.cp-sidebar:not(.open) { width: 0; border-right: none; }
.cp-backdrop { display: none; }
.cp-new-chat {
  margin: 12px; padding: 10px 14px;
  display: flex; align-items: center; gap: 8px;
  border: 1px dashed var(--border-base); border-radius: 10px;
  background: var(--bg-primary); color: var(--text-primary);
  cursor: pointer; font-size: 14px; white-space: nowrap;
  transition: all 0.15s;
}
.cp-new-chat:hover { border-color: var(--color-primary); color: var(--color-primary); }
.cp-new-chat .ic { font-size: 16px; font-weight: 600; }
.cp-conv-list { flex: 1; overflow-y: auto; padding: 0 8px; }
.cp-conv-item {
  display: flex; align-items: center; justify-content: space-between; gap: 6px;
  padding: 9px 12px; margin-bottom: 2px; border-radius: 8px;
  cursor: pointer; color: var(--text-secondary); font-size: 13px;
  transition: background 0.12s;
  white-space: nowrap;
}
.cp-conv-item:hover { background: var(--bg-hover); }
.cp-conv-item.active { background: var(--bg-active); color: var(--color-primary); }
.conv-title { overflow: hidden; text-overflow: ellipsis; flex: 1; }
.conv-del {
  opacity: 0; border: none; background: none; cursor: pointer;
  color: var(--text-tertiary); padding: 2px; flex-shrink: 0;
  display: flex; align-items: center;
}
.cp-conv-item:hover .conv-del { opacity: 1; }
.conv-del:hover { color: var(--color-danger); }
.cp-conv-empty { text-align: center; color: var(--text-tertiary); font-size: 12px; padding: 24px 0; }
.cp-sidebar-footer { padding: 8px; border-top: 1px solid var(--border-light); }
.cp-side-btn {
  width: 100%; display: flex; align-items: center; gap: 8px;
  padding: 8px 12px; border: none; background: none; border-radius: 8px;
  color: var(--text-secondary); font-size: 13px; cursor: pointer; white-space: nowrap;
}
.cp-side-btn:hover { background: var(--bg-hover); }

/* 主区域 */
.cp-main { flex: 1; display: flex; flex-direction: column; min-width: 0; background: var(--bg-primary); }

/* 顶栏 */
.cp-topbar {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 20px; border-bottom: 1px solid var(--border-light);
  background: var(--bg-primary);
}
.topbar-spacer { flex: 1; }
.cp-icon-btn {
  width: 34px; height: 34px; border: none; border-radius: 8px; cursor: pointer;
  background: none; color: var(--text-secondary); font-size: 17px;
  display: flex; align-items: center; justify-content: center;
}
.cp-icon-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.cp-model-select { width: 260px; }
.model-option-label { margin-right: 8px; }
.model-option-tag { font-size: 12px; color: var(--text-tertiary); }
.engine-badge {
  font-size: 12px; padding: 2px 10px; border-radius: 999px;
  border: 1px solid var(--border-base);
}
.engine-badge.on { color: var(--color-success); border-color: var(--color-success); }
.engine-badge.off { color: var(--text-tertiary); border-color: var(--border-base); }

.cp-settings-pop .setting-item { margin-bottom: 16px; }
.cp-settings-pop label { display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 8px; }

/* 消息区 */
.cp-messages { flex: 1; overflow-y: auto; scroll-behavior: auto; }
.cp-welcome {
  max-width: 680px; margin: 0 auto; padding: 80px 24px 40px; text-align: center;
}
.welcome-logo {
  width: 64px; height: 64px; margin: 0 auto 20px; border-radius: 20px;
  display: flex; align-items: center; justify-content: center;
  font-size: 30px; color: #fff;
  background: linear-gradient(135deg, #409eff, #7c5cff);
}
.cp-welcome h2 { margin: 0 0 10px; color: var(--text-primary); font-size: 24px; }
.welcome-sub { margin: 0 0 28px; color: var(--text-tertiary); font-size: 13px; }
.welcome-chips { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
.chip {
  padding: 9px 16px; border-radius: 12px; font-size: 13px; cursor: pointer;
  border: 1px solid var(--border-base); background: var(--bg-primary);
  color: var(--text-secondary); transition: all 0.15s;
}
.chip:hover { border-color: var(--color-primary); color: var(--color-primary); transform: translateY(-1px); }

/* 消息行 */
.cp-msg {
  display: flex; gap: 14px; max-width: 820px; margin: 0 auto;
  padding: 14px 24px;
}
.msg-avatar {
  width: 34px; height: 34px; border-radius: 10px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 600;
}
.msg-avatar.user { background: var(--bg-active); color: var(--color-primary); }
.msg-avatar.assistant { background: linear-gradient(135deg, #409eff, #7c5cff); color: #fff; }
.msg-body { flex: 1; min-width: 0; }
.msg-content {
  font-size: 14.5px; line-height: 1.75; color: var(--text-primary);
  word-break: break-word; overflow-wrap: anywhere;
}
.msg-content :deep(p) { margin: 0 0 10px; }
.msg-content :deep(p:last-child) { margin-bottom: 0; }
.msg-content :deep(ul), .msg-content :deep(ol) { padding-left: 22px; margin: 8px 0; }
.msg-content :deep(blockquote) {
  margin: 10px 0; padding: 6px 14px; border-left: 3px solid var(--color-primary);
  background: var(--bg-secondary); border-radius: 0 8px 8px 0; color: var(--text-secondary);
}
.msg-content :deep(table) { border-collapse: collapse; margin: 10px 0; width: 100%; }
.msg-content :deep(th), .msg-content :deep(td) {
  border: 1px solid var(--border-base); padding: 6px 12px; font-size: 13px;
}
.msg-content :deep(a) { color: var(--link-color); }
.msg-content :deep(code:not(.hljs)) {
  background: var(--code-inline-bg); padding: 2px 6px; border-radius: 5px;
  font-size: 13px; font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.msg-content.streaming::after {
  content: '▊'; color: var(--color-primary); animation: blink 1s steps(1) infinite;
}
@keyframes blink { 50% { opacity: 0; } }

/* 代码块 */
.msg-content :deep(.code-block) {
  margin: 12px 0; border-radius: 12px; overflow: hidden;
  border: 1px solid var(--border-base);
}
.msg-content :deep(.code-head) {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 14px; background: #1e1e1e; border-bottom: 1px solid #333;
}
.msg-content :deep(.code-lang) { font-size: 12px; color: #888; }
.msg-content :deep(.code-copy) {
  border: none; background: none; color: #aaa; font-size: 12px; cursor: pointer;
  padding: 2px 6px; border-radius: 4px;
}
.msg-content :deep(.code-copy:hover) { color: #fff; background: #333; }
.msg-content :deep(pre.hljs) { margin: 0; padding: 14px; overflow-x: auto; font-size: 13px; }
.msg-content :deep(pre.hljs code) { background: transparent; padding: 0; }

/* 思考过程 */
.msg-reasoning {
  margin-bottom: 10px; border: 1px solid var(--border-lighter);
  border-radius: 10px; overflow: hidden; background: var(--bg-tertiary);
}
.msg-reasoning summary {
  cursor: pointer; padding: 7px 12px; font-size: 12.5px; color: var(--text-tertiary);
  user-select: none;
}
.reasoning-content {
  padding: 4px 14px 12px; font-size: 13px; line-height: 1.7;
  color: var(--text-secondary); white-space: pre-wrap;
}

/* 操作行 */
.msg-actions { display: flex; gap: 4px; margin-top: 6px; opacity: 0; transition: opacity 0.15s; }
.cp-msg:hover .msg-actions { opacity: 1; }
.action-btn {
  border: none; background: none; cursor: pointer; color: var(--text-tertiary);
  padding: 4px; border-radius: 6px; display: flex; align-items: center;
}
.action-btn:hover { background: var(--bg-hover); color: var(--color-primary); }

/* 输入区 */
.cp-composer { padding: 10px 24px 14px; }
.cp-error {
  max-width: 820px; margin: 0 auto 10px; display: flex; align-items: center; gap: 8px;
  padding: 9px 14px; border-radius: 10px; font-size: 13px;
  background: var(--danger-light); color: var(--danger-dark);
}
.err-close { margin-left: auto; border: none; background: none; cursor: pointer; color: inherit; }
.composer-box {
  max-width: 820px; margin: 0 auto;
  display: flex; align-items: flex-end; gap: 10px;
  border: 1.5px solid var(--border-base); border-radius: 16px;
  padding: 10px 12px; background: var(--bg-primary);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.composer-box.focused { border-color: var(--color-primary); box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.12); }
.composer-input { flex: 1; }
.composer-input :deep(.el-textarea__inner) {
  background: transparent; border: none; box-shadow: none;
  color: var(--text-primary); font-size: 14.5px; padding: 4px 6px;
}
.send-btn {
  width: 36px; height: 36px; flex-shrink: 0; border: none; border-radius: 12px;
  background: linear-gradient(135deg, #409eff, #7c5cff); color: #fff;
  font-size: 16px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: opacity 0.15s, transform 0.15s;
}
.send-btn:hover:not(:disabled) { transform: scale(1.05); }
.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.send-btn.stop { background: var(--color-danger); }
.stop-square { width: 12px; height: 12px; background: #fff; border-radius: 2px; }
.composer-hint {
  max-width: 820px; margin: 8px auto 0; text-align: center;
  font-size: 11.5px; color: var(--text-quaternary);
}

/* 窄屏适配：侧栏改抽屉、模型选择器收缩、输入区安全区、防 iOS 聚焦缩放 */
@media (max-width: 768px) {
  .chatplus-page { height: 100%; } /* 高度由 body.chatplus-active 的 .mobile-main 精确约束 */
  .cp-backdrop {
    display: block; position: fixed; top: 56px; left: 0; right: 0; bottom: 0;
    z-index: 1999; background: rgba(0, 0, 0, 0.45);
  }
  .cp-sidebar {
    position: fixed; top: 56px; bottom: 0; left: 0; z-index: 2000;
    width: 78vw; max-width: 300px;
    transform: translateX(-105%);
    box-shadow: 6px 0 24px var(--shadow-heavy);
    border-right: 1px solid var(--border-light);
  }
  .cp-sidebar.open { width: 78vw; transform: translateX(0); }
  .cp-sidebar:not(.open) { width: 78vw; border-right: 1px solid var(--border-light); }
  .cp-topbar { padding: 8px 10px; gap: 8px; }
  .cp-model-select { width: min(52vw, 220px); }
  .engine-badge { display: none; }
  .cp-icon-btn { width: 38px; height: 38px; font-size: 19px; }
  .cp-msg { padding: 12px 12px; gap: 10px; }
  .msg-avatar { width: 30px; height: 30px; border-radius: 9px; font-size: 12px; }
  .msg-content { font-size: 15px; line-height: 1.7; }
  .cp-welcome { padding: 48px 16px 24px; }
  .cp-welcome h2 { font-size: 20px; }
  .welcome-chips { gap: 8px; }
  .chip { padding: 8px 12px; font-size: 13px; }
  .cp-messages { -webkit-overflow-scrolling: touch; }
  .cp-composer { padding: 8px 10px calc(10px + env(safe-area-inset-bottom)); }
  .composer-box { padding: 8px 8px 8px 10px; border-radius: 14px; }
  .composer-input :deep(.el-textarea__inner) { font-size: 16px; } /* <16px 会触发 iOS 聚焦自动放大 */
  .composer-hint { display: none; }
  .msg-content :deep(pre.hljs) { font-size: 12.5px; }
}

/* 超窄屏 */
@media (max-width: 380px) {
  .cp-model-select { width: 46vw; }
  .cp-msg { padding: 10px 8px; }
}
</style>

<style>
/* 聊天页自管滚动（沿用 ChatRoom 的 body.chat-room-active 模式）：
   锁定外层滚动、隐藏页脚；移动端让 .mobile-main 精确填满固定头部(56px)之下区域，
   避免自然滚动把输入栏推出屏幕。仅在本页激活（含 keep-alive 唤醒）时生效。 */
body.chatplus-active .main-content {
  overflow: hidden;
}
body.chatplus-active .page-footer {
  display: none;
}
/* 密码门阶段隐藏全局宠物（避免压住登录卡片） */
body.chatplus-gate .pet-window,
body.chatplus-gate .pet-launcher {
  display: none !important;
}
@media (max-width: 768px) {
  body.chatplus-active .mobile-main {
    margin-top: 56px;
    height: calc(100dvh - 56px);
    height: calc(100svh - 56px); /* 键盘弹起时用可视区小高度，防输入栏被顶出屏幕 */
    padding: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
}
</style>
