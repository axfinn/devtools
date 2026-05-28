<template>
  <div class="chat-share-page" :class="{ dark: isDark }">
    <div v-if="loading" class="chat-share-loading">
      <div class="loading-spinner"></div>
      <span>加载对话中...</span>
    </div>

    <div v-else-if="error" class="chat-share-error">
      <div class="error-icon">😵</div>
      <p>{{ error }}</p>
      <a href="/">返回首页</a>
    </div>

    <template v-else>
      <div class="chat-share-header">
        <div class="header-brand">
          <div class="brand-icon">✨</div>
          <span class="brand-name">AskIt</span>
        </div>
        <h1 class="header-title">{{ chatData.title }}</h1>
      </div>

      <div class="chat-share-messages">
        <div
          v-for="(msg, idx) in chatData.messages"
          :key="idx"
          class="chat-bubble-row"
          :class="msg.role"
        >
          <div class="bubble-avatar">{{ msg.role === 'user' ? '👤' : '✨' }}</div>
          <div class="bubble-content" :class="msg.role">
            <div class="bubble-body" v-html="renderContent(msg.content, msg.role)"></div>
            <div v-if="msg.timestamp" class="bubble-time">{{ formatTime(msg.timestamp) }}</div>
          </div>
        </div>
      </div>

      <div class="chat-share-footer">
        <span>共 {{ chatData.messages.length }} 条消息</span>
        <span class="footer-dot">·</span>
        <span>{{ formatDate(chatData.exportedAt) }}</span>
        <span class="footer-dot">·</span>
        <span>Powered by AskIt</span>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import MarkdownIt from 'markdown-it'
import hljs from '../../utils/highlight'
import 'highlight.js/styles/github-dark.css'
import { API_BASE } from '../../api'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return '<pre class="hljs"><code>' + hljs.highlight(str, { language: lang, ignoreIllegals: true }).value + '</code></pre>'
      } catch {}
    }
    return '<pre class="hljs"><code>' + md.utils.escapeHtml(str) + '</code></pre>'
  }
})

const route = useRoute()
const loading = ref(true)
const error = ref('')
const chatData = ref({ messages: [], title: '', exportedAt: '' })
const isDark = computed(() => document.documentElement.classList.contains('dark'))

function renderContent(content, role) {
  if (role === 'user') {
    return escapeHtml(content).replace(/\n/g, '<br>')
  }
  const mediaPlaceholders = []
  let processedContent = content.replace(/<audio\s+controls\s+src="([^"]+)"><\/audio>/gi, (_, url) => {
    const idx = mediaPlaceholders.length
    mediaPlaceholders.push(`<div class="audio-player"><div class="audio-icon">🎵</div><audio controls src="${url}"></audio></div>`)
    return `ASKIT_MEDIA_${idx}`
  })
  processedContent = processedContent.replace(/<video\s+controls\s+src="([^"]+)"><\/video>/gi, (_, url) => {
    const idx = mediaPlaceholders.length
    mediaPlaceholders.push(`<video controls src="${url}" class="chat-video"></video>`)
    return `ASKIT_MEDIA_${idx}`
  })

  let rendered = md.render(processedContent)
  rendered = rendered.replace(/ASKIT_MEDIA_(\d+)/g, (_, idx) => mediaPlaceholders[parseInt(idx)])
  return rendered
}

function escapeHtml(text) {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

function formatTime(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN')
}

function unescapeHtmlEntities(str) {
  const textarea = document.createElement('textarea')
  textarea.innerHTML = str
  return textarea.value
}

function filterSystemMessages(messages) {
  return messages.filter(m => {
    if (!m.content) return false
    if (m.content.startsWith('📤 对话已分享')) return false
    if (m.content.startsWith('📤 正在生成分享链接')) return false
    if (m.content.includes('正在搜索...') && m.content.trim() === '🔍 正在搜索...') return false
    return true
  })
}

onMounted(async () => {
  const id = route.params.id
  if (!id) { error.value = '无效的分享链接'; loading.value = false; return }

  try {
    const resp = await fetch(`${API_BASE}/api/paste/${id}`)
    if (!resp.ok) { error.value = resp.status === 404 ? '对话不存在或已过期' : '加载失败'; loading.value = false; return }
    const data = await resp.json()

    if (data.language === 'askit-chat') {
      const content = unescapeHtmlEntities(data.content)
      const parsed = JSON.parse(content)
      parsed.messages = filterSystemMessages(parsed.messages)
      chatData.value = parsed
    } else if (data.content_type === 'markdown' || data.language === 'markdown') {
      chatData.value = parseMarkdownChat(data.content, data.title)
    } else {
      error.value = '不支持的分享格式'
    }
  } catch (e) {
    error.value = '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
})

function parseMarkdownChat(content, title) {
  const messages = []
  const blocks = content.split(/---\n+/)
  for (const block of blocks) {
    const userMatch = block.match(/\*\*👤 用户：\*\*\s*\n+>\s*(.+?)(?:\n\n|\n*$)/s)
    if (userMatch) {
      messages.push({ role: 'user', content: userMatch[1].replace(/\n>\s*/g, '\n').trim() })
    }
    const aiParts = block.split(/\*\*🤖 AI：\*\*\s*\n+/)
    for (let i = 1; i < aiParts.length; i++) {
      const aiContent = aiParts[i].replace(/\n*$/, '').trim()
      if (aiContent) messages.push({ role: 'assistant', content: aiContent })
    }
  }
  return { messages, title: title || 'AskIt 对话', exportedAt: new Date().toISOString() }
}
</script>

<style scoped>
.chat-share-page {
  min-height: 100vh;
  background: linear-gradient(180deg, #f0f4ff 0%, #fafbff 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', sans-serif;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 16px;
}
.chat-share-page.dark {
  background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%);
}

.chat-share-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  gap: 16px;
  color: #6b7280;
}
.loading-spinner {
  width: 32px; height: 32px;
  border: 3px solid #e5e7eb;
  border-top-color: #7c3aed;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.chat-share-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  gap: 12px;
  color: #6b7280;
}
.error-icon { font-size: 48px; }
.chat-share-error a { color: #7c3aed; text-decoration: none; }

.chat-share-header {
  width: 100%;
  max-width: 720px;
  padding: 24px 0 16px;
  display: flex;
  align-items: center;
  gap: 16px;
}
.header-brand {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px 4px 8px;
  background: linear-gradient(135deg, #7c3aed, #6366f1);
  border-radius: 20px;
  flex-shrink: 0;
}
.brand-icon { font-size: 14px; }
.brand-name { font-size: 12px; font-weight: 600; color: #fff; }
.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dark .header-title { color: #f3f4f6; }

.chat-share-messages {
  width: 100%;
  max-width: 720px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0 24px;
}

.chat-bubble-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.chat-bubble-row.user {
  flex-direction: row-reverse;
}

.bubble-avatar {
  width: 36px; height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}
.dark .bubble-avatar { background: #2d2d3d; }

.bubble-content {
  max-width: 80%;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.bubble-body {
  padding: 12px 16px;
  border-radius: 18px;
  font-size: 14px;
  line-height: 1.7;
  word-break: break-word;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.bubble-content.user .bubble-body {
  background: linear-gradient(135deg, #7c3aed, #6366f1);
  color: #fff;
  border-bottom-right-radius: 6px;
}
.bubble-content.assistant .bubble-body {
  background: #fff;
  color: #1f2937;
  border-bottom-left-radius: 6px;
}
.dark .bubble-content.assistant .bubble-body {
  background: #2d2d3d;
  color: #e5e7eb;
}

.bubble-time {
  font-size: 11px;
  color: #9ca3af;
  padding: 0 4px;
}
.chat-bubble-row.user .bubble-time { text-align: right; }

/* Markdown content inside bubbles */
.bubble-body :deep(p) { margin: 0 0 8px; }
.bubble-body :deep(p:last-child) { margin: 0; }
.bubble-body :deep(h1),
.bubble-body :deep(h2),
.bubble-body :deep(h3) {
  margin: 12px 0 8px;
  font-size: 1.1em;
  font-weight: 600;
}
.bubble-body :deep(ul), .bubble-body :deep(ol) {
  padding-left: 1.5em;
  margin: 4px 0;
}
.bubble-body :deep(pre) {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 10px;
  overflow-x: auto;
  font-size: 13px;
  margin: 8px 0;
}
.bubble-body :deep(code) {
  font-family: 'Fira Code', 'JetBrains Mono', monospace;
  font-size: 0.9em;
}
.bubble-body :deep(pre code) { background: none; padding: 0; }
.bubble-body :deep(code:not(pre code)) {
  background: rgba(0,0,0,0.06);
  padding: 2px 6px;
  border-radius: 4px;
}
.dark .bubble-body :deep(code:not(pre code)) { background: rgba(255,255,255,0.1); }
.bubble-body :deep(a) { color: #7c3aed; text-decoration: none; }
.bubble-body :deep(blockquote) {
  border-left: 3px solid #7c3aed;
  padding-left: 12px;
  margin: 8px 0;
  color: #6b7280;
}
.bubble-body :deep(img) {
  max-width: 100%;
  border-radius: 12px;
  margin: 8px 0;
}

/* Audio player */
.bubble-body :deep(.audio-player) {
  display: flex;
  align-items: center;
  gap: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 12px 16px;
  border-radius: 12px;
  margin: 8px 0;
}
.bubble-body :deep(.audio-icon) { font-size: 24px; }
.bubble-body :deep(.audio-player audio) {
  flex: 1;
  height: 36px;
  border-radius: 18px;
}
.bubble-body :deep(audio) {
  width: 100%;
  margin: 8px 0;
  border-radius: 8px;
}
.bubble-body :deep(.chat-video) {
  width: 100%;
  max-height: 300px;
  border-radius: 12px;
  margin: 8px 0;
}

.chat-share-footer {
  width: 100%;
  max-width: 720px;
  padding: 16px 0 32px;
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.footer-dot { opacity: 0.5; }

@media (max-width: 640px) {
  .chat-share-page { padding: 0 12px; }
  .bubble-content { max-width: 85%; }
  .bubble-body { padding: 10px 14px; font-size: 13px; }
  .header-title { font-size: 14px; }
}
</style>
