<template>
  <div class="skills-tool tool-container">
    <div v-if="!ready" class="loading">
      <el-card shadow="hover">
        <template #header>
          <div class="card-header">
            <span><el-icon><Connection /></el-icon> Skills 工具入口</span>
          </div>
        </template>
        <div class="hero">
          <div class="hero-title">正在加载 Skill 清单…</div>
          <div class="hero-subtitle">{{ statusMessage }}</div>
        </div>
      </el-card>
    </div>

    <div v-else class="main-content">
      <el-card class="hero-card">
        <template #header>
          <div class="card-header">
            <span><el-icon><Connection /></el-icon> Skills 工具入口</span>
            <div class="header-actions">
              <el-tag size="small" type="info">{{ tools.length }} tools</el-tag>
              <el-button size="small" @click="loadManifest" :loading="loadingManifest">刷新清单</el-button>
            </div>
          </div>
        </template>
        <div class="hero">
          <div class="hero-title">把 DevTools 基础能力开放给 codex / Claude Code / Cursor 等外部 AI</div>
          <div class="hero-subtitle">
            支持 OpenAI function-calling 风格清单({{ serverInfo.protocol }}),无鉴权。
            默认未启用 — 部署方需在 config.yaml 设 skills.enabled=true 才生效。
          </div>
        </div>
      </el-card>

      <div class="grid-2">
        <el-card shadow="hover" class="skill-list-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><List /></el-icon> 可用 Skill</span>
              <el-input
                v-model="searchKeyword"
                size="small"
                placeholder="搜索 name / description"
                clearable
                class="search-input"
              />
            </div>
          </template>
          <div class="skill-grid">
            <div
              v-for="tool in filteredTools"
              :key="tool.name"
              class="skill-card"
              :class="{ active: selectedSkill && selectedSkill.name === tool.name }"
              @click="selectSkill(tool)"
            >
              <div class="skill-name">
                <code>{{ tool.name }}</code>
                <el-tag :type="tool.risk === 'write' ? 'danger' : 'success'" size="small" effect="plain">
                  {{ tool.risk }}
                </el-tag>
              </div>
              <div class="skill-desc">{{ tool.description }}</div>
              <div v-if="tool.input_schema && tool.input_schema.required && tool.input_schema.required.length" class="skill-required">
                必填: {{ tool.input_schema.required.join(', ') }}
              </div>
            </div>
          </div>
        </el-card>

        <el-card shadow="hover" class="install-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Promotion /></el-icon> 一键安装片段</span>
            </div>
          </template>

          <div class="install-section">
            <div class="section-title">
              <span>OpenAI tools 风格清单</span>
              <el-button size="small" @click="copyText(openAIToolsJSON, 'OpenAI tools JSON')">复制</el-button>
            </div>
            <pre class="code-block"><code>{{ openAIToolsJSON }}</code></pre>
            <div class="hint">OpenAI / Cursor / Continue.dev 等按 <code>tools</code> 字段注入。</div>
          </div>

          <el-divider />

          <div class="install-section">
            <div class="section-title">
              <span>MCP Streamable HTTP 配置</span>
              <el-button size="small" @click="copyText(mcpConfig, 'MCP 配置')">复制</el-button>
            </div>
            <pre class="code-block"><code>{{ mcpConfig }}</code></pre>
            <div class="hint">
              粘贴到 <code>~/.claude.json</code> 的 <code>mcpServers</code> 里,
              或用 <code>claude mcp add --transport http devtools {{ serverEndpoint }}</code> 一键注册。
            </div>
          </div>

          <el-divider />

          <div class="install-section">
            <div class="section-title">
              <span>各客户端一键安装命令</span>
              <el-button size="small" @click="copyText(installLines, '安装命令')">复制全部</el-button>
            </div>
            <pre class="code-block"><code>{{ installLines }}</code></pre>
            <div class="hint">
              也可以 <code>curl -s {{ serverEndpoint }}/install</code> 直接拿后端实时生成的纯文本,
              或 <code>curl {{ serverEndpoint }}/install.sh | bash</code> 一键执行。
            </div>
          </div>

          <el-divider />

          <div class="install-section" v-if="selectedSkill">
            <div class="section-title">
              <span>
                <el-icon><Promotion /></el-icon>
                <code>{{ selectedSkill.name }}</code> 调用模板
              </span>
              <el-button size="small" @click="copyText(skillCurl, 'curl 模板')">复制</el-button>
            </div>
            <pre class="code-block"><code>{{ skillCurl }}</code></pre>
          </div>
        </el-card>
      </div>

      <el-card shadow="hover" class="probe-card">
        <template #header>
          <div class="card-header">
            <span><el-icon><Position /></el-icon> 实时试调</span>
            <span class="muted">
              POST {{ serverEndpoint }}/invoke
              <span class="dot">·</span>
              选定 skill → 填 arguments → 调
            </span>
          </div>
        </template>

        <div v-if="!selectedSkill" class="probe-empty">
          从上方选一个 Skill 开始试调。
        </div>

        <div v-else class="probe-form">
          <el-form :inline="true" size="default" label-position="top">
            <el-form-item
              v-for="prop in formFields"
              :key="prop.name"
              :label="prop.name + (prop.required ? ' *' : '')"
            >
              <el-input
                v-if="prop.type === 'string' && !prop.enum"
                v-model="formState[prop.name]"
                :type="prop.long ? 'textarea' : 'text'"
                :placeholder="prop.description || prop.name"
                :rows="prop.long ? 4 : undefined"
                style="min-width: 240px"
              />
              <el-select
                v-else-if="prop.enum"
                v-model="formState[prop.name]"
                :placeholder="prop.description || prop.name"
                style="min-width: 200px"
              >
                <el-option v-for="e in prop.enum" :key="e" :label="e" :value="e" />
              </el-select>
              <el-input-number
                v-else-if="prop.type === 'number'"
                v-model.number="formState[prop.name]"
                style="min-width: 200px"
              />
            </el-form-item>
          </el-form>

          <div class="probe-actions">
            <el-button type="primary" :loading="probing" @click="probeSkill">试调</el-button>
            <el-button @click="fillSample" v-if="hasSample">填入示例</el-button>
          </div>

          <div v-if="probeResult" class="probe-result" :class="{ ok: probeResult.ok === true }">
            <pre><code>{{ JSON.stringify(probeResult, null, 2) }}</code></pre>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

const serverEndpoint = '/api/skills'

const ready = ref(false)
const loadingManifest = ref(false)
const statusMessage = ref('正在请求 /api/skills/manifest ...')
const tools = ref([])
const serverInfo = ref({ protocol: 'openai-tools + mcp-streamable-http' })
const searchKeyword = ref('')
const selectedSkill = ref(null)
const probing = ref(false)
const probeResult = ref(null)

const formState = reactive({})
const formFields = computed(() => {
  if (!selectedSkill.value) return []
  const props = (selectedSkill.value.input_schema && selectedSkill.value.input_schema.properties) || {}
  const required = (selectedSkill.value.input_schema && selectedSkill.value.input_schema.required) || []
  return Object.entries(props).map(([name, sch]) => {
    const type = Array.isArray(sch.type) ? sch.type[0] : (sch.type || 'string')
    return {
      name,
      type,
      required: required.includes(name),
      description: sch.description || '',
      enum: Array.isArray(sch.enum) ? sch.enum : null,
      long: type === 'string' && name !== 'text',
    }
  })
})

const filteredTools = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  if (!kw) return tools.value
  return tools.value.filter(t =>
    t.name.includes(kw) || (t.description || '').toLowerCase().includes(kw)
  )
})

const openAIToolsJSON = computed(() => JSON.stringify(
  { tools: tools.value },
  null, 2
))

const mcpConfig = computed(() => JSON.stringify(
  {
    mcpServers: {
      devtools: {
        type: 'http',
        url: `${window.location.origin}${serverEndpoint}/mcp`,
      },
    },
  },
  null, 2
))

// 各客户端的一键安装命令(纯文本,直接复制粘贴即可)
const installLines = computed(() => {
  const base = window.location.origin
  return [
    `# === Claude Code ===`,
    `claude mcp add --transport http devtools ${base}/api/skills/mcp`,
    ``,
    `# === OpenAI Codex CLI ===`,
    `codex mcp add devtools --url ${base}/api/skills/mcp`,
    ``,
    `# === Cursor (~/.cursor/mcp.json) ===`,
    `# 把下面的 JSON 写进 ~/.cursor/mcp.json:`,
    mcpConfig.value,
    ``,
    `# === VS Code (Copilot Chat) ===`,
    `code --add-mcp '{"name":"devtools","url":"${base}/api/skills/mcp"}'`,
    ``,
    `# === 纯 cURL(任意 HTTP 客户端) ===`,
    `curl -s ${base}/api/skills/manifest`,
    ``,
    `# === 完整安装脚本(piped to bash) ===`,
    `curl -s ${base}/api/skills/install.sh | bash`,
    ``,
    `# === AI agent 自发现 ===`,
    `curl -s ${base}/.well-known/skills`,
  ].join('\n')
})

const skillCurl = computed(() => {
  if (!selectedSkill.value) return ''
  const argsObj = {}
  for (const p of formFields.value) {
    if (formState[p.name] !== '' && formState[p.name] !== undefined && formState[p.name] !== null) {
      argsObj[p.name] = formState[p.name]
    }
  }
  const body = { name: selectedSkill.value.name, arguments: argsObj }
  return `curl -X POST ${serverEndpoint}/invoke \\
  -H 'Content-Type: application/json' \\
  -d '${JSON.stringify(body).replace(/'/g, "'\\''")}'`
})

const hasSample = computed(() => {
  if (!selectedSkill.value) return false
  return Object.keys(selectedSkill.value.input_schema?.properties || {}).length > 0
})

async function loadManifest() {
  loadingManifest.value = true
  statusMessage.value = '正在请求 /api/skills/manifest ...'
  try {
    const res = await fetch(`${serverEndpoint}/manifest`, { credentials: 'omit' })
    if (res.status === 404) {
      ready.value = true
      tools.value = []
      statusMessage.value = '/api/skills/manifest 返回 404 —— 后端 skills.enabled=false 或路由未开启。'
      return
    }
    if (!res.ok) throw new Error('HTTP ' + res.status)
    const data = await res.json()
    serverInfo.value = data.server || serverInfo.value
    tools.value = data.tools || []
    ready.value = true
    statusMessage.value = ''
  } catch (e) {
    statusMessage.value = '加载失败: ' + e.message + '。请确认 config.yaml 已设 skills.enabled=true。'
    ready.value = true
  } finally {
    loadingManifest.value = false
  }
}

function selectSkill(tool) {
  selectedSkill.value = tool
  probeResult.value = null
  for (const p of formFields.value) {
    formState[p.name] = p.enum ? p.enum[0] : p.type === 'number' ? 0 : ''
  }
}

async function probeSkill() {
  if (!selectedSkill.value) return
  probing.value = true
  probeResult.value = null
  try {
    const argsObj = {}
    for (const p of formFields.value) {
      const v = formState[p.name]
      if (p.required && (v === '' || v === null || v === undefined)) {
        ElMessage.warning(`缺少必填参数 ${p.name}`)
        probing.value = false
        return
      }
      if (v !== '' && v !== null && v !== undefined) argsObj[p.name] = v
    }
    const res = await fetch(`${serverEndpoint}/invoke`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: selectedSkill.value.name, arguments: argsObj }),
    })
    const data = await res.json()
    probeResult.value = data
  } catch (e) {
    probeResult.value = { ok: false, error: e.message }
  } finally {
    probing.value = false
  }
}

function fillSample() {
  if (!selectedSkill.value) return
  const samples = {
    base64_encode: { text: 'hello world' },
    base64_decode: { text: 'aGVsbG8gd29ybGQ=' },
    url_encode: { text: 'https://example.com/?q=hello world' },
    url_decode: { text: 'https%3A%2F%2Fexample.com%2F%3Fq%3Dhello+world' },
    json_format: { text: '{"a":1,"b":[2,3],"c":{"d":4}}' },
    json_validate: { text: '{"a":1}' },
    timestamp_to_date: { ts: Date.now(), unit: 'ms' },
    date_to_timestamp: { iso: new Date().toISOString() },
    uuid_v4: {},
    hash_sha256: { text: 'abc' },
    regex_test: { pattern: '(\\w+)@([\\w.]+)', input: 'bob@example.com', flags: '' },
    ip_lookup: {},
    shorturl_create: { url: 'https://t.jaxiu.cn' },
    paste_create: { content: 'Hello from skills paste_create!', language: 'text' },
  }
  const sample = samples[selectedSkill.value.name]
  if (!sample) return
  for (const k of Object.keys(formState)) formState[k] = ''
  for (const [k, v] of Object.entries(sample)) formState[k] = v
}

async function copyText(text, label) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(label + ' 已复制')
  } catch (e) {
    ElMessage.error('复制失败: ' + e.message)
  }
}

onMounted(() => {
  loadManifest()
})
</script>

<style scoped>
.skills-tool {
  max-width: 1280px;
  margin: 0 auto;
  padding: 16px;
}

.hero-card {
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
  gap: 12px;
  flex-wrap: wrap;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.search-input {
  width: 280px;
}

.hero-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
}

.hero-subtitle {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1.2fr;
  gap: 16px;
  margin-bottom: 16px;
}

@media (max-width: 960px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}

.skill-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 580px;
  overflow-y: auto;
}

.skill-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.skill-card:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 1px 6px rgba(64, 158, 255, 0.15);
}

.skill-card.active {
  border-color: var(--el-color-primary);
  background: rgba(64, 158, 255, 0.06);
}

.skill-name {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  font-weight: 600;
}

.skill-name code {
  font-family: 'Fira Code', 'SF Mono', Consolas, monospace;
  font-size: 13px;
}

.skill-desc {
  font-size: 12.5px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
  margin-bottom: 4px;
}

.skill-required {
  font-size: 11.5px;
  color: var(--el-text-color-secondary);
}

.install-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
  font-size: 14px;
  gap: 8px;
}

.code-block {
  background: #f7f7f9;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
  font-family: 'Fira Code', 'SF Mono', Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre;
  margin: 0;
}

.code-block code {
  background: none;
  padding: 0;
  color: inherit;
}

.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.hint code {
  background: rgba(64, 158, 255, 0.08);
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11.5px;
}

.probe-card {
  margin-bottom: 24px;
}

.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
}

.dot {
  margin: 0 6px;
}

.probe-empty {
  text-align: center;
  color: var(--el-text-color-secondary);
  padding: 40px 16px;
  font-size: 13px;
}

.probe-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.probe-actions {
  display: flex;
  gap: 12px;
}

.probe-result {
  border-radius: 6px;
  padding: 12px;
  font-family: 'Fira Code', 'SF Mono', Consolas, monospace;
  font-size: 12px;
  background: #fafafa;
  border: 1px solid var(--el-border-color-lighter);
  max-height: 360px;
  overflow: auto;
}

.probe-result.ok {
  border-color: #67c23a;
  background: rgba(103, 194, 58, 0.05);
}

.probe-result pre,
.probe-result pre code {
  margin: 0;
  background: none;
  padding: 0;
}

.loading,
.status-message {
  padding: 40px 16px;
  text-align: center;
}
</style>
