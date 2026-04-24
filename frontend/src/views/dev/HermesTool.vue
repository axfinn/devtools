<template>
  <div class="hermes-tool">
    <div v-if="!authenticated" class="login-card">
      <el-card shadow="hover">
        <template #header>
          <div class="card-header">
            <span>Hermes Agent</span>
          </div>
        </template>
        <div class="login-copy">
          <div class="hero-title">Hermes 接入控制台</div>
          <div class="hero-subtitle">按站内管理员密码模式进入，统一查看 Dashboard、Gateway 和对话联通性。</div>
        </div>
        <el-form @submit.prevent="login">
          <el-form-item label="管理员密码">
            <el-input
              v-model="passwordInput"
              type="password"
              show-password
              placeholder="请输入管理员密码"
              @keyup.enter="login"
            />
          </el-form-item>
          <el-button type="primary" :loading="loggingIn" @click="login">进入模块</el-button>
        </el-form>
      </el-card>
    </div>

    <div v-else class="main-content">
      <el-card class="status-card">
        <template #header>
          <div class="card-header">
            <span>服务状态</span>
            <div class="header-actions">
              <el-button size="small" @click="refreshAll" :loading="loadingStatus">刷新</el-button>
              <el-button size="small" @click="logout">退出</el-button>
            </div>
          </div>
        </template>

        <div v-if="status" class="status-grid">
          <div class="status-item">
            <div class="status-title">Dashboard</div>
            <el-tag :type="status.dashboard?.ok ? 'success' : 'danger'">
              {{ status.dashboard?.ok ? '在线' : '异常' }}
            </el-tag>
            <div class="status-link">{{ status.dashboard_url }}</div>
            <div class="status-actions">
              <el-button size="small" @click="openLink(status.dashboard_url)">打开</el-button>
              <el-button size="small" @click="copy(status.dashboard_url)">复制地址</el-button>
            </div>
          </div>

          <div class="status-item">
            <div class="status-title">Gateway API</div>
            <el-tag :type="status.gateway?.ok ? 'success' : 'danger'">
              {{ status.gateway?.ok ? '在线' : '异常' }}
            </el-tag>
            <div class="status-link">{{ status.api_base_url }}</div>
            <div class="status-actions">
              <el-button size="small" @click="copy(status.api_base_url)">复制地址</el-button>
              <el-button size="small" @click="loadModels" :loading="loadingModels">读取模型</el-button>
            </div>
          </div>

          <div class="status-item">
            <div class="status-title">默认模型</div>
            <el-tag type="info">{{ status.model || 'hermes-agent' }}</el-tag>
            <div class="status-link">API Key：{{ status.api_key_set ? '已配置' : '未配置' }}</div>
            <div class="status-actions">
              <el-button size="small" type="primary" @click="fillSample">填入测试问题</el-button>
            </div>
          </div>
        </div>
        <div v-else class="empty-text">正在读取 Hermes 状态...</div>
      </el-card>

      <el-card class="models-card">
        <template #header>
          <div class="card-header">
            <span>模型列表</span>
            <span class="small-text">来自 Hermes `/v1/models`</span>
          </div>
        </template>
        <div v-if="models.length" class="models-wrap">
          <el-tag v-for="item in models" :key="item.id || item" class="model-tag">
            {{ item.id || item }}
          </el-tag>
        </div>
        <div v-else class="empty-text">点击“读取模型”后显示</div>
      </el-card>

      <el-card class="chat-card">
        <template #header>
          <div class="card-header">
            <span>联通性测试</span>
            <span class="small-text">通过 devtools 后端直连 Hermes Gateway，不走容器代理</span>
          </div>
        </template>

        <el-form label-position="top">
          <el-form-item label="系统提示词（可选）">
            <el-input
              v-model="systemPrompt"
              type="textarea"
              :rows="2"
              placeholder="例如：你是一个简洁的运维助手。"
            />
          </el-form-item>
          <el-form-item label="测试消息">
            <el-input
              v-model="message"
              type="textarea"
              :rows="4"
              placeholder="输入一条测试消息，例如：只回复 ok"
            />
          </el-form-item>
          <div class="status-actions">
            <el-button type="primary" :loading="sending" @click="sendMessage">发送测试</el-button>
            <el-button @click="fillSample">示例</el-button>
            <el-button @click="clearResult">清空结果</el-button>
          </div>
        </el-form>

        <div v-if="answer" class="result-block">
          <div class="result-title">回复结果</div>
          <pre>{{ answer }}</pre>
        </div>

        <div v-if="usage" class="usage-block">
          <span>Usage:</span>
          <code>{{ JSON.stringify(usage) }}</code>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'

const API_BASE = '/api/hermes'
const SESSION_KEY = 'hermes_admin_password'

const authenticated = ref(false)
const passwordInput = ref('')
const loggingIn = ref(false)

const loadingStatus = ref(false)
const loadingModels = ref(false)
const sending = ref(false)

const status = ref(null)
const models = ref([])
const systemPrompt = ref('')
const message = ref('')
const answer = ref('')
const usage = ref(null)

function adminPassword() {
  return sessionStorage.getItem(SESSION_KEY) || ''
}

async function request(path, options = {}) {
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  url.searchParams.set('admin_password', adminPassword())
  const response = await fetch(url, options)
  const data = await response.json().catch(() => ({}))
  if (!response.ok || data.error) {
    throw new Error(data.error || `请求失败 (${response.status})`)
  }
  return data
}

async function login() {
  if (!passwordInput.value.trim()) {
    ElMessage.warning('请输入密码')
    return
  }
  loggingIn.value = true
  try {
    const response = await fetch(`${API_BASE}/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: passwordInput.value })
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok || data.error) {
      throw new Error(data.error || '密码错误')
    }
    sessionStorage.setItem(SESSION_KEY, passwordInput.value)
    authenticated.value = true
    await refreshAll()
    ElMessage.success('Hermes 模块已接入')
  } catch (error) {
    ElMessage.error(error.message || '登录失败')
  } finally {
    loggingIn.value = false
  }
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  authenticated.value = false
  passwordInput.value = ''
  status.value = null
  models.value = []
  clearResult()
}

async function loadStatus() {
  loadingStatus.value = true
  try {
    status.value = await request('/status')
  } catch (error) {
    ElMessage.error(error.message || '加载状态失败')
  } finally {
    loadingStatus.value = false
  }
}

async function loadModels() {
  loadingModels.value = true
  try {
    const data = await request('/models')
    models.value = Array.isArray(data.data) ? data.data : []
  } catch (error) {
    ElMessage.error(error.message || '加载模型失败')
  } finally {
    loadingModels.value = false
  }
}

async function sendMessage() {
  if (!message.value.trim()) {
    ElMessage.warning('请输入测试消息')
    return
  }
  sending.value = true
  try {
    const response = await fetch(`${API_BASE}/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        password: adminPassword(),
        message: message.value,
        system: systemPrompt.value
      })
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok || data.error) {
      throw new Error(data.error || '发送失败')
    }
    answer.value = data.answer || ''
    usage.value = data.usage || null
    ElMessage.success('Hermes 返回成功')
  } catch (error) {
    ElMessage.error(error.message || 'Hermes 请求失败')
  } finally {
    sending.value = false
  }
}

async function refreshAll() {
  await loadStatus()
}

function fillSample() {
  if (!message.value.trim()) {
    message.value = '只回复 ok'
  }
}

function clearResult() {
  answer.value = ''
  usage.value = null
}

function openLink(url) {
  if (!url) return
  window.open(url, '_blank', 'noopener')
}

async function copy(text) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  const saved = adminPassword()
  if (!saved) return
  authenticated.value = true
  refreshAll()
})
</script>

<style scoped>
.hermes-tool {
  max-width: 1180px;
  margin: 0 auto;
  padding: 20px;
}

.login-card {
  max-width: 460px;
  margin: 48px auto;
}

.login-copy {
  margin-bottom: 18px;
}

.hero-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.hero-subtitle {
  color: var(--text-secondary);
  line-height: 1.6;
}

.main-content {
  display: grid;
  gap: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.header-actions,
.status-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.small-text {
  color: var(--text-tertiary);
  font-size: 12px;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}

.status-item {
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  background: var(--bg-overlay);
  display: grid;
  gap: 10px;
}

.status-title,
.result-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.status-link {
  word-break: break-all;
  color: var(--text-secondary);
  font-size: 13px;
}

.models-wrap {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.model-tag {
  margin: 0;
}

.result-block {
  margin-top: 16px;
  border-radius: 12px;
  background: #0f172a;
  color: #e2e8f0;
  padding: 16px;
}

.result-block pre {
  white-space: pre-wrap;
  word-break: break-word;
  margin: 10px 0 0;
  font-family: Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1.7;
}

.usage-block {
  margin-top: 12px;
  color: var(--text-secondary);
  font-size: 13px;
}

.usage-block code {
  display: block;
  margin-top: 6px;
  white-space: pre-wrap;
  word-break: break-word;
}

.empty-text {
  color: var(--text-tertiary);
  font-size: 13px;
}

@media (max-width: 640px) {
  .hermes-tool {
    padding: 14px;
  }

  .card-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
