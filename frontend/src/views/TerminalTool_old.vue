<template>
  <div class="tool-container">
    <!-- 未登录状态 -->
    <div v-if="!isLoggedIn" class="login-panel">
      <el-card class="login-card">
        <template #header>
          <h3>SSH 终端</h3>
        </template>
        <el-form @submit.prevent="handleLogin" label-position="top">
          <el-form-item label="请输入您的口令">
            <el-input v-model="userToken" placeholder="口令" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" native-type="submit" style="width: 100%">进入</el-button>
          </el-form-item>
        </el-form>
        <p class="tip">口令用于保护您的会话列表，不同口令看到不同的会话</p>
      </el-card>
    </div>

    <!-- 已登录状态 -->
    <template v-else>
      <div class="tool-header">
        <div class="header-left">
          <h2>SSH 终端</h2>
          <el-button type="warning" size="small" @click="showAdminDialog = true">管理</el-button>
          <el-button size="small" @click="logout">切换用户</el-button>
        </div>
        <div class="header-actions">
          <el-button v-if="!isConnected" type="primary" @click="showLoginDialog = true">新建连接</el-button>
          <el-button v-else type="danger" @click="disconnect">断开</el-button>
        </div>
      </div>

      <!-- 历史会话列表 -->
      <div v-if="!isConnected && sessions.length > 0" class="sessions-panel">
        <h3>我的会话</h3>
        <div class="session-list">
          <div v-for="session in sessions" :key="session.id" class="session-item" @click="connectToSession(session)">
            <div class="session-info">
              <div class="session-name">{{ session.name }}</div>
              <div class="session-host">{{ session.username }}@{{ session.host }}:{{ session.port }}</div>
            </div>
            <div class="session-status">
              <el-tag :type="session.connected ? 'success' : 'info'" size="small">
                {{ session.connected ? '在线' : '离线' }}
              </el-tag>
            </div>
          </div>
        </div>
      </div>

      <!-- 连接状态 -->
      <el-alert v-if="!isConnected" type="info" :closable="false" class="status-alert">
        <template #title>
          <el-icon><InfoFilled /></el-icon>
          <span>SSH 远程终端。输入口令后可创建和管理 SSH 连接。</span>
      </template>
    </el-alert>

    <el-alert v-else type="success" :closable="false" class="status-alert">
      <template #title>
        <span>已连接: {{ currentHost }}</span>
      </template>
    </el-alert>

    <!-- 终端容器 -->
    <div ref="terminalContainer" class="terminal-wrapper" :class="{ active: isConnected }"></div>

    <!-- 新建连接对话框 -->
    <el-dialog v-model="showLoginDialog" title="新建 SSH 连接" width="450px" :close-on-click-modal="false">
      <el-form @submit.prevent="handleSubmit" label-position="top">
        <el-form-item label="服务器地址" required>
          <el-input v-model="formData.host" placeholder="例如: 192.168.1.100" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="用户名" required>
          <el-input v-model="formData.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-radio-group v-model="authType">
            <el-radio value="password">密码</el-radio>
            <el-radio value="privateKey">私钥</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="authType === 'password'" label="密码" required>
          <el-input v-model="formData.password" type="password" placeholder="SSH 密码" show-password />
        </el-form-item>
        <el-form-item v-else label="私钥" required>
          <el-input v-model="formData.privateKey" type="textarea" :rows="5" placeholder="私钥内容" />
        </el-form-item>
        <el-form-item label="连接名称">
          <el-input v-model="formData.name" placeholder="我的服务器" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showLoginDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="loading">连接</el-button>
      </template>
    </el-dialog>

    <!-- 登录对话框（离线会话） -->
    <el-dialog v-model="showPwdDialog" title="输入密码" width="400px">
      <el-form @submit.prevent="doLogin" label-position="top">
        <el-form-item label="密码">
          <el-input v-model="loginPwd" :type="loginAuthType === 'password' ? 'password' : 'textarea'" :rows="loginAuthType === 'privateKey' ? 5 : 1" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPwdDialog = false">取消</el-button>
        <el-button type="primary" @click="doLogin" :loading="loading">连接</el-button>
      </template>
    </el-dialog>

    <!-- 管理对话框 -->
    <el-dialog v-model="showAdminDialog" title="管理" width="500px">
      <el-form @submit.prevent="loadAdminSessions" inline>
        <el-form-item label="管理员密码">
          <el-input v-model="adminPassword" type="password" placeholder="管理员密码" style="width: 180px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadAdminSessions">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="adminSessions" style="width: 100%; margin-top: 15px" v-if="adminSessions.length > 0">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="host" label="服务器" width="120" />
        <el-table-column prop="username" label="用户" width="80" />
        <el-table-column label="状态" width="70">
          <template #default="{ row }">
            <el-tag :type="row.connected ? 'success' : 'info'" size="small">{{ row.connected ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="70">
          <template #default="{ row }">
            <el-button type="danger" size="small" @click="adminDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import '@xterm/xterm/css/xterm.css'

const API_BASE = '/api'

const terminalContainer = ref(null)
const showLoginDialog = ref(false)
const showPwdDialog = ref(false)
const showAdminDialog = ref(false)
const loading = ref(false)
const isConnected = ref(false)
const isLoggedIn = ref(false)
const authType = ref('password')
const loginAuthType = ref('password')

const userToken = ref('')
const currentHost = ref('')
const sessions = ref([])
const adminSessions = ref([])
const adminPassword = ref('')
const selectedSession = ref(null)
const loginPwd = ref('')
const sessionId = ref('')

const formData = ref({
  host: '',
  port: 22,
  username: '',
  password: '',
  privateKey: '',
  name: '',
  expiresIn: 365
})

let ws = null
let term = null
let fitAddon = null

const initTerminal = () => {
  console.log('initTerminal called, container:', terminalContainer.value)
  if (!terminalContainer.value) {
    console.error('terminal container is null!')
    return
  }

  // 如果终端已存在，先清理
  if (term) {
    console.log('disposing existing term')
    term.dispose()
    term = null
  }

  console.log('creating new Terminal')
  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: { background: '#1e1e1e', foreground: '#d4d4d4', cursor: '#ffffff', selection: '#264f78' },
    allowProposedApi: true,
    cursorStyle: 'block',
    scrollback: 10000,
    copyOnSelection: true,
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.loadAddon(new WebLinksAddon())
  term.loadAddon(new Unicode11Addon())
  term.unicode.activeVersion = '11'

  console.log('opening terminal in container')
  term.open(terminalContainer.value)

  console.log('fitting terminal')
  fitAddon.fit()
  console.log('focusing terminal')
  term.focus()

  // 绑定输入事件
  term.onData((data) => {
    console.log('xterm input:', data)
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    } else {
      console.log('WebSocket not ready, state:', ws ? ws.readyState : 'ws is null')
    }
  })

  window.addEventListener('resize', handleResize)
  console.log('terminal initialized successfully')
}

const handleResize = () => {
  if (fitAddon) {
    fitAddon.fit()
    if (ws && ws.readyState === WebSocket.OPEN) {
      const { cols, rows } = term
      ws.send(JSON.stringify({ type: 'resize', width: cols, height: rows }))
    }
  }
}

const connect = (id) => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/terminal/${id}/ws?user_token=${encodeURIComponent(userToken.value)}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket connected!')
    isConnected.value = true
    // 连接成功后初始化 xterm（此时容器才可见）
    nextTick(() => {
      initTerminal()
      setTimeout(() => {
        if (term) term.focus()
      }, 200)
    })
  }

  ws.onmessage = (event) => {
    if (!event.data) return
    // 如果是文本消息，尝试解析为 JSON 检查错误
    if (typeof event.data === 'string' && event.data.startsWith('{')) {
      try {
        const msg = JSON.parse(event.data)
        if (msg.error) {
          ElMessage.error(msg.error)
          return
        }
      } catch (e) {
        // 不是 JSON，当作文本处理
      }
    }
    if (event.data instanceof Blob) {
      event.data.arrayBuffer().then(b => term.write(new Uint8Array(b)))
    } else if (event.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(event.data))
    } else {
      term.write(event.data)
    }
  }

  ws.onclose = (e) => {
    isConnected.value = false
    if (term) {
      term.write('\r\n\x1b[33m连接已断开\x1b[0m\r\n')
    }
  }

  ws.onerror = (e) => {
    console.error('WebSocket error:', e)
    if (term) {
      term.write('\r\n\x1b[31m连接错误\x1b[0m\r\n')
    }
  }
}

const disconnect = () => {
  if (ws) { ws.close(); ws = null }
  isConnected.value = false
}

// 加载历史会话（从服务器）
const loadSessions = async () => {
  if (!userToken.value) return
  try {
    const res = await fetch(`${API_BASE}/terminal/list?user_token=${encodeURIComponent(userToken.value)}`)
    if (res.ok) {
      const data = await res.json()
      sessions.value = data.sessions || []
    }
  } catch (e) {
    console.error('加载会话失败:', e)
    sessions.value = []
  }
}

// 登录/切换用户
const handleLogin = async () => {
  if (!userToken.value) {
    ElMessage.warning('请输入口令')
    return
  }
  // 保存到 localStorage
  localStorage.setItem('ssh_user_token', userToken.value)
  isLoggedIn.value = true
  await loadSessions()
}

// 退出登录
const logout = () => {
  localStorage.removeItem('ssh_user_token')
  userToken.value = ''
  sessions.value = []
  isLoggedIn.value = false
}

// 连接会话
const connectToSession = async (session) => {
  loading.value = true
  try {
    // 从服务器获取会话详情（密码由后端直接使用）
    const res = await fetch(`${API_BASE}/terminal/${session.id}?user_token=${encodeURIComponent(userToken.value)}`)
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '获取会话失败')
      return
    }

    sessionId.value = session.id
    currentHost.value = `${session.username}@${session.host}:${session.port}`
    // 密码由后端从数据库获取，不需要前端传递
    connect(session.id)
  } catch (e) {
    ElMessage.error('连接失败')
  } finally {
    loading.value = false
  }
}

// 执行登录（不再需要，因为密码已存储在服务器）
const doLogin = async () => {
  ElMessage.info('密码已保存在服务器，直接点击会话即可连接')
  showPwdDialog.value = false
}

// 创建新连接
const handleSubmit = async () => {
  if (!formData.value.host || !formData.value.username || (!formData.value.password && !formData.value.privateKey)) {
    ElMessage.error('请填写完整信息')
    return
  }

  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/terminal?user_token=${encodeURIComponent(userToken.value)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: formData.value.name || `${formData.value.username}@${formData.value.host}`,
        host: formData.value.host,
        port: formData.value.port,
        username: formData.value.username,
        password: formData.value.password,
        private_key: formData.value.privateKey,
        expires_in: formData.value.expiresIn
      })
    })

    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '创建失败')

    const password = formData.value.password || formData.value.privateKey

    // 刷新会话列表
    await loadSessions()

    sessionId.value = data.id
    currentHost.value = `${formData.value.username}@${formData.value.host}:${formData.value.port}`
    showLoginDialog.value = false

    connect(data.id)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

// 管理员查询
const loadAdminSessions = async () => {
  if (!adminPassword.value) { ElMessage.warning('请输入管理员密码'); return }
  try {
    const res = await fetch(`${API_BASE}/terminal/admin/list?admin_password=${adminPassword.value}`)
    const data = await res.json()
    if (res.ok) {
      adminSessions.value = data.sessions || []
    } else {
      ElMessage.error(data.error || '查询失败')
    }
  } catch (e) {
    ElMessage.error('查询失败')
  }
}

// 管理员删除
const adminDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`删除会话 "${row.name}"?`, '提示', { type: 'warning' })
    const res = await fetch(`${API_BASE}/terminal/admin/${row.id}?admin_password=${adminPassword.value}`, { method: 'DELETE' })
    if (res.ok) {
      ElMessage.success('删除成功')
      loadAdminSessions()
    } else {
      ElMessage.error('删除失败')
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

onMounted(() => {
  const savedToken = localStorage.getItem('ssh_user_token')
  if (savedToken) {
    userToken.value = savedToken
    isLoggedIn.value = true
    loadSessions()
  }
})
onUnmounted(() => {
  disconnect()
  window.removeEventListener('resize', handleResize)
  if (term) term.dispose()
})
</script>

<style scoped>
.tool-container { padding: 20px; height: calc(100vh - 60px); display: flex; flex-direction: column; }
.login-panel { flex: 1; display: flex; align-items: center; justify-content: center; }
.login-card { width: 400px; }
.login-card h3 { margin: 0; text-align: center; }
.login-card .tip { text-align: center; color: #909399; font-size: 12px; margin-top: 15px; }
.tool-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
.header-left { display: flex; align-items: center; gap: 10px; }
.tool-header h2 { margin: 0; font-size: 24px; color: #303133; }
.sessions-panel { margin-bottom: 15px; padding: 15px; background: #fff; border-radius: 8px; }
.sessions-panel h3 { margin: 0 0 10px 0; font-size: 16px; }
.session-list { display: flex; flex-direction: column; gap: 10px; }
.session-item { display: flex; justify-content: space-between; align-items: center; padding: 12px; background: #f5f7fa; border-radius: 8px; cursor: pointer; transition: all 0.2s; }
.session-item:hover { background: #e6f7ff; }
.session-name { font-weight: 500; color: #303133; }
.session-host { font-size: 12px; color: #909399; margin-top: 4px; }
.status-alert { margin-bottom: 15px; }
.terminal-wrapper { flex: 1; background: #1e1e1e; border-radius: 8px; padding: 10px; min-height: 400px; display: none; }
.terminal-wrapper.active { display: block; }
.terminal-wrapper :deep(.xterm) { height: 100%; }
.terminal-wrapper :deep(.xterm) { height: 100%; }
.terminal-wrapper :deep(.xterm-viewport) { overflow-y: auto; }
.header-actions { display: flex; gap: 10px; }
</style>
