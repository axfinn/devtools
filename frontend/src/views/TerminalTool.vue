<template>
  <div class="tool-container terminal-tool">
    <!-- 用户令牌登录界面 -->
    <div v-if="!state.userToken" class="login-panel">
      <el-card class="login-card">
        <template #header>
          <h3><el-icon><Monitor /></el-icon> SSH 终端</h3>
        </template>
        <el-alert type="info" :closable="false" class="mb-4">
          <p>输入用户令牌以访问您的 SSH 会话。</p>
          <p>首次访问将自动生成新令牌。</p>
        </el-alert>
        <el-form @submit.prevent="handleUserLogin">
          <el-form-item label="用户令牌（可选）">
            <el-input v-model="loginForm.userToken" placeholder="留空自动生成" show-password clearable />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" native-type="submit" style="width: 100%">
              进入终端
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <!-- 主界面 -->
    <template v-else>
      <div class="tool-header">
        <div class="header-left">
          <h2><el-icon><Monitor /></el-icon> SSH 终端</h2>
          <el-tag v-if="state.sessions.length > 0" type="info" size="small">
            {{ state.sessions.length }} 个会话
          </el-tag>
        </div>
        <div class="header-actions">
          <el-button v-if="state.connectionStatus === 'disconnected'" type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon> 新建连接
          </el-button>
          <el-button v-else-if="state.connectionStatus === 'connected'" type="warning" @click="disconnectSession">
            <el-icon><SwitchButton /></el-icon> 断开连接
          </el-button>
          <el-button type="info" @click="loadSessions">
            <el-icon><Refresh /></el-icon>
          </el-button>
          <el-button @click="showSettings = true">
            <el-icon><Setting /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 会话列表（当未连接时显示） -->
      <div v-if="state.connectionStatus === 'disconnected'" class="sessions-panel">
        <el-empty v-if="state.sessions.length === 0" description="暂无SSH会话">
          <el-button type="primary" @click="showCreateDialog">创建第一个连接</el-button>
        </el-empty>

        <div v-else class="session-grid">
          <el-card
            v-for="session in state.sessions"
            :key="session.id"
            class="session-card"
            shadow="hover"
            @click="connectToSession(session)"
          >
            <div class="session-header">
              <div class="session-name">
                <el-icon><Connection /></el-icon>
                {{ session.name }}
              </div>
              <el-tag :type="getStatusType(session.status)" size="small">
                {{ getStatusText(session.status) }}
              </el-tag>
            </div>
            <div class="session-info">
              <div class="info-line">
                <el-icon><User /></el-icon>
                <span>{{ session.username }}@{{ session.host }}:{{ session.port }}</span>
              </div>
              <div class="info-line">
                <el-icon><Clock /></el-icon>
                <span>{{ formatTime(session.last_active_at || session.created_at) }}</span>
              </div>
            </div>
            <div class="session-actions" @click.stop>
              <el-button-group size="small">
                <el-button @click="renameSession(session)">
                  <el-icon><Edit /></el-icon>
                </el-button>
                <el-button @click="deleteSession(session)" type="danger">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-button-group>
            </div>
          </el-card>
        </div>
      </div>

      <!-- 终端容器 -->
      <div v-show="state.connectionStatus !== 'disconnected'" class="terminal-container">
        <div class="terminal-header">
          <div class="terminal-title">
            <el-icon><Monitor /></el-icon>
            <span v-if="state.activeSession">{{ state.activeSession.name }} - {{ state.activeSession.username }}@{{ state.activeSession.host }}</span>
          </div>
          <div class="terminal-status">
            <el-tag :type="state.connectionStatus === 'connected' ? 'success' : 'warning'" size="small">
              {{ state.connectionStatus === 'connected' ? '已连接' : '连接中...' }}
            </el-tag>
          </div>
        </div>
        <div ref="terminalRef" class="terminal-wrapper"></div>

        <!-- 虚拟键盘工具栏 -->
        <div v-if="settings.showVirtualKeyboard" class="virtual-keyboard">
          <!-- 收起/展开按钮 -->
          <div class="vk-toggle" @click="vkExpanded = !vkExpanded">
            <el-icon><ArrowUp v-if="vkExpanded" /><ArrowDown v-else /></el-icon>
            <span>{{ vkExpanded ? '收起键盘' : '显示键盘' }}</span>
          </div>

          <!-- 键盘主体 -->
          <div v-show="vkExpanded" class="vk-content">
            <!-- 关键操作键 - 回车/退格放在最前面 -->
            <div class="vk-row vk-row-critical">
              <button class="vk-btn vk-btn-enter" @click="sendKey('\r')">
                <el-icon><Check /></el-icon>
                <span>回车</span>
              </button>
              <button class="vk-btn vk-btn-backspace" @click="sendKey('\x7f')">
                <el-icon><Back /></el-icon>
                <span>退格</span>
              </button>
              <button class="vk-btn vk-btn-space" @click="sendKey(' ')">
                <span>空格</span>
              </button>
            </div>

            <!-- 方向键 + Tab + Esc -->
            <div class="vk-row vk-row-title">
              <span class="vk-row-label">方向键</span>
            </div>
            <div class="vk-row">
              <button class="vk-btn vk-btn-arrow vk-btn-large" @click="sendKey('\x1b[A')">↑</button>
            </div>
            <div class="vk-row">
              <button class="vk-btn vk-btn-arrow vk-btn-large" @click="sendKey('\x1b[D')">←</button>
              <button class="vk-btn vk-btn-arrow vk-btn-large" @click="sendKey('\x1b[B')">↓</button>
              <button class="vk-btn vk-btn-arrow vk-btn-large" @click="sendKey('\x1b[C')">→</button>
            </div>

            <!-- 功能键 -->
            <div class="vk-row vk-row-title">
              <span class="vk-row-label">功能键</span>
            </div>
            <div class="vk-row">
              <button class="vk-btn vk-btn-fn vk-btn-large" @click="sendKey('\t')">Tab</button>
              <button class="vk-btn vk-btn-fn vk-btn-large" @click="sendKey('\x1b')">Esc</button>
              <button class="vk-btn vk-btn-fn" @click="sendKey('\x1b[H')">Home</button>
              <button class="vk-btn vk-btn-fn" @click="sendKey('\x1b[F')">End</button>
            </div>

            <!-- Ctrl 组合键 -->
            <div class="vk-row vk-row-title">
              <span class="vk-row-label">Ctrl 组合</span>
            </div>
            <div class="vk-row">
              <button class="vk-btn vk-btn-shortcut" @click="sendKey('\x03')">中断</button>
              <button class="vk-btn vk-btn-shortcut" @click="sendKey('\x1a')">挂起</button>
              <button class="vk-btn vk-btn-shortcut" @click="sendShortcut('l')">清屏</button>
              <button class="vk-btn vk-btn-shortcut" @click="sendShortcut('u')">清行</button>
              <button class="vk-btn vk-btn-shortcut" @click="sendShortcut('a')">行首</button>
              <button class="vk-btn vk-btn-shortcut" @click="sendShortcut('e')">行尾</button>
            </div>

            <!-- 更多符号 -->
            <div class="vk-row vk-row-title">
              <span class="vk-row-label">常用符号</span>
            </div>
            <div class="vk-row">
              <button class="vk-btn vk-btn-symbol" @click="sendKey('|')">|</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey('&')">&</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey(';')">;</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey(':')">:</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey('/')">/</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey('-')">-</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey('_')">_</button>
              <button class="vk-btn vk-btn-symbol" @click="sendKey('.')">.</button>
            </div>

            <!-- 操作按钮行 -->
            <div class="vk-row vk-actions">
              <button
                class="vk-btn vk-btn-action"
                :class="{ 'vk-btn-active': voiceInputActive }"
                @click="toggleVoiceInput"
                :disabled="!settings.voiceInputEnabled"
              >
                <el-icon><Microphone /></el-icon>
                <span>{{ voiceInputActive ? '录音中...' : '语音' }}</span>
              </button>
              <button class="vk-btn vk-btn-action" @click="showPasteInput">
                <el-icon><Document /></el-icon>
                <span>粘贴</span>
              </button>
              <button class="vk-btn vk-btn-action" @click="showMoreOptions = true">
                <el-icon><MoreFilled /></el-icon>
                <span>更多</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 粘贴输入框弹窗 -->
        <el-dialog v-model="showPasteDialog" title="粘贴内容" width="400px">
          <el-input
            v-model="pasteText"
            type="textarea"
            :rows="5"
            placeholder="请输入或粘贴要发送的内容"
          />
          <template #footer>
            <el-button @click="showPasteDialog = false">取消</el-button>
            <el-button type="primary" @click="sendPasteText">发送</el-button>
          </template>
        </el-dialog>

        <!-- 更多选项弹窗 -->
        <el-dialog v-model="showMoreOptions" title="更多选项" width="400px">
          <el-form label-width="100px">
            <el-form-item label="快速命令">
              <el-button size="small" @click="sendCommand('ls')">ls</el-button>
              <el-button size="small" @click="sendCommand('cd')">cd</el-button>
              <el-button size="small" @click="sendCommand('pwd')">pwd</el-button>
              <el-button size="small" @click="sendCommand('clear')">clear</el-button>
              <el-button size="small" @click="sendCommand('exit')">exit</el-button>
            </el-form-item>
            <el-form-item label="自定义输入">
              <el-input
                v-model="customCommand"
                placeholder="输入自定义命令"
                @keyup.enter="sendCustomCommand"
              >
                <template #append>
                  <el-button @click="sendCustomCommand">发送</el-button>
                </template>
              </el-input>
            </el-form-item>
          </el-form>
        </el-dialog>
      </div>
    </template>

    <!-- 新建SSH连接对话框 -->
    <el-dialog v-model="dialogs.create" title="新建 SSH 连接" width="500px" @close="resetCreateForm">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="连接名称" prop="name">
          <el-input v-model="createForm.name" placeholder="我的服务器" />
        </el-form-item>
        <el-form-item label="主机地址" prop="host">
          <el-input v-model="createForm.host" placeholder="192.168.1.100" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="createForm.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="createForm.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-radio-group v-model="createForm.authType">
            <el-radio label="password">密码</el-radio>
            <el-radio label="key">私钥</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="createForm.authType === 'password'" label="密码" prop="password">
          <el-input v-model="createForm.password" type="password" placeholder="SSH 密码" show-password />
        </el-form-item>
        <el-form-item v-else label="私钥" prop="privateKey">
          <el-input v-model="createForm.privateKey" type="textarea" :rows="6" placeholder="-----BEGIN RSA PRIVATE KEY-----" />
        </el-form-item>
        <el-form-item label="保持连接">
          <el-switch v-model="createForm.keepAlive" />
          <span class="form-tip">刷新页面后自动重连</span>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-select v-model="createForm.expiresIn" placeholder="选择过期时间">
            <el-option label="永不过期" :value="0" />
            <el-option label="1天" :value="1" />
            <el-option label="7天" :value="7" />
            <el-option label="30天" :value="30" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogs.create = false">取消</el-button>
        <el-button type="primary" @click="createAndConnect" :loading="state.loading">创建并连接</el-button>
      </template>
    </el-dialog>

    <!-- 重命名对话框 -->
    <el-dialog v-model="dialogs.rename" title="重命名会话" width="400px">
      <el-form @submit.prevent="submitRename">
        <el-form-item label="新名称">
          <el-input v-model="renameForm.name" placeholder="输入新名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogs.rename = false">取消</el-button>
        <el-button type="primary" @click="submitRename">确定</el-button>
      </template>
    </el-dialog>

    <!-- 设置对话框 -->
    <el-dialog v-model="showSettings" title="设置" width="500px">
      <el-form label-width="120px">
        <el-form-item label="用户令牌">
          <el-input v-model="state.userToken" readonly>
            <template #append>
              <el-button @click="copyUserToken">复制</el-button>
            </template>
          </el-input>
          <div class="form-tip">保存此令牌以在其他设备访问您的会话</div>
        </el-form-item>
        <el-form-item label="终端字体大小">
          <el-slider v-model="settings.fontSize" :min="12" :max="24" @change="updateTerminalSettings" />
        </el-form-item>
        <el-form-item label="终端主题">
          <el-select v-model="settings.theme" @change="updateTerminalSettings">
            <el-option label="默认" value="default" />
            <el-option label="暗色" value="dark" />
          </el-select>
        </el-form-item>
        <el-form-item label="虚拟键盘">
          <el-switch v-model="settings.showVirtualKeyboard" />
          <span class="form-tip">在终端底部显示虚拟键盘</span>
        </el-form-item>
        <el-form-item label="语音输入">
          <el-switch v-model="settings.voiceInputEnabled" />
          <span class="form-tip">启用语音识别输入命令</span>
        </el-form-item>
        <el-form-item>
          <el-button type="danger" @click="clearAllSessions">清除所有会话</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Monitor, Plus, Refresh, Setting, Connection, User, Clock,
  Edit, Delete, SwitchButton, ArrowUp, ArrowDown,
  Microphone, Document, MoreFilled, Check, Back
} from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

// ==================== State ====================
const state = reactive({
  userToken: localStorage.getItem('ssh_user_token') || '',
  sessions: [],
  activeSession: null,
  wsConnection: null,
  terminal: null,
  connectionStatus: 'disconnected', // disconnected, connecting, connected
  loading: false
})

const loginForm = reactive({
  userToken: ''
})

const createForm = reactive({
  name: '',
  host: '',
  port: 22,
  username: '',
  authType: 'password',
  password: '',
  privateKey: '',
  keepAlive: true,
  expiresIn: 0
})

const renameForm = reactive({
  sessionId: '',
  name: '',
  creatorKey: ''
})

const settings = reactive({
  fontSize: 14,
  theme: 'default',
  showVirtualKeyboard: true,
  voiceInputEnabled: true
})

const dialogs = reactive({
  create: false,
  rename: false
})

const showSettings = ref(false)
const terminalRef = ref(null)
const createFormRef = ref(null)

// 虚拟键盘状态
const vkExpanded = ref(true)
const showPasteDialog = ref(false)
const showMoreOptions = ref(false)
const pasteText = ref('')
const customCommand = ref('')
const voiceInputActive = ref(false)
let recognition = null

let terminal = null
let fitAddon = null
let reconnectTimer = null
let heartbeatTimer = null

// ==================== Validation Rules ====================
const createRules = {
  name: [{ required: false }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, type: 'number', message: '请输入端口', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  privateKey: [{ required: true, message: '请输入私钥', trigger: 'blur' }]
}

// ==================== User Login ====================
async function handleUserLogin() {
  try {
    const token = loginForm.userToken.trim() || generateUserToken()

    // 保存令牌
    state.userToken = token
    localStorage.setItem('ssh_user_token', token)

    // 加载会话列表
    await loadSessions()

    ElMessage.success('登录成功')
  } catch (error) {
    ElMessage.error('登录失败: ' + error.message)
  }
}

function generateUserToken() {
  return 'user_' + Math.random().toString(36).substring(2, 15) +
         Math.random().toString(36).substring(2, 15)
}

// ==================== Sessions Management ====================
async function loadSessions() {
  try {
    const response = await fetch(`/api/terminal/list?user_token=${state.userToken}`)
    const data = await response.json()

    if (response.ok) {
      state.sessions = data.sessions || []
    } else {
      throw new Error(data.error || '加载会话列表失败')
    }
  } catch (error) {
    ElMessage.error('加载会话失败: ' + error.message)
  }
}

function showCreateDialog() {
  dialogs.create = true
}

function resetCreateForm() {
  Object.assign(createForm, {
    name: '',
    host: '',
    port: 22,
    username: '',
    authType: 'password',
    password: '',
    privateKey: '',
    keepAlive: true,
    expiresIn: 0
  })
}

async function createAndConnect() {
  if (!createFormRef.value) return

  try {
    await createFormRef.value.validate()
  } catch {
    return
  }

  state.loading = true

  try {
    // 创建会话
    const response = await fetch('/api/terminal', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: createForm.name || `${createForm.username}@${createForm.host}`,
        host: createForm.host,
        port: createForm.port,
        username: createForm.username,
        password: createForm.authType === 'password' ? createForm.password : '',
        private_key: createForm.authType === 'key' ? createForm.privateKey : '',
        user_token: state.userToken,
        keep_alive: createForm.keepAlive,
        expires_in: createForm.expiresIn || 0,
        width: 80,
        height: 24
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '创建会话失败')
    }

    // 保存创建者密钥
    localStorage.setItem(`ssh_creator_${data.id}`, data.creator_key)

    dialogs.create = false
    ElMessage.success('会话创建成功')

    // 连接到会话
    await connectToSession(data)

  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    state.loading = false
  }
}

// ==================== SSH Connection ====================
async function connectToSession(session) {
  state.activeSession = session
  state.connectionStatus = 'connecting'

  await nextTick()

  // 初始化终端
  initTerminal()

  // 建立 WebSocket 连接
  connectWebSocket(session.id)
}

function initTerminal() {
  if (terminal) {
    terminal.dispose()
  }

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: settings.fontSize,
    fontFamily: 'Consolas, "Courier New", monospace',
    theme: {
      background: settings.theme === 'dark' ? '#1e1e1e' : '#ffffff',
      foreground: settings.theme === 'dark' ? '#d4d4d4' : '#000000'
    },
    rows: 24,
    cols: 80
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  terminal.open(terminalRef.value)
  fitAddon.fit()

  // 监听输入
  terminal.onData(data => {
    if (state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
      state.wsConnection.send(JSON.stringify({
        type: 'input',
        data: data
      }))
    }
  })

  // 监听大小变化
  terminal.onResize(({ cols, rows }) => {
    if (state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
      state.wsConnection.send(JSON.stringify({
        type: 'resize',
        cols,
        rows
      }))
    }
  })

  // 窗口大小变化时调整终端大小
  window.addEventListener('resize', handleResize)

  state.terminal = terminal
}

function handleResize() {
  if (fitAddon) {
    fitAddon.fit()
  }
}

function connectWebSocket(sessionId) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/terminal/${sessionId}/ws?user_token=${state.userToken}`

  const ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket connected')
    state.connectionStatus = 'connected'
    state.wsConnection = ws
    startHeartbeat()
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)

      switch (msg.type) {
        case 'output':
          if (terminal) {
            terminal.write(msg.data)
          }
          break

        case 'status':
          if (msg.status === 'connected') {
            ElMessage.success(msg.message || 'SSH 连接成功')
          }
          break

        case 'history':
          if (terminal && msg.data) {
            terminal.write(msg.data)
          }
          break

        case 'error':
          if (msg.decryptError) {
            // 解密失败，提示用户删除会话
            ElMessageBox.confirm(
              msg.message + '。是否删除该会话？',
              '连接失败',
              {
                confirmButtonText: '删除会话',
                cancelButtonText: '取消',
                type: 'warning',
              }
            ).then(async () => {
              // 删除会话
              await deleteSession(msg.sessionId)
              state.activeSession = null
            }).catch(() => {})
          } else {
            ElMessage.error(msg.message || 'SSH 连接错误')
          }
          break

        case 'ping':
          ws.send(JSON.stringify({ type: 'pong' }))
          break
      }
    } catch (error) {
      console.error('WebSocket message parse error:', error)
    }
  }

  ws.onerror = (error) => {
    console.error('WebSocket error:', error)
    ElMessage.error('WebSocket 连接错误')
  }

  ws.onclose = () => {
    console.log('WebSocket closed')
    state.connectionStatus = 'disconnected'
    state.wsConnection = null
    stopHeartbeat()

    // 尝试重连（如果是意外断开）
    if (state.activeSession) {
      ElMessage.warning('连接已断开')
    }
  }
}

function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    if (state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
      state.wsConnection.send(JSON.stringify({ type: 'ping' }))
    }
  }, 30000) // 30秒心跳
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

function disconnectSession() {
  if (state.wsConnection) {
    state.wsConnection.close()
    state.wsConnection = null
  }

  if (terminal) {
    terminal.dispose()
    terminal = null
  }

  state.activeSession = null
  state.connectionStatus = 'disconnected'

  loadSessions()
}

// ==================== Session Actions ====================
function renameSession(session) {
  renameForm.sessionId = session.id
  renameForm.name = session.name
  renameForm.creatorKey = localStorage.getItem(`ssh_creator_${session.id}`) || ''
  dialogs.rename = true
}

async function submitRename() {
  if (!renameForm.name || !renameForm.creatorKey) {
    ElMessage.error('请输入新名称')
    return
  }

  try {
    const response = await fetch(`/api/terminal/${renameForm.sessionId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'rename',
        name: renameForm.name,
        creator_key: renameForm.creatorKey
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '重命名失败')
    }

    dialogs.rename = false
    ElMessage.success('重命名成功')
    await loadSessions()
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function deleteSession(session) {
  const creatorKey = localStorage.getItem(`ssh_creator_${session.id}`)

  if (!creatorKey) {
    ElMessage.error('无权删除此会话（缺少创建者密钥）')
    return
  }

  try {
    await ElMessageBox.confirm(`确定要删除会话 "${session.name}" 吗？`, '确认删除', {
      type: 'warning'
    })

    const response = await fetch(`/api/terminal/${session.id}?creator_key=${creatorKey}`, {
      method: 'DELETE'
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '删除失败')
    }

    localStorage.removeItem(`ssh_creator_${session.id}`)
    ElMessage.success('删除成功')
    await loadSessions()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// ==================== Utility Functions ====================
function getStatusType(status) {
  switch (status) {
    case 'active': return 'success'
    case 'idle': return 'info'
    case 'expired': return 'danger'
    default: return 'info'
  }
}

function getStatusText(status) {
  switch (status) {
    case 'active': return '在线'
    case 'idle': return '离线'
    case 'expired': return '已过期'
    default: return '未知'
  }
}

function formatTime(time) {
  if (!time) return '未知'
  const date = new Date(time)
  const now = new Date()
  const diff = Math.floor((now - date) / 1000)

  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  if (diff < 604800) return `${Math.floor(diff / 86400)} 天前`

  return date.toLocaleDateString('zh-CN')
}

function copyUserToken() {
  navigator.clipboard.writeText(state.userToken).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

function updateTerminalSettings() {
  if (terminal) {
    terminal.options.fontSize = settings.fontSize
    terminal.options.theme = {
      background: settings.theme === 'dark' ? '#1e1e1e' : '#ffffff',
      foreground: settings.theme === 'dark' ? '#d4d4d4' : '#000000'
    }
    if (fitAddon) {
      fitAddon.fit()
    }
  }
}

// 虚拟键盘方法
function sendKey(key) {
  if (terminal && state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
    state.wsConnection.send(JSON.stringify({
      type: 'input',
      data: key
    }))
  }
}

function sendShortcut(key) {
  // Ctrl + 字母键
  const ctrlKey = key.toLowerCase().charCodeAt(0) - 96
  sendKey(String.fromCharCode(ctrlKey))
}

function showPasteInput() {
  // 尝试读取剪贴板
  if (navigator.clipboard) {
    navigator.clipboard.readText().then(text => {
      pasteText.value = text || ''
      showPasteDialog.value = true
    }).catch(() => {
      pasteText.value = ''
      showPasteDialog.value = true
    })
  } else {
    pasteText.value = ''
    showPasteDialog.value = true
  }
}

function sendPasteText() {
  if (pasteText.value && terminal && state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
    state.wsConnection.send(JSON.stringify({
      type: 'input',
      data: pasteText.value
    }))
  }
  pasteText.value = ''
  showPasteDialog.value = false
}

function sendCommand(cmd) {
  if (terminal && state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
    state.wsConnection.send(JSON.stringify({
      type: 'input',
      data: cmd + '\r'
    }))
  }
  showMoreOptions.value = false
}

function sendCustomCommand() {
  if (customCommand.value && terminal && state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
    state.wsConnection.send(JSON.stringify({
      type: 'input',
      data: customCommand.value + '\r'
    }))
    customCommand.value = ''
  }
  showMoreOptions.value = false
}

// 语音输入
function toggleVoiceInput() {
  if (!settings.voiceInputEnabled) {
    ElMessage.warning('请先在设置中开启语音输入')
    return
  }

  if (voiceInputActive.value) {
    stopVoiceInput()
  } else {
    startVoiceInput()
  }
}

function startVoiceInput() {
  // 检查浏览器是否支持语音识别
  const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition

  if (!SpeechRecognition) {
    ElMessage.error('您的浏览器不支持语音识别功能')
    return
  }

  try {
    recognition = new SpeechRecognition()
    recognition.lang = 'zh-CN'
    recognition.continuous = true
    recognition.interimResults = true

    recognition.onstart = () => {
      voiceInputActive.value = true
      ElMessage.success('语音识别已启动')
    }

    recognition.onresult = (event) => {
      let finalTranscript = ''
      let interimTranscript = ''

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const transcript = event.results[i][0].transcript
        if (event.results[i].isFinal) {
          finalTranscript += transcript
        } else {
          interimTranscript += transcript
        }
      }

      // 发送识别的文本
      if (finalTranscript) {
        sendVoiceText(finalTranscript)
      }
    }

    recognition.onerror = (event) => {
      console.error('语音识别错误:', event.error)
      voiceInputActive.value = false
      if (event.error === 'not-allowed') {
        ElMessage.error('请允许麦克风权限')
      } else if (event.error === 'no-speech') {
        ElMessage.warning('未检测到语音')
      } else {
        ElMessage.error('语音识别错误: ' + event.error)
      }
    }

    recognition.onend = () => {
      voiceInputActive.value = false
    }

    recognition.start()
  } catch (error) {
    console.error('启动语音识别失败:', error)
    ElMessage.error('启动语音识别失败')
  }
}

function stopVoiceInput() {
  if (recognition) {
    recognition.stop()
    recognition = null
  }
  voiceInputActive.value = false
}

function sendVoiceText(text) {
  // 将识别的文本发送到终端
  if (terminal && state.wsConnection && state.wsConnection.readyState === WebSocket.OPEN) {
    state.wsConnection.send(JSON.stringify({
      type: 'input',
      data: text
    }))
  }
}

async function clearAllSessions() {
  try {
    await ElMessageBox.confirm('确定要清除所有会话吗？此操作不可恢复！', '警告', {
      type: 'warning',
      confirmButtonText: '确定清除',
      cancelButtonText: '取消'
    })

    // 删除所有会话
    for (const session of state.sessions) {
      const creatorKey = localStorage.getItem(`ssh_creator_${session.id}`)
      if (creatorKey) {
        try {
          await fetch(`/api/terminal/${session.id}?creator_key=${creatorKey}`, {
            method: 'DELETE'
          })
          localStorage.removeItem(`ssh_creator_${session.id}`)
        } catch (error) {
          console.error('Failed to delete session:', session.id, error)
        }
      }
    }

    showSettings.value = false
    ElMessage.success('已清除所有会话')
    await loadSessions()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('清除失败')
    }
  }
}

// ==================== Lifecycle ====================
onMounted(() => {
  if (state.userToken) {
    loadSessions()
  }
})

onBeforeUnmount(() => {
  stopVoiceInput()
  disconnectSession()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.terminal-tool {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 600px;
}

/* Login Panel */
.login-panel {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 500px;
}

.login-card {
  width: 100%;
  max-width: 450px;
}

.login-card h3 {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.mb-4 {
  margin-bottom: 16px;
}

/* Header */
.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-left h2 {
  margin: 0;
  font-size: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

/* Sessions Panel */
.sessions-panel {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.session-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.session-card {
  cursor: pointer;
  transition: all 0.3s;
}

.session-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.session-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.session-name {
  font-size: 16px;
  font-weight: bold;
  display: flex;
  align-items: center;
  gap: 6px;
}

.session-info {
  margin-bottom: 12px;
}

.info-line {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  margin-bottom: 4px;
}

.session-actions {
  display: flex;
  justify-content: flex-end;
}

/* Terminal Container */
.terminal-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #000;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #2d2d2d;
  color: #fff;
  flex-shrink: 0;
}

.terminal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.terminal-wrapper {
  flex: 1;
  overflow: hidden;
  padding: 8px;
}

.terminal-wrapper :deep(.xterm) {
  height: 100%;
  width: 100%;
}

.terminal-wrapper :deep(.xterm-viewport) {
  overflow-y: auto;
}

/* Form Tips */
.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 8px;
}

/* 虚拟键盘 */
.virtual-keyboard {
  background: #2d2d2d;
  border-top: 1px solid #444;
  flex-shrink: 0;
  padding-bottom: env(safe-area-inset-bottom);
}

.vk-row-critical {
  margin-bottom: 10px;
}

.vk-btn-enter {
  flex: 1;
  height: 48px;
  background: #2e7d32;
  font-size: 14px;
  min-width: auto;
}

.vk-btn-enter:hover {
  background: #3e8d42;
}

.vk-btn-backspace {
  height: 48px;
  background: #c62828;
  font-size: 14px;
  min-width: 80px;
}

.vk-btn-backspace:hover {
  background: #d63838;
}

.vk-btn-space {
  flex: 2;
  height: 48px;
  background: #424242;
  font-size: 14px;
  min-width: auto;
}

.vk-btn-space:hover {
  background: #525252;
}

.vk-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  background: #3d3d3d;
  transition: background 0.2s;
}

.vk-toggle:hover {
  background: #4d4d4d;
}

.vk-content {
  padding: 8px;
}

.vk-row {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
  flex-wrap: wrap;
  justify-content: center;
}

.vk-row-title {
  margin-bottom: 2px;
}

.vk-row-label {
  color: #888;
  font-size: 11px;
  text-transform: uppercase;
}

.vk-row:last-child {
  margin-bottom: 0;
}

.vk-btn {
  min-width: 44px;
  height: 40px;
  padding: 0 12px;
  border: none;
  border-radius: 4px;
  background: #4a4a4a;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.vk-btn:hover {
  background: #5a5a5a;
}

.vk-btn:active {
  background: #3a3a3a;
  transform: scale(0.95);
}

.vk-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.vk-btn-large {
  min-width: 60px;
  height: 48px;
  font-size: 16px;
  font-weight: bold;
}

.vk-btn-fn {
  min-width: 50px;
  background: #3d5a80;
}

.vk-btn-fn:hover {
  background: #4d6a90;
}

.vk-btn-arrow {
  min-width: 60px;
  height: 48px;
  background: #1565c0;
  font-size: 22px;
  font-weight: bold;
}

.vk-btn-arrow:hover {
  background: #4d6a90;
}

.vk-btn-shortcut {
  min-width: 48px;
  background: #5c4033;
  font-size: 12px;
}

.vk-btn-symbol {
  min-width: 40px;
  height: 36px;
  background: #37474f;
  font-size: 14px;
  font-weight: bold;
}

.vk-btn-shortcut:hover {
  background: #6c5043;
}

.vk-btn-action {
  min-width: 70px;
  background: #2d5a3d;
}

.vk-btn-action:hover {
  background: #3d6a4d;
}

.vk-btn-active {
  background: #d32f2f !important;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

.vk-actions {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #444;
}

/* Responsive */
@media (max-width: 768px) {
  .session-grid {
    grid-template-columns: 1fr;
  }

  /* 虚拟键盘移动端优化 */
  .virtual-keyboard {
    padding: 6px;
  }

  .vk-content {
    padding: 4px;
  }

  .vk-row {
    gap: 4px;
    margin-bottom: 4px;
  }

  .vk-btn {
    min-width: 36px;
    height: 36px;
    font-size: 12px;
    padding: 0 6px;
  }

  .vk-btn-large {
    min-width: 50px;
    height: 42px;
    font-size: 14px;
  }

  .vk-btn-arrow {
    min-width: 50px;
    height: 42px;
    font-size: 18px;
  }

  .vk-btn-enter {
    height: 42px;
    font-size: 12px;
  }

  .vk-btn-backspace {
    height: 42px;
    min-width: 60px;
    font-size: 12px;
  }

  .vk-btn-space {
    height: 42px;
  }

  .vk-btn-shortcut {
    min-width: 40px;
    font-size: 11px;
  }

  .vk-btn-symbol {
    min-width: 32px;
    height: 32px;
    font-size: 12px;
  }

  .vk-toggle {
    padding: 6px;
    font-size: 12px;
  }

  .vk-row-label {
    font-size: 10px;
  }

  .tool-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
