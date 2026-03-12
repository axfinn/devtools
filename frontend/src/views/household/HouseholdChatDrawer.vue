<template>
  <el-drawer
    v-model="visible"
    title="AI 助手"
    direction="rtl"
    size="400px"
    :close-on-click-modal="false"
    :destroy-on-close="false"
  >
    <div class="chat-container">
      <div class="chat-messages" ref="chatMessagesRef">
        <div v-if="chatHistory.length === 0" class="chat-welcome">
          <el-icon :size="48" color="#67c23a"><ChatDotRound /></el-icon>
          <p>你好！我是家庭物品管理助手</p>
          <p class="hint">可以对我说：</p>
          <ul>
            <li>"帮我买三瓶洗衣液"</li>
            <li>"家里还有什么缺的？"</li>
            <li>"查看库存情况"</li>
            <li>"删除那瓶过期的酱油"</li>
          </ul>
        </div>
        <div
          v-for="(msg, idx) in chatHistory"
          :key="idx"
          class="chat-message"
          :class="msg.role"
        >
          <div class="message-avatar">
            <el-icon v-if="msg.role === 'user'"><User /></el-icon>
            <el-icon v-else><Service /></el-icon>
          </div>
          <div class="message-content">
            <div class="message-text" v-html="formatMessage(msg.content)"></div>
            <div v-if="msg.actions && msg.actions.length > 0" class="message-actions">
              <el-tag v-for="(action, aIdx) in msg.actions" :key="aIdx" size="small" :type="actionTagType(action)">
                {{ formatChatAction(action) }}
              </el-tag>
            </div>
            <template v-for="(action, aIdx) in msg.actions" :key="`loc-${aIdx}`">
            <div
              v-if="action.type === 'suggest_location' && action.candidates && action.candidates.length > 0"
              class="location-candidates"
            >
              <div class="location-candidates-title">为「{{ action.name || '该物品' }}」选择位置：</div>
              <div class="location-candidates-actions">
                <el-button
                  v-for="(loc, lIdx) in action.candidates"
                  :key="lIdx"
                  size="small"
                  @click="applyLocationCandidate(action, loc)"
                >
                  {{ loc }}
                </el-button>
              </div>
            </div>
            </template>
          </div>
        </div>
        <div v-if="chatLoading" class="chat-message assistant">
          <div class="message-avatar"><el-icon><Service /></el-icon></div>
          <div class="message-content">
            <div class="message-text typing">
              <span class="dot">.</span><span class="dot">.</span><span class="dot">.</span>
            </div>
          </div>
        </div>
      </div>

      <div class="chat-quick-actions">
        <el-button size="small" @click="sendQuickPrompt('查看库存')" :disabled="chatLoading">查看库存</el-button>
        <el-button size="small" @click="sendQuickPrompt('有什么要买的？')" :disabled="chatLoading">看看缺什么</el-button>
        <el-button size="small" @click="sendQuickPrompt('删除过期的物品')" :disabled="chatLoading">清理过期</el-button>
      </div>

      <div class="chat-input-area">
        <el-input
          v-model="chatInput"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          placeholder="输入消息或点击麦克风说话..."
          @keydown.enter.exact.prevent="sendMessage"
          :disabled="chatLoading"
        >
          <template #append>
            <el-button @click="toggleVoice" :class="{ 'voice-active': isRecording }">
              <el-icon><Microphone /></el-icon>
            </el-button>
          </template>
        </el-input>
        <el-button type="primary" @click="sendMessage" :loading="chatLoading" :disabled="!chatInput.trim()">发送</el-button>
      </div>

      <div class="chat-actions">
        <el-button text size="small" @click="clearHistory">
          <el-icon><Delete /></el-icon>
          清除历史
        </el-button>
      </div>
    </div>
  </el-drawer>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ChatDotRound, User, Service, Microphone, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: Boolean,
  profileId: String,
  creatorKey: String
})
const emit = defineEmits(['update:modelValue', 'refresh', 'refresh-todos'])

const visible = ref(props.modelValue)
const chatInput = ref('')
const chatHistory = ref([])
const chatLoading = ref(false)
const chatMessagesRef = ref(null)
const isRecording = ref(false)
let recognition = null

watch(() => props.modelValue, v => {
  visible.value = v
  if (v && props.profileId) loadHistory()
})
watch(visible, v => { emit('update:modelValue', v) })

async function loadHistory() {
  try {
    const res = await fetch(`/api/household/chat/history?profile_id=${props.profileId}`)
    const data = await res.json()
    if (data.code === 0 && data.history) {
      chatHistory.value = data.history.map(h => ({
        role: h.role,
        content: typeof h.content === 'string' ? h.content : '',
        actions: Array.isArray(h.actions) ? h.actions : []
      }))
    }
  } catch {}
}

async function clearHistory() {
  try {
    await fetch(`/api/household/chat/history?profile_id=${props.profileId}`, { method: 'DELETE' })
    chatHistory.value = []
    ElMessage.success('对话历史已清除')
  } catch { ElMessage.error('清除失败') }
}

async function sendMessage() {
  if (!chatInput.value.trim() || chatLoading.value) return
  const message = chatInput.value.trim()
  chatInput.value = ''
  chatHistory.value.push({ role: 'user', content: message })
  scrollToBottom()
  chatLoading.value = true
  try {
    const res = await fetch('/api/household/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, profile_id: props.profileId, creator_key: props.creatorKey })
    })
    const data = await res.json()
    if (data.code === 0) {
      let replyContent = data.reply
      try {
        let jsonStr = replyContent
        if (typeof jsonStr === 'string') {
          jsonStr = jsonStr.replace(/```json\s*/g, '').replace(/```\s*/g, '').trim()
        }
        if (jsonStr.startsWith('{')) {
          const parsed = JSON.parse(jsonStr)
          replyContent = parsed.reply || parsed.content || replyContent
        }
      } catch {}
      chatHistory.value.push({ role: 'assistant', content: replyContent, actions: data.actions || [] })
      if ((data.actions && data.actions.length > 0) || (data.items_added && data.items_added.length > 0)) {
        emit('refresh')
        if (data.actions && data.actions.some(a => a.type === 'todo')) emit('refresh-todos')
      }
    } else {
      chatHistory.value.push({ role: 'assistant', content: '抱歉，出了点问题：' + (data.error || '未知错误') })
    }
  } catch {
    chatHistory.value.push({ role: 'assistant', content: '网络错误，请稍后重试' })
  } finally {
    chatLoading.value = false
    scrollToBottom()
  }
}

function sendQuickPrompt(text) {
  if (!text || chatLoading.value) return
  chatInput.value = text
  sendMessage()
}

function scrollToBottom() {
  setTimeout(() => {
    if (chatMessagesRef.value) chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight
  }, 100)
}

function formatMessage(text) {
  if (typeof text !== 'string') return ''
  return text.replace(/\n/g, '<br>')
}

function formatChatAction(action) {
  if (!action || !action.type) return '操作'
  const name = action.name || action.target || action.item_id || '物品'
  const quantity = action.quantity ? ` x${action.quantity}` : ''
  const map = { add: '添加', restock: '补充', use: '使用', update: '更新', todo: '待购', suggest_location: '位置候选', delete: '删除', query: '查询' }
  return `${map[action.type] || '操作'} ${name}${quantity}`
}

function actionTagType(action) {
  if (!action || !action.type) return 'info'
  const map = { add: 'success', restock: 'warning', use: 'info', update: 'primary', todo: 'success', suggest_location: 'warning', delete: 'danger' }
  return map[action.type] || 'info'
}

async function applyLocationCandidate(action, location) {
  if (!action || !location || !props.profileId) return
  const name = action.name || action.target
  if (!name) { ElMessage.warning('未找到物品名称'); return }
  emit('apply-location', { name, location })
}

function toggleVoice() {
  if (isRecording.value) stopRecording()
  else startRecording()
}

function startRecording() {
  const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!SpeechRecognition) { ElMessage.warning('当前浏览器不支持语音输入'); return }
  recognition = new SpeechRecognition()
  recognition.lang = 'zh-CN'
  recognition.continuous = false
  recognition.interimResults = true
  recognition.onresult = (event) => {
    chatInput.value = Array.from(event.results).map(r => r[0]).map(r => r.transcript).join('')
  }
  recognition.onerror = (event) => {
    isRecording.value = false
    if (event.error !== 'no-speech') ElMessage.warning('语音识别出错')
  }
  recognition.onend = () => { isRecording.value = false }
  recognition.start()
  isRecording.value = true
}

function stopRecording() {
  if (recognition) { recognition.stop(); isRecording.value = false }
}
</script>

<style scoped>
.chat-container { display: flex; flex-direction: column; height: 100%; }
.chat-messages { flex: 1; overflow-y: auto; padding: 16px; background: #f5f7fa; }
.chat-welcome { text-align: center; padding: 40px 20px; color: #606266; }
.chat-welcome p { margin: 10px 0; }
.chat-welcome .hint { color: #909399; font-size: 12px; margin-top: 20px; }
.chat-welcome ul { text-align: left; padding-left: 20px; font-size: 13px; color: #606266; }
.chat-welcome li { margin: 8px 0; }
.chat-message { display: flex; margin-bottom: 16px; }
.chat-message.user { flex-direction: row-reverse; }
.message-avatar { width: 36px; height: 36px; border-radius: 50%; background: #409eff; color: white; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.chat-message.assistant .message-avatar { background: #67c23a; }
.message-content { max-width: 75%; margin: 0 10px; }
.message-actions { margin-top: 6px; display: flex; flex-wrap: wrap; gap: 6px; }
.location-candidates { margin-top: 8px; padding: 8px; background: #f5f7fa; border-radius: 6px; }
.location-candidates-title { font-size: 12px; color: #606266; margin-bottom: 6px; }
.location-candidates-actions { display: flex; flex-wrap: wrap; gap: 6px; }
.message-text { padding: 10px 14px; border-radius: 8px; background: white; font-size: 14px; line-height: 1.5; word-break: break-word; }
.chat-message.user .message-text { background: #409eff; color: white; }
.message-text.typing { color: #909399; }
.message-text .dot { animation: dot 1.4s infinite; }
.message-text .dot:nth-child(2) { animation-delay: 0.2s; }
.message-text .dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes dot { 0%, 20% { content: '.'; } 40% { content: '..'; } 60%, 100% { content: '...'; } }
.chat-input-area { display: flex; gap: 10px; padding: 12px; background: white; border-top: 1px solid #ebeef5; }
.chat-quick-actions { display: flex; gap: 8px; flex-wrap: wrap; padding: 10px 12px 0; background: #f5f7fa; border-top: 1px solid #ebeef5; }
.chat-input-area .el-input { flex: 1; }
.voice-active { color: #f56c6c !important; background: #fef0f0; }
.chat-actions { padding: 8px 12px; background: #f5f7fa; border-top: 1px solid #ebeef5; text-align: center; }
</style>
