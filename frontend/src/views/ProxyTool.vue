<template>
  <div class="proxy-tool">
    <!-- 管理员登录 -->
    <div v-if="!isAdmin" class="login-card">
      <el-card>
        <template #header><span>科学上网 — 管理员登录</span></template>
        <el-form @submit.prevent="login">
          <el-form-item label="管理员密码">
            <el-input v-model="passwordInput" type="password" show-password
              placeholder="请输入管理员密码" @keyup.enter="login" />
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
            <el-input v-model="subscribeURL" placeholder="输入 Clash 订阅链接" clearable />
          </el-tab-pane>
          <el-tab-pane label="粘贴 YAML" name="yaml">
            <el-input v-model="yamlContent" type="textarea" :rows="6"
              placeholder="粘贴 Clash YAML 配置内容（proxies: 段）" />
          </el-tab-pane>
        </el-tabs>
        <div class="action-row">
          <el-button type="primary" @click="loadConfig" :loading="loadingConfig">解析节点</el-button>
          <el-button @click="speedTest" :loading="testingSpeed" :disabled="nodes.length === 0">全部测速</el-button>
          <el-button type="success" @click="startProxy" :loading="startingProxy" :disabled="!selectedNode">
            启动代理
          </el-button>
          <el-button type="danger" @click="stopProxy" :loading="stoppingProxy" v-if="proxyRunning">
            停止代理
          </el-button>
        </div>
      </el-card>

      <!-- 节点列表 -->
      <el-card v-if="nodes.length > 0" class="nodes-card">
        <template #header><span>节点列表（{{ nodes.length }} 个）</span></template>
        <el-table :data="sortedNodes" @row-click="selectNode" highlight-current-row
          :row-class-name="rowClass" size="small" max-height="300">
          <el-table-column label="节点名" prop="name" min-width="200" show-overflow-tooltip />
          <el-table-column label="类型" prop="type" width="80" />
          <el-table-column label="服务器" prop="server" min-width="140" show-overflow-tooltip />
          <el-table-column label="端口" prop="port" width="70" />
          <el-table-column label="延迟" width="90">
            <template #default="{ row }">
              <el-tag v-if="row.latency >= 0" :type="latencyType(row.latency)" size="small">
                {{ row.latency }} ms
              </el-tag>
              <el-tag v-else-if="testedOnce" type="danger" size="small">超时</el-tag>
              <span v-else class="text-gray">—</span>
            </template>
          </el-table-column>
          <el-table-column label="" width="70">
            <template #default="{ row }">
              <el-button size="small" :type="selectedNode?.name === row.name ? 'primary' : ''"
                @click.stop="selectedNode = row">选</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- 代理信息（启动后显示） -->
      <el-card v-if="proxyRunning" class="config-card">
        <template #header><span>代理已启动 — {{ activeNode }}</span></template>

        <!-- 方案一：Go 客户端（推荐，无需第三方工具） -->
        <el-alert type="success" :closable="false" style="margin-bottom:12px">
          <template #title><span>方案一：Go 客户端（推荐）</span></template>
          <div style="margin-top:6px;font-size:13px;">
            下载客户端，本地运行后浏览器配 SOCKS5 代理 <b>127.0.0.1:1080</b>
          </div>
          <div style="margin-top:8px;display:flex;gap:8px;flex-wrap:wrap">
            <el-button size="small" type="primary" @click="downloadClient('darwin','arm64')">Mac (Apple Silicon)</el-button>
            <el-button size="small" type="primary" @click="downloadClient('darwin','amd64')">Mac (Intel)</el-button>
            <el-button size="small" type="primary" @click="downloadClient('linux','amd64')">Linux</el-button>
            <el-button size="small" type="primary" @click="downloadClient('windows','amd64')">Windows</el-button>
          </div>
          <div class="proxy-addr-row" style="margin-top:8px">
            <el-input :value="goClientCmd" readonly size="small" class="proxy-addr-input">
              <template #append>
                <el-button @click="copy(goClientCmd)">复制</el-button>
              </template>
            </el-input>
          </div>
          <div class="proxy-hint">运行后浏览器/系统代理配置 SOCKS5 127.0.0.1:1080。流量走 WebSocket 隧道，支持 nginx 反代，防探测。</div>
        </el-alert>

        <!-- 方案二：wstunnel（需要第三方工具） -->
        <el-alert type="info" :closable="false" style="margin-bottom:12px">
          <template #title><span>方案二：wstunnel</span></template>
          <div style="margin-top:6px;font-size:13px;">
            本地运行 wstunnel，浏览器配 SOCKS5 代理 <b>127.0.0.1:1080</b>
          </div>
          <div class="proxy-addr-row" style="margin-top:8px">
            <el-input :value="wstunnelCmd" readonly size="small" class="proxy-addr-input">
              <template #append>
                <el-button @click="copy(wstunnelCmd)">复制</el-button>
              </template>
            </el-input>
          </div>
          <div class="proxy-hint">
            下载 wstunnel：<a href="https://github.com/erebe/wstunnel/releases" target="_blank">github.com/erebe/wstunnel/releases</a>
          </div>
          <el-button type="default" size="small" style="margin-top:10px" @click="downloadExtension">
            下载 Chrome 插件（自动配置，无需 wstunnel）
          </el-button>
        </el-alert>

        <!-- 方案三：直接 HTTP 代理（需要直连服务器，不经过 nginx） -->
        <el-alert type="warning" :closable="false">
          <template #title><span>方案三：直接 HTTP 代理（仅限直连服务器，不经过 nginx）</span></template>
          <div class="proxy-addr-row" style="margin-top:8px">
            <span class="proxy-addr-label">代理地址</span>
            <el-input :value="externalProxyURL" readonly size="small" class="proxy-addr-input">
              <template #append>
                <el-button @click="copy(externalProxyURL)">复制</el-button>
              </template>
            </el-input>
          </div>
          <div class="proxy-addr-row">
            <span class="proxy-addr-label">认证密码</span>
            <el-input :value="adminPassword()" readonly size="small" type="password" show-password class="proxy-addr-input">
              <template #append>
                <el-button @click="copy(adminPassword())">复制</el-button>
              </template>
            </el-input>
          </div>
          <div class="proxy-hint">用户名随意填，密码填上方密码。无认证请求会返回假页面，防止 GFW 探测。nginx 反代下此方式不可用。</div>
        </el-alert>
      </el-card>

      <!-- 内嵌浏览器（登录后始终显示） -->
      <el-card class="browser-card">
        <template #header>
          <div class="card-header">
            <span>内嵌浏览器</span>
            <el-tag v-if="proxyRunning" type="success" size="small">代理中 — {{ activeNode }}</el-tag>
            <el-tag v-else type="info" size="small">未启动代理（直连）</el-tag>
          </div>
        </template>

        <!-- Tab 栏 -->
        <div class="tab-bar">
          <div
            v-for="(tab, i) in browserTabs"
            :key="tab.id"
            class="tab-item"
            :class="{ active: activeTabId === tab.id }"
            @click="activeTabId = tab.id"
          >
            <span class="tab-title" :title="tab.url || '新标签页'">
              {{ tab.title || tab.url || '新标签页' }}
            </span>
            <el-icon class="tab-close" @click.stop="closeTab(i)"><Close /></el-icon>
          </div>
          <el-button class="tab-new" size="small" circle @click="newTab">
            <el-icon><Plus /></el-icon>
          </el-button>
        </div>

        <!-- 地址栏 -->
        <div v-if="activeTab" class="browser-bar">
          <el-button size="small" @click="goBack">
            <el-icon><ArrowLeft /></el-icon>
          </el-button>
          <el-button size="small" @click="reload(activeTab)">
            <el-icon><Refresh /></el-icon>
          </el-button>
          <el-input
            v-model="activeTab.inputURL"
            placeholder="输入网址，如 https://www.google.com"
            @keyup.enter="navigate(activeTab)"
            clearable
            class="url-input"
          />
          <el-button type="primary" size="small" @click="navigate(activeTab)" :loading="activeTab.loading">
            访问
          </el-button>
        </div>

        <!-- iframe 区域 -->
        <div class="frames-wrap">
          <template v-for="tab in browserTabs" :key="tab.id">
            <div class="frame-container" :class="{ hidden: activeTabId !== tab.id }">
              <iframe
                v-if="tab.src"
                :src="tab.src"
                class="browser-frame"
                @load="onFrameLoad(tab, $event)"
              />
              <div v-else class="browser-placeholder text-gray">
                输入网址后点击"访问"
              </div>
            </div>
          </template>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Close, Plus, ArrowLeft, Refresh } from '@element-plus/icons-vue'

const SESSION_KEY = 'proxy_admin_password'

const passwordInput = ref('')
const isAdmin = ref(false)
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

// 多 Tab 浏览器
let tabIdCounter = 1
function makeTab() {
  return { id: tabIdCounter++, inputURL: '', url: '', src: '', title: '', loading: false, canBack: false }
}
const browserTabs = ref([makeTab()])
const activeTabId = ref(browserTabs.value[0].id)
const activeTab = computed(() => browserTabs.value.find(t => t.id === activeTabId.value))

function newTab() {
  const t = makeTab()
  browserTabs.value.push(t)
  activeTabId.value = t.id
}

function closeTab(i) {
  if (browserTabs.value.length === 1) {
    browserTabs.value[0] = makeTab()
    return
  }
  const wasActive = browserTabs.value[i].id === activeTabId.value
  browserTabs.value.splice(i, 1)
  if (wasActive) {
    activeTabId.value = browserTabs.value[Math.min(i, browserTabs.value.length - 1)].id
  }
}

function navigate(tab) {
  let u = tab.inputURL.trim()
  if (!u) return
  if (!u.startsWith('http://') && !u.startsWith('https://')) u = 'https://' + u
  tab.url = u
  tab.title = new URL(u).hostname
  tab.loading = true
  const p = adminPassword()
  tab.src = `/api/proxy/fetch?admin_password=${encodeURIComponent(p)}&url=${encodeURIComponent(u)}`
}

function reload(tab) {
  if (!tab.url) return
  const src = tab.src
  tab.src = ''
  nextTick(() => { tab.src = src })
}

function goBack() {
  // iframe history.back() via postMessage or just clear
  const frame = document.querySelector(`.frame-container:not(.hidden) iframe`)
  if (frame) {
    try { frame.contentWindow.history.back() } catch (e) {}
  }
}

function onFrameLoad(tab, e) {
  tab.loading = false
  try {
    const title = e.target.contentDocument?.title
    if (title) tab.title = title
  } catch (_) {}
}

function adminPassword() {
  return sessionStorage.getItem(SESSION_KEY) || ''
}

function applyStatus(data) {
  if (data.nodes && data.nodes.length > 0) {
    nodes.value = data.nodes
    testedOnce.value = data.nodes.some(n => n.latency >= 0)
  }
  if (data.source_url) {
    subscribeURL.value = data.source_url
    configTab.value = 'url'
  }
  if (data.running) {
    proxyRunning.value = true
    httpPort.value = data.http_port
    proxyURL.value = data.proxy_url
    activeNode.value = data.node
  }
}

function login() {
  if (!passwordInput.value) { ElMessage.warning('请输入密码'); return }
  loginLoading.value = true
  fetch(`/api/proxy/status?admin_password=${encodeURIComponent(passwordInput.value)}`)
    .then(r => r.json())
    .then(data => {
      if (data.error) {
        ElMessage.error('密码错误')
      } else {
        sessionStorage.setItem(SESSION_KEY, passwordInput.value)
        isAdmin.value = true
        applyStatus(data)
      }
    })
    .catch(() => ElMessage.error('请求失败'))
    .finally(() => { loginLoading.value = false })
}

// 页面加载时，如果 sessionStorage 有密码，自动恢复状态
onMounted(() => {
  const saved = sessionStorage.getItem(SESSION_KEY)
  if (!saved) return
  fetch(`/api/proxy/status?admin_password=${encodeURIComponent(saved)}`)
    .then(r => r.json())
    .then(data => {
      if (data.error) {
        sessionStorage.removeItem(SESSION_KEY)
      } else {
        isAdmin.value = true
        applyStatus(data)
      }
    })
    .catch(() => {})
})

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  isAdmin.value = false
  passwordInput.value = ''
}

async function loadConfig() {
  loadingConfig.value = true
  try {
    const body = { admin_password: adminPassword() }
    if (configTab.value === 'url') body.source_url = subscribeURL.value
    else body.yaml_content = yamlContent.value
    const r = await fetch('/api/proxy/config', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await r.json()
    if (data.error) ElMessage.error(data.error)
    else {
      nodes.value = data.nodes
      selectedNode.value = null
      testedOnce.value = false
      ElMessage.success(`解析成功，共 ${data.count} 个节点`)
    }
  } catch (e) { ElMessage.error('请求失败') }
  finally { loadingConfig.value = false }
}

async function speedTest() {
  testingSpeed.value = true
  try {
    const r = await fetch('/api/proxy/speedtest', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) ElMessage.error(data.error)
    else {
      nodes.value = data.results
      testedOnce.value = true
      const ok = data.results.filter(n => n.latency >= 0).length
      ElMessage.success(`测速完成，${ok}/${data.results.length} 个节点可用`)
    }
  } catch (e) { ElMessage.error('测速失败') }
  finally { testingSpeed.value = false }
}

async function startProxy() {
  if (!selectedNode.value) return
  startingProxy.value = true
  try {
    const r = await fetch('/api/proxy/start', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword(), node_name: selectedNode.value.name })
    })
    const data = await r.json()
    if (data.error) ElMessage.error(data.error)
    else {
      proxyRunning.value = true
      httpPort.value = data.http_port
      proxyURL.value = data.proxy_url
      activeNode.value = data.node
      ElMessage.success(`代理已启动，端口 ${data.http_port}`)
    }
  } catch (e) { ElMessage.error('启动失败') }
  finally { startingProxy.value = false }
}

async function stopProxy() {
  stoppingProxy.value = true
  try {
    const r = await fetch('/api/proxy/stop', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) ElMessage.error(data.error)
    else {
      proxyRunning.value = false
      httpPort.value = 0; proxyURL.value = ''; activeNode.value = ''
      ElMessage.success('代理已停止')
    }
  } catch (e) { ElMessage.error('停止失败') }
  finally { stoppingProxy.value = false }
}

function selectNode(row) { selectedNode.value = row }
function rowClass({ row }) { return selectedNode.value?.name === row.name ? 'selected-row' : '' }
function latencyType(ms) { return ms < 100 ? 'success' : ms < 300 ? 'warning' : 'danger' }
function copy(text) { navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制')) }

function downloadExtension() {
  const host = window.location.host
  const p = adminPassword()
  const url = `/api/proxy/extension?admin_password=${encodeURIComponent(p)}&host=${encodeURIComponent(host)}`
  const a = document.createElement('a')
  a.href = url
  a.download = 'devtools-proxy.zip'
  a.click()
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

// 对外代理地址：用当前页面的 host（即 DevTools 服务器地址）
// 浏览器/系统代理配置：http://yourserver:PORT，密码用 Proxy-Authorization
const externalProxyURL = computed(() => {
  return `${window.location.protocol}//${window.location.host}`
})

// wstunnel 命令：通过 WebSocket 隧道，支持 nginx 反代
const wstunnelCmd = computed(() => {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const host = window.location.host
  const pass = adminPassword()
  return `wstunnel client -L 'socks5://127.0.0.1:1080' ${proto}://${host}/api/proxy/ws-tunnel?p=${encodeURIComponent(pass)}`
})

// Go 客户端命令
const goClientCmd = computed(() => {
  const proto = window.location.protocol === 'https:' ? 'https' : 'http'
  const host = window.location.host
  const pass = adminPassword()
  return `./proxy-client -server ${proto}://${host} -password ${pass}`
})

function downloadClient(os, arch) {
  const pass = adminPassword()
  const url = `/api/proxy/client/download?os=${os}&arch=${arch}&admin_password=${encodeURIComponent(pass)}`
  window.open(url, '_blank')
}
</script>

<style scoped>
.proxy-tool { max-width: 1100px; margin: 0 auto; padding: 16px; }
.login-card { max-width: 400px; margin: 60px auto; }
.main-content { display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.proxy-info-inline { display: flex; align-items: center; gap: 8px; }
.action-row { margin-top: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.nodes-card :deep(.selected-row) { background-color: var(--el-color-primary-light-9); }

/* Tab 栏 */
.tab-bar {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 0 0;
  border-bottom: 1px solid var(--el-border-color);
  margin-bottom: 8px;
  overflow-x: auto;
  flex-wrap: nowrap;
}
.tab-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 4px 4px 0 0;
  border: 1px solid transparent;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
  max-width: 160px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.tab-item.active {
  background: var(--el-bg-color);
  border-color: var(--el-border-color);
  border-bottom-color: var(--el-bg-color);
  color: var(--el-text-color-primary);
  margin-bottom: -1px;
}
.tab-title { overflow: hidden; text-overflow: ellipsis; flex: 1; }
.tab-close { font-size: 12px; opacity: 0.5; flex-shrink: 0; }
.tab-close:hover { opacity: 1; }
.tab-new { margin-left: 4px; flex-shrink: 0; }

/* 地址栏 */
.browser-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.url-input { flex: 1; }

/* iframe */
.frames-wrap { position: relative; }
.frame-container { display: block; }
.frame-container.hidden { display: none; }
.browser-frame {
  width: 100%;
  height: 600px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  display: block;
  background: #fff;
}
.browser-placeholder {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--el-border-color);
  border-radius: 4px;
}
.text-gray { color: var(--el-text-color-secondary); }
.proxy-alert { margin-bottom: 12px; }
.proxy-addr-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}
.proxy-addr-label {
  width: 60px;
  flex-shrink: 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.proxy-addr-input { flex: 1; }
.proxy-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
