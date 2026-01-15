<template>
  <div class="tool-container">
    <!-- 房间列表界面 -->
    <div v-if="!currentRoom" class="room-list-view">
      <div class="tool-header">
        <h2>聊天室</h2>
        <div class="actions">
          <el-button type="primary" @click="showCreateDialog = true">
            <el-icon><Plus /></el-icon>
            创建房间
          </el-button>
        </div>
      </div>

      <div class="join-section">
        <el-input
          v-model="joinRoomId"
          placeholder="输入房间ID直接加入"
          class="join-input"
          @keyup.enter="joinRoomById"
        >
          <template #append>
            <el-button @click="joinRoomById">加入</el-button>
          </template>
        </el-input>
      </div>

      <div class="room-list">
        <div class="section-title">公开房间</div>
        <div v-if="rooms.length === 0" class="empty-tip">暂无房间，创建一个吧</div>
        <div
          v-for="room in rooms"
          :key="room.id"
          class="room-item"
          @click="handleJoinRoom(room)"
        >
          <div class="room-info">
            <span class="room-name">
              <el-icon v-if="room.has_password"><Lock /></el-icon>
              <el-icon v-else><House /></el-icon>
              {{ room.name }}
            </span>
            <span class="room-id">ID: {{ room.id }}</span>
          </div>
          <el-button type="primary" size="small">进入</el-button>
        </div>
      </div>
    </div>

    <!-- 聊天界面 -->
    <div v-else class="chat-view">
      <div class="chat-header">
        <div class="room-title">
          <el-icon v-if="currentRoom.has_password"><Lock /></el-icon>
          <el-icon v-else><House /></el-icon>
          {{ currentRoom.name }}
          <span class="room-id-tag">{{ currentRoom.id }}</span>
        </div>
        <el-button @click="leaveRoom" type="danger" size="small">退出</el-button>
      </div>

      <div class="messages-container" ref="messagesContainer">
        <div
          v-for="msg in messages"
          :key="msg.id || msg.created_at"
          :class="['message-item', msg.type === 'system' ? 'system-message' : (msg.nickname === nickname ? 'my-message' : '')]"
        >
          <template v-if="msg.type === 'system'">
            <span class="system-text">{{ msg.content }}</span>
          </template>
          <template v-else>
            <div class="message-header">
              <span class="nickname">{{ msg.nickname }}</span>
              <span class="time">{{ formatTime(msg.created_at) }}</span>
            </div>
            <div class="message-content">{{ msg.content }}</div>
          </template>
        </div>
      </div>

      <div class="input-area">
        <el-popover
          placement="top-start"
          :width="320"
          trigger="click"
          v-model:visible="showEmoji"
        >
          <template #reference>
            <el-button class="emoji-btn">
              <el-icon><Sugar /></el-icon>
            </el-button>
          </template>
          <div class="emoji-panel">
            <span
              v-for="emoji in emojis"
              :key="emoji"
              class="emoji-item"
              @click="insertEmoji(emoji)"
            >{{ emoji }}</span>
          </div>
        </el-popover>
        <el-input
          v-model="inputMessage"
          placeholder="输入消息..."
          @keyup.enter="sendMessage"
          class="message-input"
        />
        <el-button type="primary" @click="sendMessage" :disabled="!inputMessage.trim()">
          发送
        </el-button>
      </div>
    </div>

    <!-- 创建房间对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建房间" width="400px">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="房间名称">
          <el-input v-model="createForm.name" placeholder="请输入房间名称" maxlength="50" />
        </el-form-item>
        <el-form-item label="密钥">
          <el-input
            v-model="createForm.password"
            type="password"
            placeholder="可选，留空为公开房间"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createRoom" :loading="creating">创建</el-button>
      </template>
    </el-dialog>

    <!-- 加入房间对话框 -->
    <el-dialog v-model="showJoinDialog" title="加入房间" width="400px">
      <el-form :model="joinForm" label-width="80px">
        <el-form-item label="昵称">
          <el-input v-model="joinForm.nickname" placeholder="请输入昵称" maxlength="20" />
        </el-form-item>
        <el-form-item v-if="joinForm.needPassword" label="密钥">
          <el-input
            v-model="joinForm.password"
            type="password"
            placeholder="请输入房间密钥"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showJoinDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmJoin" :loading="joining">加入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'

const API_BASE = ''

const rooms = ref([])
const currentRoom = ref(null)
const messages = ref([])
const inputMessage = ref('')
const nickname = ref('')
const joinRoomId = ref('')

const showCreateDialog = ref(false)
const showJoinDialog = ref(false)
const showEmoji = ref(false)
const creating = ref(false)
const joining = ref(false)

const createForm = ref({ name: '', password: '' })
const joinForm = ref({ nickname: '', password: '', needPassword: false, roomId: '' })

const messagesContainer = ref(null)
let ws = null

const emojis = [
  '😀', '😁', '😂', '🤣', '😃', '😄', '😅', '😆', '😉', '😊',
  '😋', '😎', '😍', '🥰', '😘', '😗', '😙', '😚', '🙂', '🤗',
  '🤔', '😐', '😑', '😶', '🙄', '😏', '😣', '😥', '😮', '🤐',
  '😯', '😪', '😫', '😴', '😌', '😛', '😜', '😝', '🤤', '😒',
  '😓', '😔', '😕', '🙃', '🤑', '😲', '🙁', '😖', '😞', '😟',
  '😤', '😢', '😭', '😦', '😧', '😨', '😩', '🤯', '😬', '😰',
  '👍', '👎', '👏', '🙌', '🤝', '❤️', '🔥', '⭐', '🎉', '💯'
]

const loadRooms = async () => {
  try {
    const response = await fetch(`${API_BASE}/api/chat/rooms`)
    const data = await response.json()
    rooms.value = data.rooms || []
  } catch (error) {
    console.error('加载房间列表失败:', error)
  }
}

const createRoom = async () => {
  if (!createForm.value.name.trim()) {
    ElMessage.warning('请输入房间名称')
    return
  }

  creating.value = true
  try {
    const response = await fetch(`${API_BASE}/api/chat/room`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(createForm.value)
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }
    ElMessage.success('房间创建成功')
    showCreateDialog.value = false
    createForm.value = { name: '', password: '' }

    // 自动加入新创建的房间
    joinForm.value = {
      nickname: '',
      password: createForm.value.password,
      needPassword: false,
      roomId: data.id
    }
    showJoinDialog.value = true
    loadRooms()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    creating.value = false
  }
}

const handleJoinRoom = (room) => {
  joinForm.value = {
    nickname: nickname.value || '',
    password: '',
    needPassword: room.has_password,
    roomId: room.id
  }
  showJoinDialog.value = true
}

const joinRoomById = async () => {
  if (!joinRoomId.value.trim()) {
    ElMessage.warning('请输入房间ID')
    return
  }

  try {
    const response = await fetch(`${API_BASE}/api/chat/room/${joinRoomId.value}`)
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '房间不存在')
    }
    joinForm.value = {
      nickname: nickname.value || '',
      password: '',
      needPassword: data.has_password,
      roomId: joinRoomId.value
    }
    showJoinDialog.value = true
  } catch (error) {
    ElMessage.error(error.message)
  }
}

const confirmJoin = async () => {
  if (!joinForm.value.nickname.trim()) {
    ElMessage.warning('请输入昵称')
    return
  }

  joining.value = true
  try {
    const response = await fetch(`${API_BASE}/api/chat/room/${joinForm.value.roomId}/join`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        nickname: joinForm.value.nickname,
        password: joinForm.value.password
      })
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '加入失败')
    }

    nickname.value = joinForm.value.nickname
    currentRoom.value = data.room
    messages.value = (data.messages || []).map(m => ({
      ...m,
      type: 'message'
    }))
    showJoinDialog.value = false

    // 连接 WebSocket
    connectWebSocket()

    nextTick(() => {
      scrollToBottom()
    })
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    joining.value = false
  }
}

const connectWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const wsUrl = `${protocol}//${host}/api/chat/room/${currentRoom.value.id}/ws?nickname=${encodeURIComponent(nickname.value)}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket 已连接')
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      messages.value.push(msg)
      nextTick(() => {
        scrollToBottom()
      })
    } catch (error) {
      console.error('消息解析失败:', error)
    }
  }

  ws.onclose = () => {
    console.log('WebSocket 已断开')
    if (currentRoom.value) {
      // 尝试重连
      setTimeout(() => {
        if (currentRoom.value) {
          connectWebSocket()
        }
      }, 3000)
    }
  }

  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error)
  }
}

const sendMessage = () => {
  if (!inputMessage.value.trim() || !ws) return

  ws.send(JSON.stringify({
    type: 'message',
    content: inputMessage.value.trim()
  }))

  inputMessage.value = ''
}

const insertEmoji = (emoji) => {
  inputMessage.value += emoji
  showEmoji.value = false
}

const leaveRoom = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  currentRoom.value = null
  messages.value = []
  loadRooms()
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  loadRooms()
})

onUnmounted(() => {
  if (ws) {
    ws.close()
    ws = null
  }
})
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 40px);
  padding: 20px;
  box-sizing: border-box;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.join-section {
  margin-bottom: 20px;
}

.join-input {
  max-width: 400px;
}

.section-title {
  color: #a0a0a0;
  font-size: 14px;
  margin-bottom: 12px;
}

.room-list {
  flex: 1;
  overflow-y: auto;
}

.empty-tip {
  color: #666;
  text-align: center;
  padding: 40px;
}

.room-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #1e1e1e;
  border-radius: 8px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

.room-item:hover {
  background: #2a2a2a;
}

.room-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.room-name {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #e0e0e0;
  font-size: 16px;
}

.room-id {
  color: #666;
  font-size: 12px;
}

/* 聊天界面 */
.chat-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 16px;
  border-bottom: 1px solid #333;
  margin-bottom: 16px;
}

.room-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #e0e0e0;
  font-size: 18px;
}

.room-id-tag {
  font-size: 12px;
  color: #666;
  background: #2a2a2a;
  padding: 2px 8px;
  border-radius: 4px;
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: #1a1a1a;
  border-radius: 8px;
  margin-bottom: 16px;
}

.message-item {
  margin-bottom: 16px;
}

.system-message {
  text-align: center;
}

.system-text {
  color: #666;
  font-size: 12px;
  background: #2a2a2a;
  padding: 4px 12px;
  border-radius: 12px;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.nickname {
  color: #409eff;
  font-weight: 500;
}

.my-message .nickname {
  color: #67c23a;
}

.time {
  color: #666;
  font-size: 12px;
}

.message-content {
  color: #e0e0e0;
  line-height: 1.5;
  word-break: break-word;
}

.input-area {
  display: flex;
  gap: 8px;
  align-items: center;
}

.emoji-btn {
  padding: 8px 12px;
}

.message-input {
  flex: 1;
}

.emoji-panel {
  display: grid;
  grid-template-columns: repeat(10, 1fr);
  gap: 4px;
}

.emoji-item {
  font-size: 20px;
  padding: 4px;
  cursor: pointer;
  text-align: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.emoji-item:hover {
  background: #f0f0f0;
}

@media (max-width: 768px) {
  .tool-container {
    padding: 12px;
    height: calc(100vh - 56px);
  }

  .emoji-panel {
    grid-template-columns: repeat(8, 1fr);
  }

  .input-area {
    flex-wrap: wrap;
  }

  .message-input {
    flex: 1 1 100%;
    order: 1;
  }

  .emoji-btn {
    order: 2;
  }

  .input-area > .el-button:last-child {
    order: 3;
    flex: 1;
  }
}
</style>
