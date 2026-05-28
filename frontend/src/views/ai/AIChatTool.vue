<template>
  <div class="tool-container ai-chat-page">
    <div class="tool-header">
      <div class="header-row">
        <div>
          <h2>AI Chat</h2>
          <p>多模型智能对话，支持流式输出。安装 <a href="#" @click.prevent="downloadExtension">AskIt 浏览器扩展</a> 获得更强大的页面分析能力。</p>
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
        </div>

        <!-- Messages -->
        <div class="messages-container" ref="messagesRef">
          <div v-if="messages.length === 0" class="empty-state">
            <div class="empty-icon">💬</div>
            <p>开始一段新对话</p>
          </div>
          <div v-for="msg in messages" :key="msg.id" class="message" :class="msg.role">
            <div class="message-avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
            <div class="message-content" v-html="renderMarkdown(msg.content)"></div>
          </div>
          <div v-if="streaming" class="message assistant">
            <div class="message-avatar">🤖</div>
            <div class="message-content">
              <span v-html="renderMarkdown(streamContent)"></span>
              <span class="cursor-blink">▊</span>
            </div>
          </div>
        </div>

        <!-- Input -->
        <div class="chat-input-area">
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="2"
            :autosize="{ minRows: 2, maxRows: 6 }"
            placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
            @keydown="handleKeydown"
            :disabled="streaming"
          />
          <div class="input-actions">
            <el-button v-if="!streaming" type="primary" :disabled="!inputText.trim()" @click="sendMessage">
              发送
            </el-button>
            <el-button v-else type="danger" @click="stopStreaming">
              停止
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Download, Plus, Delete, Setting } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'
import { API_BASE } from '@/api.js'

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

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
let abortController = null

onMounted(() => {
  loadConversations()
  if (conversations.value.length === 0) {
    newConversation()
  } else {
    currentConvId.value = conversations.value[0].id
    loadMessages()
  }
})

function generateId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

function loadConversations() {
  const raw = localStorage.getItem('ai-chat-conversations')
  conversations.value = raw ? JSON.parse(raw) : []
}

function saveConversations() {
  localStorage.setItem('ai-chat-conversations', JSON.stringify(conversations.value))
}

function loadMessages() {
  const raw = localStorage.getItem(`ai-chat-messages-${currentConvId.value}`)
  messages.value = raw ? JSON.parse(raw) : []
}

function saveMessages() {
  localStorage.setItem(`ai-chat-messages-${currentConvId.value}`, JSON.stringify(messages.value))
}

function newConversation() {
  const id = generateId()
  conversations.value.unshift({ id, title: '新对话', createdAt: Date.now() })
  currentConvId.value = id
  messages.value = []
  saveConversations()
}

function switchConversation(id) {
  currentConvId.value = id
  loadMessages()
}

function deleteConversation(id) {
  conversations.value = conversations.value.filter(c => c.id !== id)
  localStorage.removeItem(`ai-chat-messages-${id}`)
  saveConversations()
  if (currentConvId.value === id) {
    if (conversations.value.length > 0) {
      currentConvId.value = conversations.value[0].id
      loadMessages()
    } else {
      newConversation()
    }
  }
}

function renderMarkdown(text) {
  if (!text) return ''
  return md.render(text)
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  })
}

function handleKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || streaming.value) return

  messages.value.push({ id: generateId(), role: 'user', content: text })
  inputText.value = ''
  scrollToBottom()

  // Update conversation title from first message
  const conv = conversations.value.find(c => c.id === currentConvId.value)
  if (conv && conv.title === '新对话') {
    conv.title = text.slice(0, 30)
    saveConversations()
  }

  streaming.value = true
  streamContent.value = ''
  abortController = new AbortController()

  const apiMessages = []
  if (systemPrompt.value.trim()) {
    apiMessages.push({ role: 'system', content: systemPrompt.value.trim() })
  }
  for (const msg of messages.value) {
    apiMessages.push({ role: msg.role, content: msg.content })
  }

  try {
    const response = await fetch(`${API_BASE}/api/internal/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: selectedModel.value,
        messages: apiMessages,
        temperature: temperature.value,
        stream: true,
      }),
      signal: abortController.signal,
    })

    if (!response.ok) {
      const err = await response.json().catch(() => ({}))
      throw new Error(err.error || `HTTP ${response.status}`)
    }

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
          if (delta) {
            streamContent.value += delta
            scrollToBottom()
          }
        } catch {}
      }
    }

    if (streamContent.value) {
      messages.value.push({ id: generateId(), role: 'assistant', content: streamContent.value })
    }
  } catch (err) {
    if (err.name !== 'AbortError') {
      ElMessage.error(`请求失败: ${err.message}`)
      if (streamContent.value) {
        messages.value.push({ id: generateId(), role: 'assistant', content: streamContent.value })
      }
    }
  } finally {
    streaming.value = false
    streamContent.value = ''
    saveMessages()
    scrollToBottom()
  }
}

function stopStreaming() {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
}

function downloadExtension() {
  window.open(`${API_BASE}/api/askit/extension`, '_blank')
}

watch(messages, () => {
  if (currentConvId.value) saveMessages()
}, { deep: true })
</script>

<style scoped>
.ai-chat-page {
  height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
}
.tool-header {
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-light, #eee);
}
.tool-header h2 { margin: 0 0 4px; font-size: 18px; }
.tool-header p { margin: 0; font-size: 13px; color: #666; }
.tool-header a { color: #409eff; text-decoration: none; }
.header-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; }

.chat-layout {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.chat-sidebar {
  width: 220px;
  border-right: 1px solid var(--border-light, #eee);
  display: flex;
  flex-direction: column;
  padding: 12px;
  background: var(--bg-sidebar, #fafafa);
}
.new-chat-btn { width: 100%; margin-bottom: 12px; }
.conversation-list { flex: 1; overflow-y: auto; }
.conversation-item {
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
  font-size: 13px;
  color: #333;
}
.conversation-item:hover { background: #f0f0f0; }
.conversation-item.active { background: #e8f0fe; color: #1a73e8; }
.conv-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.conv-delete { opacity: 0; }
.conversation-item:hover .conv-delete { opacity: 1; }

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.chat-settings {
  padding: 8px 16px;
  border-bottom: 1px solid var(--border-light, #eee);
  display: flex;
  align-items: center;
  gap: 8px;
}
.model-select { width: 200px; }
.settings-popover .setting-item { margin-bottom: 12px; }
.settings-popover label { display: block; font-size: 12px; color: #666; margin-bottom: 4px; }

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
}
.empty-state { text-align: center; padding: 60px 0; color: #999; }
.empty-icon { font-size: 48px; margin-bottom: 12px; }

.message {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  max-width: 85%;
}
.message.user { margin-left: auto; flex-direction: row-reverse; }
.message-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
  background: #f5f5f5;
}
.message-content {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
}
.message.user .message-content { background: #e8f0fe; color: #1a1a1a; }
.message.assistant .message-content { background: #f5f5f5; color: #1a1a1a; }
.message-content :deep(pre) {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  font-size: 13px;
}
.message-content :deep(code) { font-family: 'Fira Code', monospace; }
.message-content :deep(p) { margin: 0 0 8px; }
.message-content :deep(p:last-child) { margin: 0; }

.cursor-blink { animation: blink 1s infinite; }
@keyframes blink { 0%, 50% { opacity: 1; } 51%, 100% { opacity: 0; } }

.chat-input-area {
  padding: 12px 24px 16px;
  border-top: 1px solid var(--border-light, #eee);
  display: flex;
  gap: 12px;
  align-items: flex-end;
}
.chat-input-area .el-textarea { flex: 1; }
.input-actions { flex-shrink: 0; }

@media (max-width: 768px) {
  .chat-sidebar { display: none; }
  .chat-layout { flex-direction: column; }
}
</style>

