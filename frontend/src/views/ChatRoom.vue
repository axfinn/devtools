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
          <!-- 连接状态指示器 -->
          <span :class="['connection-status', connectionStatus]">
            <span class="status-dot"></span>
            {{ connectionStatusText }}
          </span>
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
            <div class="message-content">
              <template v-if="msg.msg_type === 'image'">
                <img
                  :src="msg.content"
                  class="message-image"
                  @click="previewImage(msg.content)"
                  loading="lazy"
                />
              </template>
              <template v-else-if="msg.msg_type === 'video'">
                <video
                  :src="msg.content"
                  class="message-video"
                  controls
                  preload="metadata"
                />
              </template>
              <template v-else-if="msg.msg_type === 'audio'">
                <audio
                  :src="msg.content"
                  class="message-audio"
                  controls
                  preload="metadata"
                />
              </template>
              <template v-else-if="msg.msg_type === 'file'">
                <a :href="msg.content" class="message-file" target="_blank" download>
                  <el-icon><Document /></el-icon>
                  <span>{{ msg.original_name || '下载文件' }}</span>
                </a>
              </template>
              <template v-else>{{ msg.content }}</template>
            </div>
          </template>
        </div>
      </div>

      <div class="input-area">
        <div class="input-tools">
          <el-popover
            placement="top-start"
            :width="320"
            trigger="click"
            v-model:visible="showEmoji"
          >
            <template #reference>
              <el-button class="tool-btn" title="表情">
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
          <el-button class="tool-btn" @click="triggerFileUpload" :loading="uploading" title="文件">
            <el-icon><FolderOpened /></el-icon>
          </el-button>
          <el-button class="tool-btn" @click="triggerImageUpload" :loading="uploading" title="图片">
            <el-icon><Picture /></el-icon>
          </el-button>
          <el-button class="tool-btn" @click="openCamera" :loading="uploading" title="拍照">
            <el-icon><Camera /></el-icon>
          </el-button>
          <el-button class="tool-btn" @click="startVideoRecording" :loading="isRecordingVideo" title="录视频">
            <el-icon><VideoCamera /></el-icon>
          </el-button>
          <el-button
            class="tool-btn"
            :class="{ recording: isRecordingAudio }"
            @click="toggleAudioRecording"
            :loading="uploading"
            title="语音"
          >
            <el-icon><Microphone /></el-icon>
          </el-button>
        </div>
        <input
          type="file"
          ref="fileInput"
          accept="*/*"
          style="display: none"
          @change="handleFileSelect"
        />
        <input
          type="file"
          ref="imageInput"
          accept="image/*"
          style="display: none"
          @change="handleImageSelect"
        />
        <el-input
          v-model="inputMessage"
          placeholder="输入消息，可粘贴图片或视频..."
          @keyup.enter="sendMessage"
          @paste="handlePaste"
          class="message-input"
        />
        <el-button type="primary" @click="sendMessage" :disabled="!inputMessage.trim()">
          发送
        </el-button>
      </div>

      <!-- 录音状态显示 -->
      <div v-if="isRecordingAudio" class="recording-indicator">
        <span class="recording-dot"></span>
        录音中 {{ recordingDuration }}s
        <el-button size="small" @click="stopAudioRecording">停止并发送</el-button>
        <el-button size="small" @click="cancelAudioRecording">取消</el-button>
      </div>

      <!-- 拍照/录像弹窗 -->
      <el-dialog v-model="showCameraDialog" title="拍照/录像" width="500px" @close="closeCamera">
        <div class="camera-container">
          <video ref="cameraVideo" autoplay playsinline class="camera-preview"></video>
          <canvas ref="cameraCanvas" style="display: none"></canvas>
        </div>
        <template #footer>
          <el-button @click="closeCamera">取消</el-button>
          <el-button v-if="!isRecordingCamera" type="primary" @click="takePhoto">拍照</el-button>
          <el-button v-if="!isRecordingCamera" type="success" @click="startCameraRecording">开始录像</el-button>
          <el-button v-else type="danger" @click="stopCameraRecording">停止录像</el-button>
        </template>
      </el-dialog>

      <!-- 图片预览 -->
      <el-image-viewer
        v-if="previewVisible"
        :url-list="[previewUrl]"
        @close="previewVisible = false"
      />
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
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElImageViewer } from 'element-plus'

const API_BASE = ''
const MAX_RECONNECT_ATTEMPTS = 5

const rooms = ref([])
const currentRoom = ref(null)
const messages = ref([])
const inputMessage = ref('')
const nickname = ref('')
const joinRoomId = ref('')

const showCreateDialog = ref(false)
const showJoinDialog = ref(false)
const showEmoji = ref(false)
const showCameraDialog = ref(false)
const creating = ref(false)
const joining = ref(false)
const uploading = ref(false)
const previewVisible = ref(false)
const previewUrl = ref('')

// 连接状态
const connectionStatus = ref('disconnected') // connected, connecting, disconnected
const reconnectAttempts = ref(0)

// 录音相关
const isRecordingAudio = ref(false)
const recordingDuration = ref(0)
let audioRecorder = null
let audioChunks = []
let recordingTimer = null

// 录视频相关
const isRecordingVideo = ref(false)
const isRecordingCamera = ref(false)
let cameraStream = null
let cameraRecorder = null
let cameraChunks = []

const createForm = ref({ name: '', password: '' })
const joinForm = ref({ nickname: '', password: '', needPassword: false, roomId: '' })

const messagesContainer = ref(null)
const fileInput = ref(null)
const imageInput = ref(null)
const cameraVideo = ref(null)
const cameraCanvas = ref(null)
let ws = null

const connectionStatusText = computed(() => {
  switch (connectionStatus.value) {
    case 'connected': return '已连接'
    case 'connecting': return '连接中...'
    case 'disconnected': return '未连接'
    default: return ''
  }
})

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
    reconnectAttempts.value = 0

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
  if (reconnectAttempts.value >= MAX_RECONNECT_ATTEMPTS) {
    connectionStatus.value = 'disconnected'
    ElMessage.error('连接失败次数过多，请刷新页面重试')
    return
  }

  connectionStatus.value = 'connecting'
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const wsUrl = `${protocol}//${host}/api/chat/room/${currentRoom.value.id}/ws?nickname=${encodeURIComponent(nickname.value)}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    connectionStatus.value = 'connected'
    reconnectAttempts.value = 0
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
    connectionStatus.value = 'disconnected'
    if (currentRoom.value) {
      reconnectAttempts.value++
      if (reconnectAttempts.value < MAX_RECONNECT_ATTEMPTS) {
        const delay = Math.min(3000 * reconnectAttempts.value, 15000)
        setTimeout(() => {
          if (currentRoom.value) {
            connectWebSocket()
          }
        }, delay)
      } else {
        ElMessage.warning('连接已断开，请刷新页面重试')
      }
    }
  }

  ws.onerror = () => {
    connectionStatus.value = 'disconnected'
  }
}

const sendMessage = () => {
  if (!inputMessage.value.trim() || !ws || ws.readyState !== WebSocket.OPEN) return

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

// 触发文件选择
const triggerFileUpload = () => {
  fileInput.value?.click()
}

// 处理文件选择
const handleFileSelect = (event) => {
  const file = event.target.files?.[0]
  if (file) {
    uploadFile(file)
  }
  event.target.value = ''
}

// 触发图片选择
const triggerImageUpload = () => {
  imageInput.value?.click()
}

// 处理图片选择
const handleImageSelect = (event) => {
  const file = event.target.files?.[0]
  if (file) {
    uploadFile(file)
  }
  event.target.value = ''
}

// 处理粘贴事件
const handlePaste = (event) => {
  const items = event.clipboardData?.items
  if (!items) return

  for (const item of items) {
    if (item.type.startsWith('image/') || item.type.startsWith('video/')) {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        uploadFile(file)
      }
      return
    }
  }
}

// 上传文件（通用）
const uploadFile = async (file) => {
  const maxSizes = {
    image: { size: 5 * 1024 * 1024, label: '5MB' },
    video: { size: 50 * 1024 * 1024, label: '50MB' },
    audio: { size: 10 * 1024 * 1024, label: '10MB' },
    file: { size: 20 * 1024 * 1024, label: '20MB' }
  }

  let fileType = 'file'
  if (file.type.startsWith('image/')) fileType = 'image'
  else if (file.type.startsWith('video/')) fileType = 'video'
  else if (file.type.startsWith('audio/')) fileType = 'audio'

  const limit = maxSizes[fileType]
  if (file.size > limit.size) {
    ElMessage.error(`文件大小不能超过 ${limit.label}`)
    return
  }

  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)

    const response = await fetch(`${API_BASE}/api/chat/upload`, {
      method: 'POST',
      body: formData
    })

    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '上传失败')
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'message',
        content: data.url,
        msg_type: data.type,
        original_name: data.original_name
      }))
      ElMessage.success('发送成功')
    }
  } catch (error) {
    ElMessage.error(error.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

// ========== 拍照功能 ==========
const openCamera = async () => {
  try {
    cameraStream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
      audio: true
    })
    showCameraDialog.value = true
    nextTick(() => {
      if (cameraVideo.value) {
        cameraVideo.value.srcObject = cameraStream
      }
    })
  } catch (error) {
    ElMessage.error('无法访问摄像头: ' + error.message)
  }
}

const closeCamera = () => {
  if (cameraStream) {
    cameraStream.getTracks().forEach(track => track.stop())
    cameraStream = null
  }
  if (cameraRecorder) {
    cameraRecorder = null
  }
  isRecordingCamera.value = false
  cameraChunks = []
  showCameraDialog.value = false
}

const takePhoto = () => {
  if (!cameraVideo.value || !cameraCanvas.value) return

  const video = cameraVideo.value
  const canvas = cameraCanvas.value
  canvas.width = video.videoWidth
  canvas.height = video.videoHeight

  const ctx = canvas.getContext('2d')
  ctx.drawImage(video, 0, 0)

  canvas.toBlob(async (blob) => {
    if (blob) {
      const file = new File([blob], `photo_${Date.now()}.jpg`, { type: 'image/jpeg' })
      closeCamera()
      await uploadFile(file)
    }
  }, 'image/jpeg', 0.9)
}

const startCameraRecording = () => {
  if (!cameraStream) return

  cameraChunks = []
  cameraRecorder = new MediaRecorder(cameraStream, {
    mimeType: MediaRecorder.isTypeSupported('video/webm') ? 'video/webm' : 'video/mp4'
  })

  cameraRecorder.ondataavailable = (e) => {
    if (e.data.size > 0) {
      cameraChunks.push(e.data)
    }
  }

  cameraRecorder.onstop = async () => {
    const blob = new Blob(cameraChunks, { type: cameraRecorder.mimeType })
    const ext = cameraRecorder.mimeType.includes('webm') ? 'webm' : 'mp4'
    const file = new File([blob], `video_${Date.now()}.${ext}`, { type: cameraRecorder.mimeType })
    closeCamera()
    await uploadFile(file)
  }

  cameraRecorder.start()
  isRecordingCamera.value = true
}

const stopCameraRecording = () => {
  if (cameraRecorder && cameraRecorder.state !== 'inactive') {
    cameraRecorder.stop()
  }
}

// ========== 录视频功能（直接开始录制） ==========
const startVideoRecording = async () => {
  if (isRecordingVideo.value) return

  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
      audio: true
    })

    cameraStream = stream
    cameraChunks = []
    cameraRecorder = new MediaRecorder(stream, {
      mimeType: MediaRecorder.isTypeSupported('video/webm') ? 'video/webm' : 'video/mp4'
    })

    cameraRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) {
        cameraChunks.push(e.data)
      }
    }

    cameraRecorder.onstop = async () => {
      stream.getTracks().forEach(track => track.stop())
      const blob = new Blob(cameraChunks, { type: cameraRecorder.mimeType })
      const ext = cameraRecorder.mimeType.includes('webm') ? 'webm' : 'mp4'
      const file = new File([blob], `video_${Date.now()}.${ext}`, { type: cameraRecorder.mimeType })
      isRecordingVideo.value = false
      await uploadFile(file)
    }

    cameraRecorder.start()
    isRecordingVideo.value = true
    ElMessage.info('录制中，再次点击停止')
  } catch (error) {
    ElMessage.error('无法访问摄像头: ' + error.message)
  }
}

// ========== 语音消息功能 ==========
const toggleAudioRecording = async () => {
  if (isRecordingAudio.value) {
    stopAudioRecording()
  } else {
    await startAudioRecording()
  }
}

const startAudioRecording = async () => {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    audioChunks = []
    audioRecorder = new MediaRecorder(stream, {
      mimeType: MediaRecorder.isTypeSupported('audio/webm') ? 'audio/webm' : 'audio/mp4'
    })

    audioRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) {
        audioChunks.push(e.data)
      }
    }

    audioRecorder.onstop = async () => {
      stream.getTracks().forEach(track => track.stop())
      const blob = new Blob(audioChunks, { type: audioRecorder.mimeType })
      const ext = audioRecorder.mimeType.includes('webm') ? 'webm' : 'mp3'
      const file = new File([blob], `audio_${Date.now()}.${ext}`, { type: audioRecorder.mimeType })
      await uploadFile(file)
    }

    audioRecorder.start()
    isRecordingAudio.value = true
    recordingDuration.value = 0

    recordingTimer = setInterval(() => {
      recordingDuration.value++
      if (recordingDuration.value >= 60) {
        stopAudioRecording()
      }
    }, 1000)
  } catch (error) {
    ElMessage.error('无法访问麦克风: ' + error.message)
  }
}

const stopAudioRecording = () => {
  if (audioRecorder && audioRecorder.state !== 'inactive') {
    audioRecorder.stop()
  }
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
  isRecordingAudio.value = false
}

const cancelAudioRecording = () => {
  if (audioRecorder) {
    audioRecorder.ondataavailable = null
    audioRecorder.onstop = null
    if (audioRecorder.state !== 'inactive') {
      audioRecorder.stop()
    }
  }
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
  isRecordingAudio.value = false
  audioChunks = []
}

// 预览图片
const previewImage = (url) => {
  previewUrl.value = url
  previewVisible.value = true
}

const leaveRoom = () => {
  // 停止所有录制
  cancelAudioRecording()
  closeCamera()

  if (ws) {
    ws.close()
    ws = null
  }
  currentRoom.value = null
  messages.value = []
  connectionStatus.value = 'disconnected'
  reconnectAttempts.value = 0
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
  cancelAudioRecording()
  closeCamera()
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
  flex-wrap: wrap;
}

.room-id-tag {
  font-size: 12px;
  color: #666;
  background: #2a2a2a;
  padding: 2px 8px;
  border-radius: 4px;
}

/* 连接状态指示器 */
.connection-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  margin-left: 8px;
}

.connection-status.connected {
  color: #67c23a;
  background: rgba(103, 194, 58, 0.1);
}

.connection-status.connecting {
  color: #e6a23c;
  background: rgba(230, 162, 60, 0.1);
}

.connection-status.disconnected {
  color: #f56c6c;
  background: rgba(245, 108, 108, 0.1);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.connection-status.connecting .status-dot {
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
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

.message-image {
  max-width: 300px;
  max-height: 200px;
  border-radius: 8px;
  cursor: pointer;
  transition: transform 0.2s;
}

.message-image:hover {
  transform: scale(1.02);
}

.message-video {
  max-width: 400px;
  max-height: 300px;
  border-radius: 8px;
  background: #000;
}

.message-audio {
  max-width: 300px;
  height: 40px;
  border-radius: 20px;
}

.message-file {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #2a2a2a;
  border-radius: 8px;
  color: #409eff;
  text-decoration: none;
  transition: background 0.2s;
}

.message-file:hover {
  background: #333;
}

.input-area {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.input-tools {
  display: flex;
  gap: 4px;
}

.tool-btn {
  padding: 8px 10px;
}

.tool-btn.recording {
  color: #f56c6c;
  border-color: #f56c6c;
  animation: recording-pulse 1s infinite;
}

@keyframes recording-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.message-input {
  flex: 1;
  min-width: 200px;
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

/* 录音状态显示 */
.recording-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(245, 108, 108, 0.1);
  border-radius: 8px;
  color: #f56c6c;
  font-size: 14px;
  margin-top: 8px;
}

.recording-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #f56c6c;
  animation: recording-pulse 1s infinite;
}

/* 拍照/录像弹窗 */
.camera-container {
  display: flex;
  justify-content: center;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.camera-preview {
  width: 100%;
  max-height: 400px;
  object-fit: contain;
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

  .input-tools {
    width: 100%;
    justify-content: space-between;
    order: 2;
    margin-top: 8px;
  }

  .message-input {
    flex: 1 1 calc(100% - 80px);
    order: 1;
  }

  .input-area > .el-button:last-child {
    order: 1;
  }

  .room-title {
    font-size: 16px;
  }

  .connection-status {
    font-size: 11px;
  }
}
</style>
