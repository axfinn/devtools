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
      <el-card class="header-card">
        <div class="me-bar">
          <div class="me-info">
            <span class="me-email">{{ me.email }}</span>
            <span class="me-meta">服务器版本 v{{ serverVersion }} · 注册于 {{ fmtTime(me.createdAt) }}</span>
          </div>
          <div class="me-actions">
            <el-button size="small" :loading="dataLoading" @click="loadSnapshot">刷新</el-button>
            <el-button size="small" @click="exportJson">导出 JSON</el-button>
            <el-button size="small" text @click="logout">退出登录</el-button>
          </div>
        </div>
        <div class="stat-row">
          <div v-for="c in cards" :key="c.key" class="stat" :class="{ active: activeTab === c.key }" @click="activeTab = c.key">
            <div class="stat-num">{{ c.count }}</div>
            <div class="stat-label">{{ c.label }}</div>
          </div>
        </div>
      </el-card>
      <el-card class="body-card" v-loading="dataLoading">
        <!-- 对话 -->
        <div v-if="activeTab === 'conversations'">
          <el-empty v-if="!conversations.length" description="暂无对话" />
          <el-collapse v-else accordion>
            <el-collapse-item v-for="conv in conversations" :key="conv.id" :name="conv.id">
              <template #title>
                <div class="row-title">
                  <span class="row-main">{{ conv.title || '未命名对话' }}</span>
                  <span class="row-sub">{{ conv.messageCount }} 条 · {{ fmtTime(conv.updatedAt) }}</span>
                </div>
              </template>
              <a v-if="conv.pageUrl" class="conv-src" :href="conv.pageUrl" target="_blank" rel="noopener">{{ conv.pageTitle || conv.pageUrl }}</a>
              <div v-for="m in conv.messages" :key="m.id" class="msg" :class="m.role">
                <div class="msg-role">{{ roleLabel(m.role) }}</div>
                <div class="msg-content">{{ m.content }}</div>
                <img v-if="m.imageUrl" :src="m.imageUrl" class="msg-img" />
              </div>
              <el-empty v-if="!conv.messages.length" :image-size="60" description="该对话仅同步了元数据" />
            </el-collapse-item>
          </el-collapse>
        </div>

        <!-- 笔记 -->
        <div v-else-if="activeTab === 'notes'">
          <el-empty v-if="!notes.length" description="暂无笔记" />
          <div v-for="n in notes" :key="n.id" class="note-item">
            <img v-if="n.image" :src="n.image" class="note-img" />
            <div class="note-body">
              <div class="note-content">{{ n.content }}</div>
              <div class="note-foot">
                <a v-if="n.source" :href="n.source" target="_blank" rel="noopener" class="note-src">{{ n.sourceTitle || n.source }}</a>
                <span class="note-time">{{ fmtTime(n.createdAt) }}</span>
                <el-tag v-for="t in (n.tags || [])" :key="t" size="small" type="info">{{ t }}</el-tag>
              </div>
            </div>
          </div>
        </div>

        <!-- 书签 -->
        <div v-else-if="activeTab === 'bookmarks'">
          <el-empty v-if="!bookmarks.length" description="暂无书签" />
          <div v-for="b in bookmarks" :key="b.id" class="bm-item">
            <img v-if="b.favicon" :src="b.favicon" class="bm-favicon" />
            <div class="bm-body">
              <a :href="b.url" target="_blank" rel="noopener" class="bm-title">{{ b.title || b.url }}</a>
              <div v-if="b.summary || b.note" class="bm-note">{{ b.summary || b.note }}</div>
              <div class="bm-foot">
                <span class="bm-time">{{ fmtTime(b.createdAt) }}</span>
                <el-tag v-for="t in (b.tags || [])" :key="t" size="small" type="info">{{ t }}</el-tag>
              </div>
            </div>
          </div>
        </div>

        <!-- 分享 -->
        <div v-else-if="activeTab === 'shares'">
          <el-empty v-if="!shares.length" description="暂无分享" />
          <el-table v-else :data="shares" size="small">
            <el-table-column label="标题" min-width="160" show-overflow-tooltip>
              <template #default="{ row }"><a :href="row.url" target="_blank" rel="noopener">{{ row.title || row.url }}</a></template>
            </el-table-column>
            <el-table-column label="类型" width="90">
              <template #default="{ row }"><el-tag size="small">{{ row.kind }}</el-tag></template>
            </el-table-column>
            <el-table-column label="创建时间" width="150">
              <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 提示词 -->
        <div v-else-if="activeTab === 'prompts'">
          <el-empty v-if="!prompts.length" description="暂无自定义提示词" />
          <div v-for="p in prompts" :key="p.id" class="prompt-item">
            <div class="prompt-title">{{ p.title }}</div>
            <div class="prompt-body">{{ p.prompt }}</div>
            <div class="prompt-time">{{ fmtTime(p.createdAt) }}</div>
          </div>
        </div>

        <!-- 设置 -->
        <div v-else-if="activeTab === 'settings'">
          <el-empty v-if="!settings" description="暂无同步的设置" />
          <pre v-else class="settings-json">{{ prettySettings }}</pre>
          <div class="tip">API Key 等密钥字段从不上云,此处不会出现。</div>
        </div>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

const API = '/api/askit/v1'
const TOKEN_KEY = 'askit_mydata_access'
const REFRESH_KEY = 'askit_mydata_refresh'

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

// SCRIPT_PLACEHOLDER

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
const prettySettings = computed(() => settings.value ? JSON.stringify(settings.value, null, 2) : '')

const cards = computed(() => [
  { key: 'conversations', label: '对话', count: conversations.value.length },
  { key: 'notes', label: '笔记', count: notes.value.length },
  { key: 'bookmarks', label: '书签', count: bookmarks.value.length },
  { key: 'shares', label: '分享', count: shares.value.length },
  { key: 'prompts', label: '提示词', count: prompts.value.length },
  { key: 'settings', label: '设置', count: settings.value ? 1 : 0 },
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
.askit-mydata { max-width: 860px; margin: 0 auto; padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.tip { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.6; margin-top: 8px; }
.code-row { display: flex; align-items: center; gap: 8px; }

.me-bar { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.me-info { display: flex; flex-direction: column; gap: 2px; }
.me-email { font-weight: 600; font-size: 15px; }
.me-meta { font-size: 12px; color: var(--el-text-color-secondary); }
.me-actions { display: flex; gap: 6px; align-items: center; }

.stat-row { display: flex; gap: 10px; margin-top: 16px; flex-wrap: wrap; }
.stat { flex: 1; min-width: 96px; padding: 10px 12px; border-radius: 8px; background: var(--el-fill-color-light);
  cursor: pointer; text-align: center; transition: all .15s; border: 1px solid transparent; }
.stat:hover { background: var(--el-fill-color); }
.stat.active { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.stat-num { font-size: 22px; font-weight: 700; line-height: 1.1; }
.stat-label { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; }
/* STYLE_PLACEHOLDER */

.row-title { display: flex; justify-content: space-between; align-items: center; width: 100%; padding-right: 12px; }
.row-main { font-weight: 500; }
.row-sub { font-size: 12px; color: var(--el-text-color-secondary); }
.conv-src { display: block; font-size: 12px; color: var(--el-color-primary); margin-bottom: 8px; word-break: break-all; }

.msg { padding: 8px 10px; border-radius: 8px; margin-bottom: 8px; background: var(--el-fill-color-lighter); }
.msg.user { background: var(--el-color-primary-light-9); }
.msg-role { font-size: 11px; color: var(--el-text-color-secondary); margin-bottom: 2px; }
.msg-content { white-space: pre-wrap; word-break: break-word; font-size: 13px; line-height: 1.6; }
.msg-img { max-width: 220px; max-height: 160px; border-radius: 6px; margin-top: 6px; object-fit: contain; }

.note-item, .bm-item { display: flex; gap: 10px; padding: 10px 0; border-bottom: 1px solid var(--el-border-color-lighter); }
.note-img { width: 64px; height: 64px; object-fit: cover; border-radius: 6px; flex-shrink: 0; }
.note-body, .bm-body { flex: 1; min-width: 0; }
.note-content { white-space: pre-wrap; word-break: break-word; font-size: 13px; line-height: 1.6; }
.note-foot, .bm-foot { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-top: 6px; }
.note-src, .bm-title { font-size: 12px; color: var(--el-color-primary); word-break: break-all; }
.note-time, .bm-time { font-size: 11px; color: var(--el-text-color-secondary); }

.bm-favicon { width: 20px; height: 20px; border-radius: 4px; flex-shrink: 0; margin-top: 2px; }
.bm-title { font-size: 14px; font-weight: 500; display: block; }
.bm-note { font-size: 12px; color: var(--el-text-color-regular); margin-top: 2px; line-height: 1.5; }

.prompt-item { padding: 10px 0; border-bottom: 1px solid var(--el-border-color-lighter); }
.prompt-title { font-weight: 600; font-size: 14px; }
.prompt-body { white-space: pre-wrap; word-break: break-word; font-size: 13px; color: var(--el-text-color-regular); margin: 4px 0; line-height: 1.6; }
.prompt-time { font-size: 11px; color: var(--el-text-color-secondary); }

.settings-json { background: var(--el-fill-color-light); padding: 12px; border-radius: 8px; font-size: 12px;
  line-height: 1.6; overflow-x: auto; white-space: pre-wrap; word-break: break-word; }
</style>
