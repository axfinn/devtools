<template>
  <div class="askit-mydata">
    <!-- 未登录:邮箱验证码登录 -->
    <el-card v-if="!loggedIn" class="login-card">
      <template #header>
        <div class="card-header">
          <span>AskIt 云同步 — 查看我的数据</span>
        </div>
      </template>
      <el-form label-width="84px" @submit.prevent>
        <el-form-item label="邮箱">
          <el-input v-model="email" type="email" placeholder="你的登录邮箱"
            :disabled="codeSent" @keyup.enter="sendCode" />
        </el-form-item>
        <el-form-item v-if="codeSent" label="验证码">
          <div class="code-row">
            <el-input v-model="code" placeholder="6 位验证码" maxlength="6"
              style="width:160px" @keyup.enter="login" />
            <el-button text :disabled="cooldown > 0" @click="sendCode">
              {{ cooldown > 0 ? `${cooldown}s 后重发` : '重新发送' }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item v-if="codeSent && needInvite" label="邀请码">
          <el-input v-model="inviteCode" placeholder="仅首次注册需要" style="width:240px" />
        </el-form-item>
        <el-form-item>
          <el-button v-if="!codeSent" type="primary" :loading="loading" @click="sendCode">发送验证码</el-button>
          <el-button v-else type="primary" :loading="loading" @click="login">登录</el-button>
        </el-form-item>
      </el-form>
      <div class="tip">
        用与扩展相同的邮箱登录,即可在此查看你同步到云端的全部数据。
        登录态仅保存在本机浏览器,密钥等敏感字段端到端加密,服务器与本页都无法解密。
      </div>
    </el-card>

    <!-- 已登录:数据总览 -->
    <template v-else>
      <el-card class="header-card" shadow="never">
        <div class="me-bar">
          <div class="me-info">
            <div class="me-avatar">{{ (me.email || '?').slice(0, 1).toUpperCase() }}</div>
            <div class="me-text">
              <span class="me-email">{{ me.email }}</span>
              <span class="me-meta">服务器版本 v{{ serverVersion }} · 注册于 {{ fmtTime(me.createdAt) }}</span>
            </div>
          </div>
          <div class="me-actions">
            <el-tooltip content="切换 Markdown 渲染 / 查看原文">
              <el-switch v-model="showRaw" inline-prompt active-text="原文" inactive-text="渲染" />
            </el-tooltip>
            <el-button size="small" :icon="Refresh" :loading="dataLoading" @click="loadSnapshot">刷新</el-button>
            <el-button size="small" :icon="Download" @click="exportJson">导出</el-button>
            <el-button size="small" text type="danger" @click="logout">退出</el-button>
          </div>
        </div>
        <div class="stat-row">
          <button v-for="c in cards" :key="c.key" class="stat" :class="{ active: activeTab === c.key }" @click="activeTab = c.key">
            <component :is="c.icon" class="stat-icon" />
            <div class="stat-text">
              <div class="stat-num">{{ c.count }}</div>
              <div class="stat-label">{{ c.label }}</div>
            </div>
          </button>
        </div>
      </el-card>

      <el-card class="body-card" shadow="never" v-loading="dataLoading">
        <!-- 对话 -->
        <div v-if="activeTab === 'conversations'">
          <el-empty v-if="!conversations.length" description="暂无对话" />
          <el-collapse v-else v-model="openConvs">
            <el-collapse-item v-for="conv in conversations" :key="conv.id" :name="conv.id">
              <template #title>
                <div class="row-title">
                  <el-icon class="row-icon"><ChatDotRound /></el-icon>
                  <span class="row-main">{{ conv.title || '未命名对话' }}</span>
                  <span class="row-sub">{{ conv.messageCount }} 条 · {{ fmtTime(conv.updatedAt) }}</span>
                </div>
              </template>
              <a v-if="conv.pageUrl" class="conv-src" :href="conv.pageUrl" target="_blank" rel="noopener">
                <el-icon><Link /></el-icon>{{ conv.pageTitle || conv.pageUrl }}
              </a>
              <div class="chat">
                <div v-for="m in conv.messages" :key="m.id" class="bubble-row" :class="m.role">
                  <div class="avatar" :class="m.role">{{ roleLabel(m.role) }}</div>
                  <div class="bubble">
                    <pre v-if="showRaw" class="raw">{{ m.content }}</pre>
                    <div v-else class="md" v-html="renderMarkdown(m.content)"></div>
                    <el-image v-if="m.imageUrl" :src="m.imageUrl" :preview-src-list="[m.imageUrl]"
                      fit="contain" class="bubble-img" hide-on-click-modal preview-teleported />
                  </div>
                </div>
              </div>
              <el-empty v-if="!conv.messages.length" :image-size="50" description="该对话仅同步了元数据" />
            </el-collapse-item>
          </el-collapse>
        </div>

        <!-- 笔记 -->
        <div v-else-if="activeTab === 'notes'" class="grid">
          <el-empty v-if="!notes.length" description="暂无笔记" />
          <el-card v-for="n in notes" :key="n.id" class="item-card" shadow="hover" :body-style="{ padding: '12px' }">
            <el-image v-if="n.image" :src="n.image" :preview-src-list="[n.image]" fit="cover"
              class="card-img" hide-on-click-modal preview-teleported />
            <pre v-if="showRaw" class="raw note-content">{{ n.content }}</pre>
            <div v-else class="md note-content" v-html="renderMarkdown(n.content)"></div>
            <div class="card-foot">
              <a v-if="n.source" :href="n.source" target="_blank" rel="noopener" class="src-link">{{ n.sourceTitle || n.source }}</a>
              <span class="time">{{ fmtTime(n.createdAt) }}</span>
            </div>
            <div v-if="n.tags && n.tags.length" class="tags">
              <el-tag v-for="t in n.tags" :key="t" size="small" type="info" effect="plain">{{ t }}</el-tag>
            </div>
          </el-card>
        </div>

        <!-- 书签 -->
        <div v-else-if="activeTab === 'bookmarks'" class="grid">
          <el-empty v-if="!bookmarks.length" description="暂无书签" />
          <el-card v-for="b in bookmarks" :key="b.id" class="item-card" shadow="hover" :body-style="{ padding: '12px' }">
            <div class="bm-head">
              <img v-if="b.favicon" :src="b.favicon" class="bm-favicon" />
              <el-icon v-else class="bm-favicon-fallback"><Link /></el-icon>
              <a :href="b.url" target="_blank" rel="noopener" class="bm-title">{{ b.title || b.url }}</a>
            </div>
            <div v-if="b.summary || b.note" class="bm-note">{{ b.summary || b.note }}</div>
            <div class="card-foot">
              <span class="time">{{ fmtTime(b.createdAt) }}</span>
            </div>
            <div v-if="b.tags && b.tags.length" class="tags">
              <el-tag v-for="t in b.tags" :key="t" size="small" type="info" effect="plain">{{ t }}</el-tag>
            </div>
          </el-card>
        </div>

        <!-- 分享 -->
        <div v-else-if="activeTab === 'shares'">
          <el-empty v-if="!shares.length" description="暂无分享" />
          <el-table v-else :data="shares" size="small" stripe>
            <el-table-column label="标题" min-width="200" show-overflow-tooltip>
              <template #default="{ row }"><a :href="row.url" target="_blank" rel="noopener" class="src-link">{{ row.title || row.url }}</a></template>
            </el-table-column>
            <el-table-column label="类型" width="90" align="center">
              <template #default="{ row }"><el-tag size="small" effect="plain">{{ row.kind }}</el-tag></template>
            </el-table-column>
            <el-table-column label="创建时间" width="160" align="center">
              <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 提示词 -->
        <div v-else-if="activeTab === 'prompts'" class="grid">
          <el-empty v-if="!prompts.length" description="暂无自定义提示词" />
          <el-card v-for="p in prompts" :key="p.id" class="item-card" shadow="hover" :body-style="{ padding: '12px' }">
            <div class="prompt-title"><el-icon><MagicStick /></el-icon>{{ p.title }}</div>
            <pre class="raw prompt-body">{{ p.prompt }}</pre>
            <div class="card-foot"><span class="time">{{ fmtTime(p.createdAt) }}</span></div>
          </el-card>
        </div>

        <!-- 设置 -->
        <div v-else-if="activeTab === 'settings'">
          <el-empty v-if="!settings" description="暂无同步的设置" />
          <template v-else>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item v-for="(val, key) in flatSettings" :key="key" :label="key">
                <span class="setting-val">{{ val }}</span>
              </el-descriptions-item>
            </el-descriptions>
          </template>
          <div class="tip">API Key 等密钥字段从不上云,此处不会出现。</div>
        </div>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Refresh, Download, ChatDotRound, Link, MagicStick,
  Document, Star, Share, Setting,
} from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'
import hljs from '../../utils/highlight'

const API = '/api/askit/v1'
const TOKEN_KEY = 'askit_mydata_access'
const REFRESH_KEY = 'askit_mydata_refresh'

// html:false —— 不放行同步内容里的原始 HTML,从源头避免 v-html XSS。
const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(code, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try { return hljs.highlight(code, { language: lang }).value } catch {}
    }
    return ''
  },
})
function renderMarkdown(text) { return text ? md.render(text) : '' }
const showRaw = ref(false)

const email = ref('')
const code = ref('')
const inviteCode = ref('')
const codeSent = ref(false)
const needInvite = ref(false)
const cooldown = ref(0)
let cooldownTimer = null

const loading = ref(false)
const loggedIn = ref(false)
const me = ref({ email: '', createdAt: 0 })

const dataLoading = ref(false)
const serverVersion = ref(0)
const collections = ref({})
const activeTab = ref('conversations')
const openConvs = ref([])

// ── 集合解析:snapshot 的 collections[col] 是 [{id,updatedAt,deleted,data}] ──
function recs(col) {
  const arr = collections.value[col]
  if (!Array.isArray(arr)) return []
  return arr.filter(r => r && !r.deleted)
}

const conversations = computed(() =>
  recs('conversations')
    .map(r => {
      const meta = r.data?.meta || {}
      const conv = r.data?.conv || {}
      return {
        id: r.id,
        title: meta.title || conv.title || '',
        updatedAt: meta.updatedAt || r.updatedAt,
        messageCount: meta.messageCount ?? (conv.messages?.length || 0),
        pageUrl: meta.pageUrl,
        pageTitle: meta.pageTitle,
        messages: Array.isArray(conv.messages) ? conv.messages : [],
      }
    })
    .sort((a, b) => b.updatedAt - a.updatedAt)
)
const notes = computed(() => recs('notes').map(r => r.data).sort((a, b) => b.createdAt - a.createdAt))
const bookmarks = computed(() => recs('bookmarks').map(r => r.data).sort((a, b) => b.createdAt - a.createdAt))
const shares = computed(() => recs('shares').map(r => r.data).sort((a, b) => b.createdAt - a.createdAt))
const prompts = computed(() => recs('prompts').map(r => r.data).sort((a, b) => b.createdAt - a.createdAt))
const settings = computed(() => recs('settings')[0]?.data || null)

// settings 打平成「键: 值」表格;对象/数组转 JSON 单行,布尔/数字直显。
const flatSettings = computed(() => {
  const s = settings.value
  if (!s || typeof s !== 'object') return {}
  const out = {}
  for (const [k, v] of Object.entries(s)) {
    if (v === null || v === undefined || v === '') continue
    out[k] = typeof v === 'object' ? JSON.stringify(v) : String(v)
  }
  return out
})

const cards = computed(() => [
  { key: 'conversations', label: '对话', count: conversations.value.length, icon: ChatDotRound },
  { key: 'notes', label: '笔记', count: notes.value.length, icon: Document },
  { key: 'bookmarks', label: '书签', count: bookmarks.value.length, icon: Star },
  { key: 'shares', label: '分享', count: shares.value.length, icon: Share },
  { key: 'prompts', label: '提示词', count: prompts.value.length, icon: MagicStick },
  { key: 'settings', label: '设置', count: settings.value ? 1 : 0, icon: Setting },
])

// ── 工具 ──────────────────────────────────────────────────────
function fmtTime(ms) {
  if (!ms) return '—'
  const d = new Date(ms)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}
function roleLabel(role) {
  return role === 'user' ? '我' : role === 'assistant' ? 'AI' : '系统'
}
function startCooldown() {
  cooldown.value = 60
  cooldownTimer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0) { clearInterval(cooldownTimer); cooldownTimer = null }
  }, 1000)
}

// ── 登录流程 ──────────────────────────────────────────────────
async function sendCode() {
  if (!email.value.trim()) { ElMessage.warning('请输入邮箱'); return }
  if (cooldown.value > 0) return
  loading.value = true
  try {
    const r = await fetch(`${API}/auth/request-code`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email.value.trim(), inviteCode: inviteCode.value.trim() }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      const map = {
        invalid_email: '邮箱格式不正确',
        registration_closed: '该邮箱未注册,且当前未开放注册',
        invalid_invite_code: '邀请码无效或已过期',
      }
      if (data.error === 'invalid_invite_code' || data.error === 'registration_closed') needInvite.value = true
      ElMessage.error(map[data.error] || `发送失败（${data.error || r.status}）`)
      return
    }
    codeSent.value = true
    startCooldown()
    ElMessage.success(data.emailSent === false ? '验证码已生成(邮件未配置,请联系管理员)' : '验证码已发送到邮箱')
  } catch {
    ElMessage.error('请求失败')
  } finally {
    loading.value = false
  }
}

async function login() {
  if (!code.value.trim()) { ElMessage.warning('请输入验证码'); return }
  loading.value = true
  try {
    const r = await fetch(`${API}/auth/login-code`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email.value.trim(), code: code.value.trim(), inviteCode: inviteCode.value.trim() }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      const map = { invalid_code: '验证码错误或已过期', registration_closed: '当前未开放注册', invalid_invite_code: '邀请码无效' }
      ElMessage.error(map[data.error] || `登录失败（${data.error || r.status}）`)
      return
    }
    localStorage.setItem(TOKEN_KEY, data.accessToken)
    if (data.refreshToken) localStorage.setItem(REFRESH_KEY, data.refreshToken)
    await afterLogin()
  } catch {
    ElMessage.error('请求失败')
  } finally {
    loading.value = false
  }
}

function authHeaders() {
  return { Authorization: `Bearer ${localStorage.getItem(TOKEN_KEY) || ''}` }
}

async function tryRefresh() {
  const refresh = localStorage.getItem(REFRESH_KEY)
  if (!refresh) return false
  try {
    const r = await fetch(`${API}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: refresh }),
    })
    if (!r.ok) return false
    const data = await r.json()
    if (data.accessToken) { localStorage.setItem(TOKEN_KEY, data.accessToken); return true }
  } catch {}
  return false
}

async function afterLogin() {
  const ok = await fetchMe()
  if (!ok) return
  loggedIn.value = true
  await loadSnapshot()
}

async function fetchMe(allowRefresh = true) {
  try {
    const r = await fetch(`${API}/auth/me`, { headers: authHeaders() })
    if (r.status === 401 && allowRefresh && (await tryRefresh())) return fetchMe(false)
    if (!r.ok) { clearSession(); return false }
    me.value = await r.json()
    return true
  } catch {
    return false
  }
}

async function loadSnapshot(allowRefresh = true) {
  dataLoading.value = true
  try {
    const r = await fetch(`${API}/sync/snapshot`, { headers: authHeaders() })
    if (r.status === 401 && allowRefresh && (await tryRefresh())) { dataLoading.value = false; return loadSnapshot(false) }
    if (!r.ok) { ElMessage.error('加载失败,请重新登录'); clearSession(); return }
    const data = await r.json()
    collections.value = data.collections || {}
    serverVersion.value = data.serverVersion || 0
  } catch {
    ElMessage.error('请求失败')
  } finally {
    dataLoading.value = false
  }
}

function exportJson() {
  const blob = new Blob([JSON.stringify({ serverVersion: serverVersion.value, collections: collections.value }, null, 2)], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `askit-data-${me.value.email || 'export'}.json`
  a.click()
  URL.revokeObjectURL(a.href)
}

function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_KEY)
  loggedIn.value = false
  collections.value = {}
}

async function logout() {
  try { await fetch(`${API}/auth/logout`, { method: 'POST', headers: authHeaders() }) } catch {}
  clearSession()
  codeSent.value = false
  code.value = ''
  ElMessage.success('已退出登录')
}

onMounted(() => {
  if (localStorage.getItem(TOKEN_KEY)) afterLogin()
})
onUnmounted(() => { if (cooldownTimer) clearInterval(cooldownTimer) })
</script>

<style scoped>
.askit-mydata { max-width: 900px; margin: 0 auto; padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; font-weight: 600; }
.tip { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.6; margin-top: 10px; }
.code-row { display: flex; align-items: center; gap: 8px; }
.login-card { max-width: 460px; margin: 8px auto; }

/* 头部用户条 */
.me-bar { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.me-info { display: flex; align-items: center; gap: 12px; }
.me-avatar { width: 40px; height: 40px; border-radius: 50%; flex-shrink: 0; display: flex; align-items: center;
  justify-content: center; font-size: 18px; font-weight: 700; color: #fff;
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-light-3)); }
.me-text { display: flex; flex-direction: column; gap: 2px; }
.me-email { font-weight: 600; font-size: 15px; }
.me-meta { font-size: 12px; color: var(--el-text-color-secondary); }
.me-actions { display: flex; gap: 8px; align-items: center; }

/* 统计卡 */
.stat-row { display: grid; grid-template-columns: repeat(6, 1fr); gap: 10px; margin-top: 18px; }
.stat { display: flex; align-items: center; gap: 8px; padding: 12px; border-radius: 10px;
  background: var(--el-fill-color-light); cursor: pointer; transition: all .15s;
  border: 1px solid transparent; font-family: inherit; }
.stat:hover { background: var(--el-fill-color); transform: translateY(-1px); }
.stat.active { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.stat-icon { width: 22px; height: 22px; color: var(--el-color-primary); flex-shrink: 0; }
.stat-text { text-align: left; line-height: 1.1; }
.stat-num { font-size: 20px; font-weight: 700; }
.stat-label { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; }
@media (max-width: 640px) { .stat-row { grid-template-columns: repeat(3, 1fr); } .stat-label { font-size: 11px; } }

/* 对话标题行 */
.row-title { display: flex; align-items: center; gap: 8px; width: 100%; padding-right: 12px; }
.row-icon { color: var(--el-color-primary); }
.row-main { font-weight: 500; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.row-sub { font-size: 12px; color: var(--el-text-color-secondary); flex-shrink: 0; }
.conv-src { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; color: var(--el-color-primary);
  margin-bottom: 12px; word-break: break-all; text-decoration: none; }

/* 聊天气泡 */
.chat { display: flex; flex-direction: column; gap: 14px; padding: 4px 0; }
.bubble-row { display: flex; gap: 10px; align-items: flex-start; }
.bubble-row.user { flex-direction: row-reverse; }
.avatar { width: 30px; height: 30px; border-radius: 50%; flex-shrink: 0; display: flex; align-items: center;
  justify-content: center; font-size: 12px; font-weight: 600; color: #fff; background: var(--el-color-info); }
.avatar.user { background: var(--el-color-primary); }
.avatar.assistant { background: var(--el-color-success); }
.bubble { max-width: 80%; padding: 10px 14px; border-radius: 12px; background: var(--el-fill-color-light);
  font-size: 14px; line-height: 1.7; }
.bubble-row.user .bubble { background: var(--el-color-primary-light-9); border-top-right-radius: 4px; }
.bubble-row.assistant .bubble { border-top-left-radius: 4px; }
.bubble-img { max-width: 240px; max-height: 200px; border-radius: 8px; margin-top: 8px; display: block; }

/* Markdown 渲染区 */
.md { word-break: break-word; }
.md :deep(p) { margin: 0 0 8px; }
.md :deep(p:last-child) { margin-bottom: 0; }
.md :deep(pre) { background: var(--el-fill-color-darker); padding: 10px 12px; border-radius: 8px;
  overflow-x: auto; font-size: 13px; margin: 8px 0; }
.md :deep(code) { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.92em; }
.md :deep(:not(pre) > code) { background: var(--el-fill-color); padding: 1px 5px; border-radius: 4px; }
.md :deep(ul), .md :deep(ol) { padding-left: 22px; margin: 6px 0; }
.md :deep(a) { color: var(--el-color-primary); }
.md :deep(blockquote) { border-left: 3px solid var(--el-border-color); padding-left: 10px; margin: 8px 0;
  color: var(--el-text-color-secondary); }
.md :deep(table) { border-collapse: collapse; margin: 8px 0; }
.md :deep(th), .md :deep(td) { border: 1px solid var(--el-border-color); padding: 4px 8px; }
.md :deep(img) { max-width: 100%; border-radius: 6px; }

/* 原文 pre */
.raw { white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px; line-height: 1.6; margin: 0; }

/* 卡片网格 */
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 14px; }
.grid .el-empty { grid-column: 1 / -1; }
.item-card { border-radius: 10px; }
.card-img { width: 100%; height: 140px; border-radius: 8px; margin-bottom: 8px; display: block; cursor: zoom-in; }
.card-foot { display: flex; justify-content: space-between; align-items: center; gap: 8px; margin-top: 8px; }
.time { font-size: 11px; color: var(--el-text-color-secondary); flex-shrink: 0; }
.src-link { font-size: 12px; color: var(--el-color-primary); text-decoration: none; overflow: hidden;
  text-overflow: ellipsis; white-space: nowrap; }
.tags { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 8px; }
.note-content { max-height: 220px; overflow-y: auto; }

/* 书签 */
.bm-head { display: flex; align-items: center; gap: 8px; }
.bm-favicon { width: 20px; height: 20px; border-radius: 4px; flex-shrink: 0; }
.bm-favicon-fallback { color: var(--el-text-color-secondary); flex-shrink: 0; }
.bm-title { font-size: 14px; font-weight: 500; color: var(--el-text-color-primary); text-decoration: none;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bm-title:hover { color: var(--el-color-primary); }
.bm-note { font-size: 12px; color: var(--el-text-color-regular); margin-top: 6px; line-height: 1.5;
  display: -webkit-box; -webkit-line-clamp: 3; line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }

/* 提示词 */
.prompt-title { display: flex; align-items: center; gap: 6px; font-weight: 600; font-size: 14px; }
.prompt-body { margin: 8px 0; color: var(--el-text-color-regular); max-height: 200px; overflow-y: auto;
  background: var(--el-fill-color-lighter); padding: 8px 10px; border-radius: 6px; }

/* 设置 */
.setting-val { word-break: break-all; font-size: 13px; }
</style>
