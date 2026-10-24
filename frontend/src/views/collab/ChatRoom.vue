<template>
  <div class="wx-app">
    <!-- 左侧深色导航栏（微信风） -->
    <aside class="wx-sidebar">
      <div class="wx-me" :title="nickname || '我'">
        <div class="wx-avatar wx-avatar-lg">{{ (nickname || '我')[0] }}</div>
      </div>
      <nav class="wx-nav">
        <button class="wx-nav-item active" title="聊天"><el-icon :size="22"><ChatDotRound /></el-icon></button>
        <button class="wx-nav-item" title="AI 机器人" @click="openBotDialog"><el-icon :size="22"><Service /></el-icon></button>
        <button class="wx-nav-item" :class="{ on: ttsEnabled }" :title="ttsEnabled ? '自动朗读已开' : '自动朗读已关'" @click="toggleTTS(!ttsEnabled)"><el-icon :size="22"><Microphone /></el-icon></button>
        <button class="wx-nav-item" title="房间管理" @click="showAdminPasswordDialog = true"><el-icon :size="22"><Setting /></el-icon></button>
      </nav>
      <div class="wx-nav-bottom">
        <button v-if="currentRoom" class="wx-nav-item" title="退出房间" @click="leaveRoom"><el-icon :size="22"><SwitchButton /></el-icon></button>
      </div>
    </aside>

    <!-- 中间会话列表 -->
    <section class="wx-list" :class="{ 'mobile-hidden': isMobile && mobileView === 'chat' }">
      <header class="wx-list-header">
        <span class="wx-list-title">聊天室</span>
        <div class="wx-list-actions">
          <el-tooltip content="创建房间" placement="bottom">
            <el-button type="primary" circle size="small" @click="showCreateDialog = true"><el-icon><Plus /></el-icon></el-button>
          </el-tooltip>
        </div>
      </header>

      <div class="wx-search">
        <el-input v-model="roomSearch" placeholder="搜索房间" :prefix-icon="Search" clearable size="default" />
      </div>

      <div class="wx-join-row">
        <el-input v-model="joinRoomId" placeholder="输入房间 ID 加入" size="default" @keyup.enter="joinRoomById">
          <template #append><el-button @click="joinRoomById">加入</el-button></template>
        </el-input>
      </div>

      <div class="wx-room-scroll">
        <div class="wx-list-subtitle">公开房间 · {{ filteredRooms.length }}</div>
        <div v-if="filteredRooms.length === 0" class="wx-empty">暂无房间，点右上角 + 创建</div>
        <div
          v-for="room in filteredRooms"
          :key="room.id"
          :class="['wx-room-item', currentRoom && currentRoom.id === room.id ? 'active' : '']"
          @click="handleJoinRoom(room)"
        >
          <div class="wx-room-avatar" :class="room.has_password ? 'lock' : 'open'">
            <el-icon v-if="room.has_password"><Lock /></el-icon>
            <el-icon v-else><House /></el-icon>
          </div>
          <div class="wx-room-body">
            <div class="wx-room-name">{{ room.name }}</div>
            <div class="wx-room-id">ID: {{ room.id }}</div>
          </div>
          <span v-if="room.has_password" class="wx-badge">密码</span>
        </div>
      </div>
    </section>

    <!-- 右侧聊天区 -->
    <section v-if="currentRoom" class="wx-main" :class="{ 'mobile-hidden': isMobile && mobileView === 'list' }">
      <header class="wx-chat-header">
        <div class="wx-chat-back" @click="mobileView = 'list'"><el-icon><ArrowLeft /></el-icon></div>
        <div class="wx-chat-title">
          <el-icon v-if="currentRoom.has_password" class="wx-lock"><Lock /></el-icon>
          <el-icon v-else class="wx-lock"><House /></el-icon>
          <span class="wx-chat-name">{{ currentRoom.name }}</span>
          <span :class="['wx-conn', connectionStatus]"><i class="dot"></i>{{ connectionStatusText }}</span>
        </div>
        <div class="wx-chat-actions">
          <el-tag v-for="bot in botConfig" :key="bot.nickname" size="small" type="success" class="wx-bot-tag" effect="dark">{{ bot.nickname }}</el-tag>
          <el-tooltip v-if="botConfig.length" content="中断全部 bot 回复" placement="bottom">
            <el-button size="small" type="warning" circle @click="stopBotReply()" :loading="stoppingBot"><el-icon><VideoPause /></el-icon></el-button>
          </el-tooltip>
          <el-tooltip :content="ttsEnabled ? '自动朗读已开' : '自动朗读已关'" placement="bottom">
            <el-button size="small" circle :type="ttsEnabled ? 'primary' : ''" @click="toggleTTS(!ttsEnabled)">{{ ttsEnabled ? '🔊' : '🔇' }}</el-button>
          </el-tooltip>
          <el-tooltip content="AI 机器人" placement="bottom">
            <el-button size="small" circle @click="openBotDialog">🤖</el-button>
          </el-tooltip>
          <el-tooltip content="扫码进房" placement="bottom">
            <el-button size="small" circle type="primary" @click="showQRCode"><el-icon><Tickets /></el-icon></el-button>
          </el-tooltip>
          <el-tooltip content="退出房间" placement="bottom">
            <el-button size="small" circle type="danger" @click="leaveRoom"><el-icon><SwitchButton /></el-icon></el-button>
          </el-tooltip>
        </div>
      </header>

      <div class="wx-messages" ref="messagesContainer">
        <div
          v-for="msg in messages"
          :key="msg.id || msg.created_at"
          :class="['wx-msg', msg.type === 'system' ? 'sys' : (msg.nickname === nickname ? 'mine' : 'other')]"
        >
          <div v-if="msg.type === 'system'" class="wx-sys-text">{{ msg.content }}</div>
          <template v-else>
            <div v-if="msg.nickname !== nickname" class="wx-avatar wx-msg-avatar">{{ (msg.nickname || '?')[0] }}</div>
            <div class="wx-msg-body">
              <div v-if="msg.nickname !== nickname" class="wx-msg-nick">
                <span class="wx-msg-nick-name">{{ msg.nickname }}</span>
                <span class="wx-msg-time">{{ formatTime(msg.created_at) }}</span>
              </div>
              <div class="wx-bubble" :class="msg.msg_type || 'text'">
                <template v-if="msg.msg_type === 'image'">
                  <img :src="msg.content" class="wx-img" @click="previewImage(msg.content)" loading="lazy" />
                </template>
                <template v-else-if="msg.msg_type === 'video'">
                  <video :src="msg.content" class="wx-video" controls preload="metadata"></video>
                </template>
                <template v-else-if="msg.msg_type === 'audio'">
                  <audio :src="msg.content" class="wx-audio" controls preload="metadata"></audio>
                </template>
                <template v-else-if="msg.msg_type === 'file'">
                  <a :href="msg.content" class="wx-file" target="_blank" :download="msg.original_name || 'file'">
                    <el-icon><Document /></el-icon><span>{{ msg.original_name || '下载文件' }}</span>
                  </a>
                </template>
                <template v-else-if="msg.nickname !== nickname">
                  <div class="wx-md" v-html="renderMd(msg.content)"></div>
                </template>
                <template v-else>
                  <span class="wx-text">{{ msg.content }}</span>
                </template>
              </div>
              <div class="wx-msg-actions">
                <el-tooltip content="复制" placement="top" v-if="!msg.msg_type || msg.msg_type === 'text'">
                  <el-icon class="wx-act" @click.stop="copyText(msg.content)"><CopyDocument /></el-icon>
                </el-tooltip>
                <el-tooltip content="复制链接" placement="top" v-if="msg.msg_type === 'image'">
                  <el-icon class="wx-act" @click.stop="copyText(msg.content)"><CopyDocument /></el-icon>
                </el-tooltip>
                <el-tooltip content="下载" placement="top" v-if="['image','video','audio','file'].includes(msg.msg_type)">
                  <el-icon class="wx-act" @click.stop="downloadFile(msg)"><Download /></el-icon>
                </el-tooltip>
                <el-tooltip
                  v-if="isBotMsg(msg) && (!msg.msg_type || msg.msg_type === 'text')"
                  :content="playingMsgId === (msg.id || msg.created_at) ? '停止朗读' : '朗读此条'"
                  placement="top"
                >
                  <el-icon class="wx-act" @click.stop="toggleMsgTTS(msg)">
                    <VideoPause v-if="playingMsgId === (msg.id || msg.created_at)" />
                    <VideoPlay v-else />
                  </el-icon>
                </el-tooltip>
              </div>
            </div>
            <div v-if="msg.nickname === nickname" class="wx-avatar wx-msg-avatar mine">{{ (nickname || '我')[0] }}</div>
          </template>
        </div>
      </div>

      <div class="wx-inputbar">
        <div class="wx-tools">
          <el-popover placement="top-start" trigger="click" v-model:visible="showEmoji" popper-class="emoji-popover">
            <template #reference>
              <el-button class="wx-tool" title="表情"><el-icon><Sugar /></el-icon></el-button>
            </template>
            <div class="emoji-panel">
              <span v-for="emoji in emojis" :key="emoji" class="emoji-item" @click="insertEmoji(emoji)">{{ emoji }}</span>
            </div>
          </el-popover>
          <el-button class="wx-tool" @click="triggerFileUpload" :loading="uploading" title="文件"><el-icon><FolderOpened /></el-icon></el-button>
          <el-button class="wx-tool" @click="triggerImageUpload" :loading="uploading" title="图片"><el-icon><Picture /></el-icon></el-button>
          <el-button class="wx-tool" @click="openCamera" :loading="uploading" title="拍照"><el-icon><Camera /></el-icon></el-button>
          <el-button class="wx-tool" :class="{ recording: isRecordingVideo }" @click="startVideoRecording" title="录视频"><el-icon><VideoCamera /></el-icon></el-button>
          <el-button class="wx-tool" :class="{ recording: isRecordingScreen }" @click="toggleScreenRecording" title="录屏"><el-icon><Monitor /></el-icon></el-button>
          <el-button class="wx-tool" :class="{ recording: isRecordingAudio }" @click="toggleAudioRecording" :loading="uploading" title="语音"><el-icon><Microphone /></el-icon></el-button>
        </div>
        <input type="file" ref="fileInput" accept="*/*" style="display: none" @change="handleFileSelect" />
        <input type="file" ref="imageInput" accept="image/*" style="display: none" @change="handleImageSelect" />
        <div class="wx-input-row">
          <el-input
            v-model="inputMessage"
            placeholder="输入消息，可粘贴图片或视频..."
            @keyup.enter="sendMessage"
            @paste="handlePaste"
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 4 }"
            class="wx-input"
            resize="none"
          />
          <el-button type="primary" class="wx-send" @click="sendMessage" :disabled="!inputMessage.trim()">发送</el-button>
        </div>
      </div>

      <div v-if="isRecordingAudio" class="wx-recording">
        <span class="wx-rec-dot"></span>录音中 {{ recordingDuration }}s
        <el-button size="small" @click="stopAudioRecording">停止并发送</el-button>
        <el-button size="small" @click="cancelAudioRecording">取消</el-button>
      </div>

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

      <el-image-viewer v-if="previewVisible" :url-list="[previewUrl]" @close="previewVisible = false" />
    </section>

    <!-- 未选择房间时的占位 -->
    <section v-else class="wx-main wx-placeholder" :class="{ 'mobile-hidden': isMobile && mobileView === 'list' }">
      <div class="wx-ph-inner">
        <el-icon :size="60" class="wx-ph-icon"><ChatDotRound /></el-icon>
        <p>从左侧选择一个房间开始聊天</p>
      </div>
    </section>

    <!-- 创建房间对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建房间" width="400px">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="房间名称">
          <el-input v-model="createForm.name" placeholder="请输入房间名称" maxlength="50" />
        </el-form-item>
        <el-form-item label="密钥">
          <el-input v-model="createForm.password" type="password" placeholder="可选，留空为公开房间" show-password />
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
          <el-input v-model="joinForm.nickname" :placeholder="joinForm.defaultNickname" maxlength="20" clearable />
        </el-form-item>
        <el-form-item v-if="joinForm.needPassword" label="密钥">
          <el-input v-model="joinForm.password" type="password" placeholder="请输入房间密钥" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showJoinDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmJoin" :loading="joining">加入</el-button>
      </template>
    </el-dialog>

    <!-- 二维码对话框 -->
    <el-dialog v-model="showQRDialog" title="扫码快速进房" width="400px">
      <div class="qr-container">
        <canvas ref="qrCanvas"></canvas>
        <div class="qr-info">
          <p>房间名称：{{ currentRoom?.name }}</p>
          <p>房间ID：{{ currentRoom?.id }}</p>
          <p class="qr-tip">使用手机扫描二维码即可快速加入房间</p>
        </div>
      </div>
    </el-dialog>

    <!-- 机器人管理对话框 -->
    <el-dialog v-model="showBotDialog" title="🤖 AI 机器人" width="520px">
      <div v-if="botConfig.length" class="bot-active-panel">
        <div v-for="bot in botConfig" :key="bot.nickname" class="bot-current">
          <span class="bot-avatar">{{ bot.nickname.split(' ')[0] }}</span>
          <div class="bot-info">
            <div class="bot-name">{{ bot.nickname }}</div>
            <div class="bot-role-label">
              {{ getBotTemplateName(bot.role) }}
              <el-tag v-if="bot.mention_only" size="small" type="info" style="margin-left:4px">@触发</el-tag>
            </div>
          </div>
          <el-button type="danger" size="small" @click="removeBot(bot.nickname)" :loading="addingBot">移除</el-button>
        </div>
      </div>
      <div class="bot-hint">{{ botConfig.length ? '继续添加机器人：' : '选择一个角色加入聊天室' }}</div>
      <div v-if="!botHasKey" class="bot-no-key-tip">⚠️ 未配置 MINIMAX_API_KEY，机器人功能不可用</div>
      <div class="bot-templates">
        <div
          v-for="tmpl in botTemplates"
          :key="tmpl.key"
          :class="['bot-template-card', selectedRole === tmpl.key ? 'selected' : '']"
          @click="selectedRole = tmpl.key"
        >
          <div class="bot-template-icon">{{ tmpl.nickname.split(' ')[0] }}</div>
          <div class="bot-template-name">{{ tmpl.name }}</div>
        </div>
      </div>
      <el-collapse v-model="showBotAdvanced" class="bot-advanced">
        <el-collapse-item title="自定义设置（可选）" name="advanced">
          <el-form label-width="80px" size="small">
            <el-form-item label="昵称">
              <el-input v-model="customBotNickname" placeholder="留空使用模板默认昵称" maxlength="20" />
            </el-form-item>
            <el-form-item label="人设提示">
              <el-input v-model="customSystemPrompt" type="textarea" :rows="3" placeholder="留空使用模板默认人设" />
            </el-form-item>
            <el-form-item label="触发方式">
              <el-switch v-model="mentionOnly" active-text="仅 @我 触发" inactive-text="全部消息触发" />
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
      <template #footer>
        <el-button @click="showBotDialog = false">关闭</el-button>
        <el-button type="primary" @click="addBot" :loading="addingBot" :disabled="!selectedRole || !botHasKey">加入房间</el-button>
      </template>
    </el-dialog>

    <!-- 管理员密码输入对话框 -->
    <el-dialog v-model="showAdminPasswordDialog" title="管理员登录" width="400px">
      <el-form label-width="80px">
        <el-form-item label="管理密码">
          <el-input v-model="adminPassword" type="password" placeholder="请输入管理员密码" show-password @keyup.enter="verifyAdminPassword" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdminPasswordDialog = false">取消</el-button>
        <el-button type="primary" @click="verifyAdminPassword" :loading="verifyingAdmin">登录</el-button>
      </template>
    </el-dialog>

    <!-- 管理员面板对话框 -->
    <el-dialog v-model="showAdminPanel" title="房间管理" width="800px">
      <div class="admin-panel">
        <div class="admin-header">
          <span>共 {{ adminRooms.length }} 个房间</span>
          <el-button size="small" @click="loadAdminRooms" :loading="loadingAdminRooms"><el-icon><Refresh /></el-icon>刷新</el-button>
        </div>
        <el-table :data="adminRooms" style="width: 100%" max-height="500">
          <el-table-column prop="id" label="房间ID" width="120" />
          <el-table-column prop="name" label="房间名称" min-width="150" />
          <el-table-column label="密码保护" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.has_password" type="warning" size="small">有密码</el-tag>
              <el-tag v-else type="success" size="small">公开</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="150">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="最后活跃" width="150">
            <template #default="{ row }">{{ formatDateTime(row.last_active_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button type="danger" size="small" @click="confirmDeleteRoom(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onActivated, onDeactivated, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElImageViewer, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { API_BASE, WS_BASE } from '../../api'
import MarkdownIt from 'markdown-it'
import hljs from '../../utils/highlight'
import { getQRCode } from '../../utils/vendor-loaders'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: false,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try { return `<pre class="hljs"><code>${hljs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>` } catch {}
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  }
})

const renderMd = (text) => md.render(text || '')

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
const showQRDialog = ref(false)
const showAdminPasswordDialog = ref(false)
const showAdminPanel = ref(false)
const creating = ref(false)
const joining = ref(false)
const uploading = ref(false)
const verifyingAdmin = ref(false)
const loadingAdminRooms = ref(false)
const previewVisible = ref(false)
const previewUrl = ref('')

// 管理员相关
const adminPassword = ref('')
const adminRooms = ref([])

// 连接状态
const connectionStatus = ref('disconnected') // connected, connecting, disconnected
const reconnectAttempts = ref(0)

// 响应式布局：移动端单栏切换
const isMobile = ref(typeof window !== 'undefined' && window.innerWidth < 860)
const mobileView = ref('list') // 'list' | 'chat'
const roomSearch = ref('')
const filteredRooms = computed(() => {
  const q = roomSearch.value.trim().toLowerCase()
  if (!q) return rooms.value
  return rooms.value.filter(r => r.name.toLowerCase().includes(q) || String(r.id).toLowerCase().includes(q))
})

// 录音相关
const isRecordingAudio = ref(false)
const recordingDuration = ref(0)
let audioRecorder = null
let audioChunks = []
let recordingTimer = null
let reconnectTimer = null

// 录视频相关
const isRecordingVideo = ref(false)
const isRecordingCamera = ref(false)
let cameraStream = null
let cameraRecorder = null
let cameraChunks = []

// 录屏相关
const isRecordingScreen = ref(false)
let screenStream = null
let screenRecorder = null
let screenChunks = []

const createForm = ref({ name: '', password: '' })
const joinForm = ref({ nickname: '', defaultNickname: '', password: '', needPassword: false, roomId: '' })

const randomNicknames = ['旅行者', '探险家', '路人甲', '神秘人', '匿名者', '游客', '过客', '行者', '漫游者', '访客']
const genNickname = () => {
  const base = randomNicknames[Math.floor(Math.random() * randomNicknames.length)]
  return base + Math.floor(Math.random() * 9000 + 1000)
}

// 机器人相关
const showBotDialog = ref(false)
const botConfig = ref([])   // 多 bot 列表
const botTemplates = ref([])
const botHasKey = ref(false)
const selectedRole = ref('')
const addingBot = ref(false)
const stoppingBot = ref(false)
const customBotNickname = ref('')
const customSystemPrompt = ref('')
const mentionOnly = ref(false)
const showBotAdvanced = ref([])
const ttsEnabled = ref(localStorage.getItem('chat_tts_enabled') !== 'false')
let ttsAudio = null
const ttsQueue = []
let ttsPlaying = false
let ttsGeneration = 0  // 每次 stop 时递增，用于取消飞行中的 fetch
const playingMsgId = ref(null)

const messagesContainer = ref(null)
const fileInput = ref(null)
const imageInput = ref(null)
const cameraVideo = ref(null)
const cameraCanvas = ref(null)
const qrCanvas = ref(null)
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
      defaultNickname: nickname.value || genNickname(),
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
    defaultNickname: nickname.value || genNickname(),
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
      defaultNickname: nickname.value || genNickname(),
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
  const finalNickname = joinForm.value.nickname.trim() || joinForm.value.defaultNickname

  joining.value = true
  try {
    const response = await fetch(`${API_BASE}/api/chat/room/${joinForm.value.roomId}/join`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        nickname: finalNickname,
        password: joinForm.value.password
      })
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '加入失败')
    }

    nickname.value = finalNickname
    currentRoom.value = data.room
    showJoinDialog.value = false
    reconnectAttempts.value = 0
    mobileView.value = 'chat'

    // 持久化，刷新后自动恢复
    sessionStorage.setItem('chat_room_id', data.room.id)
    sessionStorage.setItem('chat_nickname', finalNickname)
    if (joinForm.value.password) {
      localStorage.setItem('chat_room_password', joinForm.value.password)
    }
    // 更新 URL，方便刷新恢复
    const newHash = `#/chat?roomId=${data.room.id}`
    if (window.location.hash !== newHash) {
      window.history.replaceState(null, '', newHash)
    }

    // 加载历史消息 + 机器人配置
    await loadHistoryMessages()
    loadBotConfig()

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

// 加载历史消息
const loadHistoryMessages = async () => {
  try {
    const response = await fetch(`${API_BASE}/api/chat/room/${currentRoom.value.id}/messages`)
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '加载历史消息失败')
    }

    // 将历史消息格式化并添加到消息列表
    messages.value = (data.messages || []).map(m => ({
      ...m,
      type: 'message'
    }))
  } catch (error) {
    console.error('加载历史消息失败:', error)
    // 不阻断流程,即使加载失败也可以继续聊天
    messages.value = []
  }
}

const connectWebSocket = () => {
  if (reconnectAttempts.value >= MAX_RECONNECT_ATTEMPTS) {
    connectionStatus.value = 'disconnected'
    ElMessage.error('连接失败次数过多，请刷新页面重试')
    return
  }

  connectionStatus.value = 'connecting'
  // 加密房间必须在 WS 握手时把密码带上（后端 HandleWebSocket 要求 query 带 password），
  // 否则会被 401 拒绝、WebSocket 升级失败。密码由加入房间时存入 localStorage。
  const roomPassword = localStorage.getItem('chat_room_password') || ''
  const wsUrl = `${WS_BASE}/api/chat/room/${currentRoom.value.id}/ws?nickname=${encodeURIComponent(nickname.value)}` + (roomPassword ? `&password=${encodeURIComponent(roomPassword)}` : '')

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    connectionStatus.value = 'connected'
    reconnectAttempts.value = 0
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type !== 'tts_chunk') messages.value.push(msg)
      nextTick(() => { scrollToBottom() })
      // bot 文本消息 → 等上一个 TTS 播完再请求
      if (msg.type === 'message' && msg.msg_type !== 'image' && msg.msg_type !== 'video' && msg.msg_type !== 'audio' && msg.msg_type !== 'file' && msg.nickname !== nickname.value) {
        const botEntry = botConfig.value.find(b => b.nickname === msg.nickname)
        if (botEntry && ttsEnabled.value) {
          const voice = botEntry.tts_voice || 'zh-CN-XiaoxiaoNeural'
          fetchAndEnqueueTTS(msg.content, voice)
        }
      }
    } catch (error) {
      console.error('消息解析失败:', error)
    }
  }

  ws.onclose = () => {
    connectionStatus.value = 'disconnected'
    if (currentRoom.value) {
      reconnectAttempts.value++
      if (reconnectAttempts.value < MAX_RECONNECT_ATTEMPTS) {
        if (reconnectTimer) clearTimeout(reconnectTimer)
        const delay = Math.min(3000 * reconnectAttempts.value, 15000)
        reconnectTimer = setTimeout(() => {
          reconnectTimer = null
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
        original_name: data.original_name || file.name
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
  // 如果正在录制，停止录制
  if (isRecordingVideo.value) {
    if (cameraRecorder && cameraRecorder.state !== 'inactive') {
      cameraRecorder.stop()
    }
    return
  }

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

// ========== 录屏功能 ==========
const toggleScreenRecording = async () => {
  // 如果正在录制，停止录制
  if (isRecordingScreen.value) {
    if (screenRecorder && screenRecorder.state !== 'inactive') {
      screenRecorder.stop()
    }
    return
  }

  try {
    // 请求屏幕共享权限
    screenStream = await navigator.mediaDevices.getDisplayMedia({
      video: { cursor: 'always' },
      audio: true
    })

    screenChunks = []
    screenRecorder = new MediaRecorder(screenStream, {
      mimeType: MediaRecorder.isTypeSupported('video/webm') ? 'video/webm' : 'video/mp4'
    })

    screenRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) {
        screenChunks.push(e.data)
      }
    }

    screenRecorder.onstop = async () => {
      screenStream.getTracks().forEach(track => track.stop())
      const blob = new Blob(screenChunks, { type: screenRecorder.mimeType })
      const ext = screenRecorder.mimeType.includes('webm') ? 'webm' : 'mp4'
      const file = new File([blob], `screen_${Date.now()}.${ext}`, { type: screenRecorder.mimeType })
      isRecordingScreen.value = false
      await uploadFile(file)
    }

    // 当用户通过浏览器UI停止共享时
    screenStream.getVideoTracks()[0].onended = () => {
      if (screenRecorder && screenRecorder.state !== 'inactive') {
        screenRecorder.stop()
      }
    }

    screenRecorder.start()
    isRecordingScreen.value = true
    ElMessage.info('录屏中，再次点击或停止共享结束')
  } catch (error) {
    if (error.name === 'NotAllowedError') {
      ElMessage.warning('用户取消了屏幕共享')
    } else {
      ElMessage.error('无法启动录屏: ' + error.message)
    }
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

// 停止录屏
const stopScreenRecording = () => {
  if (screenRecorder && screenRecorder.state !== 'inactive') {
    screenRecorder.stop()
  }
  if (screenStream) {
    screenStream.getTracks().forEach(track => track.stop())
    screenStream = null
  }
  isRecordingScreen.value = false
  screenChunks = []
}

// ========== 机器人管理 ==========
const openBotDialog = async () => {
  await loadBotConfig()
  showBotDialog.value = true
}

const loadBotConfig = async () => {
  if (!currentRoom.value) return
  try {
    const res = await fetch(`${API_BASE}/api/chat/room/${currentRoom.value.id}/bot`)
    const data = await res.json()
    botConfig.value = data.bots || []
    botTemplates.value = data.templates || []
    botHasKey.value = data.has_key === true
    if (!selectedRole.value && botTemplates.value.length) {
      selectedRole.value = botTemplates.value[0].key
    }
  } catch (e) {
    console.error('加载机器人配置失败', e)
  }
}

const getBotTemplateName = (key) => {
  return botTemplates.value.find(t => t.key === key)?.name || key
}

const addBot = async () => {
  if (!selectedRole.value) return
  addingBot.value = true
  try {
    const body = { role: selectedRole.value, enable_tts: true, mention_only: mentionOnly.value }
    if (customBotNickname.value.trim()) body.nickname = customBotNickname.value.trim()
    if (customSystemPrompt.value.trim()) body.system_prompt = customSystemPrompt.value.trim()
    const res = await fetch(`${API_BASE}/api/chat/room/${currentRoom.value.id}/bot`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '添加失败')
    // 更新本地列表（同 nickname 覆盖）
    const idx = botConfig.value.findIndex(b => b.nickname === data.bot.nickname)
    if (idx >= 0) botConfig.value[idx] = data.bot
    else botConfig.value.push(data.bot)
    customBotNickname.value = ''
    customSystemPrompt.value = ''
    mentionOnly.value = false
    showBotAdvanced.value = []
    ElMessage.success(`${data.bot.nickname} 已加入房间`)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    addingBot.value = false
  }
}

const removeBot = async (nickname) => {
  addingBot.value = true
  try {
    const url = `${API_BASE}/api/chat/room/${currentRoom.value.id}/bot${nickname ? '?nickname=' + encodeURIComponent(nickname) : ''}`
    const res = await fetch(url, { method: 'DELETE' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '移除失败')
    if (nickname) {
      botConfig.value = botConfig.value.filter(b => b.nickname !== nickname)
    } else {
      botConfig.value = []
    }
    ElMessage.success(`${nickname || '全部机器人'} 已移除`)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    addingBot.value = false
  }
}

const stopBotReply = async (nickname) => {
  if (!currentRoom.value) return
  stoppingBot.value = true
  try {
    const url = `${API_BASE}/api/chat/room/${currentRoom.value.id}/bot/stop${nickname ? '?nickname=' + encodeURIComponent(nickname) : ''}`
    await fetch(url, { method: 'POST' })
    stopTTSAudio()
    ElMessage.success('已中断机器人回复')
  } catch (e) {
    ElMessage.error('中断失败')
  } finally {
    stoppingBot.value = false
  }
}

// ========== TTS 语音 ==========
const toggleTTS = (val) => {
  ttsEnabled.value = val
  localStorage.setItem('chat_tts_enabled', val ? 'true' : 'false')
  if (!val) {
    stopTTSAudio()
    return
  }
  if (ttsQueue.length > 0 && !ttsPlaying) playNextTTS()
}

// 判断是否是 bot 消息（用 nickname 匹配，兼容历史消息）
const isBotMsg = (msg) => {
  if (msg.type !== 'message') return false
  if (botConfig.value.some(b => b.nickname === msg.nickname)) return true
  // 历史消息：botConfig 可能已清空，用 msg.is_bot 字段兜底（如果后端有的话）
  return !!msg.is_bot
}

// 每条消息独立朗读/停止
const toggleMsgTTS = (msg) => {
  const msgId = msg.id || msg.created_at
  if (playingMsgId.value === msgId) {
    stopTTSAudio()
    return
  }
  stopTTSAudio()
  playingMsgId.value = msgId
  const botEntry = botConfig.value.find(b => b.nickname === msg.nickname)
  const voice = botEntry?.tts_voice || 'zh-CN-XiaoxiaoNeural'
  fetchAndEnqueueTTS(msg.content, voice, () => {
    if (playingMsgId.value === msgId) playingMsgId.value = null
  }, true)
}

// 清理 markdown 特殊字符，避免 TTS 合成失败
const stripMarkdown = (text) => {
  return text
    .replace(/```[\s\S]*?```/g, '代码块')   // 代码块
    .replace(/`[^`]*`/g, '')                // 行内代码
    .replace(/#{1,6}\s/g, '')               // 标题
    .replace(/\*\*([^*]+)\*\*/g, '$1')      // 粗体
    .replace(/\*([^*]+)\*/g, '$1')          // 斜体
    .replace(/~~([^~]+)~~/g, '$1')          // 删除线
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')// 链接
    .replace(/!\[[^\]]*\]\([^)]+\)/g, '')   // 图片
    .replace(/^[-*+]\s/gm, '')              // 无序列表
    .replace(/^\d+\.\s/gm, '')              // 有序列表
    .replace(/^>\s/gm, '')                  // 引用
    .replace(/[-]{3,}/g, '')                // 分割线
    .replace(/[\u2018\u2019]/g, "'")        // 弯单引号 → '
    .replace(/[\u201c\u201d]/g, '"')        // 弯双引号 → "
    .replace(/[\u2014\u2013]/g, '，')       // 破折号/短横线 → 逗号
    .replace(/[\u2026]/g, '。')             // 省略号 → 句号
    .trim()
}

// 按句拆分文本（中英文句末标点）
const splitSentences = (text) => {
  const sentences = []
  let cur = ''
  for (const ch of text) {
    cur += ch
    if ('。！？!?\n'.includes(ch)) {
      const s = cur.trim()
      if (s && s.replace(/[。！？!?\s]/g, '').length > 0) sentences.push(s)
      cur = ''
    }
    if (cur.length >= 80) {
      const s = cur.trim()
      if (s && s.replace(/[。！？!?\s]/g, '').length > 0) sentences.push(s)
      cur = ''
    }
  }
  if (cur.trim() && cur.trim().replace(/[。！？!?\s]/g, '').length > 0) sentences.push(cur.trim())
  return sentences
}

const fetchAndEnqueueTTS = (text, voice, onDone, force = false) => {
  const clean = stripMarkdown(text)
  if (!clean) { onDone?.(); return }
  const sentences = splitSentences(clean).filter(s => {
    const readable = s.replace(/https?:\/\/\S+/g, '').replace(/[^\u4e00-\u9fa5a-zA-Z0-9]/g, '').trim()
    return readable.length >= 2
  })
  if (!sentences.length) { onDone?.(); return }

  const total = sentences.length
  const gen = ttsGeneration  // 捕获当前 generation，stop 后 gen 不匹配则丢弃
  const pendingUrls = new Array(total).fill(undefined)
  let flushed = 0

  const tryFlush = () => {
    if (gen !== ttsGeneration) return  // 已被 stop，丢弃
    while (flushed < total && pendingUrls[flushed] !== undefined) {
      const url = pendingUrls[flushed]
      flushed++
      if (url) {
        const isLast = flushed === total
        enqueueTTS(url, isLast ? () => { onDone?.() } : null, force)
      } else if (flushed === total) {
        onDone?.()
      }
    }
  }

  const FALLBACK_VOICE = 'zh-CN-XiaoxiaoNeural'
  sentences.forEach(async (s, i) => {
    let url = null
    const voices = voice === FALLBACK_VOICE ? [voice] : [voice, FALLBACK_VOICE]
    for (const v of voices) {
      if (gen !== ttsGeneration) return
      try {
        const res = await fetch(`${API_BASE}/api/edge-tts/tts`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: s, voice: v })
        })
        const data = await res.json()
        if (res.ok && data.url) { url = data.url; break }
      } catch (e) {
        console.warn('TTS fetch failed:', e)
      }
    }
    pendingUrls[i] = url
    tryFlush()
  })
}

const enqueueTTS = (audioUrl, onPlayed, force = false) => {
  const url = audioUrl.startsWith('/') ? `${location.origin}${audioUrl}` : audioUrl
  ttsQueue.push({ url, onPlayed, force })
  if ((ttsEnabled.value || force) && !ttsPlaying) playNextTTS()
}

const playNextTTS = () => {
  if (ttsQueue.length === 0) {
    ttsPlaying = false
    return
  }
  const { url, onPlayed, force } = ttsQueue[0]
  if (!ttsEnabled.value && !force) {
    ttsPlaying = false
    return
  }
  ttsQueue.shift()
  ttsPlaying = true
  ttsAudio = new Audio(url)
  ttsAudio.onended = () => { onPlayed?.(); playNextTTS() }
  ttsAudio.onerror = () => { onPlayed?.(); playNextTTS() }
  ttsAudio.play().catch(() => { onPlayed?.(); playNextTTS() })
}

const stopTTSAudio = () => {
  ttsGeneration++  // 使所有飞行中的 fetch 失效
  ttsQueue.length = 0
  ttsPlaying = false
  playingMsgId.value = null
  if (ttsAudio) {
    ttsAudio.onended = null
    ttsAudio.onerror = null
    ttsAudio.pause()
    ttsAudio.src = ''
    ttsAudio = null
  }
}

// ========== 复制/下载 ==========
const copyText = async (text) => {
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      // 兼容 HTTP / 旧浏览器
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.cssText = 'position:fixed;top:0;left:0;opacity:0;pointer-events:none'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(ta)
      if (!ok) throw new Error('execCommand failed')
    }
    ElMessage.success('已复制')
  } catch (e) {
    // 最后兜底：提示用户手动复制
    ElMessage.warning('自动复制失败，请手动选中文字后 Ctrl+C')
  }
}

const downloadFile = (msg) => {
  const url = msg.content.startsWith('/') ? `${API_BASE}${msg.content}` : msg.content
  const a = document.createElement('a')
  a.href = url
  a.download = msg.original_name || msg.content.split('/').pop() || 'file'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

const leaveRoom = () => {
  // 停止所有录制
  cancelAudioRecording()
  closeCamera()
  stopScreenRecording()

  if (ws) {
    ws.close()
    ws = null
  }
  stopTTSAudio()
  sessionStorage.removeItem('chat_room_id')
  sessionStorage.removeItem('chat_nickname')
  localStorage.removeItem('chat_room_password')
  window.history.replaceState(null, '', '#/chat')
  currentRoom.value = null
  messages.value = []
  botConfig.value = []
  connectionStatus.value = 'disconnected'
  reconnectAttempts.value = 0
  mobileView.value = 'list'
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

// 显示二维码
const showQRCode = async () => {
  if (!currentRoom.value) return

  showQRDialog.value = true

  await nextTick()

  if (!qrCanvas.value) return

  // 生成房间链接
  const roomUrl = `${window.location.origin}${window.location.pathname}#/chat?roomId=${currentRoom.value.id}`

  try {
    const QRCode = await getQRCode()
    await QRCode.toCanvas(qrCanvas.value, roomUrl, {
      width: 280,
      margin: 2,
      color: {
        dark: '#000000',
        light: '#ffffff'
      }
    })
  } catch (error) {
    console.error('生成二维码失败:', error)
    ElMessage.error('生成二维码失败')
  }
}

// 验证管理员密码
const verifyAdminPassword = async () => {
  if (!adminPassword.value.trim()) {
    ElMessage.warning('请输入管理员密码')
    return
  }

  verifyingAdmin.value = true
  try {
    const response = await fetch(
      `${API_BASE}/api/chat/admin/rooms?admin_password=${encodeURIComponent(adminPassword.value)}`
    )
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '验证失败')
    }

    // 验证成功，显示管理面板
    adminRooms.value = data.rooms || []
    showAdminPasswordDialog.value = false
    showAdminPanel.value = true

    localStorage.setItem('chat_admin_password', adminPassword.value)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    verifyingAdmin.value = false
  }
}

// 加载管理员房间列表
const loadAdminRooms = async () => {
  const password = adminPassword.value || localStorage.getItem('chat_admin_password')
  if (!password) {
    ElMessage.error('请先登录')
    showAdminPanel.value = false
    showAdminPasswordDialog.value = true
    return
  }

  loadingAdminRooms.value = true
  try {
    const response = await fetch(
      `${API_BASE}/api/chat/admin/rooms?admin_password=${encodeURIComponent(password)}`
    )
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '获取房间列表失败')
    }

    adminRooms.value = data.rooms || []
  } catch (error) {
    ElMessage.error(error.message)
    // 如果密码错误，清除缓存并要求重新登录
    if (error.message.includes('密码')) {
      localStorage.removeItem('chat_admin_password')
      adminPassword.value = ''
      showAdminPanel.value = false
      showAdminPasswordDialog.value = true
    }
  } finally {
    loadingAdminRooms.value = false
  }
}

// 确认删除房间
const confirmDeleteRoom = (room) => {
  ElMessageBox.confirm(
    `确定要删除房间 "${room.name}" (${room.id}) 吗？此操作不可撤销。`,
    '警告',
    {
      confirmButtonText: '确定删除',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(() => {
    deleteRoom(room.id)
  }).catch(() => {
    // 取消删除
  })
}

// 删除房间
const deleteRoom = async (roomId) => {
  const password = adminPassword.value || localStorage.getItem('chat_admin_password')
  if (!password) {
    ElMessage.error('请先登录')
    return
  }

  try {
    const response = await fetch(
      `${API_BASE}/api/chat/admin/room/${roomId}?admin_password=${encodeURIComponent(password)}`,
      { method: 'DELETE' }
    )
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '删除失败')
    }
    ElMessage.success('房间删除成功')
    loadAdminRooms()
  } catch (error) {
    ElMessage.error(error.message)
  }
}

// 格式化日期时间
const formatDateTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(async () => {
  // 通过 body class 控制外层滚动，而不是直接改全局 DOM：
  // 本应用所有页面都是 keep-alive 缓存，离开时只触发 onDeactivated 而不会触发
  // onUnmounted，直接改 .main-content 的 style 会残留、导致后续页面无法滚动。
  document.body.classList.add('chat-room-active')
  window.addEventListener('resize', onResize)
  loadRooms()

  const urlParams = new URLSearchParams(window.location.hash.split('?')[1])
  const urlRoomId = urlParams.get('roomId')
  const savedRoomId = sessionStorage.getItem('chat_room_id')
  const savedNickname = sessionStorage.getItem('chat_nickname')
  const savedRoomPassword = localStorage.getItem('chat_room_password') || ''

  // 检查是否有保存的管理员密码
  const savedPassword = localStorage.getItem('chat_admin_password')
  if (savedPassword) adminPassword.value = savedPassword

  // 刷新恢复：URL roomId 与 session 一致，或无 URL roomId 但有 session
  const restoreRoomId = (urlRoomId && urlRoomId === savedRoomId) || (!urlRoomId && savedRoomId)
    ? (savedRoomId || urlRoomId)
    : null

  if (restoreRoomId && savedNickname) {
    try {
      const res = await fetch(`${API_BASE}/api/chat/room/${restoreRoomId}/join`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ nickname: savedNickname, password: savedRoomPassword })
      })
      const data = await res.json()
      if (res.ok) {
        nickname.value = savedNickname
        currentRoom.value = data.room
        reconnectAttempts.value = 0
        mobileView.value = 'chat'
        window.history.replaceState(null, '', `#/chat?roomId=${data.room.id}`)
        await loadHistoryMessages()
        loadBotConfig()
        connectWebSocket()
        nextTick(() => scrollToBottom())
        return
      }
    } catch (e) { /* fall through */ }
    // 恢复失败，清掉 session
    sessionStorage.removeItem('chat_room_id')
    sessionStorage.removeItem('chat_nickname')
    localStorage.removeItem('chat_room_password')
  }

  // 扫码进房（无 session 或恢复失败）
  if (urlRoomId) {
    joinRoomId.value = urlRoomId
    joinRoomById()
  }
})

const onResize = () => {
  isMobile.value = window.innerWidth < 860
}

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  // 还原外层滚动（与 onDeactivated 一起确保在 keep-alive 下也能还原）
  document.body.classList.remove('chat-room-active')
  cancelAudioRecording()
  closeCamera()
  stopScreenRecording()
  stopTTSAudio()
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
  if (recordingTimer) { clearInterval(recordingTimer); recordingTimer = null }
  if (ws) {
    ws.close()
    ws = null
  }
})

onDeactivated(() => {
  // keep-alive 缓存时暂停资源消耗
  stopTTSAudio()
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
  if (recordingTimer) { clearInterval(recordingTimer); recordingTimer = null }
  ttsQueue.length = 0
  ttsPlaying = false
  // 离开聊天室（缓存）时还原外层滚动，避免影响其它页面
  document.body.classList.remove('chat-room-active')
})

onActivated(() => {
  // 重新进入缓存页面时确保外层滚动被锁定（聊天室自己管理滚动）
  document.body.classList.add('chat-room-active')
  if (currentRoom.value && (!ws || ws.readyState !== WebSocket.OPEN)) {
    reconnectAttempts.value = 0
    connectWebSocket()
  }
  nextTick(() => scrollToBottom())
})
</script>

<style scoped>
.wx-app {
  display: flex;
  height: calc(100vh - 40px);
  height: calc(100dvh - 40px);
  background: #ededed;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 18px rgba(0, 0, 0, 0.08);
}

/* ============ 左侧深色导航栏 ============ */
.wx-sidebar {
  width: 64px;
  flex-shrink: 0;
  background: #1e1e26;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 18px 0;
  gap: 22px;
}
.wx-me { margin-bottom: 6px; }
.wx-avatar {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: #07c160;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0;
  user-select: none;
}
.wx-avatar-lg { width: 40px; height: 40px; border-radius: 10px; }
.wx-nav {
  display: flex;
  flex-direction: column;
  gap: 18px;
  flex: 1;
}
.wx-nav-bottom { margin-top: auto; }
.wx-nav-item {
  width: 44px;
  height: 44px;
  border: none;
  background: transparent;
  color: #9a9aa3;
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}
.wx-nav-item:hover { background: rgba(255, 255, 255, 0.08); color: #fff; }
.wx-nav-item.active { color: #07c160; }
.wx-nav-item.on { color: #07c160; }

/* ============ 中间会话列表 ============ */
.wx-list {
  width: 300px;
  flex-shrink: 0;
  background: #e6e6e6;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #d8d8d8;
  min-width: 0;
}
.wx-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 10px;
}
.wx-list-title { font-size: 20px; font-weight: 600; color: #1a1a1a; }
.wx-search { padding: 0 12px 8px; }
.wx-join-row { padding: 0 12px 10px; }
.wx-room-scroll { flex: 1; overflow-y: auto; padding: 0 8px 12px; }
.wx-list-subtitle {
  font-size: 12px;
  color: #8a8a8a;
  padding: 6px 8px;
}
.wx-empty {
  text-align: center;
  color: #9a9a9a;
  font-size: 13px;
  padding: 40px 12px;
}
.wx-room-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
}
.wx-room-item:hover { background: #dcdcdc; }
.wx-room-item.active { background: #ffffff; box-shadow: 0 1px 3px rgba(0,0,0,0.06); }
.wx-room-avatar {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #fff;
  flex-shrink: 0;
}
.wx-room-avatar.open { background: linear-gradient(135deg, #07c160, #04a34e); }
.wx-room-avatar.lock { background: linear-gradient(135deg, #f0a020, #e8890c); }
.wx-room-body { flex: 1; min-width: 0; }
.wx-room-name {
  font-size: 15px;
  color: #1a1a1a;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.wx-room-id {
  font-size: 12px;
  color: #9a9a9a;
  margin-top: 3px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.wx-badge {
  font-size: 11px;
  color: #e8890c;
  background: #fdf1e0;
  border: 1px solid #f6d9a8;
  border-radius: 4px;
  padding: 1px 6px;
  flex-shrink: 0;
}

/* ============ 右侧聊天区 ============ */
.wx-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: #ededed;
}
.wx-placeholder { align-items: center; justify-content: center; }
.wx-ph-inner { text-align: center; color: #b0b0b0; }
.wx-ph-icon { color: #c8c8c8; margin-bottom: 12px; }
.wx-ph-inner p { font-size: 15px; margin: 0; }

.wx-chat-header {
  height: 60px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  background: #f5f5f5;
  border-bottom: 1px solid #e0e0e0;
}
.wx-chat-back { display: none; cursor: pointer; color: #333; }
.wx-chat-title {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  font-size: 17px;
  font-weight: 600;
  color: #1a1a1a;
}
.wx-lock { color: #f0a020; }
.wx-chat-name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.wx-conn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 400;
  padding: 1px 8px;
  border-radius: 10px;
  margin-left: 6px;
}
.wx-conn .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.wx-conn.connected { color: #07c160; background: #e3f6ea; }
.wx-conn.connecting { color: #e8890c; background: #fdf1e0; }
.wx-conn.disconnected { color: #d03050; background: #fbe4e8; }
.wx-chat-actions { display: flex; align-items: center; gap: 6px; }
.wx-bot-tag { border-radius: 10px; }

/* ============ 消息区 ============ */
.wx-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px 18px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.wx-msg { display: flex; align-items: flex-start; gap: 10px; position: relative; }
.wx-msg.sys { justify-content: center; }
.wx-sys-text {
  font-size: 12px;
  color: #999;
  background: #e2e2e2;
  border-radius: 6px;
  padding: 3px 12px;
  max-width: 80%;
  text-align: center;
}
.wx-msg-avatar { width: 38px; height: 38px; border-radius: 8px; font-size: 16px; }
.wx-msg-avatar.mine { background: #07c160; }

.wx-msg-body { max-width: 62%; display: flex; flex-direction: column; }
.wx-msg.mine { flex-direction: row-reverse; }
.wx-msg.mine .wx-msg-body { align-items: flex-end; }

.wx-msg-nick {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 4px;
  padding: 0 2px;
}
.wx-msg-nick-name { font-size: 12px; color: #8a8a8a; }
.wx-msg-time { font-size: 11px; color: #b0b0b0; }

.wx-bubble {
  position: relative;
  padding: 9px 13px;
  border-radius: 8px;
  font-size: 15px;
  line-height: 1.5;
  word-break: break-word;
  overflow-wrap: anywhere;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}
.wx-msg.other .wx-bubble { background: #ffffff; color: #1a1a1a; border-top-left-radius: 2px; }
.wx-msg.mine .wx-bubble { background: #95ec69; color: #0b3d1e; border-top-right-radius: 2px; }
.wx-bubble.image, .wx-bubble.video { padding: 4px; background: transparent !important; box-shadow: none; }
.wx-bubble.audio { padding: 6px 10px; }
.wx-bubble.file { padding: 10px 14px; }
.wx-text { white-space: pre-wrap; }

.wx-img {
  max-width: 240px;
  max-height: 240px;
  border-radius: 8px;
  cursor: zoom-in;
  display: block;
}
.wx-video { max-width: 280px; border-radius: 8px; display: block; }
.wx-audio { width: 220px; }

.wx-file {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1a1a1a;
  text-decoration: none;
  font-size: 14px;
}
.wx-file:hover { color: #07c160; }

/* markdown 内容 */
.wx-md { font-size: 14px; }
.wx-md :deep(pre) {
  background: #2d2d2d;
  color: #f8f8f2;
  padding: 10px 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 13px;
  margin: 6px 0;
}
.wx-md :deep(code) { font-family: 'SFMono-Regular', Consolas, monospace; }
.wx-md :deep(p) { margin: 4px 0; }
.wx-md :deep(a) { color: #576b95; }

/* 消息操作（hover 显示） */
.wx-msg-actions {
  display: flex;
  gap: 10px;
  margin-top: 4px;
  padding: 0 2px;
  opacity: 0;
  transition: opacity 0.15s;
}
.wx-msg:hover .wx-msg-actions { opacity: 1; }
.wx-act {
  cursor: pointer;
  color: #9a9a9a;
  font-size: 15px;
}
.wx-act:hover { color: #07c160; }

/* ============ 输入区 ============ */
.wx-inputbar {
  flex-shrink: 0;
  background: #f5f5f5;
  border-top: 1px solid #e0e0e0;
  padding: 8px 16px calc(12px + env(safe-area-inset-bottom));
}

/* 移动端：单栏布局，可见面板撑满剩余宽度，避免固定 300px 在小屏溢出 */
@media (max-width: 860px) {
  .wx-sidebar { width: 54px; padding: 14px 0; gap: 16px; }
  .wx-list { width: auto; flex: 1; min-width: 0; }
  .wx-main { width: auto; flex: 1; min-width: 0; }
  .wx-chat-header { padding: 0 12px; }
  .wx-chat-back { display: flex; align-items: center; }
  .wx-messages { padding: 14px 12px; }
  .wx-msg-body { max-width: 76%; }
  .wx-img { max-width: 180px; max-height: 180px; }
  .wx-video { max-width: 200px; }
}
.wx-tools { display: flex; gap: 4px; margin-bottom: 6px; }
.wx-tool {
  border: none;
  background: transparent;
  color: #555;
  font-size: 19px;
  width: 34px;
  height: 34px;
  border-radius: 6px;
}
.wx-tool:hover { background: #e6e6e6; color: #07c160; }
.wx-tool.recording { color: #d03050; }
.wx-input-row { display: flex; gap: 10px; align-items: flex-end; }
.wx-input { flex: 1; }
.wx-input :deep(.el-textarea__inner) {
  border-radius: 8px;
  resize: none;
  font-size: 15px;
}
.wx-send {
  background: #07c160;
  border-color: #07c160;
  border-radius: 8px;
  height: 40px;
  padding: 0 22px;
  font-size: 15px;
}
.wx-send:hover { background: #06ad56; border-color: #06ad56; }
.wx-send.is-disabled { background: #9adfbb; border-color: #9adfbb; }

.wx-recording {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  background: #fdf1e0;
  color: #b06a00;
  font-size: 14px;
  border-top: 1px solid #f0d9b0;
}
.wx-rec-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #d03050;
  animation: wx-pulse 1s infinite;
}
@keyframes wx-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.camera-preview { width: 100%; border-radius: 8px; background: #000; }
.qr-container { text-align: center; }
.qr-container canvas { border-radius: 8px; }
.qr-info { margin-top: 14px; font-size: 14px; color: #555; }
.qr-tip { color: #999; font-size: 12px; }

/* 机器人对话框 */
.bot-hint { font-size: 13px; color: #666; margin: 10px 0; }
.bot-no-key-tip {
  font-size: 13px;
  color: #b06a00;
  background: #fdf1e0;
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 10px;
}
.bot-active-panel { margin-bottom: 10px; }
.bot-current {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}
.bot-avatar {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: #07c160;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
}
.bot-info { flex: 1; }
.bot-name { font-size: 14px; font-weight: 500; }
.bot-role-label { font-size: 12px; color: #999; margin-top: 2px; }
.bot-templates { display: flex; flex-wrap: wrap; gap: 10px; }
.bot-template-card {
  width: 88px;
  padding: 12px 6px;
  border: 1px solid #e0e0e0;
  border-radius: 10px;
  text-align: center;
  cursor: pointer;
  transition: all 0.15s;
}
.bot-template-card:hover { border-color: #07c160; }
.bot-template-card.selected { border-color: #07c160; background: #e3f6ea; }
.bot-template-icon {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  background: #07c160;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  margin: 0 auto 6px;
}
.bot-template-name { font-size: 12px; color: #333; }
.bot-advanced { margin-top: 12px; }

.admin-panel { min-height: 300px; }
.admin-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 14px;
  color: #555;
}

/* 表情面板 */
.emoji-panel {
  display: grid;
  grid-template-columns: repeat(10, 1fr);
  gap: 4px;
  max-width: 320px;
}
.emoji-item { font-size: 20px; cursor: pointer; text-align: center; padding: 3px; border-radius: 4px; }
.emoji-item:hover { background: #f0f0f0; }

/* ============ 移动端适配 ============ */
@media (max-width: 859px) {
  .wx-app { border-radius: 0; }
  .wx-list { width: 100%; border-right: none; }
  .wx-main { width: 100%; }
  .wx-list.mobile-hidden, .wx-main.mobile-hidden { display: none !important; }
  .wx-chat-back { display: flex; align-items: center; }
  .wx-msg-body { max-width: 72%; }
}
</style>
