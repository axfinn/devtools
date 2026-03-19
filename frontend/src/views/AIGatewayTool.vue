<template>
  <div class="tool-container ai-gateway-page">
    <div class="hero">
      <div>
        <h2>AI Gateway</h2>
        <p>统一发放 API Key，对外开放文本与图片/视频模型能力。</p>
      </div>
      <div class="hero-actions">
        <el-input
          v-model="superAdminPassword"
          type="password"
          show-password
          placeholder="超级管理员密码"
          @keyup.enter="init"
        />
        <el-button type="primary" @click="init">进入管理</el-button>
        <el-button @click="loadDocs">查看文档</el-button>
        <el-button type="success" @click="loadAnthropicDocs">Anthropic 接入</el-button>
      </div>
    </div>

    <el-row :gutter="18">
      <el-col :lg="10" :md="24">
        <el-card>
          <template #header>
            <span>签发 API Key</span>
          </template>
          <el-form label-position="top">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="如 marketing-service" />
            </el-form-item>
            <el-form-item label="允许模型（每行一个，留空表示全部）">
              <el-input v-model="modelsText" type="textarea" :rows="6" placeholder="deepseek-chat&#10;qwen-image-2.0-pro" />
            </el-form-item>
            <el-form-item label="允许能力">
              <el-checkbox-group v-model="form.allowed_scopes">
                <el-checkbox value="chat">chat</el-checkbox>
                <el-checkbox value="media">media</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item label="有效期（天）">
              <el-input-number v-model="form.expires_days" :min="1" :max="3650" class="w-full" />
            </el-form-item>
            <el-form-item label="每小时限制">
              <el-input-number v-model="form.rate_limit_per_hour" :min="1" :max="100000" class="w-full" />
            </el-form-item>
            <el-form-item label="预算上限">
              <el-input-number v-model="form.budget_limit" :min="0" :precision="4" class="w-full" />
            </el-form-item>
            <el-form-item label="告警阈值">
              <el-input-number v-model="form.alert_threshold" :min="0.1" :max="1" :step="0.05" :precision="2" class="w-full" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.notes" type="textarea" :rows="3" />
            </el-form-item>
            <el-button type="primary" :loading="creating" @click="createKey">生成 Key</el-button>
          </el-form>

          <el-alert
            v-if="createdPlainKey"
            title="新 Key 已生成，明文只显示一次"
            type="success"
            :closable="false"
            style="margin-top: 16px;"
          >
            <template #default>
              <div class="plain-key-box">
                <code>{{ createdPlainKey }}</code>
                <el-button size="small" @click="copyPlainKey">复制</el-button>
              </div>
            </template>
          </el-alert>
        </el-card>
      </el-col>

      <el-col :lg="14" :md="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>API Key 列表</span>
              <el-button text @click="loadKeys">刷新</el-button>
            </div>
          </template>

          <el-table :data="keys" v-loading="loadingKeys" stripe>
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="key_prefix" label="前缀" min-width="140" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="total_requests" label="总请求" width="90" />
            <el-table-column prop="total_tokens" label="总 Tokens" width="110" />
            <el-table-column label="累计费用" width="120">
              <template #default="{ row }">{{ formatCost(row.total_cost, row.billing_currency) }}</template>
            </el-table-column>
            <el-table-column label="过期时间" width="180">
              <template #default="{ row }">{{ formatTime(row.expires_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="180">
              <template #default="{ row }">
                <el-button text @click="viewKey(row)">详情</el-button>
                <el-button v-if="row.status === 'active'" text type="danger" @click="revokeKey(row)">吊销</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="18">
      <el-col :lg="10" :md="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>预算告警</span>
              <el-button text @click="loadAlerts">刷新</el-button>
            </div>
          </template>
          <el-table :data="alerts" size="small" max-height="340">
            <el-table-column prop="name" label="Key" min-width="120" />
            <el-table-column label="进度" min-width="200">
              <template #default="{ row }">
                <el-progress :percentage="Math.min(row.usage_ratio * 100, 100)" :status="row.level === 'critical' ? 'exception' : 'warning'" />
              </template>
            </el-table-column>
            <el-table-column label="费用" width="120">
              <template #default="{ row }">{{ formatCost(row.total_cost, row.currency) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :lg="14" :md="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>账单报表</span>
              <div class="card-header">
                <el-select v-model="reportGroupBy" style="width: 110px" @change="loadReports">
                  <el-option label="按天" value="day" />
                  <el-option label="按月" value="month" />
                </el-select>
                <el-input-number v-model="reportDays" :min="1" :max="3650" @change="loadReports" />
              </div>
            </div>
          </template>
          <el-table :data="reports" size="small" max-height="340">
            <el-table-column prop="period" label="周期" width="110" />
            <el-table-column prop="api_key_id" label="Key ID" width="90" />
            <el-table-column prop="provider" label="Provider" width="90" />
            <el-table-column prop="model" label="模型" min-width="180" />
            <el-table-column prop="request_count" label="请求数" width="90" />
            <el-table-column prop="total_tokens" label="Tokens" width="100" />
            <el-table-column label="费用" width="120">
              <template #default="{ row }">{{ formatCost(row.total_cost, row.currency) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 模型快速测试 -->
    <el-card class="test-card">
      <template #header>
        <div class="card-header">
          <span>模型快速测试</span>
          <div class="card-header">
            <el-input
              v-model="testPrompt"
              placeholder="自定义测试问题（可选）"
              style="width: 280px"
            />
            <el-button @click="loadCatalog">刷新模型</el-button>
            <el-button type="primary" :loading="testingAll" @click="testAllModels">全部测试</el-button>
          </div>
        </div>
      </template>

      <el-table :data="catalogChat" v-loading="loadingCatalog" stripe size="small">
        <el-table-column prop="model" label="模型" min-width="200" />
        <el-table-column prop="brand" label="品牌" width="80">
          <template #default="{ row }">{{ row.brand || row.provider }}</template>
        </el-table-column>
        <el-table-column prop="description" label="能力" min-width="200" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag
              :type="testResults[row.model]?.status === 'ok' ? 'success' : testResults[row.model]?.status === 'err' ? 'danger' : testResults[row.model]?.status === 'running' ? 'warning' : 'info'"
              size="small"
            >
              {{ testResults[row.model]?.status === 'ok' ? '✓ 成功' : testResults[row.model]?.status === 'err' ? '✗ 失败' : testResults[row.model]?.status === 'running' ? '测试中' : '待测' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }">
            <span v-if="testResults[row.model]?.latency">{{ testResults[row.model].latency }}ms</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="Tokens" width="80">
          <template #default="{ row }">{{ testResults[row.model]?.tokens ?? '-' }}</template>
        </el-table-column>
        <el-table-column label="回复" min-width="260">
          <template #default="{ row }">
            <span v-if="testResults[row.model]?.status === 'err'" class="test-err">{{ testResults[row.model].error }}</span>
            <span v-else class="test-reply">{{ testResults[row.model]?.reply || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button
              text
              type="primary"
              size="small"
              :loading="testResults[row.model]?.status === 'running'"
              @click="testModel(row.model)"
            >测试</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="docs-card">
      <template #header>
        <div class="card-header">
          <span>接入文档</span>
          <el-button text @click="loadDocs">刷新文档</el-button>
        </div>
      </template>

      <div class="docs-grid">
        <div class="docs-panel">
          <h4>接入方式</h4>
          <p class="docs-text">
            业务方使用平台签发的 API Key 直接调用统一网关。文本能力走
            `/api/ai-gateway/v1/chat/completions`，图片和视频走
            `/api/ai-gateway/v1/media/generations`，所有请求都会进入统一记账、限流和审计。
          </p>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="鉴权 Header">Authorization: Bearer &lt;API_KEY&gt;</el-descriptions-item>
            <el-descriptions-item label="文本接口">POST /api/ai-gateway/v1/chat/completions</el-descriptions-item>
            <el-descriptions-item label="媒体接口">POST /api/ai-gateway/v1/media/generations</el-descriptions-item>
            <el-descriptions-item label="任务查询">GET /api/ai-gateway/v1/media/tasks/:id</el-descriptions-item>
            <el-descriptions-item label="接口总文档">GET /api/ai-gateway/docs</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="docs-panel">
          <h4>返回与计费</h4>
          <ul class="docs-list">
            <li>文本接口会回传上游结果，并附带 `usage_summary`，包含输入、输出、总 Tokens 和费用。</li>
            <li>媒体接口返回统一任务对象，支持轮询查看状态、结果地址和失败原因。</li>
            <li>每个 API Key 都会累计请求数、Token、费用，并支持预算上限、告警阈值、日报和月报。</li>
          </ul>
        </div>
      </div>

      <el-tabs class="docs-tabs">
        <el-tab-pane label="快速开始">
          <div class="docs-panel-stack">
            <h4>接入步骤</h4>
            <ol class="docs-steps">
              <li>在当前页面生成 API Key，并确认允许的模型、作用域、限流和预算。</li>
              <li>业务服务端保存 API Key，不要暴露到浏览器前端。</li>
              <li>调用聊天接口或媒体接口，Header 带上 `Authorization: Bearer &lt;API_KEY&gt;`。</li>
              <li>读取响应中的 `usage_summary`、任务状态和错误信息，接入你自己的日志系统。</li>
              <li>在本页查看请求明细、累计 Token、费用、预算告警和日报月报。</li>
            </ol>
          </div>
        </el-tab-pane>
        <el-tab-pane label="字段说明">
          <el-table :data="gatewayFieldDocs" size="small" stripe>
            <el-table-column prop="field" label="字段" width="220" />
            <el-table-column prop="required" label="必填" width="80" />
            <el-table-column prop="description" label="说明" min-width="260" />
            <el-table-column prop="example" label="示例" min-width="220" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="cURL">
          <pre class="doc-json">{{ gatewayCurlExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="JavaScript">
          <pre class="doc-json">{{ gatewayJsExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="Python">
          <pre class="doc-json">{{ gatewayPythonExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="媒体示例">
          <pre class="doc-json">{{ gatewayMediaExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="响应示例">
          <pre class="doc-json">{{ gatewayResponseExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="错误码">
          <el-table :data="gatewayErrorDocs" size="small" stripe>
            <el-table-column prop="code" label="状态码" width="100" />
            <el-table-column prop="scene" label="场景" width="180" />
            <el-table-column prop="meaning" label="含义" min-width="220" />
            <el-table-column prop="fix" label="处理建议" min-width="260" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="常见问题">
          <div class="faq-list">
            <div v-for="item in gatewayFaqs" :key="item.q" class="faq-item">
              <h4>{{ item.q }}</h4>
              <p>{{ item.a }}</p>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <div v-if="docs" class="docs-api-list">
        <h4>接口清单</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in docs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-card>

    <el-dialog v-model="detailVisible" width="920px" title="Key 详情">
      <div v-if="currentKey">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ currentKey.name }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ currentKey.status }}</el-descriptions-item>
          <el-descriptions-item label="前缀">{{ currentKey.key_prefix }}</el-descriptions-item>
          <el-descriptions-item label="总请求">{{ currentKey.total_requests }}</el-descriptions-item>
          <el-descriptions-item label="输入 Tokens">{{ currentKey.total_input_tokens }}</el-descriptions-item>
          <el-descriptions-item label="输出 Tokens">{{ currentKey.total_output_tokens }}</el-descriptions-item>
          <el-descriptions-item label="总 Tokens">{{ currentKey.total_tokens }}</el-descriptions-item>
          <el-descriptions-item label="累计费用">{{ formatCost(currentKey.total_cost, currentKey.billing_currency) }}</el-descriptions-item>
          <el-descriptions-item label="允许模型">{{ currentKey.allowed_models }}</el-descriptions-item>
          <el-descriptions-item label="允许能力">{{ currentKey.allowed_scopes }}</el-descriptions-item>
        </el-descriptions>

        <div class="logs-section">
          <h4>最近请求明细</h4>
          <el-table :data="currentLogs" max-height="420">
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column prop="provider" label="Provider" width="100" />
            <el-table-column prop="model" label="模型" min-width="180" />
            <el-table-column prop="request_type" label="类型" width="90" />
            <el-table-column prop="status_code" label="状态码" width="90" />
            <el-table-column prop="total_tokens" label="Tokens" width="90" />
            <el-table-column label="费用" width="100">
              <template #default="{ row }">{{ formatCost(row.estimated_cost, row.currency) }}</template>
            </el-table-column>
            <el-table-column prop="latency_ms" label="耗时(ms)" width="100" />
            <el-table-column prop="error_message" label="错误信息" min-width="220" />
          </el-table>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="docsVisible" width="920px" title="AI Gateway 接入文档">
      <div v-if="docs" class="docs-content">
        <el-alert :title="docs.summary" type="info" :closable="false" />

        <!-- 可用模型列表 -->
        <div v-if="docs.billing && docs.billing.pricing" style="margin-top: 16px;">
          <h4>可用模型</h4>
          <el-table :data="docs.billing.pricing" size="small" max-height="300">
            <el-table-column prop="Model" label="模型" />
            <el-table-column prop="Provider" label="提供商" width="100">
              <template #default="{ row }">
                <el-tag size="small">{{ row.Provider }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="InputPer1KTokens" label="输入/千token" width="100" />
            <el-table-column prop="OutputPer1KTokens" label="输出/千token" width="100" />
            <el-table-column prop="Currency" label="货币" width="80" />
          </el-table>
        </div>

        <el-divider />

        <h4>API 路由</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in docs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 16px;">调用示例</h4>
        <pre class="doc-json">{{ prettyJSON(docs.examples) }}</pre>
      </div>
    </el-dialog>

    <!-- Anthropic 协议接入文档对话框 -->
    <el-dialog v-model="anthropicDocsVisible" width="960px" title="Anthropic 协议接入文档">
      <div v-if="anthropicDocs" class="docs-content">
        <el-alert :title="anthropicDocs.summary" type="info" :closable="false" />

        <h4 style="margin-top: 16px;">支持的提供商</h4>
        <el-table :data="anthropicDocs.providers" size="small" stripe>
          <el-table-column prop="name" label="提供商" width="120">
            <template #default="{ row }">
              <el-tag size="small" type="success">{{ row.name }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="base_url" label="DevTools 端点" min-width="200">
            <template #default="{ row }">
              <code style="font-size: 12px;">{{ row.base_url }}/v1/messages</code>
            </template>
          </el-table-column>
          <el-table-column prop="upstream" label="上游地址" min-width="220">
            <template #default="{ row }">
              <code style="font-size: 11px; color: #909399;">{{ row.upstream }}</code>
            </template>
          </el-table-column>
          <el-table-column label="支持模型" min-width="280">
            <template #default="{ row }">
              <el-tag v-for="m in row.models" :key="m" size="small" style="margin-right: 4px; margin-bottom: 2px;">{{ m }}</el-tag>
            </template>
          </el-table-column>
        </el-table>

        <el-divider />

        <h4>API 路由</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in anthropicDocs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 16px;">请求示例</h4>
        <el-tabs>
          <el-tab-pane label="MiniMax">
            <pre class="doc-json">{{ JSON.stringify(anthropicDocs.examples.minimax.request, null, 2) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="DashScope">
            <pre class="doc-json">{{ JSON.stringify(anthropicDocs.examples.dashscope.request, null, 2) }}</pre>
          </el-tab-pane>
        </el-tabs>

        <h4 style="margin-top: 16px;">SDK 调用示例</h4>
        <el-tabs>
          <el-tab-pane label="Python SDK">
            <pre class="doc-code">{{ anthropicDocs.examples.python_sdk.code }}</pre>
          </el-tab-pane>
          <el-tab-pane label="JavaScript SDK">
            <pre class="doc-code">{{ anthropicDocs.examples.javascript_sdk.code }}</pre>
          </el-tab-pane>
          <el-tab-pane label="cURL">
            <pre class="doc-code">{{ anthropicDocs.examples.curl.code }}</pre>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const API_BASE = '/api/ai-gateway'
const PASSWORD_KEY = 'ai_gateway_super_admin_password'

const superAdminPassword = ref(sessionStorage.getItem(PASSWORD_KEY) || '')
const creating = ref(false)
const loadingKeys = ref(false)
const keys = ref([])
const currentKey = ref(null)
const currentLogs = ref([])
const reports = ref([])
const alerts = ref([])
const detailVisible = ref(false)
const docsVisible = ref(false)
const docs = ref(null)
const anthropicDocsVisible = ref(false)
const anthropicDocs = ref(null)
const createdPlainKey = ref('')
const modelsText = ref('')

const form = ref({
  name: '',
  allowed_scopes: ['chat', 'media'],
  expires_days: 90,
  rate_limit_per_hour: 1000,
  budget_limit: 0,
  alert_threshold: 0.8,
  notes: ''
})
const reportGroupBy = ref('day')
const reportDays = ref(30)
const origin = typeof window !== 'undefined' ? window.location.origin : ''

// 模型快速测试
const testPrompt = ref('')
const testResults = ref({})    // { [model]: { status, reply, error, latency, tokens } }
const loadingCatalog = ref(false)
const catalogChat = ref([])
const testingAll = ref(false)

const loadCatalog = async () => {
  loadingCatalog.value = true
  try {
    const res = await fetch(`${API_BASE}/catalog`)
    const data = await res.json()
    catalogChat.value = (data.models || []).filter(m => m.type === 'chat')
  } finally {
    loadingCatalog.value = false
  }
}

const testModel = async (model) => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  testResults.value = { ...testResults.value, [model]: { status: 'running' } }
  try {
    const res = await fetch(`${API_BASE}/admin/test-model`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Super-Admin-Password': superAdminPassword.value },
      body: JSON.stringify({ super_admin_password: superAdminPassword.value, model, prompt: testPrompt.value || undefined })
    })
    const data = await res.json()
    if (data.status === 'ok') {
      testResults.value = { ...testResults.value, [model]: { status: 'ok', reply: data.reply, latency: data.latency, tokens: data.tokens } }
    } else {
      testResults.value = { ...testResults.value, [model]: { status: 'err', error: data.error || '未知错误', latency: data.latency } }
    }
  } catch (err) {
    testResults.value = { ...testResults.value, [model]: { status: 'err', error: err.message } }
  }
}

const testAllModels = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  testingAll.value = true
  try {
    await Promise.allSettled(catalogChat.value.map(m => testModel(m.model)))
  } finally {
    testingAll.value = false
  }
}

const authHeaders = () => ({
  'Content-Type': 'application/json',
  'X-Super-Admin-Password': superAdminPassword.value
})

const init = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请输入超级管理员密码')
    return
  }
  sessionStorage.setItem(PASSWORD_KEY, superAdminPassword.value)
  await loadKeys()
  await Promise.all([loadReports(), loadAlerts()])
}

const createKey = async () => {
  if (!form.value.name.trim()) {
    ElMessage.error('名称必填')
    return
  }
  creating.value = true
  try {
    const res = await fetch(`${API_BASE}/admin/keys`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({
        ...form.value,
        super_admin_password: superAdminPassword.value,
        allowed_models: modelsText.value.split('\n').map(item => item.trim()).filter(Boolean)
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')
    createdPlainKey.value = data.plain_key
    ElMessage.success('API Key 已生成')
    await Promise.all([loadKeys(), loadAlerts()])
  } catch (err) {
    ElMessage.error(err.message || '生成失败')
  } finally {
    creating.value = false
  }
}

const loadReports = async () => {
  const params = new URLSearchParams({
    group_by: reportGroupBy.value,
    days: String(reportDays.value)
  })
  const res = await fetch(`${API_BASE}/admin/reports?${params.toString()}`, {
    headers: authHeaders()
  })
  const data = await res.json()
  if (res.ok) reports.value = data.rows || []
}

const loadAlerts = async () => {
  const res = await fetch(`${API_BASE}/admin/alerts`, {
    headers: { 'X-Super-Admin-Password': superAdminPassword.value }
  })
  const data = await res.json()
  if (res.ok) alerts.value = data.alerts || []
}

const loadKeys = async () => {
  loadingKeys.value = true
  try {
    const res = await fetch(`${API_BASE}/admin/keys`, {
      headers: authHeaders()
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    keys.value = data.keys || []
  } catch (err) {
    ElMessage.error(err.message || '加载失败')
  } finally {
    loadingKeys.value = false
  }
}

const viewKey = async (row) => {
  const res = await fetch(`${API_BASE}/admin/keys/${row.id}`, {
    headers: { 'X-Super-Admin-Password': superAdminPassword.value }
  })
  const data = await res.json()
  if (!res.ok) {
    ElMessage.error(data.error || '加载详情失败')
    return
  }
  currentKey.value = data.key
  currentLogs.value = data.logs || []
  detailVisible.value = true
}

const revokeKey = async (row) => {
  await ElMessageBox.confirm(`确认吊销 ${row.name} ?`, '提示', { type: 'warning' })
  const res = await fetch(`${API_BASE}/admin/keys/${row.id}/revoke`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ super_admin_password: superAdminPassword.value })
  })
  const data = await res.json()
  if (!res.ok) {
    ElMessage.error(data.error || '吊销失败')
    return
  }
  ElMessage.success('已吊销')
  loadKeys()
}

const loadDocs = async (showDialog = true) => {
  const res = await fetch(`${API_BASE}/docs`)
  const data = await res.json()
  docs.value = data
  if (showDialog) {
    docsVisible.value = true
  }
}

const loadAnthropicDocs = async () => {
  const res = await fetch(`${API_BASE}/docs/anthropic`)
  const data = await res.json()
  // 替换文档中的占位符为当前域名
  const docStr = JSON.stringify(data)
  const replaced = docStr.replace(/your-devtools:8080/g, window.location.host)
  anthropicDocs.value = JSON.parse(replaced)
  anthropicDocsVisible.value = true
}

const copyPlainKey = async () => {
  await navigator.clipboard.writeText(createdPlainKey.value)
  ElMessage.success('已复制')
}

const gatewayCurlExample = `curl -X POST ${origin}/api/ai-gateway/v1/chat/completions \\
  -H "Authorization: Bearer ai_xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "deepseek-chat",
    "messages": [
      { "role": "system", "content": "你是一个业务助手" },
      { "role": "user", "content": "帮我生成一段活动宣传文案" }
    ]
  }'`

const gatewayJsExample = `const response = await fetch("${origin}/api/ai-gateway/v1/chat/completions", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer ai_xxx"
  },
  body: JSON.stringify({
    model: "deepseek-chat",
    messages: [
      { role: "user", content: "输出一份周报摘要" }
    ]
  })
})

const data = await response.json()
console.log(data.usage_summary)
console.log(data.choices?.[0]?.message?.content)`

const gatewayPythonExample = `import requests

resp = requests.post(
    "${origin}/api/ai-gateway/v1/chat/completions",
    headers={
        "Authorization": "Bearer ai_xxx",
        "Content-Type": "application/json",
    },
    json={
        "model": "deepseek-chat",
        "messages": [
            {"role": "user", "content": "生成 3 条商品卖点"}
        ],
    },
    timeout=120,
)
data = resp.json()
print(data.get("usage_summary"))
print(data["choices"][0]["message"]["content"])`

const gatewayMediaExample = `curl -X POST ${origin}/api/ai-gateway/v1/media/generations \\
  -H "Authorization: Bearer ai_xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "qwen-image-2.0-pro",
    "prompt": "一张北欧客厅海报，柔和自然光",
    "images": [],
    "parameters": {
      "size": "1328x1328",
      "watermark": false
    }
  }'

curl ${origin}/api/ai-gateway/v1/media/tasks/task_xxx \\
  -H "Authorization: Bearer ai_xxx"`

const gatewayResponseExample = `{
  "id": "chatcmpl_xxx",
  "model": "deepseek-chat",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "这是生成结果"
      }
    }
  ],
  "usage_summary": {
    "input_tokens": 123,
    "output_tokens": 456,
    "total_tokens": 579,
    "estimated_cost": 0.0347,
    "currency": "CNY"
  }
}`

const gatewayFieldDocs = [
  { field: 'model', required: '是', description: '要调用的模型名，必须在当前 API Key 白名单内。', example: 'deepseek-chat' },
  { field: 'messages', required: '聊天必填', description: '聊天消息数组，兼容 OpenAI 风格。', example: '[{"role":"user","content":"你好"}]' },
  { field: 'prompt', required: '媒体必填', description: '图片或视频生成提示词。', example: '北欧风客厅海报' },
  { field: 'images', required: '按模型', description: '媒体模型输入图，支持 base64 或公网 URL。', example: '["data:image/png;base64,..."]' },
  { field: 'parameters', required: '否', description: '模型高级参数，原样透传给对应提供商。', example: '{"size":"1328x1328"}' },
  { field: 'Authorization', required: '是', description: '统一鉴权头，格式为 Bearer API Key。', example: 'Bearer ai_xxx' },
  { field: 'usage_summary', required: '响应', description: '网关追加的统一计费信息，便于业务记账。', example: '{"total_tokens":579}' }
]

const gatewayErrorDocs = [
  { code: '400', scene: '参数错误', meaning: '缺少 model、messages、prompt 等必要字段。', fix: '先按字段说明补齐必填项，并确认 JSON 格式合法。' },
  { code: '401', scene: '鉴权失败', meaning: 'API Key 缺失、错误、被吊销或已过期。', fix: '确认 Authorization 头是否正确，必要时重新签发 Key。' },
  { code: '403', scene: '权限不足', meaning: '当前 Key 没有访问对应 scope 或模型。', fix: '检查 Key 的 allowed_scopes 和 allowed_models 配置。' },
  { code: '429', scene: '触发限流', meaning: '超过每小时请求限制。', fix: '降低调用频率，或提高该 Key 的 rate limit。' },
  { code: '500', scene: '上游异常', meaning: '模型供应商返回错误或网关内部调用失败。', fix: '查看请求明细里的 error_message 和 latency，结合日志排查。' }
]

const gatewayFaqs = [
  { q: 'API Key 应该放在哪里？', a: '放在服务端配置或密钥中心，不要直接放到浏览器页面、App 安装包或公开仓库里。' },
  { q: '如何统计每个业务方的费用？', a: '建议每个业务系统单独签发一个 Key，这样后台会自动按 Key 统计请求数、Token、费用和报表。' },
  { q: '为什么有些模型没有 Token？', a: '图片和视频模型通常按次计费，没有文本 Token 概念，网关会写入 request_cost 并累计 total_cost。' },
  { q: '如何排查某次调用失败？', a: '在 Key 详情里查看最近请求明细，重点看状态码、错误信息、模型名、耗时和费用字段。' }
]

const prettyJSON = (value) => JSON.stringify(value, null, 2)
const formatTime = (value) => value ? new Date(value).toLocaleString() : '-'
const formatCost = (value, currency = 'CNY') => `${currency} ${Number(value || 0).toFixed(4)}`

onMounted(() => {
  loadDocs(false)
  loadCatalog()
})
</script>

<style scoped>
.ai-gateway-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.hero {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 24px;
  border-radius: 22px;
  background: linear-gradient(135deg, #142033 0%, #365f5f 55%, #c88044 100%);
  color: #fff;
}

.hero h2 {
  margin: 0 0 10px;
}

.hero-actions {
  display: grid;
  grid-template-columns: minmax(220px, 280px);
  gap: 10px;
}

.card-header,
.plain-key-box {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.plain-key-box code,
.doc-json {
  display: block;
  padding: 14px;
  background: #0f1724;
  color: #dce7f7;
  border-radius: 14px;
  white-space: pre-wrap;
  word-break: break-all;
}

.logs-section {
  margin-top: 18px;
}

.docs-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.docs-card {
  border-radius: 22px;
}

.docs-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.docs-panel h4,
.docs-api-list h4 {
  margin: 0 0 12px;
}

.docs-text {
  margin: 0;
  color: #475569;
  line-height: 1.75;
}

.docs-list {
  margin: 0;
  padding-left: 18px;
  color: #475569;
  line-height: 1.8;
}

.docs-tabs,
.docs-api-list {
  margin-top: 18px;
}

.docs-panel-stack h4 {
  margin: 0 0 12px;
}

.docs-steps {
  margin: 0;
  padding-left: 18px;
  color: #475569;
  line-height: 1.9;
}

.faq-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.faq-item h4 {
  margin: 0 0 8px;
}

.faq-item p {
  margin: 0;
  color: #475569;
  line-height: 1.8;
}

.test-card {
  border-radius: 22px;
}

.test-reply {
  color: #334155;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.test-err {
  color: #dc2626;
  font-size: 12px;
}

@media (max-width: 900px) {
  .hero {
    flex-direction: column;
  }

  .docs-grid {
    grid-template-columns: 1fr;
  }
}
</style>
