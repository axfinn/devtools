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
            <el-input
              v-model="subscribeURLsText"
              type="textarea"
              :rows="4"
              placeholder="输入 Clash 订阅链接，每行一个 URL"
            />
          </el-tab-pane>
          <el-tab-pane label="粘贴 YAML" name="yaml">
            <el-input v-model="yamlContent" type="textarea" :rows="6"
              placeholder="粘贴 Clash YAML 配置内容（proxies: 段）" />
          </el-tab-pane>
        </el-tabs>
        <div class="route-settings">
          <el-select v-model="routeMode" class="route-field">
            <el-option label="AI 优先分流（默认）" value="ai_priority" />
            <el-option label="智能分流" value="smart" />
            <el-option label="全局代理" value="global" />
          </el-select>
          <el-select
            v-model="defaultNodeName"
            clearable
            filterable
            class="route-field"
            placeholder="手动选择默认线路，留空则自动选择"
          >
            <el-option
              v-for="node in nodeOptions"
              :key="node.name"
              :label="node.name"
              :value="node.name"
            />
          </el-select>
          <el-select
            v-model="aiNodeName"
            clearable
            filterable
            class="route-field"
            placeholder="手动选择 AI 专线，优先级高于 AI 正则"
          >
            <el-option
              v-for="node in nodeOptions"
              :key="node.name"
              :label="node.name"
              :value="node.name"
            />
          </el-select>
          <el-input
            v-model="defaultNodeRegex"
            class="route-field"
            placeholder="默认线路正则，可留空；例如 (?i)日本|韩国|IEPL"
          />
          <el-input
            v-model="aiNodeRegex"
            class="route-field"
            placeholder="AI 专线正则；例如 (?i)(chatgpt|gpt|claude|anthropic|gemini|ai)"
          />
        </div>
        <div class="current-route-row">
          <el-tag type="success" effect="dark">默认当前线路：{{ currentDefaultNode || '未确定' }}</el-tag>
          <el-tag type="warning">AI 当前线路：{{ currentAINode || '未确定' }}</el-tag>
        </div>
        <div v-if="subscriptionRefresh.enabled" class="proxy-hint">
          托管订阅：{{ subscriptionRefresh.last_refresh_status || '未执行' }}
          <span v-if="subscriptionRefresh.last_refresh_source"> · 刷新通道：{{ subscriptionRefresh.last_refresh_source }}</span>
          <span v-if="subscriptionRefresh.resolved_site_url"> · 入口站点：{{ subscriptionRefresh.resolved_site_url }}</span>
          <span v-if="subscriptionRefresh.last_subscribe_url"> · 当前订阅：{{ subscriptionRefresh.last_subscribe_url }}</span>
          <span v-if="subscriptionRefresh.last_refresh_node_hint"> · 接管线路：{{ subscriptionRefresh.last_refresh_node_hint }}</span>
          <span v-if="subscriptionRefresh.last_refresh_at"> · 上次执行：{{ subscriptionRefresh.last_refresh_at }}</span>
        </div>
        <div class="proxy-hint">
          默认线路支持手动指定或按当前线路/正则自动选择；AI 线路支持手动指定或按 AI 正则自动选择。运行时会尽量保持同一条线路，仅在不可用时才切换。
        </div>
        <div v-if="reachabilityCheck.last_check_status" class="proxy-hint">
          节点验证：{{ reachabilityCheck.last_check_status }}
          <span v-if="reachabilityCheck.last_check_at"> · 上次执行：{{ reachabilityCheck.last_check_at }}</span>
        </div>
        <div class="action-row">
          <el-button type="primary" @click="loadConfig" :loading="loadingConfig">解析节点</el-button>
          <el-button @click="copyExportConfig">复制配置</el-button>
          <el-button @click="speedTest" :loading="testingSpeed" :disabled="nodes.length === 0">全部测速</el-button>
          <el-button @click="checkNodes" :loading="checkingNodes" :disabled="nodes.length === 0">
            验证可用（curl google）
          </el-button>
          <el-button
            v-if="subscriptionRefresh.enabled"
            type="info"
            @click="triggerSubscriptionRefresh"
            :loading="refreshingSubscription"
          >
            手动刷新托管订阅
          </el-button>
          <el-button type="success" @click="startProxy" :loading="startingProxy" :disabled="!selectedNode">
            启动代理
          </el-button>
          <el-button type="warning" @click="autoStart" :loading="autoStarting" :disabled="nodes.length === 0">
            自动选线路
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
          <el-table-column label="真实可用" width="90" v-if="checkedOnce">
            <template #default="{ row }">
              <el-tag v-if="checkResults[row.name] === true" type="success" size="small">✓ 可用</el-tag>
              <el-tag v-else-if="checkResults[row.name] === false" type="danger" size="small">✗ 不可用</el-tag>
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
          <div class="proxy-hint" style="margin-top:6px">
            下载后将二进制和 config.txt 放同一目录。
            Mac/Linux 需先 <code>chmod +x</code> 再双击，或终端运行。
            Windows 直接双击 .exe。
          </div>
          <div class="proxy-addr-row" style="margin-top:8px">
            <el-input :value="goClientConfigText" readonly size="small" type="textarea" :rows="3" class="proxy-addr-input" />
            <el-button @click="copy(goClientConfigText)">复制 config.txt</el-button>
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
        </el-alert>

        <!-- 方案三：NPS 隧道（需配置 NPS，流量经 NPS 服务器转发） -->
        <el-alert type="warning" :closable="false">
          <template #title>
            <span>方案三：NPS 隧道</span>
            <el-tag v-if="npcRunning" type="success" size="small" style="margin-left:8px">npc 运行中</el-tag>
            <el-tag v-else type="info" size="small" style="margin-left:8px">npc 未运行</el-tag>
          </template>
          <div v-if="npsTunnelAddr" style="margin-top:8px">
            <div class="proxy-addr-row">
              <span class="proxy-addr-label">代理地址</span>
              <el-input :value="npsTunnelAddr" readonly size="small" class="proxy-addr-input">
                <template #append>
                  <el-button @click="copy(npsTunnelAddr)">复制</el-button>
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
          </div>
          <div v-else class="proxy-hint" style="margin-top:8px">
            未配置 NPS（需在 config.yaml 中设置 nps.vkey 和 proxy.tunnel_port）
          </div>
          <div class="proxy-hint" style="margin-top:6px">
            <b>使用方式：</b>浏览器/系统代理配置 HTTP 代理，地址填上方代理地址，用户名随意，密码填上方密码。<br>
            <b>原理：</b>npc 客户端在服务器内运行，将代理端口通过 NPS 隧道暴露到公网。流量经 NPS 服务器 TCP 转发，不经过 nginx，适合 nginx 无法访问的场景。<br>
            <b>安全：</b>无认证请求返回伪装页面，不暴露代理特征。需在 NPS 管理页面提前创建端口映射（可用 NPS 工具页面一键映射）。
          </div>
          <div v-if="npsTunnelAddr" style="margin-top:10px;display:flex;gap:8px;flex-wrap:wrap">
            <el-button type="primary" size="small" @click="downloadExtension(npsTunnelAddr)">
              下载 Chrome 插件（自动配置代理）
            </el-button>
            <el-button type="warning" size="small" :loading="creatingTunnel" @click="createNPSTunnel">
              一键创建端口映射
            </el-button>
          </div>
        </el-alert>
      </el-card>

      <!-- 终端快速接入 -->
      <el-card v-if="npsTunnelAddr" class="config-card">
        <template #header>
          <div class="card-header">
            <span>终端快速接入</span>
            <span class="text-gray" style="font-size:12px">HTTP 代理 {{ npsTunnelAddr }}，用户名 proxy，密码见上方</span>
          </div>
        </template>

        <el-tabs v-model="quickAccessTab">
          <!-- macOS / Linux -->
          <el-tab-pane label="macOS / Linux" name="unix">
            <div class="qa-section">
              <div class="qa-label">临时生效（当前终端会话）</div>
              <el-input :value="unixExportCmd" readonly size="small" type="textarea" :rows="3" class="qa-code" />
              <el-button size="small" @click="copy(unixExportCmd)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">永久生效（写入 ~/.zshrc 或 ~/.bashrc）</div>
              <el-input :value="unixPermanentCmd" readonly size="small" type="textarea" :rows="5" class="qa-code" />
              <el-button size="small" @click="copy(unixPermanentCmd)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">macOS 系统代理（网络设置）</div>
              <div class="qa-hint">系统设置 → 网络 → 代理 → HTTP 代理<br>主机：<b>{{ npsTunnelAddr.split(':')[0] }}</b>　端口：<b>{{ npsTunnelAddr.split(':')[1] }}</b>　用户名：<b>proxy</b>　密码：<b>{{ adminPassword() }}</b></div>
            </div>
          </el-tab-pane>

          <!-- Windows -->
          <el-tab-pane label="Windows" name="win">
            <div class="qa-section">
              <div class="qa-label">CMD（临时）</div>
              <el-input :value="winCmdExport" readonly size="small" type="textarea" :rows="3" class="qa-code" />
              <el-button size="small" @click="copy(winCmdExport)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">PowerShell（临时）</div>
              <el-input :value="winPsExport" readonly size="small" type="textarea" :rows="3" class="qa-code" />
              <el-button size="small" @click="copy(winPsExport)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">系统代理（图形界面）</div>
              <div class="qa-hint">设置 → 网络和 Internet → 代理 → 手动代理设置<br>地址：<b>{{ npsTunnelAddr.split(':')[0] }}</b>　端口：<b>{{ npsTunnelAddr.split(':')[1] }}</b><br>（Windows 系统代理不支持认证，建议用 Chrome 插件或 Clash）</div>
            </div>
          </el-tab-pane>

          <!-- Git / npm / pip -->
          <el-tab-pane label="Git / npm / pip" name="tools">
            <div class="qa-section">
              <div class="qa-label">Git</div>
              <el-input :value="gitCmd" readonly size="small" type="textarea" :rows="2" class="qa-code" />
              <el-button size="small" @click="copy(gitCmd)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">npm</div>
              <el-input :value="npmCmd" readonly size="small" type="textarea" :rows="2" class="qa-code" />
              <el-button size="small" @click="copy(npmCmd)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">pip</div>
              <el-input :value="pipCmd" readonly size="small" type="textarea" :rows="2" class="qa-code" />
              <el-button size="small" @click="copy(pipCmd)" style="margin-top:6px">复制</el-button>
            </div>
            <div class="qa-section">
              <div class="qa-label">curl（单次）</div>
              <el-input :value="curlCmd" readonly size="small" type="textarea" :rows="2" class="qa-code" />
              <el-button size="small" @click="copy(curlCmd)" style="margin-top:6px">复制</el-button>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-card>

      <!-- 自定义代理域名 -->
      <el-card class="config-card">
        <template #header>
          <div class="card-header">
            <span>自定义代理域名</span>
            <span class="text-gray" style="font-size:12px">加入后强制走代理，不走国内直连</span>
          </div>
        </template>
        <div style="display:flex;gap:8px;margin-bottom:12px">
          <el-input v-model="newCustomDomain" placeholder="example.com" clearable
            @keyup.enter="addCustomDomain" style="max-width:320px" />
          <el-button type="primary" @click="addCustomDomain" :loading="addingDomain">添加</el-button>
        </div>
        <el-table :data="customDomains" size="small" empty-text="暂无自定义域名">
          <el-table-column prop="domain" label="域名" />
          <el-table-column label="" width="80">
            <template #default="{ row }">
              <el-button size="small" type="danger" @click="removeCustomDomain(row.domain)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
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
const subscribeURLsText = ref('')
const yamlContent = ref('')
const routeMode = ref('ai_priority')
const defaultNodeName = ref('')
const defaultNodeRegex = ref('')
const aiNodeName = ref('')
const aiNodeRegex = ref('(?i)(chatgpt|gpt|openai|claude|anthropic|gemini|copilot|ai)')
const loadingConfig = ref(false)
const testingSpeed = ref(false)
const testedOnce = ref(false)
const checkingNodes = ref(false)
const checkedOnce = ref(false)
const checkResults = ref({}) // name -> true/false
const refreshingSubscription = ref(false)

const nodes = ref([])
const selectedNode = ref(null)

const startingProxy = ref(false)
const stoppingProxy = ref(false)
const autoStarting = ref(false)
const creatingTunnel = ref(false)
const proxyRunning = ref(false)
const httpPort = ref(0)
const proxyURL = ref('')
const activeNode = ref('')
const currentDefaultNode = ref('')
const currentAINode = ref('')
const npcRunning = ref(false)
const npcTunnelPort = ref('')
const npcServerAddr = ref('')
const subscriptionRefresh = ref({
  enabled: false,
  resolved_site_url: '',
  last_subscribe_url: '',
  last_refresh_status: '',
  last_refresh_at: '',
  last_refresh_source: '',
  last_refresh_node_hint: ''
})
const reachabilityCheck = ref({
  running: false,
  last_check_status: '',
  last_check_at: '',
  results: []
})

const customDomains = ref([])
const newCustomDomain = ref('')
const addingDomain = ref(false)

// 多 Tab 浏览器
let tabIdCounter = 1
function makeTab() {
  return { id: tabIdCounter++, inputURL: '', url: '', src: '', title: '', loading: false, canBack: false }
}
const browserTabs = ref([makeTab()])
const activeTabId = ref(browserTabs.value[0].id)
const activeTab = computed(() => browserTabs.value.find(t => t.id === activeTabId.value))
const nodeOptions = computed(() => nodes.value.filter(node => node && node.name))

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

function parseSourceURLs() {
  return subscribeURLsText.value
    .split(/\r?\n/)
    .map(item => item.trim())
    .filter(Boolean)
}

function applyStatus(data) {
  if (data.nodes && data.nodes.length > 0) {
    nodes.value = data.nodes
    testedOnce.value = data.nodes.some(n => n.latency >= 0)
  }
  if (Array.isArray(data.source_urls) && data.source_urls.length > 0) {
    subscribeURLsText.value = data.source_urls.join('\n')
    configTab.value = 'url'
  } else if (data.source_url) {
    subscribeURLsText.value = data.source_url
    configTab.value = 'url'
  }
  if (typeof data.yaml_content === 'string' && data.yaml_content && !subscribeURLsText.value) {
    yamlContent.value = data.yaml_content
    configTab.value = 'yaml'
  }
  routeMode.value = data.routing_mode || 'ai_priority'
  defaultNodeName.value = data.default_node_name || ''
  defaultNodeRegex.value = data.default_node_regex || ''
  aiNodeName.value = data.ai_node_name || ''
  aiNodeRegex.value = data.ai_node_regex || aiNodeRegex.value
  currentDefaultNode.value = data.default_node || data.node || ''
  currentAINode.value = data.effective_ai_node || data.ai_node_name || ''
  subscriptionRefresh.value = {
    enabled: !!data.subscription_refresh?.enabled,
    resolved_site_url: data.subscription_refresh?.resolved_site_url || '',
    last_subscribe_url: data.subscription_refresh?.last_subscribe_url || '',
    last_refresh_status: data.subscription_refresh?.last_refresh_status || '',
    last_refresh_at: data.subscription_refresh?.last_refresh_at || '',
    last_refresh_source: data.subscription_refresh?.last_refresh_source || '',
    last_refresh_node_hint: data.subscription_refresh?.last_refresh_node_hint || ''
  }
  reachabilityCheck.value = {
    running: !!data.check_reachability?.running,
    last_check_status: data.check_reachability?.last_check_status || '',
    last_check_at: data.check_reachability?.last_check_at || '',
    results: Array.isArray(data.check_reachability?.results) ? data.check_reachability.results : []
  }
  if (reachabilityCheck.value.results.length > 0) {
    const map = {}
    for (const item of reachabilityCheck.value.results) {
      map[item.name] = item.reachable
    }
    checkResults.value = map
    checkedOnce.value = true
  }
  if (data.running) {
    proxyRunning.value = true
    httpPort.value = data.http_port
    proxyURL.value = data.proxy_url
    activeNode.value = data.node
  }
  npcRunning.value = !!data.npc_running
  if (data.npc_tunnel_port) npcTunnelPort.value = data.npc_tunnel_port
  if (data.npc_server_addr) npcServerAddr.value = data.npc_server_addr
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
        fetchCustomDomains()
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
        fetchCustomDomains()
      }
    })
    .catch(() => {})
})

function fetchCustomDomains() {
  fetch(`/api/proxy/custom-domains?admin_password=${encodeURIComponent(adminPassword())}`)
    .then(r => r.json())
    .then(data => { customDomains.value = (data.domains || []).map(d => ({ domain: d })) })
    .catch(() => {})
}

async function addCustomDomain() {
  const d = newCustomDomain.value.trim()
  if (!d) return
  addingDomain.value = true
  try {
    const r = await fetch(`/api/proxy/custom-domains?admin_password=${encodeURIComponent(adminPassword())}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain: d })
    })
    const data = await r.json()
    if (data.error) { ElMessage.error(data.error); return }
    newCustomDomain.value = ''
    fetchCustomDomains()
    ElMessage.success('已添加')
  } finally {
    addingDomain.value = false
  }
}

async function removeCustomDomain(domain) {
  await fetch(`/api/proxy/custom-domains?admin_password=${encodeURIComponent(adminPassword())}`, {
    method: 'DELETE', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ domain })
  })
  fetchCustomDomains()
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  isAdmin.value = false
  passwordInput.value = ''
}

async function loadConfig() {
  loadingConfig.value = true
  try {
    const body = {
      admin_password: adminPassword(),
      routing_mode: routeMode.value,
      default_node_name: defaultNodeName.value,
      default_node_regex: defaultNodeRegex.value.trim(),
      ai_node_name: aiNodeName.value,
      ai_node_regex: aiNodeRegex.value.trim()
    }
    if (configTab.value === 'url') body.source_urls = parseSourceURLs()
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
      currentDefaultNode.value = data.default_node || ''
      currentAINode.value = data.effective_ai_node || data.ai_node_name || ''
      ElMessage.success(`解析成功，共 ${data.count} 个节点`)
    }
  } catch (e) { ElMessage.error('请求失败') }
  finally { loadingConfig.value = false }
}

function copyExportConfig() {
  const text = JSON.stringify({
    source_urls: parseSourceURLs(),
    yaml_content: configTab.value === 'yaml' ? yamlContent.value : '',
    routing_mode: routeMode.value,
    default_node_name: defaultNodeName.value,
    default_node_regex: defaultNodeRegex.value.trim(),
    ai_node_name: aiNodeName.value,
    ai_node_regex: aiNodeRegex.value.trim()
  }, null, 2)
  copy(text)
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

async function checkNodes() {
  checkingNodes.value = true
  try {
    const r = await fetch('/api/proxy/check', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) { ElMessage.error(data.error); return }
    if (data.check_reachability) {
      reachabilityCheck.value = {
        running: !!data.check_reachability.running,
        last_check_status: data.check_reachability.last_check_status || '',
        last_check_at: data.check_reachability.last_check_at || '',
        results: Array.isArray(data.check_reachability.results) ? data.check_reachability.results : []
      }
    }
    if (data.started) {
      ElMessage.success('节点验证已启动，正在后台执行')
      for (let i = 0; i < 30; i++) {
        await new Promise(resolve => setTimeout(resolve, 2000))
        const statusResp = await fetch(`/api/proxy/status?admin_password=${encodeURIComponent(adminPassword())}`)
        const statusData = await statusResp.json()
        if (statusData.error) {
          continue
        }
        applyStatus(statusData)
        if (!statusData.check_reachability?.running) {
          const results = Array.isArray(statusData.check_reachability?.results) ? statusData.check_reachability.results : []
          const ok = results.filter(item => item.reachable).length
          ElMessage.success(statusData.check_reachability?.last_check_status || `验证完成，${ok}/${results.length} 个节点真实可用（curl google）`)
          return
        }
      }
      ElMessage.warning('节点验证仍在执行，请稍后查看状态')
      return
    }
  } catch (e) { ElMessage.error('验证失败') }
  finally { checkingNodes.value = false }
}

async function autoStart() {
  autoStarting.value = true
  try {
    const r = await fetch('/api/proxy/auto-start', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) { ElMessage.error(data.error); return }
    nodes.value = data.results
    testedOnce.value = true
    proxyRunning.value = true
    httpPort.value = data.http_port
    proxyURL.value = data.proxy_url
    activeNode.value = data.node
    ElMessage.success(`已自动选择节点「${data.node}」（延迟 ${data.latency}ms）`)
  } catch (e) { ElMessage.error('自动选线路失败') }
  finally { autoStarting.value = false }
}

async function createNPSTunnel() {
  creatingTunnel.value = true
  try {
    const r = await fetch('/api/proxy/nps-tunnel', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) ElMessage.error(data.error)
    else ElMessage.success(`端口映射已创建，公网端口 ${data.port}`)
  } catch (e) { ElMessage.error('创建失败') }
  finally { creatingTunnel.value = false }
}

async function triggerSubscriptionRefresh() {
  refreshingSubscription.value = true
  try {
    const r = await fetch('/api/proxy/subscription-refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ admin_password: adminPassword() })
    })
    const data = await r.json()
    if (data.error) {
      ElMessage.error(data.error)
      return
    }
    if (data.subscription_refresh) {
      subscriptionRefresh.value = {
        enabled: !!data.subscription_refresh.enabled,
        resolved_site_url: data.subscription_refresh.resolved_site_url || '',
        last_subscribe_url: data.subscription_refresh.last_subscribe_url || '',
        last_refresh_status: data.subscription_refresh.last_refresh_status || '',
        last_refresh_at: data.subscription_refresh.last_refresh_at || '',
        last_refresh_source: data.subscription_refresh.last_refresh_source || '',
        last_refresh_node_hint: data.subscription_refresh.last_refresh_node_hint || ''
      }
    }
    if (data.node) {
      activeNode.value = data.node
    }
    currentDefaultNode.value = data.default_node || currentDefaultNode.value
    currentAINode.value = data.effective_ai_node || currentAINode.value
    if (data.started) {
      ElMessage.success('托管订阅刷新已启动，正在后台执行')
      for (let i = 0; i < 30; i++) {
        await new Promise(resolve => setTimeout(resolve, 2000))
        const statusResp = await fetch(`/api/proxy/status?admin_password=${encodeURIComponent(adminPassword())}`)
        const statusData = await statusResp.json()
        if (statusData.error) {
          continue
        }
        applyStatus(statusData)
        const status = statusData.subscription_refresh?.last_refresh_status || ''
        if (!status.includes('正在后台执行')) {
          if (status.includes('未执行') || status.includes('失败')) {
            ElMessage.error(status)
          } else {
            ElMessage.success(status || '托管订阅刷新完成')
          }
          return
        }
      }
      ElMessage.warning('后台刷新仍在执行，请稍后查看状态')
      return
    }
    ElMessage.success(subscriptionRefresh.value.last_refresh_status || '托管订阅刷新完成')
  } catch (e) { ElMessage.error('手动刷新失败') }
  finally { refreshingSubscription.value = false }
}

function selectNode(row) { selectedNode.value = row }
function rowClass({ row }) { return selectedNode.value?.name === row.name ? 'selected-row' : '' }
function latencyType(ms) { return ms < 100 ? 'success' : ms < 300 ? 'warning' : 'danger' }
function copy(text) { navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制')) }

function downloadExtension(customHost) {
  const host = customHost || window.location.host
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

// NPS 隧道代理地址：npc 将代理端口暴露到 NPS 服务器的 tunnel_port
const npsTunnelAddr = computed(() => {
  if (!npcServerAddr.value || !npcTunnelPort.value) return ''
  const host = npcServerAddr.value.split(':')[0]
  return `${host}:${npcTunnelPort.value}`
})

// 终端快速接入
const quickAccessTab = ref('unix')

const unixExportCmd = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  const url = `http://proxy:${pass}@${addr}`
  return `export http_proxy="${url}"\nexport https_proxy="${url}"\nexport no_proxy="localhost,127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"`
})

const unixPermanentCmd = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  const url = `http://proxy:${pass}@${addr}`
  return `# 追加到 ~/.zshrc 或 ~/.bashrc\nexport http_proxy="${url}"\nexport https_proxy="${url}"\nexport no_proxy="localhost,127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"\n# 生效：source ~/.zshrc`
})

const winCmdExport = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  const url = `http://proxy:${pass}@${addr}`
  return `set http_proxy=${url}\nset https_proxy=${url}`
})

const winPsExport = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  const url = `http://proxy:${pass}@${addr}`
  return `$env:http_proxy="${url}"\n$env:https_proxy="${url}"`
})

const gitCmd = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  return `git config --global http.proxy "http://proxy:${pass}@${addr}"\ngit config --global https.proxy "http://proxy:${pass}@${addr}"`
})

const npmCmd = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  return `npm config set proxy "http://proxy:${pass}@${addr}"\nnpm config set https-proxy "http://proxy:${pass}@${addr}"`
})

const pipCmd = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  return `pip install <package> --proxy "http://proxy:${pass}@${addr}"`
})

const curlCmd = computed(() => {
  if (!npsTunnelAddr.value) return ''
  const addr = npsTunnelAddr.value
  const pass = adminPassword()
  return `curl -x "http://proxy:${pass}@${addr}" https://www.google.com`
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

const goClientConfigText = computed(() => {
  const host = window.location.host
  const proto = window.location.protocol === 'https:' ? 'https' : 'http'
  const pass = adminPassword()
  return `server=${proto}://${host}\npassword=${pass}\nlisten=127.0.0.1:1080\n`
})

function downloadClient(os, arch) {
  const pass = adminPassword()
  // 先下载二进制
  const url = `/api/proxy/client/download?os=${os}&arch=${arch}&admin_password=${encodeURIComponent(pass)}`
  window.open(url, '_blank')
  // 再下载 config.txt，让用户双击直接用
  const blob = new Blob([goClientConfigText.value], { type: 'text/plain' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = 'config.txt'
  a.click()
}
</script>

<style scoped>
.proxy-tool { max-width: 1100px; margin: 0 auto; padding: 16px; }
.login-card { max-width: 400px; margin: 60px auto; }
.main-content { display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.proxy-info-inline { display: flex; align-items: center; gap: 8px; }
.route-settings { display: grid; gap: 8px; margin-top: 12px; }
.current-route-row { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 10px; }
.route-field { width: 100%; }
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
.qa-section {
  margin-bottom: 16px;
}
.qa-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  margin-bottom: 6px;
}
.qa-code :deep(textarea) {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.6;
}
.qa-hint {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.8;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
}
</style>
