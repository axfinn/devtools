<template>
  <div class="proxy-tool">
    <!-- 管理员登录 -->
    <div v-if="!isAdmin" class="login-card">
      <el-card>
        <template #header>
          <span>科学上网 — 管理员登录</span>
        </template>
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
          <el-button type="primary" @click="login" :loading="loginLoading">登录</el-button>
        </el-form>
      </el-card>
    </div>

    <!-- 主界面 -->
    <div v-else class="main-content">
      <!-- 配置区 -->
      <el-card class="config-card">
        <template #header>
          <div class="card-header">
            <span>节点配置</span>
            <el-button size="small" @click="logout">退出</el-button>
          </div>
        </template>

        <el-tabs v-model="configTab">
          <el-tab-pane label="订阅 URL" name="url">
            <el-input
              v-model="subscribeURL"
              placeholder="输入 Clash 订阅链接"
              clearable
            />
          </el-tab-pane>
          <el-tab-pane label="粘贴 YAML" name="yaml">
            <el-input
              v-model="yamlContent"
              type="textarea"
              :rows="8"
              placeholder="粘贴 Clash YAML 配置内容"
            />
          </el-tab-pane>
        </el-tabs>

        <div class="action-row">
          <el-button type="primary" @click="loadConfig" :loading="loadingConfig">
            解析节点
          </el-button>
          <el-button @click="speedTest" :loading="testingSpeed" :disabled="nodes.length === 0">
            全部测速
          </el-button>
          <el-button
            type="success"
            @click="startProxy"
            :loading="startingProxy"
            :disabled="!selectedNode"
          >
            启动代理
          </el-button>
          <el-button
            type="danger"
            @click="stopProxy"
            :loading="stoppingProxy"
            v-if="proxyRunning"
          >
            停止代理
          </el-button>
        </div>
      </el-card>

      <!-- 节点列表 -->
      <el-card v-if="nodes.length > 0" class="nodes-card">
        <template #header>
          <span>节点列表（{{ nodes.length }} 个）</span>
        </template>
        <el-table
          :data="sortedNodes"
          @row-click="selectNode"
          highlight-current-row
          :row-class-name="rowClass"
          size="small"
        >
          <el-table-column label="节点名" prop="name" min-width="200" show-overflow-tooltip />
          <el-table-column label="类型" prop="type" width="80" />
          <el-table-column label="服务器" prop="server" min-width="140" show-overflow-tooltip />
          <el-table-column label="端口" prop="port" width="70" />
          <el-table-column label="延迟" width="90">
            <template #default="{ row }">
              <el-tag
                v-if="row.latency >= 0"
                :type="latencyType(row.latency)"
                size="small"
              >{{ row.latency }} ms</el-tag>
              <el-tag v-else-if="row.latency === -1 && testedOnce" type="danger" size="small">超时</el-tag>
              <span v-else class="text-gray">—</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80">
            <template #default="{ row }">
              <el-button
                size="small"
                :type="selectedNode?.name === row.name ? 'primary' : ''"
                @click.stop="selectedNode = row"
              >选择</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- 代理信息 -->
      <el-card v-if="proxyRunning" class="proxy-info-card">
        <template #header>
          <span>代理信息 — {{ activeNode }}</span>
        </template>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="HTTP 代理">
            <el-input :value="proxyURL" readonly>
              <template #append>
                <el-button @click="copy(proxyURL)">复制</el-button>
              </template>
            </el-input>
          </el-descriptions-item>
          <el-descriptions-item label="代理端口">{{ httpPort }}</el-descriptions-item>
        </el-descriptions>

        <!-- 内嵌浏览器 -->
        <div class="browser-section">
          <div class="browser-bar">
            <el-input
              v-model="browserURL"
              placeholder="输入要访问的网址，如 https://www.google.com"
              @keyup.enter="fetchPage"
              clearable
            >
              <template #prepend>
                <el-icon><ChromeFilled /></el-icon>
              </template>
              <template #append>
                <el-button @click="fetchPage" :loading="fetchingPage">访问</el-button>
              </template>
            </el-input>
          </div>
          <div v-if="pageHTML" class="browser-frame-wrap">
            <iframe
              ref="browserFrame"
              :srcdoc="pageHTML"
              sandbox="allow-same-origin allow-scripts"
              class="browser-frame"
            />
          </div>
          <div v-else-if="fetchingPage" class="browser-placeholder">
            <el-icon class="is-loading"><Loading /></el-icon> 加载中...
          </div>
          <div v-else class="browser-placeholder text-gray">
            输入网址后点击"访问"，页面将在此处显示
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { ChromeFilled, Loading } from '@element-plus/icons-vue'

const SESSION_KEY = 'proxy_admin_password'

const passwordInput = ref('')
const isAdmin = ref(!!sessionStorage.getItem(SESSION_KEY))
const loginLoading = ref(false)

const configTab = ref('url')
const subscribeURL = ref('')
const yamlContent = ref('')
const loadingConfig = ref(false)
const testingSpeed = ref(false)
const testedOnce = ref(false)

const nodes = ref([])
const selectedNode = ref(null)

const startingProxy = ref(false)
const stoppingProxy = ref(false)
const proxyRunning = ref(false)
const httpPort = ref(0)
const proxyURL = ref('')
const activeNode = ref('')

const browserURL = ref('')
const fetchingPage = ref(false)
const pageHTML = ref('')

function adminPassword() {
  return sessionStorage.getItem(SESSION_KEY) || ''
}

function login() {
  if (!passwordInput.value) {
    ElMessage.warning('请输入密码')
    return
  }
  loginLoading.value = true
  // 用 status 接口验证密码
  fetch(`/api/proxy/status?admin_password=${encodeURIComponent(passwordInput.value)}`)
    .then(r => r.json())
    .then(data => {
      if (data.error) {
        ElMessage.error('密码错误')
      } else {
        sessionStorage.setItem(SESSION_KEY, passwordInput.value)
        isAdmin.value = true
        if (data.running) {
          proxyRunning.value = true
          httpPort.value = data.http_port
          proxyURL.value = data.proxy_url
          activeNode.value = data.node
        }
      }
    })
    .catch(() => ElMessage.error('请求失败'))
    .finally(() => { loginLoading.value = false })
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  isAdmin.value = false
  passwordInput.value = ''
}

async function loadConfig() {
  loadingConfig.value = true
  try {
    const body = { admin_password: adminPassword() }
    if (configTab.value === 'url') {
      body.source_url = subscribeURL.value
    } else {
      body.yaml_content = yamlContent.value
    }
    const r = await fetch('/api/proxy/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await r.json()
    if (data.error) {
      ElMessage.error(data.error)
    } else {
      nodes.value = data.nodes
      selectedNode.value = null
      testedOnce.value = false
      ElMessage.success(`解析成功，共 ${data.count} 个节点`)
    }
  } catch (e) {
    ElMessage.error('请求失败')
  } finally {
    loadingConfig.value = false
  }
}

async function speedTest() {
  testingSpeed.value = true
  try {
    const r = await fetch('/api/proxy/speedtest', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) {
      ElMessage.error(data.error)
    } else {
      nodes.value = data.results
      testedOnce.value = true
      const ok = data.results.filter(n => n.latency >= 0).length
      ElMessage.success(`测速完成，${ok}/${data.results.length} 个节点可用`)
    }
  } catch (e) {
    ElMessage.error('测速失败')
  } finally {
    testingSpeed.value = false
  }
}

async function startProxy() {
  if (!selectedNode.value) return
  startingProxy.value = true
  try {
    const r = await fetch('/api/proxy/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPassword(),
        node_name: selectedNode.value.name
      })
    })
    const data = await r.json()
    if (data.error) {
      ElMessage.error(data.error)
    } else {
      proxyRunning.value = true
      httpPort.value = data.http_port
      proxyURL.value = data.proxy_url
      activeNode.value = data.node
      ElMessage.success(`代理已启动，端口 ${data.http_port}`)
    }
  } catch (e) {
    ElMessage.error('启动失败')
  } finally {
    startingProxy.value = false
  }
}

async function stopProxy() {
  stoppingProxy.value = true
  try {
    const r = await fetch('/api/proxy/stop', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) {
      ElMessage.error(data.error)
    } else {
      proxyRunning.value = false
      httpPort.value = 0
      proxyURL.value = ''
      activeNode.value = ''
      pageHTML.value = ''
      ElMessage.success('代理已停止')
    }
  } catch (e) {
    ElMessage.error('停止失败')
  } finally {
    stoppingProxy.value = false
  }
}

async function fetchPage() {
  if (!browserURL.value) return
  fetchingPage.value = true
  pageHTML.value = ''
  try {
    const params = new URLSearchParams({
      url: browserURL.value,
      admin_password: adminPassword()
    })
    const r = await fetch(`/api/proxy/fetch?${params}`)
    if (r.headers.get('content-type')?.includes('text/html')) {
      pageHTML.value = await r.text()
    } else {
      const data = await r.json()
      ElMessage.error(data.error || '请求失败')
    }
  } catch (e) {
    ElMessage.error('访问失败: ' + e.message)
  } finally {
    fetchingPage.value = false
  }
}

function selectNode(row) {
  selectedNode.value = row
}

function rowClass({ row }) {
  return selectedNode.value?.name === row.name ? 'selected-row' : ''
}

function latencyType(ms) {
  if (ms < 100) return 'success'
  if (ms < 300) return 'warning'
  return 'danger'
}

function copy(text) {
  navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制'))
}

const sortedNodes = computed(() => {
  if (!testedOnce.value) return nodes.value
  return [...nodes.value].sort((a, b) => {
    if (a.latency < 0 && b.latency < 0) return 0
    if (a.latency < 0) return 1
    if (b.latency < 0) return -1
    return a.latency - b.latency
  })
})
</script>

<style scoped>
.proxy-tool {
  max-width: 900px;
  margin: 0 auto;
  padding: 16px;
}
.login-card {
  max-width: 400px;
  margin: 60px auto;
}
.main-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.action-row {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.nodes-card :deep(.selected-row) {
  background-color: var(--el-color-primary-light-9);
}
.browser-section {
  margin-top: 16px;
}
.browser-bar {
  margin-bottom: 12px;
}
.browser-frame-wrap {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;
}
.browser-frame {
  width: 100%;
  height: 500px;
  border: none;
  display: block;
}
.browser-placeholder {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  border: 1px dashed var(--el-border-color);
  border-radius: 4px;
  gap: 8px;
}
.text-gray {
  color: var(--el-text-color-secondary);
}
</style>
